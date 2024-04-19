// Package fileloader
/*
 文件加载的扩展名和目录字典
*/
package fileloader

type Extension string

const (
	ExtJson       Extension = ".json"
	ExtTxt        Extension = ".txt"
	ExtProperties Extension = ".properties"
	ExtGraphql    Extension = ".graphql"
	ExtPrisma     Extension = ".prisma"
	ExtYaml       Extension = ".yaml"
	ExtYml        Extension = ".yml"
	ExtKey        Extension = ".key"
)

var rootDirectories map[string]string

func GetRootDirectories() map[string]string {
	return rootDirectories
}

func init() {
	rootDirectories = make(map[string]string)
}
