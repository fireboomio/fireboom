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
)

func init() {
	configs.WsMsgHandlerMap[hookReportChannel] = func(msg *configs.WsMsgBody) any {
		switch msg.Event {
		case configs.PullEvent:
			return utils.GetTimeWithLockViper(consts.HookReportTime)
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
				Data:    utils.GetTimeWithLockViper(consts.HookReportTime),
			}
		},
	})

	utils.RegisterInitMethod(40, func() {
		hookClient := hooks.NewHealthClient(zap.L())
		workdir, _ := os.Getwd()
		AddOnFirstStartedFunc(func() {
			var (
				reportInterval int
				reportMutex    sync.Mutex
				reportStatus   int
				reportTime     time.Time
				restartInvoked bool
			)
			reportPrinter := func() {
				logger.Info("health report changed",
					zap.Int(consts.HookReportStatus, reportStatus),
					zap.Time(consts.HookReportTime, reportTime),
					zap.Bool("restartInvoked", restartInvoked))
			}
			reportTicker, reportCtx := time.NewTicker(time.Second), context.Background()
			for range reportTicker.C {
				reportMutex.Lock()
				if models.GetEnabledServerSdk() == nil {
					reportMutex.Unlock()
					reportInterval = resetTicker(reportTicker, reportInterval, 5)
					continue
				}
				var report healthReport
				buf := pool.GetBytesBuffer()
				hookClient.ResetServerUrl(models.GetHookServerUrl())
				// 调用钩子的健康检查接口
				if hookClient.DoHealthCheckRequest(reportCtx, buf) {
					var health Health
					_ = json.Unmarshal(buf.Bytes(), &health)
					if len(health.Workdir) >= 0 && !strings.HasPrefix(health.Workdir, workdir) {
						reportMutex.Unlock()
						continue
					}
					report, reportStatus = health.Report, http.StatusOK
					reportInterval = resetTicker(reportTicker, reportInterval, 10)
				} else {
					report.Time = reportTime
					if reportStatus == http.StatusOK {
						report.Time, reportStatus = time.Now(), http.StatusInternalServerError
					}
					reportInterval = resetTicker(reportTicker, reportInterval, 1)
				}
				pool.PutBytesBuffer(buf)

				restartInvoked = false
				if utils.GetBoolWithLockViper(consts.EnableHookReport) {
					// 仅有dev模式会触发热重启
					var affectCount int
					affectCount += migrateCustomizes(report.Customizes)
					affectCount += migrateOperations(report.Functions, wgpb.OperationExecutionEngine_ENGINE_FUNCTION, consts.HookFunctionParent)
					affectCount += migrateOperations(report.Proxys, wgpb.OperationExecutionEngine_ENGINE_PROXY, consts.HookProxyParent)
					if affectCount > 0 || report.Time.After(reportTime) {
						restartInvoked = true
						AddOnEveryStartedFunc(reportPrinter)
						go utils.BuildAndStart()
					}
				}
				if !reportTime.Equal(report.Time) {
					reportTime = report.Time
					utils.SetWithLockViper(consts.HookReportTime, reportTime)
					if !restartInvoked {
						reportPrinter()
					}
				}
				reportMutex.Unlock()
			}
		})
	})
}

func resetTicker(ticker *time.Ticker, now, expected int) int {
	if now != expected {
		ticker.Reset(time.Second * time.Duration(expected))
	}
	return expected
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
