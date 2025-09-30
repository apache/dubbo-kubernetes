package inject

import (
	"text/template"
)

var InjectionFuncmap = createInjectionFuncmap()

func createInjectionFuncmap() template.FuncMap {
	return template.FuncMap{}
}
