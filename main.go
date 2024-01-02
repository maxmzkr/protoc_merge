package main

import (
	// "bytes"
	"cmp"
	// "encoding/json"
	// "fmt"
	"io"
	"log"
	"os"
	// "slices"
	"strings"

	"github.com/maxmzkr/protoc_merge/merge"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func ptr[T any](v T) *T {
	return &v
}

func max[T cmp.Ordered](a, b T) T {
	if b > a {
		return b
	}
	return a
}

type mergeSpec struct {
	basePaths map[string]bool
	prefixes  []string
	packages  []string
}

type matchedFiles struct {
	base   *descriptorpb.FileDescriptorProto
	merge  *descriptorpb.FileDescriptorProto
	merged *descriptorpb.FileDescriptorProto
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

	inputPrefix := []string{}
	packageMapping := []string{}
	paths := map[string]bool{}
	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value = param[i+1:]
			param = param[0:i]
		}
		switch param {
		case "":
		case "prefix":
			inputPrefix = append(inputPrefix, value)
		case "package":
			packageMapping = append(packageMapping, value)
		case "paths":
			paths[value] = true
		default:
			log.Printf("Unknown parameter: %s", param)
			os.Exit(1)
		}
	}

	if len(inputPrefix) != 3 {
		log.Printf("Expected exactly three prefix parameters, got %d", len(inputPrefix))
		os.Exit(1)
	}

	if len(packageMapping) != 3 {
		log.Printf("Expected exactly three package parameters, got %d", len(packageMapping))
		os.Exit(1)
	}

	mergeSpec := mergeSpec{
		basePaths: paths,
		prefixes:  inputPrefix,
		packages:  packageMapping,
	}

	resp := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}

	filesToGenerate := make(map[string]bool)
	for _, name := range req.GetFileToGenerate() {
		filesToGenerate[name] = true
	}

	matchedFiles := map[string]matchedFiles{}
	for _, file := range req.GetProtoFile() {
		if !filesToGenerate[file.GetName()] {
			continue
		}
		for i, prefix := range inputPrefix {
			if strings.HasPrefix(file.GetName(), prefix) {
				suffix := file.GetName()[len(prefix):]
				matchedFile := matchedFiles[suffix]
				switch i {
				case 0:
					matchedFile.base = file
				case 1:
					matchedFile.merge = file
				case 2:
					matchedFile.merged = file
				}
				matchedFiles[suffix] = matchedFile
			}
		}
	}

	for _, matchedFile := range matchedFiles {
		if matchedFile.base == nil {
			continue
		}
		mergeSpec.makeModel(matchedFile)
		// newFile, err := mergeSpec.merge(matchedFile)
		// if err != nil {
		// 	log.Printf("Failed to merge content for %s: %v", matchedFile.merged, err)
		// 	os.Exit(1)
		// }
		// resp.File = append(resp.File, newFile)
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

func (s mergeSpec) makeModel(matchedFiles matchedFiles) (merge.File, error) {
	// j, err := json.MarshalIndent(matchedFiles.base.GetSourceCodeInfo(), "", "  ")
	// if err != nil {
	// 	return merge.File{}, fmt.Errorf("failed to marshal source code info: %w", err)
	// }
	// log.Println(string(j))

	baseF := merge.ParseFile(matchedFiles.base)
	mergeF := merge.ParseFile(matchedFiles.merge)
	mergedF := merge.ParseFile(matchedFiles.merged)

	mergeSpec := merge.MergeSpec{
		BasePaths:     s.basePaths,
		BasePackage:   s.packages[0],
		MergePackage:  s.packages[1],
		MergedPackage: s.packages[2],

		MergePrefix:  s.prefixes[1],
		MergedPrefix: s.prefixes[2],
	}
	outF := mergeSpec.MergeFile(baseF, mergeF, mergedF)

	log.Println(merge.Serialize(outF))
	// j, err := json.MarshalIndent(outF, "", "  ")
	// if err != nil {
	// 	return merge.File{}, fmt.Errorf("failed to marshal model: %w", err)
	// }
	// log.Println(string(j))
	return merge.File{}, nil
}

// func (s mergeSpec) merge(matchedFiles matchedFiles) (*pluginpb.CodeGeneratorResponse_File, error) {
// 	j, err := json.MarshalIndent(matchedFiles.base.GetSourceCodeInfo(), "", "  ")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal source code info: %w", err)
// 	}
// 	log.Println(string(j))
// 	buf := &bytes.Buffer{}
// 	content := indentWriter{
// 		writer: buf,
// 	}
// 	_, err = content.WriteString("syntax = \"proto3\";\n\n")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to write syntax: %w", err)
// 	}
//
// 	_, err = content.WriteString("package " + strings.Replace(matchedFiles.base.GetPackage(), s.packages[0], s.packages[2], 1) + ";\n\n")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to write package: %w", err)
// 	}
//
// 	s.mergeOptions(content, matchedFiles)
//
// 	return &pluginpb.CodeGeneratorResponse_File{
// 		Name:    ptr(strings.TrimPrefix(matchedFiles.base.GetName(), s.prefixes[0])),
// 		Content: ptr(buf.String()),
// 	}, nil
// }
//
// func (s mergeSpec) mergeOptions(content indentWriter, matchedFiles matchedFiles) error {
// 	baseOptions := matchedFiles.base.GetOptions()
// 	mergeOptions := baseOptions
// 	if matchedFiles.merge != nil {
// 		mergeOptions = matchedFiles.merge.GetOptions()
// 	}
//
// 	if mergeOptions.GoPackage != nil {
// 		_, err := content.WriteString("option go_package = \"" + mergeOptions.GetGoPackage() + "\";\n\n")
// 		if err != nil {
// 			return fmt.Errorf("failed to write go_package: %w", err)
// 		}
// 	} else if baseOptions.GoPackage != nil {
// 		_, err := content.WriteString("option go_package = \"" + baseOptions.GetGoPackage() + "\";\n\n")
// 		if err != nil {
// 			return fmt.Errorf("failed to write go_package: %w", err)
// 		}
// 	}
//
// 	dependencySet := map[string]bool{}
// 	for _, dependency := range matchedFiles.base.GetDependency() {
// 		dependencySet[strings.Replace(dependency, s.prefixes[0], s.prefixes[2], 1)] = true
// 	}
// 	for _, dependency := range matchedFiles.merge.GetDependency() {
// 		dependencySet[strings.Replace(dependency, s.prefixes[1], s.prefixes[2], 1)] = true
// 	}
//
// 	dependencies := []string{}
// 	for dependency := range dependencySet {
// 		dependencies = append(dependencies, dependency)
// 	}
// 	slices.Sort(dependencies)
//
// 	for _, dependency := range dependencies {
// 		_, err := content.WriteString("import \"" + dependency + "\";\n")
// 		if err != nil {
// 			return fmt.Errorf("failed to write dependency: %w", err)
// 		}
// 	}
//
// 	err := s.mergeEnums(content, matchedFiles.base.GetEnumType(), matchedFiles.merge.GetEnumType(), matchedFiles.merged.GetEnumType())
// 	if err != nil {
// 		return fmt.Errorf("failed to merge enums: %w", err)
// 	}
//
// 	err = s.mergeMessages(content, matchedFiles.base.GetMessageType(), matchedFiles.merge.GetMessageType(), matchedFiles.merged.GetMessageType())
// 	if err != nil {
// 		return fmt.Errorf("failed to merge messages: %w", err)
// 	}
//
// 	return nil
// }
//
// func (s mergeSpec) mergeEnums(content indentWriter, baseEnums, mergeEnums, mergedEnums []*descriptorpb.EnumDescriptorProto) error {
// 	baseEnumMap := map[string]*descriptorpb.EnumDescriptorProto{}
// 	for _, enum := range baseEnums {
// 		baseEnumMap[enum.GetName()] = enum
// 	}
//
// 	mergeEnumMap := map[string]*descriptorpb.EnumDescriptorProto{}
// 	for _, enum := range mergeEnums {
// 		mergeEnumMap[enum.GetName()] = enum
// 	}
//
// 	mergedEnumMap := map[string]*descriptorpb.EnumDescriptorProto{}
// 	for _, enum := range mergedEnums {
// 		mergedEnumMap[enum.GetName()] = enum
// 	}
//
// 	allEnumSet := map[string]bool{}
// 	for enum := range baseEnumMap {
// 		allEnumSet[enum] = true
// 	}
// 	for enum := range mergeEnumMap {
// 		allEnumSet[enum] = true
// 	}
//
// 	allEnums := []string{}
// 	for enum := range allEnumSet {
// 		allEnums = append(allEnums, enum)
// 	}
//
// 	slices.Sort(allEnums)
//
// 	for _, enum := range allEnums {
// 		err := s.mergeEnum(content, baseEnumMap[enum], mergeEnumMap[enum], mergedEnumMap[enum])
// 		if err != nil {
// 			return fmt.Errorf("failed to merge enum: %w", err)
// 		}
// 	}
//
// 	return nil
// }
//
// func (s mergeSpec) mergeEnum(content indentWriter, baseEnum, mergeEnum, mergedEnum *descriptorpb.EnumDescriptorProto) error {
// 	var name string
// 	if baseEnum != nil {
// 		name = baseEnum.GetName()
// 	}
// 	if mergeEnum != nil {
// 		name = mergeEnum.GetName()
// 	}
//
// 	_, err := content.WriteString("\nenum " + name + " {\n")
// 	if err != nil {
// 		return fmt.Errorf("failed to write enum: %w", err)
// 	}
//
// 	content.Indent()
//
// 	maxFieldNumber := int32(0)
// 	fieldNumbers := map[string]int32{}
// 	numberSet := map[int32]bool{}
// 	for _, value := range mergedEnum.GetValue() {
// 		maxFieldNumber = max(maxFieldNumber, value.GetNumber())
// 		fieldNumbers[value.GetName()] = value.GetNumber()
// 		numberSet[value.GetNumber()] = true
// 	}
//
// 	baseNameSet := map[string]bool{}
// 	baseNames := []string{}
// 	for _, value := range baseEnum.GetValue() {
// 		if _, ok := fieldNumbers[value.GetName()]; ok {
// 			baseNameSet[value.GetName()] = true
// 			baseNames = append(baseNames, value.GetName())
// 			continue
// 		}
// 		fieldNumber := value.GetNumber()
// 		if numberSet[fieldNumber] {
// 			fieldNumber = maxFieldNumber + 1
// 		}
// 		maxFieldNumber = max(maxFieldNumber, fieldNumber)
// 		fieldNumbers[value.GetName()] = fieldNumber
// 		numberSet[fieldNumber] = true
// 		baseNameSet[value.GetName()] = true
// 		baseNames = append(baseNames, value.GetName())
// 	}
//
// 	mergeNames := []string{}
// 	for _, value := range mergeEnum.GetValue() {
// 		if baseNameSet[value.GetName()] {
// 			continue
// 		}
// 		if _, ok := fieldNumbers[value.GetName()]; ok {
// 			mergeNames = append(mergeNames, value.GetName())
// 			continue
// 		}
// 		fieldNumber := value.GetNumber()
// 		if numberSet[fieldNumber] {
// 			fieldNumber = maxFieldNumber + 1
// 		}
// 		maxFieldNumber = max(maxFieldNumber, fieldNumber)
// 		fieldNumbers[value.GetName()] = fieldNumber
// 		numberSet[fieldNumber] = true
// 		mergeNames = append(mergeNames, value.GetName())
// 	}
//
// 	for _, name := range baseNames {
// 		_, err := content.WriteString(name + " = " + fmt.Sprint(fieldNumbers[name]) + ";\n")
// 		if err != nil {
// 			return fmt.Errorf("failed to write enum value: %w", err)
// 		}
// 	}
//
// 	for _, name := range mergeNames {
// 		_, err := content.WriteString(name + " = " + fmt.Sprint(fieldNumbers[name]) + ";\n")
// 		if err != nil {
// 			return fmt.Errorf("failed to write enum value: %w", err)
// 		}
// 	}
//
// 	content.Outdent()
// 	content.WriteString("}\n")
// 	return nil
// }
//
// func (s mergeSpec) mergeMessages(content indentWriter, baseMessages, mergeMessages, mergedMessages []*descriptorpb.DescriptorProto) error {
// 	baseMessageMap := map[string]*descriptorpb.DescriptorProto{}
// 	for _, message := range baseMessages {
// 		baseMessageMap[message.GetName()] = message
// 	}
//
// 	mergeMessageMap := map[string]*descriptorpb.DescriptorProto{}
// 	for _, message := range mergeMessages {
// 		mergeMessageMap[message.GetName()] = message
// 	}
//
// 	mergedMessageMap := map[string]*descriptorpb.DescriptorProto{}
// 	for _, message := range mergedMessages {
// 		mergedMessageMap[message.GetName()] = message
// 	}
//
// 	allMessageSet := map[string]bool{}
// 	for message := range baseMessageMap {
// 		allMessageSet[message] = true
// 	}
//
// 	allMessages := []string{}
// 	for message := range allMessageSet {
// 		allMessages = append(allMessages, message)
// 	}
//
// 	slices.Sort(allMessages)
//
// 	for _, message := range allMessages {
// 		err := s.mergeMessage(content, baseMessageMap[message], mergeMessageMap[message], mergedMessageMap[message])
// 		if err != nil {
// 			return fmt.Errorf("failed to merge message: %w", err)
// 		}
// 	}
//
// 	return nil
// }
//
// func (s mergeSpec) mergeMessage(content indentWriter, baseMessage, mergeMessage, mergedMessage *descriptorpb.DescriptorProto) error {
// 	var name string
// 	if baseMessage != nil {
// 		name = baseMessage.GetName()
// 	}
// 	if mergeMessage != nil {
// 		name = mergeMessage.GetName()
// 	}
//
// 	_, err := content.WriteString("\nmessage " + name + " {\n")
// 	if err != nil {
// 		return fmt.Errorf("failed to write message: %w", err)
// 	}
//
// 	content.Indent()
//
// 	s.mergeEnums(content, baseMessage.GetEnumType(), mergeMessage.GetEnumType(), mergedMessage.GetEnumType())
// 	s.mergeMessages(content, baseMessage.GetNestedType(), mergeMessage.GetNestedType(), mergedMessage.GetNestedType())
//
// 	maxFieldNumber := int32(0)
// 	fieldDescriptors := map[string]*descriptorpb.FieldDescriptorProto{}
// 	fieldNumbers := map[string]int32{}
// 	numberSet := map[int32]bool{}
// 	for _, field := range mergedMessage.GetField() {
// 		maxFieldNumber = max(maxFieldNumber, field.GetNumber())
// 		fieldNumbers[field.GetName()] = field.GetNumber()
// 		numberSet[field.GetNumber()] = true
// 	}
//
// 	baseOneofIndex := map[string][]string{}
// 	baseNameSet := map[string]bool{}
// 	baseNames := []string{}
// 	for _, field := range baseMessage.GetField() {
// 		fieldDescriptors[field.GetName()] = field
// 		baseNameSet[field.GetName()] = true
// 		baseNames = append(baseNames, field.GetName())
// 		if field.OneofIndex != nil {
// 			oneof := baseMessage.GetOneofDecl()[field.GetOneofIndex()]
// 			baseOneofIndex[oneof.GetName()] = append(baseOneofIndex[oneof.GetName()], field.GetName())
// 		}
// 		if _, ok := fieldNumbers[field.GetName()]; ok {
// 			baseNames = append(baseNames, field.GetName())
// 			continue
// 		}
// 		fieldNumber := field.GetNumber()
// 		if numberSet[fieldNumber] {
// 			fieldNumber = maxFieldNumber + 1
// 		}
// 		maxFieldNumber = max(maxFieldNumber, fieldNumber)
// 		fieldNumbers[field.GetName()] = fieldNumber
// 		numberSet[fieldNumber] = true
// 	}
//
// 	mergeOneofIndex := map[string][]string{}
// 	mergeNames := []string{}
// 	for _, field := range mergeMessage.GetField() {
// 		fieldDescriptors[field.GetName()] = field
// 		if baseNameSet[field.GetName()] {
// 			continue
// 		}
// 		if field.OneofIndex != nil {
// 			oneof := mergeMessage.GetOneofDecl()[field.GetOneofIndex()]
// 			mergeOneofIndex[oneof.GetName()] = append(mergeOneofIndex[oneof.GetName()], field.GetName())
// 		}
// 		if _, ok := fieldNumbers[field.GetName()]; ok {
// 			mergeNames = append(mergeNames, field.GetName())
// 			continue
// 		}
// 		fieldNumber := field.GetNumber()
// 		if numberSet[fieldNumber] {
// 			fieldNumber = maxFieldNumber + 1
// 		}
// 		maxFieldNumber = max(maxFieldNumber, fieldNumber)
// 		fieldNumbers[field.GetName()] = fieldNumber
// 		numberSet[fieldNumber] = true
// 		mergeNames = append(mergeNames, field.GetName())
// 	}
//
// 	for _, name := range baseNames {
// 		field := fieldDescriptors[name]
// 		if field.OneofIndex != nil {
// 			continue
// 		}
// 		err := s.writeField(content, field, fieldNumbers[name])
// 		if err != nil {
// 			return fmt.Errorf("failed to write field: %w", err)
// 		}
// 	}
//
// 	for _, name := range mergeNames {
// 		field := fieldDescriptors[name]
// 		if field.OneofIndex != nil {
// 			continue
// 		}
// 		err := s.writeField(content, fieldDescriptors[name], fieldNumbers[name])
// 		if err != nil {
// 			return fmt.Errorf("failed to write field: %w", err)
// 		}
// 	}
//
// 	baseOneOfSet := map[string]bool{}
// 	for _, oneof := range baseMessage.GetOneofDecl() {
// 		baseOneOfSet[oneof.GetName()] = true
// 		content.WriteString("\noneof " + oneof.GetName() + " {\n")
// 		content.Indent()
// 		for _, field := range baseOneofIndex[oneof.GetName()] {
// 			err := s.writeField(content, fieldDescriptors[field], fieldNumbers[field])
// 			if err != nil {
// 				return fmt.Errorf("failed to write field: %w", err)
// 			}
// 		}
//
// 		for _, field := range mergeOneofIndex[oneof.GetName()] {
// 			err := s.writeField(content, fieldDescriptors[field], fieldNumbers[field])
// 			if err != nil {
// 				return fmt.Errorf("failed to write field: %w", err)
// 			}
// 		}
// 		content.Outdent()
// 		content.WriteString("}\n")
// 	}
//
// 	for _, oneof := range mergeMessage.GetOneofDecl() {
// 		if baseOneOfSet[oneof.GetName()] {
// 			continue
// 		}
// 		content.WriteString("\noneof " + oneof.GetName() + " {\n")
// 		content.Indent()
// 		for _, field := range mergeOneofIndex[oneof.GetName()] {
// 			err := s.writeField(content, fieldDescriptors[field], fieldNumbers[field])
// 			if err != nil {
// 				return fmt.Errorf("failed to write field: %w", err)
// 			}
// 		}
// 		content.Outdent()
// 		content.WriteString("}\n")
// 	}
//
// 	content.Outdent()
// 	content.WriteString("}\n")
// 	return nil
// }
//
// func (s mergeSpec) writeField(content indentWriter, field *descriptorpb.FieldDescriptorProto, number int32) error {
// 	if field.GetTypeName() != "" {
// 		_, err := content.WriteString(field.GetTypeName())
// 		if err != nil {
// 			return fmt.Errorf("failed to write field type: %w", err)
// 		}
// 	} else {
// 		_, err := content.WriteString(strings.ToLower(strings.Split(field.GetType().String(), "_")[1]))
// 		if err != nil {
// 			return fmt.Errorf("failed to write field type: %w", err)
// 		}
// 	}
// 	_, err := content.WriteString(" " + field.GetName() + " = " + fmt.Sprint(number) + ";\n")
// 	if err != nil {
// 		return fmt.Errorf("failed to write field: %w", err)
// 	}
// 	return nil
// }
