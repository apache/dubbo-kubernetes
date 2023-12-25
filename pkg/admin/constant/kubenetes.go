package constant

const baseLabel = "dubbo.io"

const (
	ApplicationLabel = baseLabel + "/application"
	ServiceKeyLabel  = baseLabel + "/serviceKey"
	GroupLabel       = baseLabel + "/group"
	VersionLabel     = baseLabel + "/version"
)

const (
	ConfigMapType             = "ConfigMap"
	CronJobType               = "CronJob"
	DaemonSetType             = "DaemonSet"
	DeploymentType            = "Deployment"
	DeploymentConfigType      = "DeploymentConfig"
	EndpointsType             = "Endpoints"
	JobType                   = "Job"
	PodType                   = "Pod"
	ReplicationControllerType = "ReplicationController"
	ReplicaSetType            = "ReplicaSet"
	ServiceType               = "Service"
	StatefulSetType           = "StatefulSet"
)
