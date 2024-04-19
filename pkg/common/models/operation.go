// Package models
/*
 使用fileloader.Model管理接口配置
 读取store/operation下的文件，支持多级目录，支持逻辑删除，变更后会触发引擎编译
 批量插入时读取额外参数即graphql文本
 支持多级目录，返回树状结构，并在子节点添加额外属性
 Invalid, Internal, OperationType, AuthorizationConfig由graphql编译时动态设置，准确时启动引擎时读取编译生成的配置后设置
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"net/http"
	"strconv"
	"sync"
)

type Operation struct {
	Enabled    bool   `json:"enabled"`
	Title      string `json:"title"`
	Remark     string `json:"remark"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
	DeleteTime string `json:"deleteTime"`

	Path               string                            `json:"path"`
	Engine             wgpb.OperationExecutionEngine     `json:"engine"`
	HooksConfiguration *wgpb.OperationHooksConfiguration `json:"hooksConfiguration"`
	RateLimit          *wgpb.OperationRateLimit          `json:"rateLimit"`
	Semaphore          *wgpb.OperationSemaphore          `json:"semaphore"`

	ConfigCustomized     bool                                `json:"configCustomized"`
	CacheConfig          *wgpb.OperationCacheConfig          `json:"cacheConfig"`
	LiveQueryConfig      *wgpb.OperationLiveQueryConfig      `json:"liveQueryConfig"`
	AuthenticationConfig *wgpb.OperationAuthenticationConfig `json:"authenticationConfig"`

	Invalid             bool                               `json:"-"`
	Internal            bool                               `json:"-"`
	OperationType       wgpb.OperationType                 `json:"-"`
	AuthorizationConfig *wgpb.OperationAuthorizationConfig `json:"-"`
}

type operationExtra struct {
	Enabled          bool                          `json:"enabled"`
	Invalid          bool                          `json:"invalid"`
	Internal         bool                          `json:"internal"`
	LiveQueryEnabled bool                          `json:"liveQueryEnabled"`
	Method           string                        `json:"method"`
	OperationType    wgpb.OperationType            `json:"operationType"`
	Engine           wgpb.OperationExecutionEngine `json:"engine"`
}

const fieldOriginContent = "originContent"

var (
	OperationRoot      *fileloader.Model[Operation]
	OperationMethodMap map[wgpb.OperationType]string
	operationResultMap sync.Map
)

func StoreOperationResult(path string, result *wgpb.Operation) {
	operationResultMap.Store(path, result)
}

func LoadOperationResult(path string) *wgpb.Operation {
	value, ok := operationResultMap.Load(path)
	if !ok {
		return nil
	}

	return value.(*wgpb.Operation)
}

func init() {
	OperationMethodMap = map[wgpb.OperationType]string{
		wgpb.OperationType_QUERY:        http.MethodGet,
		wgpb.OperationType_MUTATION:     http.MethodPost,
		wgpb.OperationType_SUBSCRIPTION: http.MethodGet,
	}

	OperationRoot = &fileloader.Model[Operation]{
		Root:                  utils.NormalizePath(consts.RootStore, consts.StoreOperationParent),
		Extension:             fileloader.ExtJson,
		InsertBatchExtraField: fieldOriginContent,
		DataHook: &fileloader.DataHook[Operation]{
			OnInsert: func(item *Operation) error {
				item.CreateTime = utils.TimeFormatNow()
				return nil
			},
			OnUpdate: func(_, dst *Operation, user string) error {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				return nil
			},
			AfterInsert: func(item *Operation, user string) bool { return item.Enabled },
			AfterBatchInsert: func(datas []*Operation, _ string, extraBytes ...[]byte) bool {
				var enabledCount int
				for i, data := range datas {
					extraQuoteStr, _ := strconv.Unquote(`"` + string(extraBytes[i]) + `"`)
					_ = OperationGraphql.Write(data.Path, fileloader.SystemUser, []byte(extraQuoteStr))
					if data.Enabled {
						enabledCount++
					}
				}
				return enabledCount > 0
			},
		},
		DataRW: &fileloader.MultipleDataRW[Operation]{
			GetDataName: func(item *Operation) string { return item.Path },
			SetDataName: func(item *Operation, name string) { item.Path = name },
			Filter:      func(item *Operation) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Operation) { item.DeleteTime = utils.TimeFormatNow() },
		},
		DataTreeExtra: func(item *Operation) any {
			return &operationExtra{
				Enabled:          item.Enabled,
				Internal:         item.Internal,
				Invalid:          item.Invalid,
				LiveQueryEnabled: item.LiveQueryConfig != nil && item.LiveQueryConfig.Enabled,
				Method:           OperationMethodMap[item.OperationType],
				OperationType:    item.OperationType,
				Engine:           item.Engine,
			}
		},
	}
	utils.RegisterInitMethod(20, func() {
		OperationRoot.DataRW.(*fileloader.MultipleDataRW[Operation]).MergeDataBytes = globalOperationDefaultText.GetFirstCache()
		OperationRoot.Init()
		configs.AddFileLoaderQuestionCollector(OperationRoot.GetModelName(), func(dataName string) map[string]any {
			data, _ := OperationRoot.GetByDataName(dataName)
			if data == nil {
				return nil
			}

			data.Invalid = true
			return map[string]any{
				fieldEnabled: data.Enabled,
				"engine":     data.Engine,
			}
		})
		utils.AddBuildAndStartFuncWatcher(func(f func()) { OperationRoot.DataHook.AfterMutate = f })
	})
}
