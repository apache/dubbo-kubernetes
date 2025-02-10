package deployer

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"os"
	template2 "text/template"
)

const (
	deployTemplateFile = "deploy.tpl"
)

//go:embed deploy.tpl
var deployTemplate string

type deploy struct{}

type Deployment struct {
	Name       string
	Namespace  string
	Image      string
	Port       int
	TargetPort int
	NodePort   int
}

type DeployerOption func(deployer *deploy)

func NewDeployer(opts ...DeployerOption) *deploy {
	d := &deploy{}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *deploy) Deploy(ctx context.Context, dc *dubbo.DubboConfig, option ...sdk.DeployOption) (sdk.DeploymentResult, error) {
	ns := dc.Deploy.Namespace

	var err error
	text, err := util.LoadTemplate(dc.Root, deployTemplateFile, deployTemplate)
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	targetPort := dc.Deploy.TargetPort
	if targetPort == 0 {
		targetPort = dc.Deploy.ContainerPort
	}

	path := dc.Root + "/" + dc.Deploy.Output
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "WARNING! The file already exists in this directory and has been overwritten.")
	}

	out, err := os.Create(path)
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	t := template2.Must(template2.New("deployTemplate").Parse(text))
	err = t.Execute(out, Deployment{
		Name:       dc.Name,
		Namespace:  ns,
		Image:      dc.Image,
		Port:       dc.Deploy.ContainerPort,
		TargetPort: targetPort,
		NodePort:   dc.Deploy.NodePort,
	})
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	return sdk.DeploymentResult{
		Status:    sdk.Deployed,
		Namespace: ns,
	}, nil
}
