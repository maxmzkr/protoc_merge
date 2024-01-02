package merge

import (
	"fmt"
	"log"
	"strings"
)

type MergeSpec struct {
	MergePackage  string
	MergedPackage string

	MergePrefix  string
	MergedPrefix string
}

type EnumHaver interface {
	GetEnums() []*Enum
}

type MessageHaver interface {
	GetMessages() []*Message
}

type FieldHaver interface {
	GetFields() []*Field
}

type numberer struct {
	reserved map[string]int32
	used     map[int32]bool
	next     int32
}

func newNumberer(start int32) *numberer {
	return &numberer{
		reserved: make(map[string]int32),
		used:     make(map[int32]bool),
		next:     start - 1,
	}
}

func (n *numberer) use(name string, number int32) {
	n.reserved[name] = number
	n.used[number] = true
}

func (n *numberer) number(name string) int32 {
	if number, ok := n.reserved[name]; ok {
		return number
	}

	for {
		n.next += 1
		if _, ok := n.used[n.next]; !ok {
			break
		}
	}

	n.reserved[name] = n.next
	return n.next
}

func (s *MergeSpec) MergeFile(base *File, merge *File, merged *File) *File {
	out := &File{}

	out.Syntax = &Syntax{
		LeadingDetachedComments: append(
			append(
				[]string{},
				base.Syntax.LeadingDetachedComments...,
			),
			merge.Syntax.LeadingDetachedComments...,
		),
		LeadingComments:  base.Syntax.LeadingComments,
		TrailingComments: base.Syntax.TrailingComments,
		Name:             "proto3",
	}
	if len(merge.Syntax.LeadingComments) > 0 {
		out.Syntax.LeadingComments = merge.Syntax.LeadingComments
	}
	if len(merge.Syntax.TrailingComments) > 0 {
		out.Syntax.TrailingComments = merge.Syntax.TrailingComments
	}

	out.Package = &Package{
		LeadingDetachedComments: append(append([]string{}, base.Package.LeadingDetachedComments...), merge.Package.LeadingDetachedComments...),
		LeadingComments:         base.Package.LeadingComments,
		TrailingComments:        base.Package.TrailingComments,
		Name:                    strings.Replace(merge.Package.Name, s.MergePackage, s.MergedPackage, 1),
	}
	if len(merge.Package.LeadingComments) > 0 {
		out.Package.LeadingComments = merge.Package.LeadingComments
	}
	if len(merge.Package.TrailingComments) > 0 {
		out.Package.TrailingComments = merge.Package.TrailingComments
	}

	outDeps := map[string]*Dependency{}

	mergeDeps := map[string]*Dependency{}
	for _, d := range merge.Dependencies {
		mergeDeps[d.Name] = d
	}

	out.Options = s.mergeOptions(base, merge)

	first := true
	for _, based := range base.Dependencies {
		outD := based
		if mergeD, ok := mergeDeps[based.Name]; ok {
			outD.LeadingDetachedComments = append(append([]string{}, based.LeadingDetachedComments...), mergeD.LeadingDetachedComments...)
			if len(mergeD.LeadingComments) > 0 {
				outD.LeadingComments = mergeD.LeadingComments
			}
			if len(mergeD.TrailingComments) > 0 {
				outD.TrailingComments = mergeD.TrailingComments
			}
		}

		if first {
			first = false
			outD.LeadingDetachedComments = append([]string{"//////\n Dependencies from base\n//////\n"}, outD.LeadingDetachedComments...)
		}

		out.Dependencies = append(out.Dependencies, outD)
		outDeps[outD.Name] = based
	}
	first = true
	for _, mergeD := range merge.Dependencies {
		if _, ok := outDeps[mergeD.Name]; ok {
			continue
		}
		outD := mergeD

		if first {
			first = false
			mergeD.LeadingDetachedComments = append([]string{"//////\n Dependencies from merge\n//////\n"}, mergeD.LeadingDetachedComments...)
		}

		if strings.HasPrefix(outD.Name, s.MergePrefix) {
			outD.Name = strings.Replace(mergeD.Name, s.MergePackage, s.MergedPackage, 1)
		}

		out.Dependencies = append(out.Dependencies, outD)
	}

	out.Enums = s.mergeEnums(base, merge, merged)
	out.Messages = s.mergeMessages(base, merge, merged)

	return out
}

