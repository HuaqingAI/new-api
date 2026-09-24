package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWebIndexUsesConfiguredBrandingForSocialPreviews(t *testing.T) {
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	previousSystemName := common.SystemName
	previousLogo := common.Logo
	common.SystemName = "Huaqing \"AI\" <Assistant>"
	common.Logo = "/uploads/huaqing-logo.png"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.SystemName = previousSystemName
		common.Logo = previousLogo
		common.OptionMapRWMutex.Unlock()
	})

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "http://hth.huaqing.run/pricing", nil)
	context.Request.Header.Set("X-Forwarded-Proto", "https")

	indexPage := []byte(strings.Join([]string{
		"<html><head>",
		webBrandingStart,
		"<title>New API</title>",
		webBrandingEnd,
		generatedFavicon,
		"</head><body></body></html>",
	}, ""))
	rendered := string(renderWebIndex(context, indexPage))

	require.Contains(t, rendered, "<title>Huaqing &#34;AI&#34; &lt;Assistant&gt;</title>")
	assert.Contains(t, rendered, "<meta property=\"og:title\" content=\"Huaqing &#34;AI&#34; &lt;Assistant&gt;\" />")
	assert.Contains(t, rendered, "<meta property=\"og:image\" content=\"https://hth.huaqing.run/uploads/huaqing-logo.png\" />")
	assert.Contains(t, rendered, "<meta property=\"og:url\" content=\"https://hth.huaqing.run/pricing\" />")
	assert.NotContains(t, rendered, generatedFavicon)
}

func TestRenderWebIndexFallsBackToDefaultBranding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	previousSystemName := common.SystemName
	previousLogo := common.Logo
	common.SystemName = ""
	common.Logo = "data:image/png;base64,invalid"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.SystemName = previousSystemName
		common.Logo = previousLogo
		common.OptionMapRWMutex.Unlock()
	})

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "https://example.com/", nil)
	indexPage := []byte(webBrandingStart + "<title>placeholder</title>" + webBrandingEnd)
	rendered := string(renderWebIndex(context, indexPage))

	assert.Contains(t, rendered, "<title>New API</title>")
	assert.Contains(t, rendered, "<meta property=\"og:image\" content=\"https://example.com/logo.png\" />")
}
