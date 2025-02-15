package sdk

import (
	"archive/zip"
	"bytes"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/tpl"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
)

func newEmbeddedTemplatesFS() util.Filesystem {
	archive, err := zip.NewReader(bytes.NewReader(tpl.TemplatesZip), int64(len(tpl.TemplatesZip)))
	if err != nil {
		panic(err)
	}
	return util.NewZipFS(archive)
}

var EmbeddedTemplatesFS = newEmbeddedTemplatesFS()
