package render

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/cmd/validation"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"io"
	"os"
	"strings"
)

func MergeInputs(filenames []string, flags []string) ([]values.Map, error) {
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
	}
	return nil, nil
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
func GenerateManifest(files []string, setFlags []string, logger clog.Logger) ([]manifest.ManifestSet, values.Map, error) {
	merged, err := MergeInputs(files, setFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("merge inputs: %v", err)
	}
}

func validateDubboOperator(dop values.Map, logger clog.Logger) error {
	warnings, errs := validation.ParseAndValidateDubboOperator(dop)
	if err := errs.ToError(); err != nil {
		return err
	}
	if logger != nil {
		for _, w := range warnings {
			logger.LogAndErrorf("%s %v", "❗", w)
		}
	}
	return nil
}
