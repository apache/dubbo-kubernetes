package protomodel

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type Mode string

var (
	ModeUnset   Mode
	ModeFile    Mode = "file"
	ModePackage Mode = "package"
	ModeNone    Mode = "none"
)

type FrontMatter struct {
	Title        string
	Overview     string
	Description  string
	HomeLocation string
	Extra        []string
	Location     LocationDescriptor
	Mode         Mode
}

const (
	titleTag       = "$title: "
	overviewTag    = "$overview: "
	descriptionTag = "$description: "
	locationTag    = "$location: "
	frontMatterTag = "$front_matter: "
	modeTag        = "$mode: "
)

func checkSingle(name string, old string, line string, tag string) string {
	result := line[len(tag):]
	if old != "" {
		_, _ = fmt.Fprintf(os.Stderr, "%v has more than one %v: %v\n", name, tag, result)
	}
	return result
}

func extractFrontMatter(name string, loc *descriptor.SourceCodeInfo_Location, file *FileDescriptor) FrontMatter {
	title := ""
	overview := ""
	description := ""
	homeLocation := ""
	mode := ""
	var extra []string

	for _, para := range loc.LeadingDetachedComments {
		lines := strings.Split(para, "\n")
		for _, l := range lines {
			l = strings.Trim(l, " ")

			if strings.HasPrefix(l, "$") {
				if strings.HasPrefix(l, titleTag) {
					title = checkSingle(name, title, l, titleTag)
				} else if strings.HasPrefix(l, overviewTag) {
					overview = checkSingle(name, overview, l, overviewTag)
				} else if strings.HasPrefix(l, descriptionTag) {
					description = checkSingle(name, description, l, descriptionTag)
				} else if strings.HasPrefix(l, locationTag) {
					homeLocation = checkSingle(name, homeLocation, l, locationTag)
				} else if strings.HasPrefix(l, frontMatterTag) {
					// old way to specify custom front-matter
					extra = append(extra, l[len(frontMatterTag):])
				} else if strings.HasPrefix(l, modeTag) {
					mode = checkSingle(name, mode, l, modeTag)
				} else {
					extra = append(extra, l[1:])
				}
			}
		}
	}

	return FrontMatter{
		Title:        title,
		Overview:     overview,
		Description:  description,
		HomeLocation: homeLocation,
		Mode:         checkMode(mode),
		Extra:        extra,
		Location:     newLocationDescriptor(loc, file),
	}
}

func checkMode(single string) Mode {
	switch Mode(single) {
	case ModeUnset, ModeFile, ModePackage, ModeNone:
		return Mode(single)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %v\n", single)
		return ModeUnset
	}
}
