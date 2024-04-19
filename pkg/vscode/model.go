package vscode

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"os"
	"path/filepath"
)

const (
	Changed FileChangeType = iota + 1
	Created
	Deleted
)

const (
	File FileType = iota + 1
	Directory
	SymbolicLink FileType = 64
)

const vscodeModelName = "vscode"

type (
	ErrorType int
	// FileType 定义了文件类型
	FileType int
	// FileChangeType 定义了文件变化类型
	FileChangeType int
	// FileStat 包装了os.FileInfo用于返回文件元数据
	FileStat struct {
		Type  FileType `json:"type"`
		Ctime int64    `json:"ctime"`
		Mtime int64    `json:"mtime"`
		Size  int64    `json:"size"`
		Name  string   `json:"name"`
	}
	// FileChangeEvent 定义了文件变化事件
	FileChangeEvent struct {
		// 文件变化类型
		Type FileChangeType // `json:"type"`
		// 文件URI
		URI string // `json:"uri"`
		// 旧URI（如重命名操作）
		OldURI string // `json:"oldUri"`
		// 文件元数据
		Metadata FileStat // `json:"metadata"`
	}
	// FileSystemProvider 是一个定义了操作文件系统方法的接口
	FileSystemProvider interface {
		OnDidChangeFile(ch chan<- []FileChangeEvent)
		Watch(uri string, recursive bool, excludes []string) error
		Stat(uri string) (FileStat, error)
		ReadDirectory(uri string) ([]FileStat, error)
		CreateDirectory(uri string) error
		ReadFile(uri string) ([]byte, error)
		WriteFile(uri string, content []byte, create, overwrite bool) error
		Delete(uri string, recursive bool) error
		Rename(oldURI, newURI string, overwrite bool) error
		Copy(sourceURI, destinationURI string, overwrite bool) error
	}
	Uri struct {
		Value string // `json:"value"`
	}
	FileSystemError struct {
		Type     ErrorType // `json:"type"`
		Uri      Uri       // `json:"uri"`
		Message  string    // `json:"message"`
		Internal error     // `json:"internal"`
	}
)

type (
	// fileSystemProvider 是FileSystemProvider的实现
	fileSystemProvider struct {
		watchers map[string]os.FileInfo
		events   chan []FileChangeEvent
	}
)

// NewFileSystemProvider 创建并返回一个新的fileSystemProvider
func NewFileSystemProvider() FileSystemProvider {
	return &fileSystemProvider{
		watchers: make(map[string]os.FileInfo),
		events:   make(chan []FileChangeEvent),
	}
}

// OnDidChangeFile 注册一个用于接收文件变化事件的通道
func (p *fileSystemProvider) OnDidChangeFile(ch chan<- []FileChangeEvent) {
	go func() {
		for events := range p.events {
			ch <- events
		}
	}()
}

// Watch 监听指定uri上的文件变化
func (p *fileSystemProvider) Watch(uri string, recursive bool, excludes []string) error {
	info, err := os.Stat(uri)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeOnlyDirectoriesCanWatchError)
	}

	watcher := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			p.events <- []FileChangeEvent{{Type: Deleted, URI: path}}
			delete(p.watchers, path)
			return nil
		}
		if !recursive && path != uri {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		eventType := Changed
		if _, ok := p.watchers[path]; !ok {
			eventType = Created
		}
		p.watchers[path] = info
		p.events <- []FileChangeEvent{{Type: eventType, URI: path, Metadata: FileStatFromOs(info)}}
		return nil
	}

	err = filepath.Walk(uri, watcher)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-p.events:
				for _, event := range event {
					watchPath := event.URI
					if event.Type == Deleted {
						watchPath = event.OldURI
					}
					if excludeMatch(watchPath, excludes) {
						continue
					}
					info, err := os.Stat(watchPath)
					if err != nil {
						continue
					}
					p.watchers[watchPath] = info
				}
			}
		}
	}()

	return nil
}

// Stat 返回指定URI的文件元数据
func (p *fileSystemProvider) Stat(uri string) (FileStat, error) {
	info, err := os.Stat(uri)
	if err != nil {
		return FileStat{}, err
	}
	return FileStatFromOs(info), nil
}

// ReadDirectory 返回指定URI下的所有文件和目录的元数据
func (p *fileSystemProvider) ReadDirectory(uri string) ([]FileStat, error) {
	fileInfos, err := os.ReadDir(uri)
	if err != nil {
		return nil, err
	}
	results := make([]FileStat, len(fileInfos))
	for i, fi := range fileInfos {
		info, _ := fi.Info()
		results[i] = FileStatFromOs(info)
	}
	return results, nil
}

