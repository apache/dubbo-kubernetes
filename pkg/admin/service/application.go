package service

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetApplicationDetail(rt core_runtime.Runtime, req *model.ApplicationDetailReq) ([]*model.ApplicationDetailResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	appMap := make(map[string]*model.ApplicationDetail)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Meta.GetLabels()[mesh_proto.AppTag]
		var applicationDetail *model.ApplicationDetail
		if _, ok := appMap[appName]; ok {
			applicationDetail = appMap[appName]
		} else {
			applicationDetail = model.NewApplicationDetail()
		}
		applicationDetail.Merge(dataplane)
		appMap[appName] = applicationDetail
	}

	resp := make([]*model.ApplicationDetailResp, 0, len(appMap))
	for appName, appDetail := range appMap {
		respItem := &model.ApplicationDetailResp{
			AppName: appName,
		}
		resp = append(resp, respItem.FromApplicationDetail(appDetail))
	}

	return resp, nil
}
