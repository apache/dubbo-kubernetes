package helm

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/manifests"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/parts"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
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
	profilesDirName     = "profiles"
)

type Warnings = util.Errors

func Reader(namespace string, directory string, dop values.Map) ([]manifest.Manifest, util.Errors, error) {
	vals, ok := dop.GetPathMap("spec.values")
	if !ok {
		return nil, nil, fmt.Errorf("failed to get values from dop: %v", ok)
	}
	path := pathJoin("charts", directory)
	pkgPath := dop.GetPathString("spec.packagePath")
	f := manifests.BuiltinDir(pkgPath)
	chrt, err := loadChart(f, path)
	output, warnings, err := readerChart(namespace, vals, chrt)
	if err != nil {
		return nil, nil, fmt.Errorf("render chart: %v", err)
	}
	mfs, err := manifest.Parse(output)
	return mfs, warnings, err
}

func readerChart(namespace string, chrtVals values.Map, chrt *chart.Chart) ([]string, Warnings, error) {
	opts := chartutil.ReleaseOptions{
		Name:      "dubbo",
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
		if strings.HasSuffix(k, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]string, 0, len(keys))
	for _, k := range keys {
		res = append(res, parts.SplitString(files[k])...)
	}
	slices.SortBy(crdFiles, func(a chart.CRD) string {
		return a.Name
	})
	for _, crd := range crdFiles {
		res = append(res, parts.SplitString(string(crd.File.Data))...)
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
	return loader.LoadFiles(bfs)
}

func stripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
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

func readProfiles(chartsDir string) (map[string]bool, error) {
	profiles := map[string]bool{}
	f := manifests.BuiltinDir(chartsDir)
	dir, err := fs.ReadDir(f, profilesDirName)
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		trimmedString := strings.TrimSuffix(f.Name(), ".yaml")
		if f.Name() != trimmedString {
			profiles[trimmedString] = true
		}
	}
	return profiles, nil
}

func ListProfiles(charts string) ([]string, error) {
	profiles, err := readProfiles(charts)
	if err != nil {
		return nil, err
	}
	return stringBoolMapToSlice(profiles), nil
}

func stringBoolMapToSlice(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			s = append(s, k)
		}
	}
	return s
}
