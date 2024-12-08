package helm

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"sort"
	"strings"
)

const (
	// NotesFileNameSuffix is the file name suffix for helm notes.
	// see https://helm.sh/docs/chart_template_guide/notes_files/
	NotesFileNameSuffix = ".txt"
)

func readerChart(namespace string, chrt *chart.Chart) ([]string, error) {
	var s []string
	opts := chartutil.ReleaseOptions{
		Namespace: namespace,
	}
	caps := *chartutil.DefaultCapabilities
	helmVals, err := chartutil.ToRenderValues(chrt, nil, opts, &caps)
	if err != nil {
		return nil, nil
	}
	files, err := engine.Render(chrt, helmVals)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		if strings.HasPrefix(k, NotesFileNameSuffix) {
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	chrt.CRDObjects()
	if chrt.Metadata.Name == "base" {
	}

	return s, nil
}

func Reader(namespace string, dir string, dop values.Map) {
	_, ok := dop.GetPathMap("spec.values")
	if !ok {
		return
	}
}