func (s *MergeSpec) mergeOptions(base, merge *File) []*FileOption {
	out := []*FileOption{}
	outMap := map[string]*FileOption{}

	mergeMap := map[string]*FileOption{}
	for _, o := range merge.Options {
		mergeMap[o.Name] = o
	}

	first := true
	for _, baseO := range base.Options {
		log.Printf("base \"%s\"", baseO.Name)
		mergeO, ok := mergeMap[baseO.Name]
		if !ok {
			mergeO = &FileOption{
				Name: baseO.Name,
			}
		}

		outO := &FileOption{
			LeadingDetachedComments: append(append([]string{}, baseO.LeadingDetachedComments...), mergeO.LeadingDetachedComments...),
			LeadingComments:         baseO.LeadingComments,
			TrailingComments:        baseO.TrailingComments,
			Name:                    baseO.Name,
			Value:                   mergeO.Value,
		}

		if first {
			first = false
			outO.LeadingDetachedComments = append([]string{"//////\n Options from base\n//////\n"}, outO.LeadingDetachedComments...)
		}

		out = append(out, outO)
		outMap[outO.Name] = outO
	}

	first = true
	for _, mergeO := range merge.Options {
		log.Printf("merge \"%s\"", mergeO.Name)
		if _, ok := outMap[mergeO.Name]; ok {
			continue
		}

		outO := &FileOption{
			LeadingDetachedComments: append([]string{}, mergeO.LeadingDetachedComments...),
			LeadingComments:         mergeO.LeadingComments,
			TrailingComments:        mergeO.TrailingComments,
			Name:                    mergeO.Name,
			Value:                   mergeO.Value,
		}

		if first {
			first = false
			outO.LeadingDetachedComments = append([]string{"//////\n Options from merge\n//////\n"}, outO.LeadingDetachedComments...)
		}

		out = append(out, outO)
		outMap[outO.Name] = outO
	}

	return out
}

func (s *MergeSpec) mergeEnums(base, merge, merged EnumHaver) []*Enum {
	out := []*Enum{}
	outMap := map[string]*Enum{}

	baseMap := map[string]*Enum{}
	for _, e := range base.GetEnums() {
		baseMap[e.Name] = e
	}

	mergeMap := map[string]*Enum{}
	for _, e := range merge.GetEnums() {
		mergeMap[e.Name] = e
	}

	mergedMap := map[string]*Enum{}
	for _, e := range merged.GetEnums() {
		mergedMap[e.Name] = e
	}

	first := true
	for _, baseE := range base.GetEnums() {
		mergeE, ok := mergeMap[baseE.Name]
		if !ok {
			mergeE = &Enum{
				Name: baseE.Name,
			}
		}

		mergedE, ok := mergedMap[baseE.Name]
		if !ok {
			mergedE = &Enum{
				Name: baseE.Name,
			}
		}

		outE := s.mergeEnum(baseE, mergeE, mergedE)

		if first {
			first = false
			outE.LeadingDetachedComments = append([]string{"//////\n Enums from base\n//////\n"}, outE.LeadingDetachedComments...)
		}

		out = append(out, outE)
		outMap[outE.Name] = outE
	}

	// first = true
	// for _, mergeE := range merge.GetEnums() {
	// 	if _, ok := outMap[mergeE.Name]; ok {
	// 		continue
	// 	}

	// 	baseE := &Enum{
	// 		Name: mergeE.Name,
	// 	}

	// 	mergedE, ok := mergedMap[mergeE.Name]
	// 	if !ok {
	// 		mergedE = &Enum{
	// 			Name: mergeE.Name,
	// 		}
	// 	}

	// 	outE := s.mergeEnum(baseE, mergeE, mergedE)

	// 	if first {
	// 		first = false
	// 		outE.LeadingDetachedComments = append([]string{"//////\n Enums from merge\n//////\n"}, outE.LeadingDetachedComments...)
	// 	}

	// 	out = append(out, mergeE)
	// }

	return out
}

