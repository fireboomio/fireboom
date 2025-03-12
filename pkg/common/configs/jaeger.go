package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/ghodss/yaml"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cast"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/wundergraph/wundergraph/pkg/logging"
	"go.uber.org/zap"
	"os"
)

type jaegerConfiguration struct {
	SpanInout bool `json:"spanInout"`
	jaegercfg.Configuration
}

func (j *jaegerConfiguration) setGlobalTracer() {
	yamlConfig := &j.Configuration
	if _, err := yamlConfig.FromEnv(); err != nil {
		logger.Error("setGlobalTracer FromEnv failed", zap.Error(err))
		return
	}
	if yamlConfig.Disabled {
		return
	}

	yamlConfig.ServiceName = utils.ReplacePlaceholderFromEnv(yamlConfig.ServiceName)
	yamlConfig.Gen128Bit = true
	tracer, _, err := yamlConfig.NewTracer(jaegercfg.Logger(jaegerlog.StdLogger))
	if err != nil {
		logger.Error("setGlobalTracer NewTracer failed", zap.Error(err))
		return
	}

	if e := os.Getenv(consts.JaegerSpanInout); e != "" {
		j.SpanInout = cast.ToBool(e)
	}
	opentracing.SetGlobalTracer(tracer)
	logging.WithSpanInout(j.SpanInout)
}

func init() {
	utils.RegisterInitMethod(15, func() {
		jaegerConfigDefaultText := &fileloader.ModelText[jaegerConfiguration]{
			Root:      embed.DefaultRoot,
			Extension: fileloader.ExtYml,
			TextRW:    &fileloader.EmbedTextRW{EmbedFiles: &embed.DefaultFs, Name: consts.JaegerConfig},
		}
		jaegerConfigDefaultText.Init()

		jaegerConfigRoot := &fileloader.Model[jaegerConfiguration]{
			Root:             utils.NormalizePath(consts.RootStore, consts.StoreConfigParent),
			Extension:        fileloader.ExtYml,
			LoadErrorIgnored: true,
			DataHook: &fileloader.DataHook[jaegerConfiguration]{
				AfterInit: func(datas map[string]*jaegerConfiguration) {
					datas[consts.JaegerConfig].setGlobalTracer()
				},
				AfterUpdate: func(data *jaegerConfiguration, _ *fileloader.DataModifies, _ string, _ ...string) {
					data.setGlobalTracer()
				},
			},
			DataRW: &fileloader.SingleDataRW[jaegerConfiguration]{
				InitDataBytes: jaegerConfigDefaultText.GetFirstCache(),
				DataName:      consts.JaegerConfig,
				Unmarshal: func(dataBytes []byte) (data *jaegerConfiguration, err error) {
					err = yaml.Unmarshal(dataBytes, &data)
					return
				},
				Marshal: func(data *jaegerConfiguration) ([]byte, error) {
					return yaml.Marshal(data)
				},
			},
		}
		jaegerConfigRoot.Init(lazyLogger)
		utils.AddBuildAndStartFuncWatcher(func(f func()) { jaegerConfigRoot.DataHook.AfterMutate = f })
	})
}
