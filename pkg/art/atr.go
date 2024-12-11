package art

import (
	_ "embed"
	"github.com/fatih/color"
)

//go:embed dubbo-ascii.txt
var dubboASCIIArt string

func DubboArt() string {
	return dubboASCIIArt
}

func DubboColoredArt() string {
	return color.New(color.FgHiCyan).Add(color.Bold).Sprint(dubboASCIIArt)
}
