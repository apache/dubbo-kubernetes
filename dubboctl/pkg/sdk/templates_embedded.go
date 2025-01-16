package sdk

import (
	"archive/zip"
	"bytes"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/generate"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
)

func newEmbeddedTemplatesFS() util.Filesystem {
	archive, err := zip.NewReader(bytes.NewReader(generate.TemplatesZip), int64(len(generate.TemplatesZip)))
	if err != nil {
		panic(err)
	}
	return util.NewZipFS(archive)
}

var EmbeddedTemplatesFS = newEmbeddedTemplatesFS()