// CreateDirectory 创建一个新目录
func (p *fileSystemProvider) CreateDirectory(uri string) error {
	if _, err := os.Stat(uri); err == nil || !os.IsNotExist(err) {
		return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeDirectoryExistError, uri)
	}
	return os.MkdirAll(uri, 0755)
}

// ReadFile 返回指定URI的文件内容
func (p *fileSystemProvider) ReadFile(uri string) ([]byte, error) {
	return utils.ReadFile(uri)
}

// WriteFile 将content写入指定URI的文件中。如果create为true，则在文件不存在的情况下创建文件；如果overwrite
// 为true，则覆盖已有文件
func (p *fileSystemProvider) WriteFile(uri string, content []byte, create, overwrite bool) error {
	_, err := os.Stat(uri)
	if os.IsNotExist(err) {
		if !create {
			return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeFileNotExistError, uri)
		}
	} else {
		if !overwrite {
			return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeFileExistError, uri)
		}
	}

	return utils.WriteFile(uri, content)
}

// Delete 删除指定URI的文件或目录。如果recursive为true，则删除所有子目录和文件
func (p *fileSystemProvider) Delete(uri string, recursive bool) error {
	info, err := os.Stat(uri)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if recursive {
			err = os.RemoveAll(uri)
		} else {
			err = os.Remove(uri)
		}
	} else {
		err = os.Remove(uri)
	}
	return err
}

// Rename 将旧URI重命名为新URI。如果overwrite为true，则覆盖同名文件
func (p *fileSystemProvider) Rename(oldURI, newURI string, overwrite bool) error {
	_, err := os.Stat(oldURI)
	if err != nil {
		return err
	}

	if _, err := os.Stat(newURI); !os.IsNotExist(err) {
		if !overwrite {
			return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeFileExistError, newURI)
		}
		err = p.Delete(newURI, false)
		if err != nil {
			return err
		}
	}

	return os.Rename(oldURI, newURI)
}

// Copy 将源URI的文件或目录复制到目标URI。如果overwrite为true，则覆盖同名文件
func (p *fileSystemProvider) Copy(sourceURI, destinationURI string, overwrite bool) error {
	sourceInfo, err := os.Stat(sourceURI)
	if err != nil {
		return err
	}
	if sourceInfo.IsDir() {
		if _, err := os.Stat(destinationURI); !os.IsNotExist(err) {
			if !overwrite {
				return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeFileExistError, destinationURI)
			}
			err = p.Delete(destinationURI, true)
			if err != nil {
				return err
			}
		}
		return copyDir(sourceURI, destinationURI)
	} else {
		if _, err := os.Stat(destinationURI); !os.IsNotExist(err) {
			if !overwrite {
				return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeFileExistError, destinationURI)
			}
			err = p.Delete(destinationURI, false)
			if err != nil {
				return err
			}
		}
		return copyFile(sourceURI, destinationURI)
	}
}

// FileStatFromOs 将os.FileInfo转换为FileStat
func FileStatFromOs(fi os.FileInfo) FileStat {
	fileType := File
	if fi.Mode() == os.ModeSymlink {
		fileType = SymbolicLink
	} else if fi.IsDir() {
		fileType = Directory
	}

	return FileStat{
		Type:  fileType,
		Ctime: fi.ModTime().UnixNano(),
		Mtime: fi.ModTime().UnixNano(),
		Name:  fi.Name(),
		Size:  fi.Size(),
	}
}

// excludeMatch 检查指定的路径是否匹配排除列表中的任意一项
func excludeMatch(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if match, _ := filepath.Match(exclude, path); match {
			return true
		}
	}
	return false
}

// copyDir 复制源目录到目标目录
func copyDir(sourceDir string, targetDir string) error {
	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}

	if !sourceInfo.IsDir() {
		return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeSourceNotDirectoryError, sourceDir)
	}

	_, err = os.Stat(targetDir)
	if err == nil {
		return i18n.NewCustomErrorWithMode(vscodeModelName, nil, i18n.VscodeTargetDirectoryExistError, targetDir)
	}
	if !os.IsNotExist(err) {
		return err
	}

	err = os.MkdirAll(targetDir, sourceInfo.Mode())
	if err != nil {
		return err
	}

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		sourceFilePath := filepath.Join(sourceDir, file.Name())
		targetFilePath := filepath.Join(targetDir, file.Name())

		if file.IsDir() {
			err = copyDir(sourceFilePath, targetFilePath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(sourceFilePath, targetFilePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile 复制源文件到目标文件
func copyFile(sourceFile string, targetFile string) error {
	return utils.CopyFile(sourceFile, targetFile)
}
