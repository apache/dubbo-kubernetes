package validate

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/cmd/validation"
	operator "github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	errFiles = errors.New(`error: you must specify resources by --filename.
Example resource specifications include:
   '-f default.yaml'
   '--filename=default.json'`)

	validFields = map[string]struct{}{
		"apiVersion": {},
		"kind":       {},
		"metadata":   {},
		"spec":       {},
		"status":     {},
	}
)

type validator struct{}

func (v *validator) validateFile(path string, dubboNamespace *string, defaultNamespace string, reader io.Reader, writer io.Writer) (validation.Warning, error) {
	yamlReader := yaml.NewYAMLReader(bufio.NewReader(reader))
	var errs error
	var warnings validation.Warnings
	for {
		doc, err := yamlReader.Read()
		if err == io.EOF {
			return warnings, errs
		}
		if err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("failed to decode file %s: ", path)))
			return warnings, errs
		}
		if len(doc) == 0 {
			continue
		}
		out := map[string]any{}
		if err := yaml.UnmarshalStrict(doc, &out); err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("failed to decode file %s: ", path)))
			return warnings, errs
		}
		un := unstructured.Unstructured{Object: out}
		warning, err := v.validateResource(*dubboNamespace, &un, writer)
		if err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("%s/%s/%s:",
				un.GetKind(), un.GetNamespace(), un.GetName())))
		}
		if warning != nil {
			warnings = multierror.Append(warnings, multierror.Prefix(warning, fmt.Sprintf("%s/%s/%s:",
				un.GetKind(), un.GetNamespace(), un.GetName())))
		}
	}
}

func (v *validator) validateResource(istioNamespace string, un *unstructured.Unstructured, writer io.Writer) (validation.Warnings, error) {
	g := config.GroupVersionKind{
		Group:   un.GroupVersionKind().Group,
		Version: un.GroupVersionKind().Version,
		Kind:    un.GroupVersionKind().Kind,
	}
	var errs error
	if errs != nil {
		return nil, errs
	}

	if un.GetAPIVersion() == operator.DubboOperatorGVK.GroupVersion().String() {
		if un.GetKind() == operator.DubboOperatorGVK.Kind {
			if err := checkFields(un); err != nil {
				return nil, err
			}
			warnings, err := operatorvalidate.ParseAndValidateIstioOperator(un.Object, nil)
			if err != nil {
				return nil, err
			}
			if len(warnings) > 0 {
				return validation.Warning(warnings.ToError()), nil
			}
		}
	}
	return nil, nil
}

func checkFields(un *unstructured.Unstructured) error {
	var errs error
	for key := range un.Object {
		if _, ok := validFields[key]; !ok {
			errs = multierror.Append(errs, fmt.Errorf("unknown field %q", key))
		}
	}
	return errs
}

func NewValidateCommand(ctx cli.Context) *cobra.Command {
	vc := &cobra.Command{
		Use:   "validate -f FILENAME [options]",
		Short: "Validate Dubbo rules files",
		Long:  "The validate command is used to validate the dubbo related rule file",
		Example: `  # Validate current deployments under 'default' namespace with in the cluster
  kubectl get deployments -o yaml | dubboctl validate -f -

  # Validate current services under 'default' namespace with in the cluster
  kubectl get services -o yaml | dubboctl validate -f -
`,
		Args:    cobra.NoArgs,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			dn := ctx.DubboNamespace()
			return validateFiles(&dn, nil, nil)
		},
	}
	return vc
}

func validateFiles(dubboNamespace *string, files []string, writer io.Writer) error {
	if len(files) == 0 {
		return errFiles
	}
	v := &validator{}
	var errs error
	var reader io.ReadCloser

	processFile := func(path string) {
		var err error
		if path == "-" {
			reader = io.NopCloser(os.Stdin)
		} else {
			reader, err = os.Open(path)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("cannot read file %q: %v", path, err))
				return
			}
		}
		warning, err := v.validateFile(path, istioNamespace, defaultNamespace, reader, writer)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		err = reader.Close()
		if err != nil {
			log.Infof("file: %s is not closed: %v", path, err)
		}
		warningsByFilename[path] = warning
	}
	return nil
}
