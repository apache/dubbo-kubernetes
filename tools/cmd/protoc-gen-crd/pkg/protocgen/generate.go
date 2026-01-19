package protocgen

import (
	"fmt"
	"io"
	"os"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"
)

// GenerateFn is a function definition for encapsulating the ore logic of code generation.
type GenerateFn func(req *plugin.CodeGeneratorRequest) (*plugin.CodeGeneratorResponse, error)

// Generate is a wrapper for a main function of a protoc generator plugin.
func Generate(fn GenerateFn) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatal("Unable to read input proto: %v\n", err)
	}

	var request plugin.CodeGeneratorRequest
	if err = proto.Unmarshal(data, &request); err != nil {
		fatal("Unable to parse input proto: %v\n", err)
	}

	response, err := fn(&request)
	if err != nil {
		fatal("%v\n", err)
	}

	data, err = proto.Marshal(response)
	if err != nil {
		fatal("Unable to serialize output proto: %v\n", err)
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		fatal("Unable to write output proto: %v\n", err)
	}
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format, args...)
	os.Exit(1)
}
