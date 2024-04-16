package handler

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
)

func GetInstances(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := rt.ResourceManager()
		dataplaneList := &mesh.DataplaneResourceList{}
		if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(dataplaneList))
	}
}

func GetMetas(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := rt.ResourceManager()
		metadataList := &mesh.MetaDataResourceList{}
		if err := manager.List(rt.AppContext(), metadataList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(metadataList))
	}
}

func SearchInstances(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.SearchInstanceReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		instances, pageData, err := service.SearchInstances(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		pageRes := &model.PageData{}
		c.JSON(http.StatusOK, model.NewSuccessResp(pageRes.WithData(instances).WithTotal(int(pageData.Total)).WithCurPage(req.CurPage).WithPageSize(req.PageSize)))
	}
}
