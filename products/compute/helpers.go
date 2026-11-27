package compute

import "google.golang.org/protobuf/proto"

func protoString(s string) *string    { return proto.String(s) }
func protoUint64(v uint64) *uint64    { return proto.Uint64(v) }
func protoInt32(v int32) *int32       { return proto.Int32(v) }
func protoInt64(v int64) *int64       { return proto.Int64(v) }
func protoBool(v bool) *bool          { return proto.Bool(v) }
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func protoUint32(v uint32) *uint32 { return proto.Uint32(v) }
