package model

type ApplicationDetailReq struct {
	AppName string `json:"appName"`
}

type ApplicationDetailResp struct {
	AppName          string   `json:"appName"`
	AppTypes         []string `json:"appTypes"`
	DeployClusters   []string `json:"deployClusters"`
	DubboPorts       []string `json:"dubboPorts"`
	DubboVersions    []string `json:"dubboVersions"`
	Images           []string `json:"images"`
	RegisterClusters []string `json:"registerClusters"`
	RegisterModes    []string `json:"registerModes"`
	RPCProtocols     []string `json:"rpcProtocols"`
	SerialProtocols  []string `json:"serialProtocols"`
	Workloads        []string `json:"workloads"`
}
