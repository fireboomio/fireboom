// Package datasource
/*
 数据源内省的通用代码，需要实现内省、编译引擎数据源配置、运行时解析数据源配置
*/
package datasource

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"sync"
)

type Action interface {
	Introspect() (string, error)
	BuildDataSourceConfiguration(*ast.SchemaDocument) (*wgpb.DataSourceConfiguration, error)
	RuntimeDataSourceConfiguration(*wgpb.DataSourceConfiguration) ([]*wgpb.DataSourceConfiguration, []*wgpb.FieldConfiguration, error)
}

type ActionExtend interface {
	ExtendDocument(*ast.SchemaDocument)
	GetFieldRealName(string) string
}

type FieldConfigurationAction interface {
	// Handle 实现特殊的字段添加逻辑
	Handle(*wgpb.FieldConfiguration)
}

// GetDatasourceAction 根据数据源类型返回不同的Action实现
// prismaSchema 默认是读取数据源的配置，不为空时使用此参数，具体需要实现方自定义处理
func GetDatasourceAction(ds *models.Datasource, prismaSchema ...string) (action Action, err error) {
	actionFunc, ok := actionMap[ds.Kind]
	if !ok {
		err = i18n.NewCustomErrorWithMode(datasourceModelName, nil, i18n.DatasourceKindNotSupportedError, ds.Kind)
		return
	}

	var content string
	if len(prismaSchema) > 0 && len(prismaSchema[0]) > 0 {
		content = prismaSchema[0]
	}
	action = actionFunc(ds, content)
	return
}

func cacheGraphqlSchema(dsName string, graphqlSchema string) {
	go func() { _ = CacheGraphqlSchemaText.Write(dsName, fileloader.SystemUser, []byte(graphqlSchema)) }()
}

// 将数据源拆解成多个，组合方式为一个rootNode+其引用的childNodes
// 有jsonField的childNode需要额外添加字段
// 从在编译期保存的根字段引用解析出实际引用的字段定义
func copyDatasourceWithRootNodes(config *wgpb.DataSourceConfiguration, copyPostFunc func(*wgpb.TypeField, *wgpb.DataSourceConfiguration) bool) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration) {
	emptyArgs, emptyRequires := make([]*wgpb.ArgumentConfiguration, 0), make([]string, 0)
	emptyCustomStatic := &wgpb.DataSourceCustom_Static{Data: utils.MakeStaticVariable("{}")}
	joinModifyFunc := func(childItem *wgpb.TypeField, fieldName string, fieldNames []string) (joinIndex int) {
		if joinIndex = slices.Index(childItem.FieldNames, fieldName); joinIndex == -1 {
			return
		}

		configs = append(configs, &wgpb.DataSourceConfiguration{
			Kind: wgpb.DataSourceKind_STATIC,
			RootNodes: []*wgpb.TypeField{{
				TypeName:   childItem.TypeName,
				FieldNames: fieldNames,
			}},
			CustomStatic: emptyCustomStatic,
		})
		fields = append(fields, &wgpb.FieldConfiguration{
			TypeName:                   childItem.TypeName,
			FieldName:                  fieldName,
			Path:                       fieldNames,
			DisableDefaultFieldMapping: true,
			ArgumentsConfiguration:     emptyArgs,
			RequiresFields:             emptyRequires,
		})
		return
	}
	clearJoinChildNodes := make([]*wgpb.TypeField, len(config.ChildNodes))
	for i, item := range config.ChildNodes {
		joinIndexes := make([]int, 0, len(joinFieldMap))
		for name, fieldNames := range joinFieldMap {
			if index := joinModifyFunc(item, name, fieldNames); index != -1 {
				joinIndexes = append(joinIndexes, index)
			}
		}
		if len(joinIndexes) == 0 {
			clearJoinChildNodes[i] = item
			continue
		}

		clearItem := &wgpb.TypeField{
			TypeName:   item.TypeName,
			FieldNames: make([]string, 0, len(item.FieldNames)-len(joinIndexes)),
		}
		for index := range item.FieldNames {
			if !slices.Contains(joinIndexes, index) {
				clearItem.FieldNames = append(clearItem.FieldNames, item.FieldNames[index])
			}
		}
		clearJoinChildNodes[i] = clearItem
	}

	for _, rootItem := range config.RootNodes {
		for index, fieldName := range rootItem.FieldNames {
			copyRootItem := &wgpb.TypeField{
				TypeName:   rootItem.TypeName,
				FieldNames: []string{fieldName},
			}
			destCopy := &wgpb.DataSourceConfiguration{
				Kind:                       config.Kind,
				KindForPrisma:              config.KindForPrisma,
				RootNodes:                  []*wgpb.TypeField{copyRootItem},
				OverrideFieldPathFromAlias: config.OverrideFieldPathFromAlias,
				Directives:                 config.Directives,
				RequestTimeoutSeconds:      config.RequestTimeoutSeconds,
				Id:                         config.Id,
			}
			if quotes, ok := rootItem.Quotes[int32(index)]; ok {
				for _, quoteIndex := range quotes.Indexes {
					destCopy.ChildNodes = append(destCopy.ChildNodes, clearJoinChildNodes[quoteIndex])
				}
			}
			if copyPostFunc(copyRootItem, destCopy) {
				configs = append(configs, destCopy)
			}
		}
	}
	return
}

const (
	JoinFieldName         = "_join"
	JoinMutationFieldName = "_join_mutation"
)

var (
	joinFieldMap = map[string][]string{
		JoinFieldName:         {JoinFieldName},
		JoinMutationFieldName: {JoinMutationFieldName},
	}
	datasourceModelName string
	logger              *zap.Logger
	actionMap           = make(map[wgpb.DataSourceKind]func(*models.Datasource, string) Action)
)

func init() {
	utils.RegisterInitMethod(30, func() {
		logger = zap.L()
		datasourceModelName = models.DatasourceRoot.GetModelName()
	})
	reloadMutexMap := sync.Map{}
	// 设置刷新prisma缓存函数
	utils.ReloadPrismaCache = func(dsName string) (err error) {
		value, existed := reloadMutexMap.LoadOrStore(dsName, &sync.Mutex{})
		reloadMutex := value.(*sync.Mutex)
		reloadMutex.Lock()
		defer reloadMutex.Unlock()
		if existed {
			return
		}

		defer reloadMutexMap.Delete(dsName)
		data, err := models.DatasourceRoot.GetByDataName(dsName)
		if err != nil {
			return
		}

		action, err := GetDatasourceAction(data)
		if err != nil {
			return
		}

		if _, err = action.Introspect(); err != nil {
			return
		}

		_ = CacheDmmfText.Remove(dsName, fileloader.SystemUser)
		return
	}
}
