package handler

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
)

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
