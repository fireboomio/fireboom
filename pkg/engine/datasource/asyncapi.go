package datasource

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/asyncapi"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/ghodss/yaml"
	json "github.com/json-iterator/go"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"path/filepath"
)

func init() {
	actionMap[wgpb.DataSourceKind_ASYNCAPI] = func(ds *models.Datasource, _ string) Action {
		return &actionAsyncapi{ds: ds}
	}
}

type actionAsyncapi struct {
	ds  *models.Datasource
	doc *asyncapi.Spec
}

func (a *actionAsyncapi) Introspect() (string, error) {
	return "", nil
}

func (a *actionAsyncapi) BuildDataSourceConfiguration(document *ast.SchemaDocument) (*wgpb.DataSourceConfiguration, error) {
	return nil, nil
}

func (a *actionAsyncapi) RuntimeDataSourceConfiguration(configuration *wgpb.DataSourceConfiguration) ([]*wgpb.DataSourceConfiguration, []*wgpb.FieldConfiguration, error) {
	return nil, nil, nil
}

// 处理rest数据源依赖的文本，支持.yaml, .yml, .json等类型文件
// 添加了对openapi2和3的支持
func (a *actionAsyncapi) fetchDocument() (err error) {
	oasFilepath := models.DatasourceUploadAsyncapi.GetPath(a.ds.Name)
	oasBytes, err := utils.ReadFileAsUTF8(oasFilepath)
	if err != nil {
		return
	}

	ext := fileloader.Extension(filepath.Ext(oasFilepath))
	if slices.Contains(yamlExtensions, ext) {
		if oasBytes, err = yaml.YAMLToJSON(oasBytes); err != nil {
			return
		}
	}

	if err = json.Unmarshal(oasBytes, &a.doc); err != nil {
		return
	}

	if err = a.doc.ResolveRefsIn(); err != nil {
		return
	}

	return a.doc.Validate()
}
