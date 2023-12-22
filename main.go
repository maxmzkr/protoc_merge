package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
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

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("Failed to read input: %v", err)
		os.Exit(1)
	}

	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		log.Printf("Failed to parse input proto: %v", err)
		os.Exit(1)
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	for _, file := range req.GetProtoFile() {
		if !file.
		newFile, err := generateFile(file)
		if err != nil {
			log.Printf("Failed to generate content for %s: %v", file.GetName(), err)
			os.Exit(1)
		}
		resp.File = append(resp.File, newFile)
	}

	data, err = proto.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal output proto: %v", err)
		os.Exit(1)
	}

	if _, err := os.Stdout.Write(data); err != nil {
		log.Printf("Failed to write output proto: %v", err)
		os.Exit(1)
	}
}

func generateFile(file *descriptorpb.FileDescriptorProto) (*pluginpb.CodeGeneratorResponse_File, error) {
	buf := &bytes.Buffer{}
	content := indentWriter{
		writer: buf,
	}
	_, err := content.WriteString("package " + file.GetPackage() + "\n")
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	for _, dependency := range file.Dependency {
		_, err := content.WriteString("import " + dependency + "\n")
		if err != nil {
			return nil, fmt.Errorf("failed to write content: %w", err)
		}
	}
	content.WriteString("\n")

	for _, message := range file.MessageType {
		writeMessage(content, message)
	}

	return &pluginpb.CodeGeneratorResponse_File{
		Name:    proto.String(file.GetName() + ".pb2"),
		Content: to.Ptr(buf.String()),
	}, nil
}

func writeEnum(content indentWriter, enum *descriptorpb.EnumDescriptorProto) {
	content.WriteString("\nenum " + enum.GetName() + " {\n")
	content.Indent()
	for _, value := range enum.Value {
		content.WriteString(value.GetName() + " = " + fmt.Sprint(value.GetNumber()) + ";\n")
	}
	content.Outdent()
}

func writeMessage(content indentWriter, message *descriptorpb.DescriptorProto) {
	content.WriteString("\nmessage " + message.GetName() + " {\n")
	content.Indent()
	for _, nestedMessage := range message.NestedType {
		writeMessage(content, nestedMessage)
	}
	for _, field := range message.Field {
		writeField(content, field)
	}
	content.Outdent()
	content.WriteString("}\n")
}

func writeField(content indentWriter, field *descriptorpb.FieldDescriptorProto) {
	if field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		content.WriteString("repeated ")
	}
	if field.GetTypeName() != "" {
		content.WriteString(field.GetTypeName())
	} else {
		content.WriteString(strings.ToLower(strings.Split(field.GetType().String(), "_")[1]))
	}
	content.WriteString(" " + field.GetName() + " = " + fmt.Sprint(field.GetNumber()) + ";\n")
}
