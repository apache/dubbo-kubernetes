package cli

import (
	"github.com/apache/dubbo-kubernetes/pkg/pointer"
	"github.com/ory/viper"
	"github.com/spf13/pflag"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KubeConfigFlag     = "kubeconfig"
	ContextFlag        = "context"
	NamespaceFlag      = "namespace"
	DubboNamespaceFlag = "dubbo-namespace"
	ChartFlag          = "charts"
)

type RootFlags struct {
	kubeconfig     *string
	Context        *string
	namespace      *string
	dubboNamespace *string
}

func AddRootFlags(flags *pflag.FlagSet) *RootFlags {
	rootFlags := &RootFlags{
		kubeconfig:     pointer.Of[string](""),
		Context:        pointer.Of[string](""),
		namespace:      pointer.Of[string](""),
		dubboNamespace: pointer.Of[string](""),
	}
	flags.StringVarP(rootFlags.kubeconfig, KubeConfigFlag, "c", "", "Kubernetes configuration file")
	flags.StringVar(rootFlags.Context, ContextFlag, "", "Kubernetes configuration context")
	flags.StringVarP(rootFlags.namespace, NamespaceFlag, "n", v1.NamespaceAll, "Kubernetes namespace")
	flags.StringVarP(rootFlags.dubboNamespace, DubboNamespaceFlag, "i", viper.GetString(DubboNamespaceFlag), "Dubbo system namespace")
	return rootFlags
}