func (s *MergeSpec) mergeEnum(base, merge, merged *Enum) *Enum {
	out := &Enum{
		LeadingDetachedComments: append(append([]string{}, base.LeadingDetachedComments...), merge.LeadingDetachedComments...),
		LeadingComments:         base.LeadingComments,
		TrailingComments:        base.TrailingComments,
		Name:                    base.Name,
	}
	if len(merge.LeadingComments) > 0 {
		out.LeadingComments = merge.LeadingComments
	}
	if len(merge.TrailingComments) > 0 {
		out.TrailingComments = merge.TrailingComments
	}

	outMap := map[string]*EnumValue{}

	usedNumbers := map[int32]bool{}

	numberer := newNumberer(0)
	for _, v := range merged.ReservedRanges {
		for i := v.Start; i <= v.End; i++ {
			numberer.use("", i)
		}
	}
	for _, v := range merged.Values {
		numberer.use(v.Name, v.Number)
	}

	for _, v := range merged.Values {
		usedNumbers[v.Number] = true
	}

	reservedNames := map[string]bool{}
	for _, r := range merge.ReservedNames {
		reservedNames[r.Name] = true
	}

	mergeMap := map[string]*EnumValue{}
	for _, v := range merge.Values {
		mergeMap[v.Name] = v
	}

	mergedMap := map[string]*EnumValue{}
	for _, v := range merged.Values {
		mergedMap[v.Name] = v
	}

	first := true
	for _, baseV := range base.Values {
		if reservedNames[baseV.Name] {
			continue
		}

		mergeV, ok := mergeMap[baseV.Name]
		if !ok {
			mergeV = &EnumValue{
				Name: baseV.Name,
			}
		}

		outV := s.mergeEnumValue(baseV, mergeV, numberer)

		if first {
			first = false
			outV.LeadingDetachedComments = append([]string{"//////\n Values from base\n//////\n"}, outV.LeadingDetachedComments...)
		}

		out.Values = append(out.Values, outV)
		outMap[outV.Name] = outV
	}

	first = true
	for _, mergeV := range merge.Values {
		if _, ok := outMap[mergeV.Name]; ok {
			continue
		}

		baseV := &EnumValue{
			Name: mergeV.Name,
		}

		outV := s.mergeEnumValue(baseV, mergeV, numberer)

		if first {
			first = false
			outV.LeadingDetachedComments = append([]string{"//////\n Values from merge\n//////\n"}, outV.LeadingDetachedComments...)
		}

		out.Values = append(out.Values, outV)
		outMap[outV.Name] = outV
	}

	out.ReservedNames = append(out.ReservedNames, merge.ReservedNames...)

	for _, mergedV := range merged.Values {
		if _, ok := outMap[mergedV.Name]; ok {
			continue
		}
		out.ReservedRanges = append(out.ReservedRanges, &ReservedRange{
			LeadingComments: fmt.Sprintf("Reserved because the field %s was removed\n", mergedV.Name),
			Start:           mergedV.Number,
			End:             mergedV.Number,
		})

		outR := &ReservedName{
			Name: mergedV.Name,
		}
		if !reservedNames[mergedV.Name] {
			out.ReservedNames = append(out.ReservedNames, outR)
		}
	}

	for _, n := range merged.ReservedNames {
		if reservedNames[n.Name] {
			continue
		}
		out.ReservedNames = append(out.ReservedNames, n)
	}
	out.ReservedRanges = append(out.ReservedRanges, merged.ReservedRanges...)

	return out
}

func (s *MergeSpec) mergeEnumValue(base, merge *EnumValue, numberer *numberer) *EnumValue {
	out := &EnumValue{
		LeadingDetachedComments: append(append([]string{}, base.LeadingDetachedComments...), merge.LeadingDetachedComments...),
		LeadingComments:         base.LeadingComments,
		TrailingComments:        base.TrailingComments,
		Name:                    base.Name,
	}
	if len(merge.LeadingComments) > 0 {
		out.LeadingComments = merge.LeadingComments
	}
	if len(merge.TrailingComments) > 0 {
		out.TrailingComments = merge.TrailingComments
	}

	out.Number = numberer.number(out.Name)

	return out
}

