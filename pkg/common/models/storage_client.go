// Package models
/*
 利用storage配置构建上传客户端，并提供缓存且在storage变更时清除
*/
package models

import (
	"context"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var (
	ClientCache *StorageClientCache
	partSize    uint64 = 1024 * 1024 * 5
)

func init() {
	ClientCache = &StorageClientCache{}
}

const (
	headerResponseContentDisposition = "response-content-disposition"
	signedUrlExpires                 = time.Second * 24 * 60 * 60
)

type (
	StorageFile struct {
		SignedUrl    string    `json:"signedUrl,omitempty"`
		Name         string    `json:"name"`                  // Name of the object
		Size         int64     `json:"size"`                  // Size in bytes of the object.
		LastModified time.Time `json:"lastModified"`          // Date and time the object was last modified.
		ContentType  string    `json:"contentType,omitempty"` // A standard MIME type describing the format of the object data.
		Extension    string    `json:"extension,omitempty"`
	}
	StorageClientCache map[string]*minio.Client
)

func buildStorageFile(info *minio.ObjectInfo) *StorageFile {
	file := &StorageFile{
		Name:         info.Key,
		Size:         info.Size,
		LastModified: info.LastModified,
		ContentType:  info.ContentType,
		Extension:    filepath.Ext(info.Key),
	}
	return file
}

func (s *StorageClientCache) removeClient(name string) {
	delete(*s, name)
}

// 构建客户端并添加来对Region的设置
func (s *StorageClientCache) buildClient(storage *Storage) (client *minio.Client, err error) {
	if !storage.Enabled {
		err = i18n.NewCustomErrorWithMode(StorageRoot.GetModelName(), nil, i18n.StorageDisabledError)
		return
	}

	client, ok := (*s)[storage.Name]
	if ok {
		return
	}

	endpoint := utils.GetVariableString(storage.Endpoint)
	region := utils.GetVariableString(storage.BucketLocation)
	accessKey, secretAccessKey := utils.GetVariableString(storage.AccessKeyID), utils.GetVariableString(storage.SecretAccessKey)
	client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretAccessKey, ""),
		Secure: storage.UseSSL,
		Region: region,
	})
	if err != nil {
		return
	}

	(*s)[storage.Name] = client
	return
}

// Ping 检查链接并在bucket不存在时尝试创建
func (s *StorageClientCache) Ping(ctx context.Context, storage *Storage) (err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil || exists {
		return
	}

	bucketLocation := utils.GetVariableString(storage.BucketLocation)
	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: bucketLocation})
	return
}

// PutDirObject 创建目录且没有多余文件产生
func (s *StorageClientCache) PutDirObject(ctx context.Context, storage *Storage, dirname string) (err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	dirname = utils.AppendIfMissSlash(dirname)
	bucketName := utils.GetVariableString(storage.BucketName)
	_, err = client.PutObject(ctx, bucketName, dirname, nil, 0, minio.PutObjectOptions{})
	return
}

// PutObject 从流中读取文件进行上传，并支持设置父目录
func (s *StorageClientCache) PutObject(ctx context.Context, storage *Storage, dirname string, multipartFile *multipart.FileHeader) (err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	file, err := multipartFile.Open()
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	filename := multipartFile.Filename
	if dirname != "" {
		filename = utils.NormalizePath(dirname, filename)
	}
	filename = strings.TrimPrefix(filename, "/")
	bucketName := utils.GetVariableString(storage.BucketName)
	contentType := multipartFile.Header.Get(echo.HeaderContentType)
	_, err = client.PutObject(ctx, bucketName, filename, file, multipartFile.Size, minio.PutObjectOptions{PartSize: partSize, ContentType: contentType})
	return
}

// RemoveObjects 移除文件或目录
func (s *StorageClientCache) RemoveObjects(ctx context.Context, storage *Storage, prefix string) (err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	options := minio.ListObjectsOptions{Prefix: strings.TrimPrefix(prefix, "/")}
	objectInfoChan := client.ListObjects(ctx, bucketName, options)

	for objectError := range client.RemoveObjects(ctx, bucketName, objectInfoChan, minio.RemoveObjectsOptions{}) {
		return objectError.Err
	}
	return
}

// RenameObject 重命名文件或目录
func (s *StorageClientCache) RenameObject(ctx context.Context, storage *Storage, mutation *fileloader.DataMutation) (err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	options := minio.ListObjectsOptions{Prefix: mutation.Src}
	isDir := strings.HasSuffix(mutation.Src, "/")
	for info := range client.ListObjects(ctx, bucketName, options) {
		srcObject, destObject := info.Key, mutation.Dst
		basename := strings.TrimPrefix(info.Key, mutation.Src)
		if isDir && basename != "" {
			destObject = utils.NormalizePath(destObject, basename)
		} else if srcObject == "" {
			srcObject = mutation.Src
		}
		srcOptions := minio.CopySrcOptions{Bucket: bucketName, Object: srcObject}
		destOptions := minio.CopyDestOptions{Bucket: bucketName, Object: destObject}
		if _, err = client.CopyObject(ctx, destOptions, srcOptions); err != nil {
			return
		}

		if err = client.RemoveObject(ctx, bucketName, srcObject, minio.RemoveObjectOptions{}); err != nil {
			return
		}
	}
	return
}

// StatObject 查看文件详情
func (s *StorageClientCache) StatObject(ctx context.Context, storage *Storage, filename string) (file *StorageFile, err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	objectInfo, err := client.StatObject(ctx, bucketName, filename, minio.StatObjectOptions{})
	if err != nil {
		return
	}

	signedUrl, err := s.PresignGetObject(ctx, storage, filename)
	if err != nil {
		return
	}

	file = buildStorageFile(&objectInfo)
	file.SignedUrl = signedUrl
	return
}

func (s *StorageClientCache) PresignGetObject(ctx context.Context, storage *Storage, filename string) (signedUrl string, err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	query := url.Values{}
	query.Set(headerResponseContentDisposition, fmt.Sprintf(consts.AttachmentFilenameFormat, filename))

	bucketName := utils.GetVariableString(storage.BucketName)
	URL, err := client.PresignedGetObject(ctx, bucketName, filename, signedUrlExpires, query)
	if err != nil {
		return
	}

	signedUrl = URL.String()
	return
}

// ListObjects 查看文件列表
func (s *StorageClientCache) ListObjects(ctx context.Context, storage *Storage, prefix string) (files []*StorageFile, err error) {
	files = make([]*StorageFile, 0)
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	if prefix = strings.TrimPrefix(prefix, "/"); prefix != "" {
		prefix = utils.AppendIfMissSlash(prefix)
	}
	options := minio.ListObjectsOptions{Prefix: prefix}
	for info := range client.ListObjects(ctx, bucketName, options) {
		files = append(files, buildStorageFile(&info))
	}
	return
}

// GetObjectReader 文件下载并返回io.ReadCloser
func (s *StorageClientCache) GetObjectReader(ctx context.Context, storage *Storage, filename string) (reader io.ReadCloser, err error) {
	client, err := s.buildClient(storage)
	if err != nil {
		return
	}

	bucketName := utils.GetVariableString(storage.BucketName)
	reader, err = client.GetObject(ctx, bucketName, filename, minio.GetObjectOptions{})
	return
}
