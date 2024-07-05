// Package build
/*
 引擎编译的通用代码，包含所生成的利用fileloader定义的文件管理
*/
package build

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"sync"
)

var (
	logger                        *zap.Logger
	GeneratedGraphqlSchemaText    *fileloader.ModelText[any]
	GeneratedGraphqlConfigRoot    *fileloader.Model[wgpb.WunderGraphConfiguration]
	GeneratedOperationsConfigRoot *fileloader.Model[OperationsConfig]
	GeneratedSwaggerText          *fileloader.ModelText[any]
	GeneratedHookSwaggerText      *fileloader.ModelText[any]
	generatedDirname              = utils.NormalizePath(consts.RootExported, consts.ExportedGeneratedParent)

	// 所有需要执行的编译，通过key排序确定执行顺序
	buildResolves = make(map[int]ResolveFetch)
	// 异步需要执行的生成
	asyncGenerates []AsyncGenerateFetch
)

type (
	Builder struct {
		FieldHashes *utils.SyncMap[string, *LazyFieldHash]
		Document    *ast.SchemaDocument
		DefinedApi  *wgpb.UserDefinedApi
	}
	LazyFieldHash struct {
		hashValue string
		lazyFunc  func() string
	}
	Resolve interface {
		Resolve(*Builder) error
	}
	AsyncGenerate interface {
		Generate(*Builder)
	}
	ResolveFetch       func() Resolve
	AsyncGenerateFetch func() AsyncGenerate
)

func (f *LazyFieldHash) Hash() string {
	if f.hashValue == "" {
		f.hashValue, f.lazyFunc = f.lazyFunc(), nil
	}
	return f.hashValue
}

func GeneratedJsonLoadErrored() bool {
	return GeneratedGraphqlConfigRoot.LoadErrored() || GeneratedOperationsConfigRoot.LoadErrored()
}

// CallAsyncGenerates 调用所有异步生成，支持waitGroup等待所有操作完成
// 支持事件订阅，当实现EventSubscribe接口时调用
// 当系统仅使用编译功能时需要传递参数waitGroup，不然系统会直接退出，不等线程完成
func CallAsyncGenerates(builder *Builder, waitGroup ...*sync.WaitGroup) {
	hasGroup := len(waitGroup) == 1
	if hasGroup {
		waitGroup[0].Add(len(asyncGenerates))
	}
	for _, generate := range asyncGenerates {
		generateItem := generate()
		go func() {
			generateItem.Generate(builder)
			eventbus.EnsureEventSubscribe(generateItem)
			if hasGroup {
				waitGroup[0].Done()
			}
		}()
	}
}

// CallRunResolves 调用编译处理，通过key排序确定执行顺序
// 支持事件订阅，当实现EventSubscribe接口时调用
func CallRunResolves(builder *Builder) (err error) {
	sorts := maps.Keys(buildResolves)
	slices.Sort(sorts)
	for _, sort := range sorts {
		resolve := buildResolves[sort]()
		if err = resolve.Resolve(builder); err != nil {
			return
		}

		eventbus.EnsureEventSubscribe(resolve)
	}
	return
}

func addResolve(sort int, resolve ResolveFetch) {
	buildResolves[sort] = resolve
}

func AddAsyncGenerate(resolve AsyncGenerateFetch) {
	asyncGenerates = append(asyncGenerates, resolve)
}

func init() {
	GeneratedGraphqlSchemaText = &fileloader.ModelText[any]{
		Root:      generatedDirname,
		Extension: fileloader.ExtGraphql,
		TextRW: &fileloader.SingleTextRW[any]{
			Name: consts.ExportedGeneratedGraphqlSchemaFilename,
		},
	}

	GeneratedGraphqlConfigRoot = &fileloader.Model[wgpb.WunderGraphConfiguration]{
		Root:      generatedDirname,
		Extension: fileloader.ExtJson,
		DataRW: &fileloader.SingleDataRW[wgpb.WunderGraphConfiguration]{
			DataName: consts.ExportedGeneratedFireboomConfigFilename,
		},
	}

	GeneratedOperationsConfigRoot = &fileloader.Model[OperationsConfig]{
		Root:      generatedDirname,
		Extension: fileloader.ExtJson,
		DataRW: &fileloader.SingleDataRW[OperationsConfig]{
			DataName: consts.ExportedGeneratedFireboomOperationsFilename,
		},
	}

	GeneratedSwaggerText = &fileloader.ModelText[any]{
		Root:      generatedDirname,
		Extension: fileloader.ExtJson,
		TextRW: &fileloader.SingleTextRW[any]{
			Name: consts.ExportedGeneratedSwaggerFilename,
		},
	}
	GeneratedHookSwaggerText = &fileloader.ModelText[any]{
		Root:      generatedDirname,
		Extension: fileloader.ExtJson,
		TextRW: &fileloader.SingleTextRW[any]{
			Name: consts.ExportedGeneratedHookSwaggerFilename,
		},
	}

	utils.RegisterInitMethod(30, func() {
		logger = zap.L()
		if utils.GetStringWithLockViper(consts.EngineFirstStatus) == consts.EngineBuilding {
			GeneratedGraphqlConfigRoot.LoadErrorIgnored, GeneratedOperationsConfigRoot.LoadErrorIgnored = true, true
		}
		GeneratedGraphqlSchemaText.Init()
		GeneratedGraphqlConfigRoot.Init()
		GeneratedOperationsConfigRoot.Init()
		GeneratedSwaggerText.Init()
		GeneratedHookSwaggerText.Init()
	})
}
