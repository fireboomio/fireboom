package assets

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

//go:embed dbml
var dbmlFiles embed.FS

//go:embed all:front
var frontFiles embed.FS

const (
	staticDirname     = "static"
	embedDbmlDirname  = "dbml"
	embedFrontDirname = "front"
)

func getFileSystem(files embed.FS, dirName string) http.FileSystem {
	subFiles, err := fs.Sub(files, dirName)
	if err != nil {
		panic(err)
	}
	return http.FS(subFiles)
}

func GetDBMLFileSystem() http.FileSystem {
	return getFileSystem(dbmlFiles, embedDbmlDirname)
}

func GetFrontFileSystem() http.FileSystem {
	fileInfo, _ := os.Stat(staticDirname)
	if fileInfo != nil && fileInfo.IsDir() {
		return http.FS(os.DirFS(staticDirname))
	}

	return getFileSystem(frontFiles, embedFrontDirname)
}
