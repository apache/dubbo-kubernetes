package render

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/manifests"
	"github.com/apache/dubbo-kubernetes/operator/cmd/validation"
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/helm"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"io"
	"os"
	"strings"
)

func MergeInputs(filenames []string, flags []string, client kube.Client) (values.Map, error) {
	ConfigBase, err := values.MapFromJSON([]byte(`{
	  "apiVersion": "install.dubbo.io/v1alpha1",
	  "kind": "DubboOperator",
	  "metadata": {},
	  "spec": {}
	}`))
	if err != nil {
		return nil, err
	}

	for i, fn := range filenames {
		var b []byte
		var err error
		if fn == "-" {
			if i != len(filenames)-1 {
				return nil, fmt.Errorf("stdin is only allowed as the last filename")
			}
			b, err = io.ReadAll(os.Stdin)
		} else {
			b, err = os.ReadFile(strings.TrimSpace(fn))
		}
		if err := checkDops(string(b)); err != nil {
			return nil, fmt.Errorf("checkDops err:%v", err)
		}
		m, err := values.MapFromYAML(b)
		if err != nil {
			return nil, fmt.Errorf("yaml Unmarshal err:%v", err)
		}
		if m["spec"] == nil {
			delete(m, "spec")
		}
		ConfigBase.MergeFrom(m)
	}

	if err := ConfigBase.SetSpecPaths(flags...); err != nil {
		return nil, err
	}

	path := ConfigBase.GetPathString("")
	profile := ConfigBase.GetPathString("spec.profile")
	value, _ := ConfigBase.GetPathMap("spec.values")
	base, err := readProfile(path, profile)
	if err != nil {
		return base, err
	}
	base.MergeFrom(ConfigBase)

	if value != nil {
		base.MergeFrom(values.Map{"spec": values.Map{"values": value}})
	}

	return base, nil
}

func readProfile(path, profile string) (values.Map, error) {
	if profile == "" {
		profile = "default"
	}
	base, err := readBuiltinProfile(path, "default")
	if err != nil {
		return nil, err
	}
	if profile == "default" {
		return base, nil
	}
	p, err := readBuiltinProfile(path, profile)
	if err != nil {
		return nil, err
	}
	base.MergeFrom(p)
	return base, nil
}

func readBuiltinProfile(path, profile string) (values.Map, error) {
	fs := manifests.BuiltinDir(path)
	file, err := fs.Open(fmt.Sprintf("profiles/%v.yaml", profile))
	if err != nil {
		return nil, fmt.Errorf("profile %q not found: %v", profile, err)
	}
	pb, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return values.MapFromYAML(pb)
}

func checkDops(s string) error {
	mfs, err := manifest.ParseMultiple(s)
	if err != nil {
		return fmt.Errorf("unable to parse file: %v", err)
	}
	if len(mfs) > 1 {
		return fmt.Errorf("")
	}
	return nil
}
func GenerateManifest(files []string, setFlags []string, logger clog.Logger, client kube.Client) ([]manifest.ManifestSet, values.Map, error) {
	var chartWarnings util.Errors
	merged, err := MergeInputs(files, setFlags, client)
	if err != nil {
		return nil, nil, fmt.Errorf("merge inputs: %v", err)
	}
	if err := validateDubboOperator(merged, client, logger); err != nil {
		return nil, nil, fmt.Errorf("validateDubboOperator err:%v", err)
	}
	allManifests := map[component.Name]manifest.ManifestSet{}
	for _, comp := range component.AllComponents {
		specs, err := comp.Get(merged)
		if err != nil {
			return nil, nil, fmt.Errorf("get component %v: %v", comp.UserFacingName, err)
		}
		for _, spec := range specs {
			compVals := applyComponentValuesToHelmValues(comp, spec, merged)
			rendered, warnings, err := helm.Reader(spec.Namespace, comp.HelmSubDir, compVals)
			if err != nil {
				return nil, nil, fmt.Errorf("helm render: %v", err)
			}
			chartWarnings = util.AppendErrs(chartWarnings, warnings)
			finalized, err := postProcess(comp, rendered, compVals)
			if err != nil {
				return nil, nil, fmt.Errorf("post process: %v", err)
			}
			mfs, found := allManifests[comp.UserFacingName]
			if found {
				mfs.Manifests = append(mfs.Manifests, finalized...)
				allManifests[comp.UserFacingName] = mfs
			} else {
				allManifests[comp.UserFacingName] = manifest.ManifestSet{
					Components: comp.UserFacingName,
					Manifests:  finalized,
				}
			}
		}
	}
	if logger != nil {
		for _, w := range chartWarnings {
			logger.LogAndErrorf("%s %v", "❗", w)
		}
	}

	val := make([]manifest.ManifestSet, 0, len(allManifests))
	for _, v := range allManifests {
		val = append(val, v)
	}
	return val, merged, nil
}

func validateDubboOperator(dop values.Map, client kube.Client, logger clog.Logger) error {
	warnings, errs := validation.ParseAndValidateDubboOperator(dop, client)
	if err := errs.ToErrors(); err != nil {
		return err
	}
	if logger != nil {
		for _, w := range warnings {
			logger.LogAndErrorf("%s %v", "❗", w)
		}
	}
	return nil
}

func applyComponentValuesToHelmValues(comp component.Component, spec apis.MetadataCompSpec, merged values.Map) values.Map {
	if spec.Namespace != "" {
		spec.Namespace = "dubbo-system"
	}
	return merged
}

func postProcess(comp component.Component, manifests []manifest.Manifest, vals values.Map) ([]manifest.Manifest, error) {
	return manifests, nil
}
