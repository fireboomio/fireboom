package utils

import (
	"bufio"
	"bytes"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileAsUTF8 以UTF8格式读取文件
func ReadFileAsUTF8(path string) (dstBytes []byte, err error) {
	srcBytes, err := ReadFile(path)
	if err != nil {
		return
	}

	detectBest, err := chardet.NewTextDetector().DetectBest(srcBytes)
	if err != nil {
		return
	}

	fromEncoding, err := ianaindex.MIME.Encoding(detectBest.Charset)
	if err != nil {
		return
	}

	toEncoding, err := ianaindex.MIME.Encoding("utf-8")
	if err != nil {
		return
	}

	transformer := transform.Chain(fromEncoding.NewDecoder(), toEncoding.NewEncoder())
	dstBytes, _, err = transform.Bytes(transformer, srcBytes)
	return
}

// ReadFile 逐行读取文件，避免读取大文件过多占用资源
func ReadFile(path string) (content []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	var (
		lineBytes     []byte
		contentBuffer bytes.Buffer
	)
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		lineBytes, err = reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return
		}
		_, _ = contentBuffer.Write(lineBytes)
		if err == io.EOF {
			err = nil
			break
		}
	}
	content = contentBuffer.Bytes()
	return
}

// ReadWithCondition 带条件读取文件
func ReadWithCondition(path string, skipFunc, breakFunc func(int, string) bool) (content []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	var (
		lineNumber       int
		lineWithoutDelim string
	)
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		lineWithoutDelim, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return
		}

		lineNumber++
		lineWithoutDelim = strings.TrimRight(lineWithoutDelim, "\n")
		if skipFunc != nil && skipFunc(lineNumber, lineWithoutDelim) {
			continue
		}
		content = append(content, lineWithoutDelim)
		if breakFunc != nil && breakFunc(lineNumber, lineWithoutDelim) {
			break
		}
		if err == io.EOF {
			err = nil
			break
		}
	}
	return
}

// MkdirAll 创建目录
func MkdirAll(dirname string) error {
	return os.MkdirAll(dirname, os.ModePerm)
}

// WriteFile 写入文本到文件，自动创建父目录
func WriteFile(path string, dataBytes []byte) error {
	if err := MkdirAll(filepath.Dir(path)); err != nil {
		return err
	}

	return os.WriteFile(path, dataBytes, 0644)
}

// CreateFile 创建文件，自动创建父目录
func CreateFile(path string) (file *os.File, err error) {
	if err = MkdirAll(filepath.Dir(path)); err != nil {
		return
	}

	_ = os.Remove(path)
	return os.Create(path)
}

// CopyFile 高效地拷贝文件，使用底层操作系统的零拷贝特性，不需要将整个文件的内容加载到内存中。
func CopyFile(srcPath, dstPath string) (err error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := CreateFile(dstPath)
	if err != nil {
		return
	}
	defer func() { _ = dstFile.Close() }()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return
	}

	err = dstFile.Sync()
	return
}

// NotExistFile 判断文件不存在
func NotExistFile(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// IsEmptyDirectory 判断目录是否为空
func IsEmptyDirectory(dir string) bool {
	dirEntries, _ := os.ReadDir(dir)
	return len(dirEntries) == 0
}
