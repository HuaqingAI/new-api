package aionui

import (
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

func ListLatestClientPackages(c *gin.Context) {
	result, err := clientPackageService().ListLatest()
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func DownloadClientPackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	url, err := clientPackageService().DownloadURL(id)
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func GetClientUpdateFeed(c *gin.Context) {
	feed, ok, err := clientPackageService().UpdateFeed(filepath.Base(c.Request.URL.Path))
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.String(http.StatusOK, serviceaionui.BuildClientUpdateFeedYAML(feed))
}

func DownloadClientUpdateArtifact(c *gin.Context) {
	url, err := clientPackageService().UpdateArtifactURL(c.Param("version"), c.Param("file"))
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func AdminListClientPackages(c *gin.Context) {
	var query dtoaionui.ClientPackageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientPackageService().List(serviceaionui.ClientPackageQuery{
		Platform: stringValue(query.Platform),
		Status:   stringValue(query.Status),
		Page:     intValue(query.Page),
		PageSize: intValue(query.PageSize),
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminUploadClientPackage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请上传客户端下载用安装包")
		return
	}
	openedFile, err := file.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer openedFile.Close()

	updateFile, updateReader, err := optionalMultipartFile(c, "update_file")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if updateReader != nil {
		defer updateReader.Close()
	}
	metadataFile, metadataReader, err := optionalMultipartFile(c, "update_metadata_file")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if metadataReader != nil {
		defer metadataReader.Close()
	}

	result, err := clientPackageService().Upload(serviceaionui.ClientPackageUploadInput{
		Platform:               c.PostForm("platform"),
		Version:                c.PostForm("version"),
		ReleaseNote:            c.PostForm("release_note"),
		FileName:               file.Filename,
		File:                   openedFile,
		UpdateFileName:         optionalFileName(updateFile),
		UpdateFile:             updateReader,
		UpdateMetadataFileName: optionalFileName(metadataFile),
		UpdateMetadataFile:     metadataReader,
		Publish:                c.PostForm("publish") == "true" || c.PostForm("publish") == "1",
		ActorUserId:            c.GetInt("id"),
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminUpdateClientPackageStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	var req dtoaionui.ClientPackageStatusRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientPackageService().UpdateStatus(id, req.Status)
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminDeleteClientPackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	if err := clientPackageService().Delete(id); err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

func clientPackageService() *serviceaionui.ClientPackageService {
	return serviceaionui.NewClientPackageService(model.DB)
}

func writeClientPackageError(c *gin.Context, err error) {
	var validationErr serviceaionui.ClientPackageValidationError
	switch {
	case errors.As(err, &validationErr):
		common.ApiErrorMsg(c, validationErr.Message)
	case errors.Is(err, serviceaionui.ErrClientPackageInvalidInput):
		common.ApiErrorMsg(c, "请求参数不正确")
	case errors.Is(err, serviceaionui.ErrClientPackageDuplicateVersion):
		common.ApiErrorMsg(c, "该客户端版本已存在")
	case errors.Is(err, serviceaionui.ErrClientPackageMetadataRequired):
		common.ApiErrorMsg(c, "发布时必须上传 electron-builder 产出的 latest*.yml 更新元数据文件")
	case errors.Is(err, serviceaionui.ErrClientPackageUpdateFileRequired):
		common.ApiErrorMsg(c, "发布 macOS 客户端时必须同时上传 electron-builder 产出的 .zip 自动更新包")
	case errors.Is(err, serviceaionui.ErrClientPackageNotFound), errors.Is(err, serviceaionui.ErrClientPackageNotPublished):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "客户端安装包不存在或未发布"})
	default:
		common.ApiError(c, err)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func optionalMultipartFile(c *gin.Context, field string) (*multipart.FileHeader, multipart.File, error) {
	file, err := c.FormFile(field)
	if err != nil {
		return nil, nil, nil
	}
	opened, err := file.Open()
	if err != nil {
		return nil, nil, err
	}
	return file, opened, nil
}

func optionalFileName(file *multipart.FileHeader) string {
	if file == nil {
		return ""
	}
	return file.Filename
}
