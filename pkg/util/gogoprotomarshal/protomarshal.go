package gogoprotomarshal

import (
	"strings"

	"github.com/gogo/protobuf/jsonpb" // nolint: depguard
	"github.com/gogo/protobuf/proto"  // nolint: depguard
)

// ApplyJSON unmarshals a JSON string into a proto message. Unknown fields are allowed
func ApplyJSON(js string, pb proto.Message) error {
	reader := strings.NewReader(js)
	m := jsonpb.Unmarshaler{}
	if err := m.Unmarshal(reader, pb); err != nil {
		m.AllowUnknownFields = true
		reader.Reset(js)
		return m.Unmarshal(reader, pb)
	}
	return nil
}

// ApplyJSONStrict unmarshals a JSON string into a proto message.
func ApplyJSONStrict(js string, pb proto.Message) error {
	reader := strings.NewReader(js)
	m := jsonpb.Unmarshaler{}
	return m.Unmarshal(reader, pb)
}