func (s *MergeSpec) mergeMessages(base, merge, merged MessageHaver) []*Message {
	out := []*Message{}
	outMap := map[string]*Message{}

	baseMap := map[string]*Message{}
	for _, m := range base.GetMessages() {
		baseMap[m.Name] = m
	}

	mergeMap := map[string]*Message{}
	for _, m := range merge.GetMessages() {
		mergeMap[m.Name] = m
	}

	mergedMap := map[string]*Message{}
	for _, m := range merged.GetMessages() {
		mergedMap[m.Name] = m
	}

	first := true
	for _, baseM := range base.GetMessages() {
		mergeM, ok := mergeMap[baseM.Name]
		if !ok {
			continue
		}

		mergedM, ok := mergedMap[baseM.Name]
		if !ok {
			mergedM = &Message{
				Name: baseM.Name,
			}
		}

		outM := s.mergeMessage(baseM, mergeM, mergedM)

		if first {
			first = false
			outM.LeadingDetachedComments = append([]string{"//////\n Messages from base\n//////\n"}, baseM.LeadingDetachedComments...)
		}

		out = append(out, outM)
		outMap[outM.Name] = outM
	}

	// first = true
	// for _, mergeM := range merge.GetMessages() {
	// 	baseM := &Message{
	// 		Name: mergeM.Name,
	// 	}

	// 	mergedM, ok := mergedMap[mergeM.Name]
	// 	if !ok {
	// 		mergedM = &Message{
	// 			Name: mergeM.Name,
	// 		}
	// 	}
	// 	outM := s.mergeMessage(baseM, mergeM, mergedM)

	// 	if first {
	// 		first = false
	// 		outM.LeadingDetachedComments = append([]string{"//////\n Messages from merge\n//////\n"}, outM.LeadingDetachedComments...)
	// 	}

	// 	out = append(out, outM)
	// }

	return out
}

func (s *MergeSpec) mergeMessage(base, merge, merged *Message) *Message {
	out := &Message{
		Name:                    base.Name,
		LeadingDetachedComments: append(append([]string{}, base.LeadingDetachedComments...), merge.LeadingDetachedComments...),
		LeadingComments:         base.LeadingComments,
		TrailingComments:        base.TrailingComments,
	}
	if len(merge.LeadingComments) > 0 {
		out.LeadingComments = merge.LeadingComments
	}
	if len(merge.TrailingComments) > 0 {
		out.TrailingComments = merge.TrailingComments
	}

	out.Enums = s.mergeEnums(base, merge, merged)
	out.Messages = s.mergeMessages(base, merge, merged)

	numberer := newNumberer(1)
	for _, f := range merged.ReservedRanges {
		for i := f.Start; i <= f.End; i++ {
			numberer.use("", i)
		}
	}

	for _, f := range merged.Fields {
		numberer.use(f.Name, f.Number)
	}

	for _, oneof := range merged.Oneofs {
		for _, f := range oneof.Fields {
			numberer.use(f.Name, f.Number)
		}
	}

	reservedNames := map[string]bool{}
	for _, r := range merge.ReservedNames {
		reservedNames[r.Name] = true
	}

	out.Fields = s.mergeFields(base, merge, numberer, reservedNames)

	// merge oneofs
	outOneofMap := map[string]*Oneof{}

	baseOneOfs := map[string]*Oneof{}
	for _, oneof := range base.Oneofs {
		baseOneOfs[oneof.Name] = oneof
	}

	mergeOneofs := map[string]*Oneof{}
	for _, oneof := range merge.Oneofs {
		mergeOneofs[oneof.Name] = oneof
	}

	mergedOneofs := map[string]*Oneof{}
	for _, oneof := range merged.Oneofs {
		mergedOneofs[oneof.Name] = oneof
	}

	first := true
	for _, baseOneof := range base.Oneofs {
		mergeOneof, ok := mergeOneofs[baseOneof.Name]
		if !ok {
			mergeOneof = &Oneof{
				Name: baseOneof.Name,
			}
		}

		outOneof := s.mergeOneof(baseOneof, mergeOneof, numberer, reservedNames)

		outOneof.LeadingDetachedComments = append([]string{"//////\n Oneofs from base\n//////\n"}, outOneof.LeadingDetachedComments...)

		out.Oneofs = append(out.Oneofs, outOneof)
		outOneofMap[outOneof.Name] = outOneof
	}

	first = true
	for _, mergeOneof := range merge.Oneofs {
		if _, ok := outOneofMap[mergeOneof.Name]; ok {
			continue
		}

		baseOneof := &Oneof{
			Name: mergeOneof.Name,
		}

		outOneof := s.mergeOneof(baseOneof, mergeOneof, numberer, reservedNames)

		if first {
			first = false
			outOneof.LeadingDetachedComments = append([]string{"//////\n Oneofs from merge\n//////\n"}, outOneof.LeadingDetachedComments...)
		}

		out.Oneofs = append(out.Oneofs, outOneof)
	}

	outFields := map[string]*Field{}
	for _, f := range out.Fields {
		outFields[f.Name] = f
	}

	for _, oneof := range out.Oneofs {
		for _, f := range oneof.Fields {
			outFields[f.Name] = f
		}
	}

	// handle removed fields
	out.ReservedNames = append(out.ReservedNames, merge.ReservedNames...)

	for _, n := range merged.ReservedNames {
		if reservedNames[n.Name] {
			continue
		}
		out.ReservedNames = append(out.ReservedNames, n)
	}
	for _, f := range merged.Fields {
		if _, ok := outFields[f.Name]; ok {
			continue
		}
		out.ReservedRanges = append(out.ReservedRanges, &ReservedRange{
			LeadingComments: fmt.Sprintf(" Reserved because the field %s was removed\n", f.Name),
			Start:           f.Number,
			End:             f.Number,
		})
		if !reservedNames[f.Name] {
			out.ReservedNames = append(out.ReservedNames, &ReservedName{
				Name: f.Name,
			})
		}
	}

	out.ReservedRanges = append(out.ReservedRanges, merged.ReservedRanges...)

	return out
}

