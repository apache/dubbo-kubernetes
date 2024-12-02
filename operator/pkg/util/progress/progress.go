package progress

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/cheggaaa/pb/v3"
	"io"
	"sync"
)

type InstallState int

const (
	StateInstalling InstallState = iota
	StatePruning
	StateComplete
	StateUninstallComplete
)

const inProgress = `{{ yellow (cycle . "-" "-" " ") }}`

type ManifestInfo struct {
	report   func()
	err      string
	waiting  []string
	finished bool
	mu       sync.Mutex
}

type Info struct {
	components map[string]*ManifestInfo
	state      InstallState
	bar        *pb.ProgressBar
	mu         sync.Mutex
	template   string
}

func NewInfo() *Info {
	return &Info{
		components: map[string]*ManifestInfo{},
		bar:        createBar(),
	}
}

func (info *Info) reportProgress(componentName string) func() {
	return func() {
		compName := component.Name(componentName)
		cliName := component.UserFacingCompName(compName)
		info.mu.Lock()
		defer info.mu.Unlock()
		comp := info.components[componentName]
		comp.mu.Lock()
		finished := comp.finished
		compErr := comp.err
		comp.mu.Unlock()
		successIcon := "🎉"
		if icon, found := component.Icons[compName]; found {
			successIcon = icon
		}
		if finished || compErr != "" {
			if finished {
				info.SetMessage(fmt.Sprintf(`{{ green "✔" }} %s install Completed %s`, cliName, successIcon), true)
			} else {
				info.SetMessage(fmt.Sprintf(`{{ read "✘" }} %s encountered an error: %s`, cliName, compErr), true)
			}
			delete(info.components, componentName)
			info.bar = createBar()
			return
		}
	}
}

func (info *Info) SetMessage(status string, finish bool) {
	if !info.bar.GetBool(pb.Terminal) && status == info.template {
		return
	}
	info.template = status
	info.bar.SetTemplateString(info.template)
	if finish {
		info.bar.Finish()
	}
	info.bar.Write()
}

func createBar() *pb.ProgressBar {
	var testWriter *io.Writer
	bar := pb.New(0)
	bar.Set(pb.Static, true)
	if testWriter != nil {
		bar.SetWriter(*testWriter)
	}
	bar.Start()
	if !bar.GetBool(pb.Terminal) {
		bar.Set(pb.ReturnSymbol, "\n")
	}
	return bar
}
