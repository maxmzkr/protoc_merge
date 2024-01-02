package merge

import (
	"slices"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

func ParseFile(f *descriptorpb.FileDescriptorProto) *File {
	out := &File{}
	locations := f.GetSourceCodeInfo().GetLocation()
	for len(locations) > 0 {
		location := locations[0]

		if len(location.GetPath()) == 0 {
			locations = locations[1:]
			continue
		}

		switch location.GetPath()[0] {
		case 2:
			var filePackage *Package
			locations, filePackage = parsePackage(locations, f.GetPackage())
			out.Package = filePackage
		case 3:
			var dependency *Dependency
			locations, dependency = parseDependency(locations, f.GetDependency()[location.GetPath()[1]])
			out.Dependencies = append(out.Dependencies, dependency)
		case 4:
			var message *Message
			locations, message = parseMessage(locations, f.GetMessageType()[location.GetPath()[1]])
			out.Messages = append(out.Messages, message)
		case 5:
			var enum *Enum
			locations, enum = parseEnum(locations, f.GetEnumType()[location.GetPath()[1]])
			out.Enums = append(out.Enums, enum)
		case 6:
			// I don't think we'll handle service
			locations = consumeLocation(locations)
		case 7:
			// I don't think we'll handle extensions
			locations = consumeLocation(locations)
		case 8:
			// TODO handle file options
			locations = consumeLocation(locations)
		case 12:
			var syntax *Syntax
			locations, syntax = parseSyntax(locations, f.GetSyntax())
			out.Syntax = syntax
		}
	}
	return out
}

func parsePackage(locations []*descriptorpb.SourceCodeInfo_Location, p string) ([]*descriptorpb.SourceCodeInfo_Location, *Package) {
	filePackage := &Package{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    p,
	}
	return locations[1:], filePackage
}

func parseDependency(locations []*descriptorpb.SourceCodeInfo_Location, d string) ([]*descriptorpb.SourceCodeInfo_Location, *Dependency) {
	dependency := &Dependency{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    d,
	}
	return locations[1:], dependency
}

func parseMessage(locations []*descriptorpb.SourceCodeInfo_Location, m *descriptorpb.DescriptorProto) ([]*descriptorpb.SourceCodeInfo_Location, *Message) {
	out := &Message{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    m.GetName(),
	}
	startLocation := locations[0]
	locations = locations[1:]
	nested := len(startLocation.GetPath())
	for len(locations) > 0 &&
		len(locations[0].GetPath()) >= nested &&
		locations[0].GetPath()[nested-2] == startLocation.GetPath()[nested-2] &&
		locations[0].GetPath()[nested-1] == startLocation.GetPath()[nested-1] {

		location := locations[0]
		switch location.GetPath()[nested] {
		case 1:
			// already parsed
			locations = locations[1:]
		case 2:
			var field *Field
			locations, field = parseField(locations, m.GetField()[location.GetPath()[nested+1]])
			out.Fields = append(out.Fields, field)
		case 3:
			var message *Message
			locations, message = parseMessage(locations, m.GetNestedType()[location.GetPath()[nested+1]])
			out.Messages = append(out.Messages, message)
		case 4:
			var enum *Enum
			locations, enum = parseEnum(locations, m.GetEnumType()[location.GetPath()[nested+1]])
			out.Enums = append(out.Enums, enum)
		case 5:
			// I don't think we'll handle message extensions
			locations = consumeLocation(locations)
		case 6:
			// I don't think we'll handle message extensions
			locations = consumeLocation(locations)
		case 7:
			// TODO handle message options
			locations = consumeLocation(locations)
		case 8:
			var oneof *Oneof
			locations, oneof = parseOneof(locations, m, m.GetOneofDecl()[location.GetPath()[nested+1]])
			out.Oneofs = append(out.Oneofs, oneof)
		case 9:
			var reservedRange *ReservedRange
			locations, reservedRange = parseReservedRange(locations, m.GetReservedRange()[location.GetPath()[nested+1]])
			out.ReservedRanges = append(out.ReservedRanges, reservedRange)
		case 10:
			if len(location.GetPath()) == nested+1 {
				locations = locations[1:]
				continue
			}
			var reservedName *ReservedName
			locations, reservedName = parseReservedName(locations, m.GetReservedName()[location.GetPath()[nested+1]])
			out.ReservedNames = append(out.ReservedNames, reservedName)
		default:
			locations = locations[1:]
		}
	}
	return locations, out
}

func parseField(locations []*descriptorpb.SourceCodeInfo_Location, f *descriptorpb.FieldDescriptorProto) ([]*descriptorpb.SourceCodeInfo_Location, *Field) {
	var fieldType string
	if f.GetTypeName() != "" {
		fieldType = f.GetTypeName()
	} else {
		fieldType = strings.ToLower(strings.Split(f.GetType().String(), "_")[1])
	}

	var label string
	if f.GetProto3Optional() {
		label = "optional"
	} else if f.GetLabel() != descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL {
		label = strings.ToLower(strings.Split(f.GetLabel().String(), "_")[1])
	}

	out := &Field{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    f.GetName(),
		Number:                  f.GetNumber(),
		Label:                   label,
		Type:                    fieldType,
	}

	locations = consumeLocation(locations)

	return locations, out
}

func parseEnum(locations []*descriptorpb.SourceCodeInfo_Location, e *descriptorpb.EnumDescriptorProto) ([]*descriptorpb.SourceCodeInfo_Location, *Enum) {
	out := &Enum{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    e.GetName(),
	}
	startLocation := locations[0]
	locations = locations[1:]
	nested := len(startLocation.GetPath())
	for len(locations) > 0 &&
		len(locations) >= nested &&
		locations[0].GetPath()[nested-2] == startLocation.GetPath()[nested-2] &&
		locations[0].GetPath()[nested-1] == startLocation.GetPath()[nested-1] {

		location := locations[0]
		switch location.GetPath()[nested] {
		case 1:
			// already parsed
			locations = locations[1:]
		case 2:
			var enumValue *EnumValue
			locations, enumValue = parseEnumValue(locations, e.GetValue()[location.GetPath()[nested+1]])
			out.Values = append(out.Values, enumValue)
		case 3:
			// TODO handle enum options
			locations = consumeLocation(locations)
		case 4:
			var reservedRange *ReservedRange
			locations, reservedRange = parseReservedRange(locations, e.GetReservedRange()[location.GetPath()[nested+1]])
			out.ReservedRanges = append(out.ReservedRanges, reservedRange)
		case 5:
			var reservedName *ReservedName
			locations, reservedName = parseReservedName(locations, e.GetReservedName()[location.GetPath()[nested+1]])
			out.ReservedNames = append(out.ReservedNames, reservedName)
		default:
			locations = locations[1:]
		}
	}
	return locations, out
}

func parseEnumValue(locations []*descriptorpb.SourceCodeInfo_Location, e *descriptorpb.EnumValueDescriptorProto) ([]*descriptorpb.SourceCodeInfo_Location, *EnumValue) {
	out := &EnumValue{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    e.GetName(),
		Number:                  e.GetNumber(),
	}

	locations = consumeLocation(locations)

	return locations, out
}

type reservedRange interface {
	GetStart() int32
	GetEnd() int32
}

func parseReservedRange(locations []*descriptorpb.SourceCodeInfo_Location, r reservedRange) ([]*descriptorpb.SourceCodeInfo_Location, *ReservedRange) {
	out := &ReservedRange{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Start:                   r.GetStart(),
		End:                     r.GetEnd(),
	}

	locations = consumeLocation(locations)

	return locations, out
}

func parseReservedName(locations []*descriptorpb.SourceCodeInfo_Location, r string) ([]*descriptorpb.SourceCodeInfo_Location, *ReservedName) {
	out := &ReservedName{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    r,
	}

	return locations[1:], out
}

func parseOneof(locations []*descriptorpb.SourceCodeInfo_Location, m *descriptorpb.DescriptorProto, o *descriptorpb.OneofDescriptorProto) ([]*descriptorpb.SourceCodeInfo_Location, *Oneof) {
	out := &Oneof{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    o.GetName(),
	}

	startLocation := locations[0]
	locations = locations[1:]
	nested := len(startLocation.GetPath())
	for len(locations) > 0 &&
		len(locations) >= nested &&
		locations[0].GetPath()[nested-2] == startLocation.GetPath()[nested-2] &&
		locations[0].GetPath()[nested-1] == startLocation.GetPath()[nested-1] {

		location := locations[0]
		switch location.GetPath()[nested] {
		case 1:
			// already parsed
			locations = locations[1:]
		case 2:
			// TODO handle oneof options
			locations = consumeLocation(locations)
		default:
			locations = locations[1:]
		}
	}

	// parse the one of fields using the spans as a guide
	endLine := startLocation.GetSpan()[0]
	endColumn := startLocation.GetSpan()[2]
	if len(startLocation.GetSpan()) > 0 {
		endLine = endColumn
		endColumn = startLocation.GetSpan()[3]
	}

	for len(locations) > 0 &&
		(locations[0].GetSpan()[0] < endLine ||
			(locations[0].GetSpan()[0] == endLine &&
				locations[0].GetSpan()[1] < endColumn)) {
		var field *Field
		locations, field = parseField(locations, m.GetField()[locations[0].GetPath()[nested-1]])
		out.Fields = append(out.Fields, field)
	}

	return locations, out
}

func parseSyntax(locations []*descriptorpb.SourceCodeInfo_Location, s string) ([]*descriptorpb.SourceCodeInfo_Location, *Syntax) {
	out := &Syntax{
		LeadingDetachedComments: locations[0].GetLeadingDetachedComments(),
		LeadingComments:         locations[0].GetLeadingComments(),
		TrailingComments:        locations[0].GetTrailingComments(),
		Name:                    s,
	}

	return locations[1:], out
}

func consumeLocation(locations []*descriptorpb.SourceCodeInfo_Location) []*descriptorpb.SourceCodeInfo_Location {
	startLocation := locations[0]
	nested := len(startLocation.GetPath())
	for len(locations) > 0 &&
		len(locations[0].GetPath()) >= nested &&
		slices.Equal(locations[0].GetPath()[:nested], startLocation.GetPath()[:nested]) &&
		slices.Equal(locations[0].GetPath()[:nested], startLocation.GetPath()[:nested]) {

		locations = locations[1:]
	}
	return locations
}
