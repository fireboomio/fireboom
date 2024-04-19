package vscode

import (
	"bytes"
	"github.com/spf13/cast"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	fsProvider FileSystemProvider
}

func NewVscodeHandler() *Handler {
	return &Handler{NewFileSystemProvider()}
}

func (h *Handler) watch(c echo.Context) error {
	param := struct {
		Uri       string   `json:"uri"`
		Recursive bool     `json:"recursive"`
		Excludes  []string `json:"excludes"`
	}{}
	err := c.Bind(&param)
	if err != nil {
		return err
	}

	err = h.fsProvider.Watch(param.Uri, param.Recursive, param.Excludes)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, nil)
}

// stat @Title stat
// @Description 返回指定URI的文件元数据
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Success 200 {object} FileStat "查询成功"
// @Failure 400	"查询失败"
// @Router /vscode/stat [GET]
func (h *Handler) state(c echo.Context) error {
	uri := c.QueryParam("uri")
	stat, err := h.fsProvider.Stat(uri)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, stat)
}

// readDirectory @Title readDirectory
// @Description 返回指定URI下的所有文件和目录的元数据
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Param ignoreNotExist formData bool false "忽略不存在报错"
// @Success 200 {object} []FileStat "查询成功"
// @Failure 400	"查询失败"
// @Router /vscode/readDirectory [GET]
func (h *Handler) readDirectory(c echo.Context) error {
	uri := c.QueryParam("uri")
	statArr, err := h.fsProvider.ReadDirectory(uri)
	if err != nil {
		if os.IsNotExist(err) && cast.ToBool(c.QueryParam("ignoreNotExist")) {
			return c.JSON(http.StatusOK, []FileStat{})
		}

		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, statArr)
}

// createDirectory @Title createDirectory
// @Description 创建一个新目录
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Success 200 "创建成功"
// @Failure 400	"创建失败"
// @Router /vscode/createDirectory [POST]
func (h *Handler) createDirectory(c echo.Context) error {
	param := struct {
		Uri string `json:"uri"`
	}{}
	err := c.Bind(&param)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = h.fsProvider.CreateDirectory(param.Uri)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

// readFile @Title readFile
// @Description 返回指定URI的文件内容
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Success 200 {string} string "查询成功"
// @Failure 400	"查询失败"
// @Router /vscode/readFile [GET]
func (h *Handler) readFile(c echo.Context) error {
	uri := c.QueryParam("uri")
	fileBytes, err := h.fsProvider.ReadFile(uri)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+filepath.Base(uri))
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(fileBytes))
}

// writeFile @Title writeFile
// @Description 将content写入指定URI的文件中。如果create为true，则在文件不存在的情况下创建文件；如果overwrite
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Param content formData file true "URI"
// @Param create formData bool false "create"
// @Param overwrite formData bool false "overwrite"
// @Success 200 "写入成功"
// @Failure 400	"写入失败"
// @Router /vscode/writeFile [POST]
func (h *Handler) writeFile(c echo.Context) error {
	fileContent, err := c.FormFile("content")
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	openFile, err := fileContent.Open()
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	fileBytes, err := io.ReadAll(openFile)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	create := strings.ToLower(c.FormValue("create")) == "true"
	overwrite := strings.ToLower(c.FormValue("overwrite")) == "true"
	err = h.fsProvider.WriteFile(c.FormValue("uri"), fileBytes, create, overwrite)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// delete @Title delete
// @Description 删除指定URI的文件或目录。如果recursive为true，则删除所有子目录和文件
// @Accept  json
// @Tags  vscode
// @Param uri formData string true "URI"
// @Param recursive formData string false "是否删除所有子目录和文件"
// @Success 200 "删除成功"
// @Failure 400	"删除失败"
// @Router /vscode/delete [DELETE]
func (h *Handler) delete(c echo.Context) error {
	param := struct {
		Uri       string `json:"uri"`
		Recursive bool   `json:"recursive"`
	}{}
	err := c.Bind(&param)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = h.fsProvider.Delete(param.Uri, param.Recursive)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

// rename @Title rename
// @Description 将旧URI重命名为新URI。如果overwrite为true，则覆盖同名文件
// @Accept  json
// @Tags  vscode
// @Param oldURI formData string true "oldURI"
// @Param newURI formData string true "newURI"
// @Param overwrite formData bool false "是否覆盖同名文件"
// @Success 200 "删除成功"
// @Failure 400	"删除失败"
// @Router /vscode/rename [PUT]
func (h *Handler) rename(c echo.Context) error {
	param := struct {
		OldURI    string `json:"oldURI,omitempty"`
		NewURI    string `json:"newURI,omitempty"`
		Overwrite bool   `json:"overwrite,omitempty"`
	}{}
	err := c.Bind(&param)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = h.fsProvider.Rename(param.OldURI, param.NewURI, param.Overwrite)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

// copy @Title copy
// @Description 将旧URI重命名为新URI。如果overwrite为true，则覆盖同名文件
// @Accept  json
// @Tags  vscode
// @Param source formData string true "source"
// @Param destination formData string true "destination"
// @Param overwrite formData bool false "是否覆盖同名文件"
// @Success 200 "复制成功"
// @Failure 400	"复制失败"
// @Router /vscode/copy [POST]
func (h *Handler) copy(c echo.Context) error {
	param := struct {
		SourceURI      string `json:"source,omitempty"`
		DestinationURI string `json:"destination,omitempty"`
		Overwrite      bool   `json:"overwrite,omitempty"`
	}{}
	err := c.Bind(&param)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = h.fsProvider.Copy(param.SourceURI, param.DestinationURI, param.Overwrite)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}
