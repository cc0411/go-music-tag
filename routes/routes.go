package routes

import (
	"go-music-tag/config"
	"go-music-tag/handlers"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	cfg := config.GetConfig()

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Range")
		c.Header("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 静态文件和首页
	frontendExists := false
	if _, err := os.Stat("./frontend"); err == nil {
		if _, err := os.Stat("./frontend/index.html"); err == nil {
			frontendExists = true
		}
	}

	if frontendExists {
		r.Static("/static", "./frontend")
		r.GET("/", func(c *gin.Context) {
			c.File("./frontend/index.html")
		})
	} else {
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": "Music Tag Manager API",
				"version": "1.0.0",
				"port":    cfg.Server.Port,
			})
		})
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format("2006-01-02 15:04:05"),
			"port":   cfg.Server.Port,
		})
	})

	// 初始化 Handler
	musicHandler := handlers.NewMusicHandlerLazy()

	// API 路由组
	v1 := r.Group("/api/v1")
	{
		// WebDAV 配置
		v1.GET("/webdav/config", musicHandler.GetWebDAVConfig)
		v1.POST("/webdav/config", musicHandler.SaveWebDAVConfig)
		v1.DELETE("/webdav/config", musicHandler.DeleteWebDAVConfig)
		v1.POST("/webdav/test", musicHandler.TestWebDAVConfig)

		// 扫描管理
		v1.POST("/scan", musicHandler.Scan)
		v1.GET("/scan/status", musicHandler.GetScanStatus)
		v1.GET("/scan/logs", musicHandler.GetScanLogs)

		// 音乐管理
		v1.GET("/music", musicHandler.List)
		v1.GET("/music/:id", musicHandler.Get)
		v1.PUT("/music/:id", musicHandler.Update)
		v1.POST("/music/batch", musicHandler.BatchUpdate)
		v1.DELETE("/music/:id", musicHandler.Delete)
		v1.DELETE("/music", musicHandler.DeleteAll)
		v1.GET("/music/search", musicHandler.Search)
		v1.GET("/music/playlist", musicHandler.GetPlaylist)
		v1.GET("/music/:id/cover", musicHandler.GetCover)
		v1.GET("/music/:id/play", musicHandler.Play)
		v1.GET("/music/:id/lyrics", musicHandler.GetLyrics)

		// 歌词和封面获取（确保这些只出现一次！）
		v1.POST("/music/:id/fetch-lyrics", musicHandler.FetchLyrics)
		v1.POST("/music/:id/fetch-cover", musicHandler.FetchCover)

		// 统计信息
		v1.GET("/statistics", musicHandler.Statistics)
	}

	// 404 处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Path not found: " + c.Request.URL.Path,
		})
	})

	return r
}
