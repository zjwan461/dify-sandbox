package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/langgenius/dify-sandbox/internal/middleware"
	"github.com/langgenius/dify-sandbox/internal/static"
)

func Setup(Router *gin.Engine) {
	PublicGroup := Router.Group("")
	PrivateGroup := Router.Group("/v1/sandbox/")

	PrivateGroup.Use(middleware.Auth())

	{
		// health check
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}

	InitRunRouter(PrivateGroup)
	InitDependencyRouter(PrivateGroup)
	InitFileRouter(PrivateGroup)
	InitCommandRouter(PrivateGroup)
}

func InitFileRouter(Router *gin.RouterGroup) {
	fileRouter := Router.Group("file")
	{
		fileRouter.POST("upload", UploadFileController)
		fileRouter.POST("download", DownloadFileController)
		fileRouter.POST("delete", DeleteFileController)
	}
}

func InitCommandRouter(Router *gin.RouterGroup) {
	// /v1/sandbox/run/command — launch a previously uploaded file via a
	// curated set of executables. Sits next to /run so the API surface
	// around script execution stays grouped under "run".
	commandRouter := Router.Group("run")
	{
		commandRouter.POST(
			"command",
			middleware.MaxRequest(static.GetDifySandboxGlobalConfigurations().MaxRequests),
			middleware.MaxWorker(static.GetDifySandboxGlobalConfigurations().MaxWorkers),
			middleware.TraceMiddleware(),
			RunCommandController,
		)
	}
}

func InitDependencyRouter(Router *gin.RouterGroup) {
	dependencyRouter := Router.Group("dependencies")
	{
		dependencyRouter.GET("", GetDependencies)
		dependencyRouter.POST("update", UpdateDependencies)
		dependencyRouter.GET("refresh", RefreshDependencies)
	}
}

func InitRunRouter(Router *gin.RouterGroup) {
	runRouter := Router.Group("")
	{
		runRouter.POST(
			"run",
			middleware.MaxRequest(static.GetDifySandboxGlobalConfigurations().MaxRequests),
			middleware.MaxWorker(static.GetDifySandboxGlobalConfigurations().MaxWorkers),
			middleware.TraceMiddleware(),
			RunSandboxController,
		)
	}
}
