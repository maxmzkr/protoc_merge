package merge

import (
	"fmt"
	"io"
	"strings"
)

type indentWriter struct {
	writer        io.StringWriter
	level         int
	writtenToLine bool
}

// WriteString writes the the given string to the underlying writer, indenting
// the first character of a newline by the current indentation level.
func (w *indentWriter) WriteString(s string) (int, error) {
	total := 0
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if !w.writtenToLine && len(line) > 0 {
			w.writtenToLine = true
			w.writer.WriteString(strings.Repeat("  ", w.level))
		}
		n, err := w.writer.WriteString(line)
		total += n
		if err != nil {
			return total, err
		}
		if i != len(lines)-1 {
			w.writtenToLine = false
			w.writer.WriteString("\n")
		}
	}
	return total, nil
}

func (w *indentWriter) Indent() {
	w.level++
}

func (w *indentWriter) Outdent() {
	w.level--
}

func Serialize(f *File) string {
	innerBuf := &strings.Builder{}
	buf := &indentWriter{writer: innerBuf}
	writeComments(buf, f.Syntax.LeadingDetachedComments)
	writeComment(buf, f.Syntax.LeadingComments)
	buf.WriteString(fmt.Sprintf("syntax = \"%s\";\n", f.Syntax.Name))
	writeTrailingComment(buf, f.Syntax.TrailingComments)

	writeComments(buf, f.Package.LeadingDetachedComments)
	writeComment(buf, f.Package.LeadingComments)
	buf.WriteString(fmt.Sprintf("package %s;\n", f.Package.Name))
	writeTrailingComment(buf, f.Package.TrailingComments)

	for _, dependency := range f.Dependencies {
		writeComments(buf, dependency.LeadingDetachedComments)
		writeComment(buf, dependency.LeadingComments)
		buf.WriteString(fmt.Sprintf("import \"%s\";\n", dependency.Name))
		writeTrailingComment(buf, dependency.TrailingComments)
	}

	for _, enum := range f.Enums {
		writeEnum(buf, enum)
	}

	for _, message := range f.Messages {
		writeMessage(buf, message)
	}

	return innerBuf.String()
}

func writeEnum(buf *indentWriter, e *Enum) {
	writeComments(buf, e.LeadingDetachedComments)
	writeComment(buf, e.LeadingComments)
	buf.WriteString("enum " + e.Name + " {\n")
	buf.Indent()

	for _, value := range e.Values {
		writeComments(buf, value.LeadingDetachedComments)
		writeComment(buf, value.LeadingComments)
		buf.WriteString(fmt.Sprintf("%s = %d;\n", value.Name, value.Number))
		writeTrailingComment(buf, value.TrailingComments)
	}

	for _, range_ := range e.ReservedRanges {
		writeComments(buf, range_.LeadingDetachedComments)
		writeComment(buf, range_.LeadingComments)
		buf.WriteString(fmt.Sprintf("reserved %d to %d;", range_.Start, range_.End))
		writeTrailingComment(buf, range_.TrailingComments)
	}

	for _, name := range e.ReservedNames {
		writeComments(buf, name.LeadingDetachedComments)
		writeComment(buf, name.LeadingComments)
		buf.WriteString(fmt.Sprintf("reserved \"%s\";", name.Name))
		writeTrailingComment(buf, name.TrailingComments)
	}

	buf.Outdent()
	buf.WriteString("}\n")
	writeTrailingComment(buf, e.TrailingComments)
}

func writeMessage(buf *indentWriter, m *Message) {
	writeComments(buf, m.LeadingDetachedComments)
	writeComment(buf, m.LeadingComments)
	buf.WriteString(fmt.Sprintf("message %s {\n", m.Name))
	buf.Indent()

	for _, nested := range m.Messages {
		writeMessage(buf, nested)
	}

	for _, nested := range m.Enums {
		writeEnum(buf, nested)
	}

	for _, field := range m.Fields {
		writeField(buf, field)
	}

	for _, oneof := range m.Oneofs {
		writeOneof(buf, oneof)
	}

	for _, range_ := range m.ReservedRanges {
		writeComments(buf, range_.LeadingDetachedComments)
		writeComment(buf, range_.LeadingComments)
		buf.WriteString(fmt.Sprintf("reserved %d to %d;", range_.Start, range_.End))
		writeTrailingComment(buf, range_.TrailingComments)
	}

	for _, name := range m.ReservedNames {
		writeComments(buf, name.LeadingDetachedComments)
		writeComment(buf, name.LeadingComments)
		buf.WriteString(fmt.Sprintf("reserved \"%s\";", name.Name))
		writeTrailingComment(buf, name.TrailingComments)
	}

	buf.Outdent()
	buf.WriteString("}")
	writeTrailingComment(buf, m.TrailingComments)
}

func writeField(buf *indentWriter, f *Field) {
	writeComments(buf, f.LeadingDetachedComments)
	writeComment(buf, f.LeadingComments)
	if f.Label == "" {
		buf.WriteString(fmt.Sprintf("%s %s = %d;\n", f.Type, f.Name, f.Number))
	} else {
		buf.WriteString(fmt.Sprintf("%s %s %s = %d;\n", f.Label, f.Type, f.Name, f.Number))
	}
	writeTrailingComment(buf, f.TrailingComments)
}

func writeOneof(buf *indentWriter, o *Oneof) {
	writeComments(buf, o.LeadingDetachedComments)
	writeComment(buf, o.LeadingComments)
	buf.WriteString(fmt.Sprintf("oneof %s {\n", o.Name))
	buf.Indent()

	for _, field := range o.Fields {
		writeField(buf, field)
	}

	buf.Outdent()
	buf.WriteString("}\n")
	writeTrailingComment(buf, o.TrailingComments)
}

func writeComments(buf *indentWriter, comments []string) {
	for _, comment := range comments {
		writeComment(buf, comment)
		buf.WriteString("\n")
	}
}

func writeComment(buf *indentWriter, comment string) {
	if comment == "" {
		return
	}
	lines := strings.Split(comment, "\n")
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		buf.WriteString(fmt.Sprintf("//%s\n", line))
	}
}

func writeTrailingComment(buf *indentWriter, comment string) {
	writeComment(buf, comment)
	buf.WriteString("\n")
}
