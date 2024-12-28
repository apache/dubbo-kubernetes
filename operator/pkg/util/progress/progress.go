package progress

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/cheggaaa/pb/v3"
	"io"
	"sort"
	"strings"
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

func (i *Info) createStatus(maxWidth int) string {
	comps := make([]string, 0, len(i.components))
	wait := make([]string, 0, len(i.components))
	for c, l := range i.components {
		comps = append(comps, component.UserFacingCompName(component.Name(c)))
		wait = append(wait, l.waitingResources()...)
	}
	sort.Strings(comps)
	sort.Strings(wait)
	msg := fmt.Sprintf(`Processing resources for %s.`, strings.Join(comps, ", "))
	if len(wait) > 0 {
		msg += fmt.Sprintf(` Waiting for %s`, strings.Join(wait, ", "))
	}
	prefix := inProgress
	if !i.bar.GetBool(pb.Terminal) {
		prefix = `{{ yellow "-" }} `
	}
	maxWidth -= 2
	if maxWidth > 0 && len(msg) > maxWidth {
		return prefix + msg[:maxWidth-3] + "..."
	}
	return prefix + msg
}

func (i *Info) NewComponent(comp string) *ManifestInfo {
	mi := &ManifestInfo{
		report: i.reportProgress(comp),
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.components[comp] = mi
	return mi
}
func (i *Info) reportProgress(componentName string) func() {
	return func() {
		compName := component.Name(componentName)
		cliName := component.UserFacingCompName(compName)
		i.mu.Lock()
		defer i.mu.Unlock()
		comp := i.components[componentName]
		comp.mu.Lock()
		finished := comp.finished
		compErr := comp.err
		comp.mu.Unlock()
		successIcon := "✅"
		if icon, found := component.Icons[compName]; found {
			successIcon = icon
		}
		if finished || compErr != "" {
			if finished {
				i.SetMessage(fmt.Sprintf(`{{ green "✔" }} %s install Completed %s`, cliName, successIcon), true)
			} else {
				i.SetMessage(fmt.Sprintf(`{{ read "✘" }} %s encountered an error: %s`, cliName, compErr), true)
			}
			delete(i.components, componentName)
			i.bar = createBar()
			return
		}
		i.SetMessage(i.createStatus(i.bar.Width()), false)
	}
}

func (i *Info) SetMessage(status string, finish bool) {
	if !i.bar.GetBool(pb.Terminal) && status == i.template {
		return
	}
	i.template = status
	i.bar.SetTemplateString(i.template)
	if finish {
		i.bar.Finish()
	}
	i.bar.Write()
}

func (i *Info) SetState(state InstallState) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.state = state
	switch i.state {
	case StatePruning:
		i.SetMessage(inProgress+`Pruning removed resources`, false)
	case StateComplete:
		// TODO Install provides the command for copying and running.
		i.SetMessage(`{{ green "✔" }} All Dubbo resources have been successfully installed to the cluster.`, true)
	case StateUninstallComplete:
		// TODO Uninstall provides a one-click command to delete CRDs.
		i.SetMessage(`{{ green "✔" }} All Dubbo resources have been successfully removed from the cluster.`, true)
	}
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

type ManifestInfo struct {
	report   func()
	err      string
	waiting  []string
	finished bool
	mu       sync.Mutex
}

func (mi *ManifestInfo) ReportProgress() {
	if mi == nil {
		return
	}
}

func (mi *ManifestInfo) ReportFinished() {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.finished = true
	mi.mu.Unlock()
	mi.report()
}

func (mi *ManifestInfo) ReportError(err string) {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.err = err
	mi.mu.Unlock()
	mi.report()
}

func (mi *ManifestInfo) ReportWaiting(resources []string) {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.waiting = resources
	mi.mu.Unlock()
	mi.report()
}

func (p *ManifestInfo) waitingResources() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waiting
}
