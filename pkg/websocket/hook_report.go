// Package websocket
/*
 钩子服务信息websocket实现
 钩子健康检查接口返回proxy和function类型operation定义
 function参数定义通过反射成schema实现
 graphql数据源通过内省自身获取响应并缓存，钩子负责存储，飞布负责加载
*/
package websocket

import (
	"context"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/hooks"
	"github.com/wundergraph/wundergraph/pkg/pool"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const hookReportChannel configs.WsChannel = "hookReport"

type (
	Health struct {
		Status  string       `json:"status"`
		Report  healthReport `json:"report"`
		Workdir string       `json:"workdir,omitempty"`
	}
	healthReport struct {
		Customizes []string  `json:"customizes"`
		Functions  []string  `json:"functions"`
		Proxys     []string  `json:"proxys"`
		Time       time.Time `json:"time"`
	}
	hookReportInfo struct {
		Time   time.Time `json:"time"`
		Status int       `json:"status"`

		workdir  string
		logger   *zap.Logger
		ctx      context.Context
		client   *hooks.Client
		ticker   *time.Ticker
		interval time.Duration
		mutex    sync.Mutex
	}
)

var hookReport hookReportInfo

func init() {
	configs.WsMsgHandlerMap[hookReportChannel] = func(msg *configs.WsMsgBody) any {
		switch msg.Event {
		case configs.PullEvent:
			return &hookReport
		}
		return nil
	}

	configs.AddCollector(&configs.LogCollector{
		MatchLevel:    []zapcore.Level{zap.InfoLevel},
		IdentifyField: consts.HookReportTime,
		Handle: func(entry zapcore.Entry, value *zapcore.Field, fieldMap map[string]*zapcore.Field) *configs.WsMsgBody {
			return &configs.WsMsgBody{
				Channel: hookReportChannel,
				Event:   configs.PushEvent,
				Data:    &hookReport,
			}
		},
	})

	utils.RegisterInitMethod(40, func() {
		hookReport = hookReportInfo{
			Time:     time.Now(),
			ctx:      context.Background(),
			logger:   zap.L(),
			client:   hooks.NewHealthClient(zap.L()),
			interval: time.Second,
		}
		hookReport.workdir, _ = os.Getwd()
		hookReport.ticker = time.NewTicker(hookReport.interval)
		AddOnFirstStartedHook(hookReport.timingReport, math.MaxInt, true)
	})
}

func (r *hookReportInfo) printReport(extras ...zap.Field) {
	fields := []zap.Field{zap.Time(consts.HookReportTime, r.Time), zap.Int(consts.HookReportStatus, r.Status)}
	fields = append(fields, extras...)
	r.logger.Info("health report changed", fields...)
}

func (r *hookReportInfo) timingReport() {
	isFirstReport := true
	for range r.ticker.C {
		r.report(isFirstReport)
		isFirstReport = false
	}
}

func (r *hookReportInfo) resetTicker(interval time.Duration) {
	if r.interval != interval {
		r.interval = interval
		r.ticker.Reset(interval)
	}
}

func (r *hookReportInfo) report(isFirstReport bool) {
	var restartInvoked bool
	r.mutex.Lock()
	defer func() {
		if !restartInvoked {
			r.mutex.Unlock()
		}
	}()
	if models.GetEnabledServerSdk() == nil || !utils.EngineStarted() {
		r.resetTicker(5 * time.Second)
		return
	}

	var report healthReport
	buf := pool.GetBytesBuffer()
	defer pool.PutBytesBuffer(buf)
	enableHookReport := utils.GetBoolWithLockViper(consts.EnableHookReport)
	r.client.ResetServerUrl(models.GetHookServerUrl())
	r.client.ResetHealthQuery(map[string]interface{}{consts.EnableHookReport: enableHookReport})
	// 调用钩子的健康检查接口
	if r.client.DoHealthCheckRequest(r.ctx, buf) {
		var health Health
		_ = json.Unmarshal(buf.Bytes(), &health)
		if len(health.Workdir) > 0 && !strings.HasPrefix(health.Workdir, r.workdir) {
			return
		}

		r.resetTicker(10 * time.Second)
		report, r.Status = health.Report, http.StatusOK
	} else {
		r.resetTicker(time.Second)
		if r.Status == http.StatusOK {
			report.Time, r.Status = time.Now(), http.StatusInternalServerError
		} else {
			report.Time = r.Time
		}
	}

	reportChanged := report.Time.After(r.Time)
	if reportChanged || isFirstReport {
		r.Time = report.Time
	}
	if enableHookReport {
		// 仅有dev模式会触发热重启
		var affectCount int
		affectCount += migrateCustomizes(report.Customizes)
		affectCount += migrateOperations(report.Functions, wgpb.OperationExecutionEngine_ENGINE_FUNCTION, consts.HookFunctionParent)
		affectCount += migrateOperations(report.Proxys, wgpb.OperationExecutionEngine_ENGINE_PROXY, consts.HookProxyParent)
		if restartInvoked = affectCount > 0 || reportChanged; restartInvoked {
			AddOnEveryStartedHook(func() {
				r.printReport(zap.Bool("restartInvoked", true))
				r.resetTicker(time.Second)
				r.mutex.Unlock()
			})
			go utils.BuildAndStart()
			return
		}
	}
	if reportChanged || isFirstReport {
		r.printReport()
	}
}

func migrateCustomizes(customizes []string) (affectCount int) {
	filter := func(item *models.Datasource) bool {
		return item.Kind == wgpb.DataSourceKind_GRAPHQL && item.CustomGraphql != nil && item.CustomGraphql.Customized
	}
	modifyEnableFunc := func(item *models.Datasource, enabled bool) bool {
		if item.Enabled == enabled {
			return true
		}

		item.Enabled = enabled
		return false
	}
	buildDataFunc := func(name string) *models.Datasource {
		return &models.Datasource{
			Name:          name,
			Enabled:       true,
			Kind:          wgpb.DataSourceKind_GRAPHQL,
			CustomGraphql: &models.CustomGraphql{Customized: true},
		}
	}
	return migrateHealthReportData(customizes, models.DatasourceRoot, filter, modifyEnableFunc, buildDataFunc)
}

func migrateOperations(operations []string, engine wgpb.OperationExecutionEngine, prefix string) (affectCount int) {
	filter := func(item *models.Operation) bool {
		return item.Engine == engine
	}
	modifyEnableFunc := func(item *models.Operation, enabled bool) bool {
		if item.Enabled == enabled {
			return true
		}

		item.Enabled = enabled
		return false
	}
	buildDataFunc := func(name string) *models.Operation {
		return &models.Operation{
			Path:    name,
			Enabled: true,
			Engine:  engine,
		}
	}
	var operationsWithPrefix []string
	for _, item := range operations {
		if item != "" {
			operationsWithPrefix = append(operationsWithPrefix, utils.NormalizePath(prefix, item))
		}
	}
	return migrateHealthReportData(operationsWithPrefix, models.OperationRoot, filter, modifyEnableFunc, buildDataFunc)
}

func migrateHealthReportData[T any](datas []string, modelRoot *fileloader.Model[T], filter func(*T) bool,
	modifyEnableFunc func(*T, bool) bool, buildDataFunc func(string) *T) (affectCount int) {
	modelName := modelRoot.GetModelName()
	existArray := modelRoot.ListByCondition(filter)
	var err error
	for _, item := range existArray {
		itemName := modelRoot.GetDataName(item)
		itemEnabled := slices.Contains(datas, itemName)
		if itemEnabled {
			index := slices.Index(datas, itemName)
			datas = slices.Delete(datas, index, index+1)
		}

		if modifyEnableFunc(item, itemEnabled) {
			continue
		}

		if err = modelRoot.InsertOrUpdate(item); err != nil {
			continue
		}

		logger.Info("health report data modified", zap.String(modelName, itemName), zap.Bool("existed", itemEnabled))
		affectCount++
	}

	for _, item := range datas {
		if err = modelRoot.ExistedDataNameThrowError(item); err != nil || item == "" {
			logger.Warn("health report data failed", zap.String(modelName, item), zap.Error(err))
			continue
		}

		if err = modelRoot.InsertOrUpdate(buildDataFunc(item)); err != nil {
			logger.Warn("health report data failed", zap.String(modelName, item), zap.Error(err))
			continue
		}

		logger.Info("health report data added", zap.String(modelName, item))
		affectCount++
	}
	return
}
