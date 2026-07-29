package agentplatform

import (
	"errors"
	"os"

	"github.com/QuantumNous/new-api/common"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func UploadAgentAvatar(c *gin.Context) {
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
	url, err := apservice.StoreAgentAvatar(file.Filename, opened)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"url": url})
}

func GetAgentAvatar(c *gin.Context) {
	path, err := apservice.AgentAvatarPath(c.Param("file"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			common.ApiErrorMsg(c, "resource not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	c.File(path)
}
