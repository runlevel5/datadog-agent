package trace

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *TraceChunk) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Priority":
			z.Priority, err = dc.ReadInt32()
			if err != nil {
				err = msgp.WrapError(err, "Priority")
				return
			}
		case "Origin":
			z.Origin, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Origin")
				return
			}
		case "Spans":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "Spans")
				return
			}
			if cap(z.Spans) >= int(zb0002) {
				z.Spans = (z.Spans)[:zb0002]
			} else {
				z.Spans = make([]*Span, zb0002)
			}
			for za0001 := range z.Spans {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						err = msgp.WrapError(err, "Spans", za0001)
						return
					}
					z.Spans[za0001] = nil
				} else {
					if z.Spans[za0001] == nil {
						z.Spans[za0001] = new(Span)
					}
					err = z.Spans[za0001].DecodeMsg(dc)
					if err != nil {
						err = msgp.WrapError(err, "Spans", za0001)
						return
					}
				}
			}
		case "Tags":
			var zb0003 uint32
			zb0003, err = dc.ReadMapHeader()
			if err != nil {
				err = msgp.WrapError(err, "Tags")
				return
			}
			if z.Tags == nil {
				z.Tags = make(map[string]string, zb0003)
			} else if len(z.Tags) > 0 {
				for key := range z.Tags {
					delete(z.Tags, key)
				}
			}
			for zb0003 > 0 {
				zb0003--
				var za0002 string
				var za0003 string
				za0002, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "Tags")
					return
				}
				za0003, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "Tags", za0002)
					return
				}
				z.Tags[za0002] = za0003
			}
		case "DroppedTrace":
			z.DroppedTrace, err = dc.ReadBool()
			if err != nil {
				err = msgp.WrapError(err, "DroppedTrace")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TraceChunk) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "Priority"
	err = en.Append(0x85, 0xa8, 0x50, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79)
	if err != nil {
		return
	}
	err = en.WriteInt32(z.Priority)
	if err != nil {
		err = msgp.WrapError(err, "Priority")
		return
	}
	// write "Origin"
	err = en.Append(0xa6, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.Origin)
	if err != nil {
		err = msgp.WrapError(err, "Origin")
		return
	}
	// write "Spans"
	err = en.Append(0xa5, 0x53, 0x70, 0x61, 0x6e, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Spans)))
	if err != nil {
		err = msgp.WrapError(err, "Spans")
		return
	}
	for za0001 := range z.Spans {
		if z.Spans[za0001] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Spans[za0001].EncodeMsg(en)
			if err != nil {
				err = msgp.WrapError(err, "Spans", za0001)
				return
			}
		}
	}
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Tags)))
	if err != nil {
		err = msgp.WrapError(err, "Tags")
		return
	}
	for za0002, za0003 := range z.Tags {
		err = en.WriteString(za0002)
		if err != nil {
			err = msgp.WrapError(err, "Tags")
			return
		}
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "Tags", za0002)
			return
		}
	}
	// write "DroppedTrace"
	err = en.Append(0xac, 0x44, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x54, 0x72, 0x61, 0x63, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBool(z.DroppedTrace)
	if err != nil {
		err = msgp.WrapError(err, "DroppedTrace")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TraceChunk) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "Priority"
	o = append(o, 0x85, 0xa8, 0x50, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79)
	o = msgp.AppendInt32(o, z.Priority)
	// string "Origin"
	o = append(o, 0xa6, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e)
	o = msgp.AppendString(o, z.Origin)
	// string "Spans"
	o = append(o, 0xa5, 0x53, 0x70, 0x61, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Spans)))
	for za0001 := range z.Spans {
		if z.Spans[za0001] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Spans[za0001].MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, "Spans", za0001)
				return
			}
		}
	}
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Tags)))
	for za0002, za0003 := range z.Tags {
		o = msgp.AppendString(o, za0002)
		o = msgp.AppendString(o, za0003)
	}
	// string "DroppedTrace"
	o = append(o, 0xac, 0x44, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x54, 0x72, 0x61, 0x63, 0x65)
	o = msgp.AppendBool(o, z.DroppedTrace)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TraceChunk) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Priority":
			z.Priority, bts, err = msgp.ReadInt32Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Priority")
				return
			}
		case "Origin":
			z.Origin, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Origin")
				return
			}
		case "Spans":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Spans")
				return
			}
			if cap(z.Spans) >= int(zb0002) {
				z.Spans = (z.Spans)[:zb0002]
			} else {
				z.Spans = make([]*Span, zb0002)
			}
			for za0001 := range z.Spans {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Spans[za0001] = nil
				} else {
					if z.Spans[za0001] == nil {
						z.Spans[za0001] = new(Span)
					}
					bts, err = z.Spans[za0001].UnmarshalMsg(bts)
					if err != nil {
						err = msgp.WrapError(err, "Spans", za0001)
						return
					}
				}
			}
		case "Tags":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Tags")
				return
			}
			if z.Tags == nil {
				z.Tags = make(map[string]string, zb0003)
			} else if len(z.Tags) > 0 {
				for key := range z.Tags {
					delete(z.Tags, key)
				}
			}
			for zb0003 > 0 {
				var za0002 string
				var za0003 string
				zb0003--
				za0002, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Tags")
					return
				}
				za0003, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Tags", za0002)
					return
				}
				z.Tags[za0002] = za0003
			}
		case "DroppedTrace":
			z.DroppedTrace, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "DroppedTrace")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *TraceChunk) Msgsize() (s int) {
	s = 1 + 9 + msgp.Int32Size + 7 + msgp.StringPrefixSize + len(z.Origin) + 6 + msgp.ArrayHeaderSize
	for za0001 := range z.Spans {
		if z.Spans[za0001] == nil {
			s += msgp.NilSize
		} else {
			s += z.Spans[za0001].Msgsize()
		}
	}
	s += 5 + msgp.MapHeaderSize
	if z.Tags != nil {
		for za0002, za0003 := range z.Tags {
			_ = za0003
			s += msgp.StringPrefixSize + len(za0002) + msgp.StringPrefixSize + len(za0003)
		}
	}
	s += 13 + msgp.BoolSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TracerPayload) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ContainerID":
			z.ContainerID, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "ContainerID")
				return
			}
		case "LanguageName":
			z.LanguageName, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "LanguageName")
				return
			}
		case "LanguageVersion":
			z.LanguageVersion, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "LanguageVersion")
				return
			}
		case "TracerVersion":
			z.TracerVersion, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "TracerVersion")
				return
			}
		case "RuntimeID":
			z.RuntimeID, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "RuntimeID")
				return
			}
		case "Chunks":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "Chunks")
				return
			}
			if cap(z.Chunks) >= int(zb0002) {
				z.Chunks = (z.Chunks)[:zb0002]
			} else {
				z.Chunks = make([]*TraceChunk, zb0002)
			}
			for za0001 := range z.Chunks {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						err = msgp.WrapError(err, "Chunks", za0001)
						return
					}
					z.Chunks[za0001] = nil
				} else {
					if z.Chunks[za0001] == nil {
						z.Chunks[za0001] = new(TraceChunk)
					}
					err = z.Chunks[za0001].DecodeMsg(dc)
					if err != nil {
						err = msgp.WrapError(err, "Chunks", za0001)
						return
					}
				}
			}
		case "Tags":
			var zb0003 uint32
			zb0003, err = dc.ReadMapHeader()
			if err != nil {
				err = msgp.WrapError(err, "Tags")
				return
			}
			if z.Tags == nil {
				z.Tags = make(map[string]string, zb0003)
			} else if len(z.Tags) > 0 {
				for key := range z.Tags {
					delete(z.Tags, key)
				}
			}
			for zb0003 > 0 {
				zb0003--
				var za0002 string
				var za0003 string
				za0002, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "Tags")
					return
				}
				za0003, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "Tags", za0002)
					return
				}
				z.Tags[za0002] = za0003
			}
		case "Env":
			z.Env, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Env")
				return
			}
		case "Hostname":
			z.Hostname, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Hostname")
				return
			}
		case "AppVersion":
			z.AppVersion, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "AppVersion")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TracerPayload) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "ContainerID"
	err = en.Append(0x8a, 0xab, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteString(z.ContainerID)
	if err != nil {
		err = msgp.WrapError(err, "ContainerID")
		return
	}
	// write "LanguageName"
	err = en.Append(0xac, 0x4c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x4e, 0x61, 0x6d, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.LanguageName)
	if err != nil {
		err = msgp.WrapError(err, "LanguageName")
		return
	}
	// write "LanguageVersion"
	err = en.Append(0xaf, 0x4c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.LanguageVersion)
	if err != nil {
		err = msgp.WrapError(err, "LanguageVersion")
		return
	}
	// write "TracerVersion"
	err = en.Append(0xad, 0x54, 0x72, 0x61, 0x63, 0x65, 0x72, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.TracerVersion)
	if err != nil {
		err = msgp.WrapError(err, "TracerVersion")
		return
	}
	// write "RuntimeID"
	err = en.Append(0xa9, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteString(z.RuntimeID)
	if err != nil {
		err = msgp.WrapError(err, "RuntimeID")
		return
	}
	// write "Chunks"
	err = en.Append(0xa6, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Chunks)))
	if err != nil {
		err = msgp.WrapError(err, "Chunks")
		return
	}
	for za0001 := range z.Chunks {
		if z.Chunks[za0001] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Chunks[za0001].EncodeMsg(en)
			if err != nil {
				err = msgp.WrapError(err, "Chunks", za0001)
				return
			}
		}
	}
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Tags)))
	if err != nil {
		err = msgp.WrapError(err, "Tags")
		return
	}
	for za0002, za0003 := range z.Tags {
		err = en.WriteString(za0002)
		if err != nil {
			err = msgp.WrapError(err, "Tags")
			return
		}
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "Tags", za0002)
			return
		}
	}
	// write "Env"
	err = en.Append(0xa3, 0x45, 0x6e, 0x76)
	if err != nil {
		return
	}
	err = en.WriteString(z.Env)
	if err != nil {
		err = msgp.WrapError(err, "Env")
		return
	}
	// write "Hostname"
	err = en.Append(0xa8, 0x48, 0x6f, 0x73, 0x74, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.Hostname)
	if err != nil {
		err = msgp.WrapError(err, "Hostname")
		return
	}
	// write "AppVersion"
	err = en.Append(0xaa, 0x41, 0x70, 0x70, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.AppVersion)
	if err != nil {
		err = msgp.WrapError(err, "AppVersion")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TracerPayload) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "ContainerID"
	o = append(o, 0x8a, 0xab, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x49, 0x44)
	o = msgp.AppendString(o, z.ContainerID)
	// string "LanguageName"
	o = append(o, 0xac, 0x4c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x4e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.LanguageName)
	// string "LanguageVersion"
	o = append(o, 0xaf, 0x4c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendString(o, z.LanguageVersion)
	// string "TracerVersion"
	o = append(o, 0xad, 0x54, 0x72, 0x61, 0x63, 0x65, 0x72, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendString(o, z.TracerVersion)
	// string "RuntimeID"
	o = append(o, 0xa9, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x49, 0x44)
	o = msgp.AppendString(o, z.RuntimeID)
	// string "Chunks"
	o = append(o, 0xa6, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Chunks)))
	for za0001 := range z.Chunks {
		if z.Chunks[za0001] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Chunks[za0001].MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, "Chunks", za0001)
				return
			}
		}
	}
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Tags)))
	for za0002, za0003 := range z.Tags {
		o = msgp.AppendString(o, za0002)
		o = msgp.AppendString(o, za0003)
	}
	// string "Env"
	o = append(o, 0xa3, 0x45, 0x6e, 0x76)
	o = msgp.AppendString(o, z.Env)
	// string "Hostname"
	o = append(o, 0xa8, 0x48, 0x6f, 0x73, 0x74, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Hostname)
	// string "AppVersion"
	o = append(o, 0xaa, 0x41, 0x70, 0x70, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendString(o, z.AppVersion)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TracerPayload) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ContainerID":
			z.ContainerID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ContainerID")
				return
			}
		case "LanguageName":
			z.LanguageName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LanguageName")
				return
			}
		case "LanguageVersion":
			z.LanguageVersion, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LanguageVersion")
				return
			}
		case "TracerVersion":
			z.TracerVersion, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "TracerVersion")
				return
			}
		case "RuntimeID":
			z.RuntimeID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "RuntimeID")
				return
			}
		case "Chunks":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Chunks")
				return
			}
			if cap(z.Chunks) >= int(zb0002) {
				z.Chunks = (z.Chunks)[:zb0002]
			} else {
				z.Chunks = make([]*TraceChunk, zb0002)
			}
			for za0001 := range z.Chunks {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Chunks[za0001] = nil
				} else {
					if z.Chunks[za0001] == nil {
						z.Chunks[za0001] = new(TraceChunk)
					}
					bts, err = z.Chunks[za0001].UnmarshalMsg(bts)
					if err != nil {
						err = msgp.WrapError(err, "Chunks", za0001)
						return
					}
				}
			}
		case "Tags":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Tags")
				return
			}
			if z.Tags == nil {
				z.Tags = make(map[string]string, zb0003)
			} else if len(z.Tags) > 0 {
				for key := range z.Tags {
					delete(z.Tags, key)
				}
			}
			for zb0003 > 0 {
				var za0002 string
				var za0003 string
				zb0003--
				za0002, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Tags")
					return
				}
				za0003, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Tags", za0002)
					return
				}
				z.Tags[za0002] = za0003
			}
		case "Env":
			z.Env, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Env")
				return
			}
		case "Hostname":
			z.Hostname, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Hostname")
				return
			}
		case "AppVersion":
			z.AppVersion, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "AppVersion")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *TracerPayload) Msgsize() (s int) {
	s = 1 + 12 + msgp.StringPrefixSize + len(z.ContainerID) + 13 + msgp.StringPrefixSize + len(z.LanguageName) + 16 + msgp.StringPrefixSize + len(z.LanguageVersion) + 14 + msgp.StringPrefixSize + len(z.TracerVersion) + 10 + msgp.StringPrefixSize + len(z.RuntimeID) + 7 + msgp.ArrayHeaderSize
	for za0001 := range z.Chunks {
		if z.Chunks[za0001] == nil {
			s += msgp.NilSize
		} else {
			s += z.Chunks[za0001].Msgsize()
		}
	}
	s += 5 + msgp.MapHeaderSize
	if z.Tags != nil {
		for za0002, za0003 := range z.Tags {
			_ = za0003
			s += msgp.StringPrefixSize + len(za0002) + msgp.StringPrefixSize + len(za0003)
		}
	}
	s += 4 + msgp.StringPrefixSize + len(z.Env) + 9 + msgp.StringPrefixSize + len(z.Hostname) + 11 + msgp.StringPrefixSize + len(z.AppVersion)
	return
}
