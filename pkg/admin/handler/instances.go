package handler

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
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
