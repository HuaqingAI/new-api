package aionui

import (
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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
	var feed serviceaionui.ClientUpdateFeed
	var ok bool
	var err error
	channel := filepath.Base(c.Request.URL.Path)
	if capability := c.GetHeader(serviceaionui.ClientUpdateCapabilityHeader); capability != "" {
		feed, ok, err = clientPackageService().UpdateFeedForCapability(channel, capability)
	} else {
		feed, ok, err = clientPackageService().UpdateFeed(channel)
	}
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
	var url string
	var err error
	if capability := c.GetHeader(serviceaionui.ClientUpdateCapabilityHeader); capability != "" {
		url, err = clientPackageService().UpdateArtifactURLForCapability(capability, c.Param("version"), c.Param("file"))
	} else {
		url, err = clientPackageService().UpdateArtifactURL(c.Param("version"), c.Param("file"))
	}
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func PrepareClientUpdateAccess(c *gin.Context) {
	var req dtoaionui.ClientUpdateAccessRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	userID := c.GetInt("aionui_user_id")
	if userID <= 0 {
		common.ApiSuccess(c, dtoaionui.ClientUpdateAccessResponse{LegacyOpen: true})
		return
	}
	result, err := clientPackageService().PrepareUpdateAccess(userID, c.GetString("aionui_device_id"), serviceaionui.ClientUpdateAccessInput{
		Platform:        req.Platform,
		CurrentVersion:  req.CurrentVersion,
		ExpectedVersion: req.ExpectedVersion,
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, dtoaionui.ClientUpdateAccessResponse{
		LegacyOpen:         result.LegacyOpen,
		Eligible:           result.Eligible,
		Release:            result.Release,
		ArtifactCapability: result.ArtifactCapability,
		ExpiresAt:          result.ExpiresAt,
	})
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

func AdminGetClientPackageDownloadURL(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	url, err := clientPackageService().AdminDownloadURL(id)
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"url": url})
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
	var scopes []dtoaionui.ClientPackageScopeInput
	if rawScopes := strings.TrimSpace(c.PostForm("scopes")); rawScopes != "" {
		if err := common.UnmarshalJsonStr(rawScopes, &scopes); err != nil {
			common.ApiErrorMsg(c, "invalid request params")
			return
		}
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
		RolloutMode:            c.PostForm("rollout_mode"),
		Scopes:                 mapClientPackageScopeInputs(scopes),
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminCreateClientPackageUpload(c *gin.Context) {
	var req dtoaionui.ClientPackageDirectUploadInitRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	files := make([]serviceaionui.ClientPackageDirectArtifactInput, 0, len(req.Files))
	for _, file := range req.Files {
		files = append(files, serviceaionui.ClientPackageDirectArtifactInput{
			Kind:     file.Kind,
			FileName: file.FileName,
			Sha256:   file.Sha256,
			Sha512:   file.Sha512,
			Size:     file.Size,
		})
	}
	result, err := clientPackageService().CreateDirectUpload(serviceaionui.ClientPackageDirectUploadInitInput{
		Platform:    req.Platform,
		Version:     req.Version,
		Publish:     req.Publish,
		Files:       files,
		ActorUserId: c.GetInt("id"),
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminCompleteClientPackageUpload(c *gin.Context) {
	var req dtoaionui.ClientPackageDirectUploadCompleteRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientPackageService().CompleteDirectUpload(serviceaionui.ClientPackageDirectUploadCompleteInput{
		Platform:           req.Platform,
		Version:            req.Version,
		ReleaseNote:        req.ReleaseNote,
		Publish:            req.Publish,
		RolloutMode:        req.RolloutMode,
		Scopes:             mapClientPackageScopeInputs(req.Scopes),
		File:               mapClientPackageUploadTarget(req.File),
		UpdateFile:         mapClientPackageUploadTarget(req.UpdateFile),
		UpdateMetadataFile: mapClientPackageUploadTarget(req.UpdateMetadataFile),
		ActorUserId:        c.GetInt("id"),
	})
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminGetClientPackageRollout(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientPackageService().GetRollout(id)
	if err != nil {
		writeClientPackageError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminUpdateClientPackageRollout(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	var req dtoaionui.ClientPackageRolloutRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientPackageService().UpdateRollout(id, req.RolloutMode, mapClientPackageScopeInputs(req.Scopes), c.GetInt("id"))
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
	case errors.Is(err, serviceaionui.ErrClientPackageRolloutInvalid):
		common.ApiErrorMsg(c, "客户端发布范围无效")
	case errors.Is(err, serviceaionui.ErrClientUpdateCapabilityInvalid):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "client update access denied"})
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

func mapClientPackageUploadTarget(input dtoaionui.ClientPackageDirectUploadTarget) serviceaionui.ClientPackageDirectArtifactInput {
	return serviceaionui.ClientPackageDirectArtifactInput{
		Kind:      input.Kind,
		FileName:  input.FileName,
		ObjectURI: input.ObjectURI,
		Sha256:    input.Sha256,
		Sha512:    input.Sha512,
		Size:      input.Size,
	}
}

func mapClientPackageScopeInputs(inputs []dtoaionui.ClientPackageScopeInput) []serviceaionui.ClientPackageScopeInput {
	result := make([]serviceaionui.ClientPackageScopeInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, serviceaionui.ClientPackageScopeInput{SubjectType: input.SubjectType, SubjectId: input.SubjectId})
	}
	return result
}
