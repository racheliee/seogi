package main

import (
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// --- tableMeta, imageMeta, IndentedBuilder 등 이미 선언되어 있음 ---
// (만약 중복 선언이 있다면 여기서만 관리)

// tableMeta 및 imageMeta는 metadata를 위한 구조체 (renderer.go에서 사용한 것과 동일)
type tableMeta struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
}

type imageMeta struct {
	Label string
}

// IndentedBuilder는 문자열 빌더에 자동 인덴테이션을 적용합니다.
type IndentedBuilder struct {
	builder *strings.Builder
	indent  string
	level   int
}

func NewIndentedBuilder(indent string) *IndentedBuilder {
	return &IndentedBuilder{
		builder: &strings.Builder{},
		indent:  indent,
		level:   0,
	}
}

func (ib *IndentedBuilder) WriteLine(line string) {
	ib.builder.WriteString(strings.Repeat(ib.indent, ib.level))
	ib.builder.WriteString(line)
	ib.builder.WriteByte('\n')
}

func (ib *IndentedBuilder) Write(text string) {
	ib.builder.WriteString(text)
}

func (ib *IndentedBuilder) Increase() {
	ib.level++
}

func (ib *IndentedBuilder) Decrease() {
	if ib.level > 0 {
		ib.level--
	}
}

func (ib *IndentedBuilder) String() string {
	return ib.builder.String()
}

// escapeString은 Typst 코드 내에서 백슬래시와 따옴표를 이스케이프합니다.
func escapeString(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if ch == '\\' || ch == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// isBeginExclude, isEndExclude, isTableMetaComment, parseTableMeta, isImageMetaComment, parseImageMeta는 아래와 같이 정의합니다.

func isBeginExclude(html string) bool {
	return strings.Contains(html, "<!--typst-begin-exclude")
}

func isEndExclude(node ast.Node) bool {
	switch x := node.(type) {
	case *ast.HTMLSpan:
		return strings.Contains(string(x.Literal), "<!--typst-end-exclude")
	case *ast.HTMLBlock:
		return strings.Contains(string(x.Literal), "<!--typst-end-exclude")
	default:
		return false
	}
}

func isTableMetaComment(node ast.Node) (string, bool) {
	var literal string
	switch x := node.(type) {
	case *ast.HTMLSpan:
		literal = string(x.Literal)
	case *ast.HTMLBlock:
		literal = string(x.Literal)
	default:
		return "", false
	}
	idx := strings.Index(literal, "<!--typst-table")
	if idx == -1 {
		return "", false
	}
	end := strings.Index(literal, "-->")
	if end == -1 {
		end = len(literal)
	}
	metaRaw := literal[idx+len("<!--typst-table") : end]
	return metaRaw, true
}

func parseTableMeta(metaRaw string) tableMeta {
	tm := tableMeta{
		Caption:   "",
		Placement: "none",
		Columns:   "",
		Align:     "",
	}
	lines := strings.Split(metaRaw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "caption":
			tm.Caption = val
		case "placement":
			tm.Placement = val
		case "columns":
			tm.Columns = val
		case "align":
			tm.Align = val
		case "label":
			tm.Label = val
		}
	}
	return tm
}

func isImageMetaComment(node ast.Node) (string, bool) {
	var literal string
	switch x := node.(type) {
	case *ast.HTMLSpan:
		literal = string(x.Literal)
	case *ast.HTMLBlock:
		literal = string(x.Literal)
	default:
		return "", false
	}
	idx := strings.Index(literal, "<!--typst-image")
	if idx == -1 {
		return "", false
	}
	end := strings.Index(literal, "-->")
	if end == -1 {
		end = len(literal)
	}
	metaRaw := literal[idx+len("<!--typst-image") : end]
	return metaRaw, true
}

func parseImageMeta(metaRaw string) imageMeta {
	im := imageMeta{}
	lines := strings.Split(metaRaw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "label" {
			im.Label = val
		}
	}
	return im
}

// --- Header Finder & Cell Counter for Table ---
type headerFinderVisitor struct {
	header ast.Node
}

func (v *headerFinderVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableHeader); ok {
			v.header = n
			return ast.Terminate
		}
	}
	return ast.GoToNext
}

type cellCounterVisitor struct {
	count int
}

func (v *cellCounterVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableCell); ok {
			v.count++
			return ast.SkipChildren
		}
	}
	return ast.GoToNext
}

func headerCellsCount(node ast.Node) int {
	visitor := &cellCounterVisitor{}
	ast.Walk(node, visitor)
	return visitor.count
}
