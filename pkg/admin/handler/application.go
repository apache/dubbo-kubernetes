package handler

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
)

func GetApplicationDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetApplicationDetail(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}
