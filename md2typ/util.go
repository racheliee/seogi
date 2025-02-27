package main

import (
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// table / image meta for md comments -> used to fill tableData / imageData
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

// custom strings.Builder that supports indentation
type IndentedBuilder struct {
	builder *strings.Builder
	indent  string
	level   int
}

// generate new IndentedBuilder with given indent string
func NewIndentedBuilder(indent string) *IndentedBuilder {
	return &IndentedBuilder{
		builder: &strings.Builder{},
		indent:  indent,
		level:   0,
	}
}

// write line with indentation
func (ib *IndentedBuilder) WriteLine(line string) {
	ib.builder.WriteString(strings.Repeat(ib.indent, ib.level))
	ib.builder.WriteString(line)
	ib.builder.WriteByte('\n')
}


// write text without indentation
func (ib *IndentedBuilder) Write(text string) {
	ib.builder.WriteString(text)
}

// increase / decrease indentation level
func (ib *IndentedBuilder) Increase() {
	ib.level++
}

func (ib *IndentedBuilder) Decrease() {
	if ib.level > 0 {
		ib.level--
	}
}

// return string representation of IndentedBuilder
func (ib *IndentedBuilder) String() string {
	return ib.builder.String()
}

// escape string
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

// special comment checkers (exclude, raw-typst, table, image)
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

func isRawTypst(html string) bool {
	return strings.Contains(html, "<!--raw-typst")
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

// parse meta data that given from comment
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

// visitor for finding table header node
type headerFinderVisitor struct {
	header ast.Node
}

// implement ast.Visitor interface for headerFinderVisitor that set v.header to table header
func (v *headerFinderVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableHeader); ok {
			v.header = n
			return ast.Terminate
		}
	}
	return ast.GoToNext
}

// visitor for counting table cell nodes
type cellCounterVisitor struct {
	count int
}

// implement ast.Visitor interface for cellCounterVisitor that count table cell nodes
func (v *cellCounterVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableCell); ok {
			v.count++
			return ast.SkipChildren
		}
	}
	return ast.GoToNext
}

// function that count and return table header cells count for set column count
func headerCellsCount(node ast.Node) int {
	visitor := &cellCounterVisitor{}
	ast.Walk(node, visitor)
	return visitor.count
}
