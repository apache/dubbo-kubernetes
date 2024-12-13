package helm

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/operator/pkg/yml"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// NotesFileNameSuffix is the file name suffix for helm notes.
	// see https://helm.sh/docs/chart_template_guide/notes_files/
	NotesFileNameSuffix = ".txt"
	BaseChartName       = "base"
)

type Warnings = util.Errors

func readerChart(namespace string, chrtVals values.Map, chrt *chart.Chart) ([]string, Warnings, error) {
	opts := chartutil.ReleaseOptions{
		Namespace: namespace,
	}
	caps := *chartutil.DefaultCapabilities
	helmVals, err := chartutil.ToRenderValues(chrt, chrtVals, opts, &caps)
	if err != nil {
		return nil, nil, fmt.Errorf("converting values: %v", err)
	}
	files, err := engine.Render(chrt, helmVals)
	if err != nil {
		return nil, nil, err
	}
	crdFiles := chrt.CRDObjects()
	if chrt.Metadata.Name == BaseChartName {
		values.GetPathHelper[bool](chrtVals, "base.enableDubboConfigCRD")
	}
	var warnings Warnings
	keys := make([]string, 0, len(files))
	for k := range files {
		if strings.HasPrefix(k, NotesFileNameSuffix) {
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]string, 0, len(keys))
	for _, k := range keys {
		res = append(res, yml.SplitString(files[k])...)
	}
	slices.SortBy(crdFiles, func(a chart.CRD) string {
		return a.Name
	})
	for _, crd := range crdFiles {
		res = append(res, yml.SplitString(string(crd.File.Data))...)
	}
	return res, warnings, nil
}

func loadChart(f fs.FS, root string) (*chart.Chart, error) {
	fnames, err := getFilesRecursive(f, root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("component does not exist")
		}
		return nil, fmt.Errorf("list files: %v", err)
	}
	var bfs []*loader.BufferedFile
	for _, fn := range fnames {
		b, err := fs.ReadFile(f, fn)
		if err != nil {
			return nil, fmt.Errorf("read file:%v", err)
		}
		name := strings.ReplaceAll(stripPrefix(fn, root), string(filepath.Separator), "/")
		bf := &loader.BufferedFile{
			Name: name,
			Data: b,
		}
		bfs = append(bfs, bf)
	}
	return loader.LoadFile(bfs)
}

func stripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}

func Reader(namespace string, directory string, dop values.Map) ([]manifest.Manifest, util.Errors, error) {
	vals, ok := dop.GetPathMap("spec.values")
	if !ok {
		return nil, nil, fmt.Errorf("failed to get values from dop: %v", ok)
	}
	path := pathJoin("charts", directory)
	chrt, err := loadChart(nil, path)
	output, warnings, err := readerChart(namespace, vals, chrt)
	if err != nil {
		return nil, nil, fmt.Errorf("render chart: %v", err)
	}
	mfs, err := manifest.Parse(output)
	return mfs, warnings, nil
}

func getFilesRecursive(f fs.FS, root string) ([]string, error) {
	result := []string{}
	err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		result = append(result, path)
		return nil
	})
	return result, err
}
