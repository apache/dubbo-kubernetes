package bootstrap

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"os"
	"strconv"
	"strings"
)

type MetadataOptions struct {
}

func ReadPodAnnotations(path string) (map[string]string, error) {
	if path == "" {
		path = constants.PodInfoAnnotationsPath
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDownwardAPI(string(b))
}

func ParseDownwardAPI(i string) (map[string]string, error) {
	res := map[string]string{}
	for _, line := range strings.Split(i, "\n") {
		sl := strings.SplitN(line, "=", 2)
		if len(sl) != 2 {
			continue
		}
		key := sl[0]
		// Strip the leading/trailing quotes
		val, err := strconv.Unquote(sl[1])
		if err != nil {
			return nil, fmt.Errorf("failed to unquote %v: %v", sl[1], err)
		}
		res[key] = val
	}
	return res, nil
}

func GetNodeMetaData(options MetadataOptions) error {
	return nil
}
