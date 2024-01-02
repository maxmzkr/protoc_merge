package main

import (
	"cmp"
	"encoding/json"
	"io"
	"log"
	"os"
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
		os.Exit(1)
	}

	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		os.Exit(1)
	}

	prefixes := []string{}
	packages := []string{}
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
			prefixes = append(prefixes, value)
		case "package":
			packages = append(packages, value)
		case "paths":
			paths[value] = true
		default:
			os.Exit(1)
		}
	}

	if len(prefixes) != 3 {
		os.Exit(1)
	}

	if len(packages) != 2 {
		os.Exit(1)
	}

	resp := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}

	filesToGenerate := make(map[string]bool)
	for _, name := range req.GetFileToGenerate() {
		filesToGenerate[name] = true
	}

	matchedMap := map[string]matchedFiles{}
	for _, file := range req.GetProtoFile() {
		for i, prefix := range prefixes {
			if strings.HasPrefix(file.GetName(), prefix) {
				suffix := file.GetName()[len(prefix):]
				matchedFile := matchedMap[suffix]
				switch i {
				case 0:
					matchedFile.base = file
				case 1:
					matchedFile.merge = file
				case 2:
					matchedFile.merged = file
				}
				matchedMap[suffix] = matchedFile
			}
		}
	}

	for _, matchedFile := range matchedMap {
		if matchedFile.base == nil {
			continue
		}

		if matchedFile.merge == nil {
			continue
		}

		mergeSpec := merge.MergeSpec{
			MergePackage:  packages[0],
			MergedPackage: packages[1],

			MergePrefix:  prefixes[1],
			MergedPrefix: prefixes[2],
		}

		j, err := json.MarshalIndent(req, "", "  ")
		if err != nil {
			os.Exit(1)
		}
		log.Println(string(j))

		baseF := merge.ParseFile(matchedFile.base)
		mergeF := merge.ParseFile(matchedFile.merge)
		mergedF := merge.ParseFile(matchedFile.merged)

		outF := mergeSpec.MergeFile(baseF, mergeF, mergedF)
		file := &pluginpb.CodeGeneratorResponse_File{
			// We can't use merged file name because it might not exist yet
			Name:    ptr(strings.Replace(matchedFile.merge.GetName(), prefixes[1], prefixes[2], 1)),
			Content: ptr(merge.Serialize(outF)),
		}
		resp.File = append(resp.File, file)
	}

	data, err = proto.Marshal(resp)
	if err != nil {
		os.Exit(1)
	}

	if _, err := os.Stdout.Write(data); err != nil {
		os.Exit(1)
	}
}
