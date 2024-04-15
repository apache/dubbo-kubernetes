package server

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/handler"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
)

func initRouter(r *gin.Engine, rt core_runtime.Runtime) {
	router := r.Group("/api/v1")
	{
		instance := router.Group("/instance")
		instance.GET("/search", handler.SearchInstances(rt))
	}

	{
		application := router.Group("/application")
		application.GET("/detail", handler.GetApplicationDetail(rt))
	}

	{
		dev := router.Group("/dev")
		dev.GET("/instances", handler.GetInstances(rt))
		dev.GET("/metas", handler.GetMetas(rt))
	}
}
