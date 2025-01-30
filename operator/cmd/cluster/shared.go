package cluster

import (
	"fmt"
	dopv1alpha1 "github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/helm"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type writerPrinter struct {
	writer io.Writer
}

type Printer interface {
	Printf(string, ...any)
	Println(string)
}

func NewPtrForWtr(w io.Writer) Printer {
	return &writerPrinter{writer: w}
}

func (w writerPrinter) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.writer, format, a...)
}

func (w writerPrinter) Println(s string) {
	_, _ = fmt.Fprintln(w.writer, s)
}

func OptionDeterminate(msg string, writer io.Writer) bool {
	for {
		_, _ = fmt.Fprintf(writer, "%s ", msg)
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil {
			return false
		}
		switch strings.ToUpper(resp) {
		case "Y", "YES", "y", "yes":
			return true
		case "N", "NO", "n", "no":
			return false
		}
	}
}

func NewPrinterForWriter(w io.Writer) Printer {
	return &writerPrinter{writer: w}
}

func GenerateConfig(filenames []string, setFlags []string, l clog.Logger) (string, *dopv1alpha1.DubboOperator, error) {
	if err := validateSetFlags(setFlags); err != nil {
		return "", nil, err
	}

	fy, profile, err := readYamlProfile(filenames, setFlags, l)
	if err != nil {
		return "", nil, err
	}
	return overlayYAMLStrings(profile, fy, setFlags, l)
}

func validateSetFlags(setFlags []string) error {
	for _, sf := range setFlags {
		pv := strings.Split(sf, "=")
		if len(pv) != 2 {
			return fmt.Errorf("set flag %s has incorrect format, must be path=value", sf)
		}
	}
	return nil
}

func overlayYAMLStrings(
	profile string, fy string,
	setFlags []string, l clog.Logger,
) (string, *dopv1alpha1.DubboOperator, error) {
	dopsString, dops, err := GenDOPFromProfile(profile, fy, setFlags, l)
	if err != nil {
		return "", nil, err
	}
	return dopsString, dops, nil
}

func GenDOPFromProfile(profileOrPath, fileOverlayYAML string, setFlags []string, l clog.Logger) (string, *dopv1alpha1.DubboOperator, error) {
	outYAML, err := helm.GetProfileYAML("", profileOrPath)
	if err != nil {
		return "", nil, err
	}
	finalIOP, err := unmarshalDOP(outYAML, false, l)
	if err != nil {
		return "", nil, err
	}

	if finalIOP.Spec.Profile == "" {
		finalIOP.Spec.Profile = DefaultProfileName
	}
	return util.ToYAMLWithJSONPB(finalIOP), finalIOP, nil
}

func unmarshalDOP(dopsYAML string, allowUnknownField bool, l clog.Logger) (*dopv1alpha1.DubboOperator, error) {
	dop := &dopv1alpha1.DubboOperator{}
	if err := util.UnmarshalWithJSONPB(dopsYAML, dop, allowUnknownField); err != nil {
		return nil, fmt.Errorf("could not unmarshal merged YAML: %s\n\nYAML:\n%s", err, dopsYAML)
	}
	return dop, nil
}

func readYamlProfile(filenames []string, setFlags []string, l clog.Logger) (string, string, error) {
	profile := DefaultProfileName
	fy, fp, err := ParseYAMLfilenames(filenames, l)
	if err != nil {
		return "", "", err
	}
	if fp != "" {
		profile = fp
	}
	psf := getValueForSetFlag(setFlags, "profile")
	if psf != "" {
		profile = psf
	}
	return fy, profile, nil
}

func ParseYAMLfilenames(filenames []string, l clog.Logger) (overlayYAML string, profile string, err error) {
	if filenames == nil {
		return "", "", nil
	}
	y, err := ReadLayeredYAMLs(filenames)
	if err != nil {
		return "", "", err
	}
	return y, profile, nil
}

func ReadLayeredYAMLs(filenames []string) (string, error) {
	return readLayeredYAMLs(filenames, os.Stdin)
}

func readLayeredYAMLs(filenames []string, stdinReader io.Reader) (string, error) {
	var ly string
	var stdin bool
	for _, fn := range filenames {
		var _ []byte
		var err error
		if fn == "-" {
			if stdin {
				continue
			}
			stdin = true
			_, err = ioutil.ReadAll(stdinReader)
		} else {
			_, err = ioutil.ReadFile(strings.TrimSpace(fn))
		}
		if err != nil {
			return "", err
		}
	}
	return ly, nil
}

func getValueForSetFlag(setFlags []string, path string) string {
	ret := ""
	for _, sf := range setFlags {
		p, v := getPV(sf)
		if p == path {
			ret = v
		}
	}
	return ret
}

func getPV(setFlag string) (path string, value string) {
	pv := strings.Split(setFlag, "=")
	if len(pv) != 2 {
		return setFlag, ""
	}
	path, value = strings.TrimSpace(pv[0]), strings.TrimSpace(pv[1])
	return
}