func (s *MergeSpec) mergeFields(base, merge FieldHaver, numberer *numberer, reservedNames map[string]bool) []*Field {
	out := []*Field{}
	outMap := map[string]*Field{}

	baseMap := map[string]*Field{}
	for _, f := range base.GetFields() {
		baseMap[f.Name] = f
	}

	mergeMap := map[string]*Field{}
	for _, f := range merge.GetFields() {
		mergeMap[f.Name] = f
	}

	first := true
	for _, baseF := range base.GetFields() {
		if reservedNames[baseF.Name] {
			continue
		}

		mergeF, ok := mergeMap[baseF.Name]
		if !ok {
			mergeF = &Field{
				Label: baseF.Label,
				Type:  baseF.Type,
				Name:  baseF.Name,
			}
		}

		outF := s.mergeField(baseF, mergeF, numberer)

		if first {
			first = false
			outF.LeadingDetachedComments = append([]string{"//////\n Fields from base\n//////\n"}, outF.LeadingDetachedComments...)
		}

		out = append(out, outF)
		outMap[outF.Name] = outF
	}

	first = true
	for _, mergeF := range merge.GetFields() {
		if _, ok := outMap[mergeF.Name]; ok {
			continue
		}

		baseF := &Field{
			Label: mergeF.Label,
			Type:  mergeF.Type,
			Name:  mergeF.Name,
		}

		outF := s.mergeField(baseF, mergeF, numberer)

		if first {
			first = false
			outF.LeadingDetachedComments = append([]string{"//////\n Fields from merge\n//////\n"}, outF.LeadingDetachedComments...)
		}

		out = append(out, outF)
		outMap[outF.Name] = outF
	}

	return out
}

func (s *MergeSpec) mergeField(base, merge *Field, numberer *numberer) *Field {
	out := &Field{
		LeadingDetachedComments: append(append([]string{}, base.LeadingDetachedComments...), merge.LeadingDetachedComments...),
		LeadingComments:         base.LeadingComments,
		TrailingComments:        base.TrailingComments,
		Name:                    base.Name,
		Label:                   merge.Label,
		Type:                    merge.Type,
	}
	if len(merge.LeadingComments) > 0 {
		out.LeadingComments = merge.LeadingComments
	}
	if len(merge.TrailingComments) > 0 {
		out.TrailingComments = merge.TrailingComments
	}

	replace := fmt.Sprintf(".%s", s.MergePackage)
	if strings.HasPrefix(merge.Type, replace) {
		out.Type = strings.Replace(out.Type, replace, fmt.Sprintf(".%s", s.MergedPackage), 1)
	}

	out.Number = numberer.number(out.Name)

	return out
}

func (s *MergeSpec) mergeOneof(base, merge *Oneof, numberer *numberer, reservedNames map[string]bool) *Oneof {
	out := &Oneof{
		LeadingDetachedComments: append(append([]string{}, base.LeadingDetachedComments...), merge.LeadingDetachedComments...),
		LeadingComments:         base.LeadingComments,
		TrailingComments:        base.TrailingComments,
		Name:                    base.Name,
	}
	if len(merge.LeadingComments) > 0 {
		out.LeadingComments = merge.LeadingComments
	}
	if len(merge.TrailingComments) > 0 {
		out.TrailingComments = merge.TrailingComments
	}

	out.Fields = s.mergeFields(base, merge, numberer, reservedNames)

	return out
}
