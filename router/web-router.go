package router

import (
	"embed"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const (
	webBrandingStart       = "<!--web-branding-start-->"
	webBrandingEnd         = "<!--web-branding-end-->"
	defaultWebDescription  = "Unified AI API gateway and admin dashboard."
	generatedFavicon       = `<link rel="icon" href="/favicon.ico">`
	generatedFaviconClosed = `<link rel="icon" href="/favicon.ico" />`
)

// WebAssets holds the embedded dashboard frontend assets.
type WebAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets WebAssets, pluginDispatcher gin.HandlerFunc) {
	frontendFS := common.EmbedFolder(assets.BuildFS, "web/dist")

	router.NoRoute(
		pluginDispatcher,
		middleware.RouteTag("web"),
		gzip.Gzip(gzip.DefaultCompression),
		middleware.AccessTokenAudit(),
		middleware.GlobalWebRateLimit(),
		middleware.Cache(),
		static.Serve("/", frontendFS),
		func(c *gin.Context) {
			if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
				controller.RelayNotFound(c)
				return
			}
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", renderWebIndex(c, assets.IndexPage))
		},
	)
}

func renderWebIndex(c *gin.Context, indexPage []byte) []byte {
	page := string(indexPage)
	start := strings.Index(page, webBrandingStart)
	end := strings.Index(page, webBrandingEnd)
	if start == -1 || end == -1 || end < start {
		return indexPage
	}

	common.OptionMapRWMutex.RLock()
	systemName := strings.TrimSpace(common.SystemName)
	logo := strings.TrimSpace(common.Logo)
	common.OptionMapRWMutex.RUnlock()

	if systemName == "" {
		systemName = "New API"
	}
	if logo == "" {
		logo = "/logo.png"
	}

	logoURL := resolveWebURL(c, logo)
	if logoURL == "" {
		logoURL = resolveWebURL(c, "/logo.png")
	}
	pageURL := resolveWebURL(c, c.Request.URL.EscapedPath())

	escapedName := html.EscapeString(systemName)
	escapedLogoURL := html.EscapeString(logoURL)
	escapedPageURL := html.EscapeString(pageURL)
	escapedDescription := html.EscapeString(defaultWebDescription)

	branding := strings.Join([]string{
		`<link rel="icon" href="` + escapedLogoURL + `" />`,
		`<title>` + escapedName + `</title>`,
		`<meta name="title" content="` + escapedName + `" />`,
		`<meta name="description" content="` + escapedDescription + `" />`,
		`<meta property="og:type" content="website" />`,
		`<meta property="og:site_name" content="` + escapedName + `" />`,
		`<meta property="og:title" content="` + escapedName + `" />`,
		`<meta property="og:description" content="` + escapedDescription + `" />`,
		`<meta property="og:image" content="` + escapedLogoURL + `" />`,
		`<meta property="og:url" content="` + escapedPageURL + `" />`,
		`<meta name="twitter:card" content="summary" />`,
		`<meta name="twitter:title" content="` + escapedName + `" />`,
		`<meta name="twitter:description" content="` + escapedDescription + `" />`,
		`<meta name="twitter:image" content="` + escapedLogoURL + `" />`,
	}, "\n    ")

	contentStart := start + len(webBrandingStart)
	page = page[:contentStart] + "\n    " + branding + "\n    " + page[end:]
	page = strings.ReplaceAll(page, generatedFavicon, "")
	page = strings.ReplaceAll(page, generatedFaviconClosed, "")
	return []byte(page)
}

func resolveWebURL(c *gin.Context, value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			return parsed.String()
		}
		return ""
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else {
		forwardedProto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
		if forwardedProto == "http" || forwardedProto == "https" {
			scheme = forwardedProto
		}
	}

	baseURL := &url.URL{Scheme: scheme, Host: c.Request.Host, Path: "/"}
	return baseURL.ResolveReference(parsed).String()
}
