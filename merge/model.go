package merge

type File struct {
	Syntax       *Syntax
	Package      *Package
	Options      []*FileOption
	Dependencies []*Dependency
	Enums        []*Enum
	Messages     []*Message
}

func (f *File) GetEnums() []*Enum {
	return f.Enums
}

func (f *File) GetMessages() []*Message {
	return f.Messages
}

type FileOption struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Value                   string
}

type Syntax struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
}

type Package struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
}

type Dependency struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
}

type Enum struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Values                  []*EnumValue
	ReservedRanges          []*ReservedRange
	ReservedNames           []*ReservedName
}

type EnumValue struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Number                  int32
}

type Message struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Enums                   []*Enum
	Messages                []*Message
	Fields                  []*Field
	Oneofs                  []*Oneof
	ReservedRanges          []*ReservedRange
	ReservedNames           []*ReservedName
}

func (m *Message) GetEnums() []*Enum {
	return m.Enums
}

func (m *Message) GetMessages() []*Message {
	return m.Messages
}

func (m *Message) GetFields() []*Field {
	return m.Fields
}

type Oneof struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Fields                  []*Field
}

func (o *Oneof) GetFields() []*Field {
	return o.Fields
}

type Field struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
	Number                  int32
	Label                   string
	Type                    string
}

type ReservedRange struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Start                   int32
	End                     int32
}

type ReservedName struct {
	LeadingDetachedComments []string
	LeadingComments         string
	TrailingComments        string
	Name                    string
}
