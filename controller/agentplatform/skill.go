package agentplatform

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListSkills(c *gin.Context) {
	var query dtoagentplatform.SkillQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := skillService().List(apservice.SkillQuery{
		OwnerUserId: query.OwnerUserId,
		TenantId:    query.TenantId,
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	items := make([]dtoagentplatform.SkillItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResourceItem(item.ResourceItem))
	}
	common.ApiSuccess(c, dtoagentplatform.SkillListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetSkill(c *gin.Context) {
	item, err := skillService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func CreateSkill(c *gin.Context) {
	var req dtoagentplatform.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := skillService().Create(apservice.SkillCreateInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
		OwnerUserId: c.GetInt("id"),
		TenantId:    0,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func CreateSkillMultipart(c *gin.Context) {
	item, err := skillService().Create(apservice.SkillCreateInput{
		DisplayName: c.PostForm("display_name"),
		Description: c.PostForm("description"),
		OwnerUserId: c.GetInt("id"),
		TenantId:    0,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiSuccess(c, mapSkillDetailItem(item))
		return
	}
	opened, err := file.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer opened.Close()
	item, err = skillService().SavePackage(item.ResourceId, file.Filename, opened)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapSkillDetailItem(item))
}

func UpdateSkill(c *gin.Context) {
	var req dtoagentplatform.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := skillService().Update(c.Param("id"), apservice.SkillUpdateInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func UploadSkillPackage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	opened, err := file.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer opened.Close()
	item, err := skillService().SavePackage(c.Param("id"), file.Filename, opened)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapSkillDetailItem(item))
}

func DownloadSkillPackage(c *gin.Context) {
	path, err := skillService().PackagePath(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	c.FileAttachment(path, "")
}

func EnableSkill(c *gin.Context) {
	item, err := skillService().SetStatus(c.Param("id"), apmodel.ResourceStatusPublished)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapSkillDetailItem(item))
}

func DisableSkill(c *gin.Context) {
	item, err := skillService().SetStatus(c.Param("id"), apmodel.ResourceStatusDisabled)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapSkillDetailItem(item))
}

func DeleteSkill(c *gin.Context) {
	if err := skillService().Delete(c.Param("id")); err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"resource_id": c.Param("id")})
}

func skillService() *apservice.SkillService {
	return apservice.NewSkillService(model.DB)
}

func CreateSkillEntry(c *gin.Context) {
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(contentType, "multipart/form-data") {
		CreateSkillMultipart(c)
		return
	}
	CreateSkill(c)
}

func mapSkillDetailItem(item apservice.SkillItem) dtoagentplatform.SkillDetailItem {
	return dtoagentplatform.SkillDetailItem{
		ResourceItem: mapResourceItem(item.ResourceItem),
		FileName:     item.FileName,
		Sha256:       item.Sha256,
		SizeBytes:    item.SizeBytes,
	}
}
