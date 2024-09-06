package db

type NodeDataInfo struct {
	Id              uint32 `json:"id"`
	NodeName        string `json:"nodeName"`
	NodeIP          string `json:"nodeIP"`
	Sn              string `json:"sn"`
	ClusterName     string `json:"clusterName"`
	ModuleName      string `json:"moduleName"`
	Reason          string `json:"reason"`
	Restart         uint32 `json:"restart"`
	Repair          uint32 `json:"repair"`
	RepairTicketUrl string `json:"repairTicketUrl"`
	FirstDate       string `json:"firstDate"`
	LastDate        string `json:"lastDate"`
	CreateTime      string `json:"createTime"`
	UpdateTime      string `json:"updateTime"`
}

type PodDataInfo struct {
	Id              uint32 `json:"id"`
	PodName         string `json:"podName"`
	PodIP           string `json:"podIP"`
	Sn              string `json:"sn"`
	NodeName        string `json:"nodeName"`
	ClusterName     string `json:"clusterName"`
	ModuleName      string `json:"moduleName"`
	Reason          string `json:"reason"`
	Restart         int32  `json:"restart"`
	Repair          int32  `json:"repair"`
	RepairTicketUrl string `json:"repairTicketUrl"`
	FirstDate       string `json:"firstDate"`
	LastDate        string `json:"lastDate"`
	CreateTime      string `json:"createTime"`
	UpdateTime      string `json:"updateTime"`
}