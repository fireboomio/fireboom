package utils

import (
	"archive/zip"
	"bytes"
	"fireboom-server/pkg/plugins/i18n"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
)

const ExtensionZip = ".zip"

// UnzipMultipartFile 解压上传请求中压缩文件
// 允许过滤解压后的文件且返回成功的文件路径
func UnzipMultipartFile(multipartFile *multipart.FileHeader, filter func(string) bool) (items []string, err error) {
	items = make([]string, 0)
	file, err := multipartFile.Open()
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	zipReader, err := zip.NewReader(file, multipartFile.Size)
	if err != nil {
		return
	}

	var itemReader io.ReadCloser
	var itemFile *os.File
	for _, item := range zipReader.File {
		itemReader, err = item.Open()
		if err != nil {
			return
		}

		if !filter(item.Name) {
			continue
		}

		itemFile, err = os.OpenFile(item.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, item.Mode())
		if err != nil {
			return
		}

		_, err = io.Copy(itemFile, itemReader)
		if err != nil {
			return
		}

		items = append(items, item.Name)
		_ = itemFile.Close()
		_ = itemReader.Close()
	}
	return
}

// ZipFilesWithBuffer 压缩文件到buffer中
// 允许仅保留最后一级路径
// 支持压缩目录
func ZipFilesWithBuffer(filenames []string, onlyBase bool) (buf *bytes.Buffer, err error) {
	if len(filenames) == 0 {
		err = i18n.NewCustomError(nil, i18n.FileZipAmountZeroError)
		return
	}

	buf = new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	defer func() { _ = zipWriter.Close() }()

	// 遍历文件列表，将每个文件添加到压缩文件中
	var itemFile os.FileInfo
	for _, item := range filenames {
		var relParent string
		itemFile, _ = os.Stat(item)
		if itemFile != nil && itemFile.IsDir() {
			if onlyBase {
				relParent = item
			}
			if err = filepath.Walk(item, func(itemPath string, info fs.FileInfo, _ error) error {
				if info == nil || info.IsDir() {
					return nil
				}

				return addFileToZip(zipWriter, itemPath, relParent)
			}); err != nil {
				return
			}

			continue
		}

		if err = addFileToZip(zipWriter, item, relParent); err != nil {
			return
		}
	}
	return
}

// 将文件添加到压缩文件中的辅助函数
func addFileToZip(zipWriter *zip.Writer, filename, relParent string) error {
	// 打开要添加的文件
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// 创建一个zip文件的头
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}

	// 设置压缩方法为默认的deflate
	header.Method = zip.Deflate
	if relParent != "" {
		header.Name, _ = filepath.Rel(relParent, filename)
	} else {
		header.Name = filename
	}

	// 创建一个zip.Writer中的文件写入器
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// 将文件内容复制到压缩文件中
	_, err = io.Copy(writer, file)
	if err != nil {
		return err
	}

	return nil
}
