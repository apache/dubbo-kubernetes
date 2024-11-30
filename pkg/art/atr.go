package art

import "github.com/fatih/color"

//go:embed dubbo-ascii.txt
var dubboASCIIArt string

func DubboArt() string {
	return dubboASCIIArt
}

func DubboColoredArt() {
	return color.New(color.FgHiCyan).Add(color.Bold).Sprint(dubboASCIIArt)
}
