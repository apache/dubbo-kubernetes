package parse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type PolicyConfig struct {
	Package             string
	Name                string
	NameLower           string
	Plural              string
	SkipRegistration    bool
	SingularDisplayName string
	PluralDisplayName   string
	Path                string
	AlternativeNames    []string
	GoModule            string
}

func Policy(path string) (PolicyConfig, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return PolicyConfig{}, err
	}

	policyName := strings.Split(filepath.Base(path), ".")[0]
	var mainStruct *ast.TypeSpec
	var mainComment *ast.CommentGroup
	var packageName string

	ast.Inspect(f, func(n ast.Node) bool {
		if file, ok := n.(*ast.File); ok {
			packageName = file.Name.String()
			return true
		}
		if gd, ok := n.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			for _, spec := range gd.Specs {
				if strings.ToLower(spec.(*ast.TypeSpec).Name.String()) == policyName {
					mainStruct = spec.(*ast.TypeSpec)
					mainComment = gd.Doc
					return false
				}
			}
			return false
		}
		return false
	})

	markers, err := parseMarkers(mainComment)
	if err != nil {
		return PolicyConfig{}, err
	}

	return newPolicyConfig(packageName, mainStruct.Name.String(), markers)
}

func parseMarkers(cg *ast.CommentGroup) (map[string]string, error) {
	result := map[string]string{}
	for _, comment := range cg.List {
		if !strings.HasPrefix(comment.Text, "// +") {
			continue
		}
		trimmed := strings.TrimPrefix(comment.Text, "// +")
		mrkr := strings.Split(trimmed, "=")
		if len(mrkr) != 2 {
			return nil, errors.Errorf("marker %s has wrong format", trimmed)
		}
		result[mrkr[0]] = mrkr[1]
	}
	return result, nil
}

func parseBool(markers map[string]string, key string) (bool, bool) {
	if v, ok := markers[key]; ok {
		vbool, err := strconv.ParseBool(v)
		if err != nil {
			return false, false
		}
		return vbool, true
	}

	return false, false
}

func newPolicyConfig(pkg, name string, markers map[string]string) (PolicyConfig, error) {
	res := PolicyConfig{
		Package:             pkg,
		Name:                name,
		NameLower:           strings.ToLower(name),
		SingularDisplayName: core_model.DisplayName(name),
		PluralDisplayName:   core_model.PluralType(core_model.DisplayName(name)),
		AlternativeNames:    []string{strings.ToLower(name)},
	}

	if v, ok := parseBool(markers, "dubbo:policy:skip_registration"); ok {
		res.SkipRegistration = v
	}

	if v, ok := markers["dubbo:policy:singular_display_name"]; ok {
		res.SingularDisplayName = v
		res.PluralDisplayName = core_model.PluralType(v)
	}

	if v, ok := markers["dubbo:policy:plural"]; ok {
		res.Plural = v
	} else {
		res.Plural = core_model.PluralType(res.Name)
	}

	res.Path = strings.ToLower(res.Plural)

	return res, nil
}
