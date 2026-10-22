package router

import (
	"embed"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the embedded frontend assets for both themes.
type ThemeAssets struct {
	DefaultBuildFS   embed.FS
	DefaultIndexPage []byte
	ClassicBuildFS   embed.FS
	ClassicIndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	router.Any("/canvas-app", proxyInfiniteCanvas)
	router.Any("/canvas-app/*filepath", proxyInfiniteCanvas)

	defaultFS := common.EmbedFolder(assets.DefaultBuildFS, "web/default/dist")
	classicFS := common.EmbedFolder(assets.ClassicBuildFS, "web/classic/dist")
	themeFS := common.NewThemeAwareFS(defaultFS, classicFS)

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", themeFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		if common.GetTheme() == "classic" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", assets.ClassicIndexPage)
		} else {
			c.Data(http.StatusOK, "text/html; charset=utf-8", assets.DefaultIndexPage)
		}
	})
}

func proxyInfiniteCanvas(c *gin.Context) {
	upstream := strings.TrimSpace(os.Getenv("INFINITE_CANVAS_UPSTREAM"))
	if upstream == "" {
		upstream = "http://infinite-canvas:3000"
	}
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		c.String(http.StatusBadGateway, "invalid infinite canvas upstream")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		trimmed := strings.TrimPrefix(req.URL.Path, "/canvas-app")
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		req.URL.Path = trimmed
		req.URL.RawPath = ""
		originalDirector(req)
		req.Header.Set("Accept-Encoding", "identity")
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		path := resp.Request.URL.Path
		contentType := resp.Header.Get("Content-Type")
		if path == "/" || strings.HasSuffix(path, "/config.js") || strings.Contains(contentType, "text/html") {
			resp.Header.Set("Cache-Control", "no-cache")
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "infinite canvas is unavailable", http.StatusBadGateway)
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
