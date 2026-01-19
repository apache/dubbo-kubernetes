package protomodel

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type FileDescriptor struct {
	*descriptor.FileDescriptorProto
	Parent       *PackageDescriptor
	AllMessages  []*MessageDescriptor                               // All the messages defined in this file
	AllEnums     []*EnumDescriptor                                  // All the enums defined in this file
	Messages     []*MessageDescriptor                               // Top-level messages defined in this file
	Enums        []*EnumDescriptor                                  // Top-level enums defined in this file
	Services     []*ServiceDescriptor                               // All services defined in this file
	Dependencies []*FileDescriptor                                  // Files imported by this file
	locations    map[pathVector]*descriptor.SourceCodeInfo_Location // Provenance
	Matter       FrontMatter                                        // Title, overview, homeLocation, front_matter
}

func newFileDescriptor(desc *descriptor.FileDescriptorProto, parent *PackageDescriptor) *FileDescriptor {
	f := &FileDescriptor{
		FileDescriptorProto: desc,
		locations:           make(map[pathVector]*descriptor.SourceCodeInfo_Location, len(desc.GetSourceCodeInfo().GetLocation())),
		Parent:              parent,
	}

	// put all the locations in a map for quick lookup
	for _, loc := range desc.GetSourceCodeInfo().GetLocation() {
		if len(loc.Path) > 0 {
			pv := newPathVector(int(loc.Path[0]))
			for _, v := range loc.Path[1:] {
				pv = pv.append(int(v))
			}
			f.locations[pv] = loc
		}
	}

	path := newPathVector(messagePath)
	for i, md := range desc.MessageType {
		f.Messages = append(f.Messages, newMessageDescriptor(md, nil, f, path.append(i)))
	}

	path = newPathVector(enumPath)
	for i, e := range desc.EnumType {
		f.Enums = append(f.Enums, newEnumDescriptor(e, nil, f, path.append(i)))
	}

	path = newPathVector(servicePath)
	for i, s := range desc.Service {
		f.Services = append(f.Services, newServiceDescriptor(s, f, path.append(i)))
	}

	// Find title/overview/etc content in comments and store it explicitly.
	loc := f.find(newPathVector(packagePath))
	if loc != nil && loc.LeadingDetachedComments != nil {
		f.Matter = extractFrontMatter(f.GetName(), loc, f)
	}

	// get the transitive close of all messages and enums
	f.aggregateMessages(f.Messages)
	f.aggregateEnums(f.Enums)

	return f
}

func (f *FileDescriptor) find(path pathVector) *descriptor.SourceCodeInfo_Location {
	loc := f.locations[path]
	return loc
}

func (f *FileDescriptor) aggregateMessages(messages []*MessageDescriptor) {
	f.AllMessages = append(f.AllMessages, messages...)
	for _, msg := range messages {
		f.aggregateMessages(msg.Messages)
		f.aggregateEnums(msg.Enums)
	}
}

func (f *FileDescriptor) aggregateEnums(enums []*EnumDescriptor) {
	f.AllEnums = append(f.AllEnums, enums...)
}
