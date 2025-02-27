package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// main rendering function for converting Markdown to Typst
func Render(md []byte, opts Options, h1Level int) (string, error) {
	extensions := parser.CommonExtensions | parser.Strikethrough | parser.Tables | parser.NoEmptyLineBeforeBlock | parser.Includes
	p := parser.NewWithExtensions(extensions)
	doc := markdown.Parse(md, p)
	r := NewTypRenderer(opts, h1Level)
	ast.Walk(doc, &typVisitor{r: r})
	return r.builder.String(), nil
}

// delegate a walker from typRenderer when calling ast.Walk
type typVisitor struct {
	r *typRenderer
}

// implements the ast.Visitor interface
func (v *typVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	return v.r.walker(node, entering)
}

// typRenderer generate Typst content from markdown AST
type typRenderer struct {
	builder           *IndentedBuilder
	opts              Options
	h1Level           int
	skipBlocks        bool             // <!--typst-begin-exclude--> ~ <!--typst-end-exclude--> 구간 건너뛰기
	altTextBuffer     *strings.Builder // 이미지 alt 텍스트 임시 저장
	currentTableMeta  *tableMeta       // 테이블 meta 정보
	currentImageMeta  *imageMeta       // 이미지 meta 정보
	rawTypstNext      bool             // raw-typst 주석 이후 다음 code block을 그대로 삽입
}

// generate a new typRenderer instance with indented builder and options
func NewTypRenderer(opts Options, h1Level int) *typRenderer {
	return &typRenderer{
		builder: NewIndentedBuilder("  "),
		opts:    opts,
		h1Level: h1Level,
	}
}

// walker function for AST nodes to render Typst content from Markdown
func (r *typRenderer) walker(node ast.Node, entering bool) ast.WalkStatus {
	if r.skipBlocks && !isEndExclude(node) {
		return ast.GoToNext
	}

	switch n := node.(type) {

	// ---------- Heading ----------
	case *ast.Heading:
		if entering {
			r.builder.WriteLine("")
			eqCount := int(n.Level-1) + r.h1Level
			header := strings.Repeat("=", eqCount) + " "
			r.builder.Write(header)
		} else {
			r.builder.WriteLine("")
		}

	// ---------- Paragraph ----------
	case *ast.Paragraph:
		if !entering {
			r.builder.WriteLine("\n")
		}

	// ---------- BlockQuote ----------
	case *ast.BlockQuote:
		if entering {
			content := r.renderChildNodes(n)
			r.builder.WriteLine("")
			r.builder.Write(fmt.Sprintf("#quote(block:true, \"%s\")", content))
			return ast.SkipChildren
		}

	// ---------- Emphasis / Strong / Strikethrough ----------
	case *ast.Emph:
		if entering {
			r.builder.Write("#emph[")
		} else {
			r.builder.Write("]")
		}
	case *ast.Strong:
		if entering {
			r.builder.Write("#strong[")
		} else {
			r.builder.Write("]")
		}
	case *ast.Del:
		if entering {
			r.builder.Write("#strike[")
		} else {
			r.builder.Write("]")
		}

	// ---------- Inline Code ----------
	case *ast.Code:
		if entering {
			r.builder.Write("#raw(block:false, \"")
			r.builder.Write(escapeString(string(n.Literal)))
			r.builder.Write("\")")
		}

	// ---------- Code Block ----------
	case *ast.CodeBlock:
		if entering {
			if r.rawTypstNext {
				r.builder.WriteLine(string(n.Literal))
				r.rawTypstNext = false
			} else {
				r.builder.Write("#raw(block:true,")
				if len(n.Info) > 0 {
					tokens := strings.Fields(string(n.Info))
					langOnly := tokens[0]
					r.builder.Write(fmt.Sprintf(" lang:\"%s\",", escapeString(langOnly)))
				}
				r.builder.Write(" \"")
				r.builder.Write(escapeString(string(n.Literal)))
				r.builder.WriteLine("\")")
			}
		}

	// ---------- Horizontal Rule ----------
	case *ast.HorizontalRule:
		if entering {
			r.builder.WriteLine("#line(length:100%)")
		}

	// ---------- Lists ----------
	case *ast.List:
		if entering {
			if (n.ListFlags & ast.ListTypeOrdered) != 0 {
				r.builder.Write("#enum(start:1,")
			} else {
				r.builder.Write("#list(")
			}
		} else {
			r.builder.WriteLine(")\n")
		}
	case *ast.ListItem:
		if entering {
			// render child nodes and trim spaces
			content := strings.TrimSpace(r.renderChildNodes(n))
			// write list item line by line
			r.builder.WriteLine("[" + content + "],")
			return ast.SkipChildren
		}

	// ---------- Tables ----------
	case *ast.Table:
		if entering {
			var meta tableMeta
			if r.currentTableMeta != nil {
				meta = *r.currentTableMeta
			} else {
				meta = tableMeta{
					Caption:   "",
					Placement: "none",
					Columns:   "",
					Align:     "",
				}
			}

			// if columns is not set, find the first TableHeader and calculate the number of cells
			if meta.Columns == "" {
				hfv := &headerFinderVisitor{}
				ast.Walk(n, hfv)
				headerCells := 0
				if hfv.header != nil {
					headerCells = headerCellsCount(hfv.header)
				}
				if headerCells == 0 {
					headerCells = 1
				}
				meta.Columns = "(" + strconv.Itoa(headerCells) + ")"
			}

			// collect table content
			tableContent := r.collectTableContent(n)
			tableData := TableData{
				Caption:   meta.Caption,
				Placement: meta.Placement,
				Columns:   meta.Columns,
				Align:     meta.Align,
				Label:     meta.Label,
				Rows:      tableContent,
			}

			// render table
			tableStr, err := RenderTable(tableData)
			if err != nil {
				r.builder.WriteLine("#table( ... )")
			} else {
				r.builder.Write(tableStr)
			}
			return ast.SkipChildren
		}

	// ---------- Table Header & Table Cell ----------
	case *ast.TableHeader:
		if entering {
			r.builder.Write("table.header(")
		} else {
			r.builder.Write("),")
		}
	case *ast.TableRow:
		if !entering {
			r.builder.WriteLine("")
			return ast.SkipChildren
		}
	case *ast.TableCell:
		if entering {
			r.builder.Write("[")
		} else {
			r.builder.Write("],")
		}

	// ---------- Links ----------
	case *ast.Link:
		if entering {
			dest := string(n.Destination)
			r.builder.Write("#link(\"")
			r.builder.Write(escapeString(dest))
			r.builder.Write("\")[")
		} else {
			r.builder.Write("]")
		}

	// ---------- Images ----------
	case *ast.Image:
		// If we're entering the image node, process its children to extract alt text
		if entering {
			// Save the current builder and create a new one for temporary use
			oldBuilder := r.builder
			r.altTextBuffer = &strings.Builder{}
			r.builder = NewIndentedBuilder("")

			// Walk through the image node's children to collect alt text
			for _, child := range n.Children {
				ast.Walk(child, &typVisitor{r: r})
			}
			// Extract and trim the alt text from the temporary builder
			altText := strings.TrimSpace(r.builder.String())
			r.builder = oldBuilder

			// Extract the destination of the image
			dest := string(n.Destination)

			// set label if exists
			var label string
			if r.currentImageMeta != nil && r.currentImageMeta.Label != "" {
				label = r.currentImageMeta.Label
				// Clear the current image meta after use
				r.currentImageMeta = nil
			}

			// render image
			figData := FigureData{
				ImagePath: dest,
				Caption:   altText,
				Label:     label,
			}
			figStr, err := RenderFigure(figData)
			if err != nil {
				r.builder.WriteLine(fmt.Sprintf("#figure( image: \"%s\" )", escapeString(dest)))
			} else {
				r.builder.Write(figStr)
			}

			// ignore children nodes due to already processed
			return ast.SkipChildren
		}

	// ---------- HTML Blocks / Spans ----------
	case *ast.HTMLSpan, *ast.HTMLBlock:
		if entering {
			htmlContent := ""
			switch x := n.(type) {
			case *ast.HTMLSpan:
				htmlContent = string(x.Literal)
			case *ast.HTMLBlock:
				htmlContent = string(x.Literal)
			}

			// check if the block should be excluded
			if isBeginExclude(htmlContent) {
				r.skipBlocks = true
				return ast.GoToNext
			}
			if isEndExclude(n) {
				r.skipBlocks = false
				return ast.GoToNext
			}

			// check if the block contains raw Typst content
			if isRawTypst(htmlContent) {
				r.rawTypstNext = true
				return ast.GoToNext
			}

			// check if the block contains metadata
			if metaRaw, ok := isTableMetaComment(n); ok {
				m := parseTableMeta(metaRaw)
				r.currentTableMeta = &m
				return ast.GoToNext
			}
			if metaRaw, ok := isImageMetaComment(n); ok {
				im := parseImageMeta(metaRaw)
				r.currentImageMeta = &im
				return ast.GoToNext
			}
		}

	// ---------- ETC. ----------
	case *ast.Text:
		if entering {
			r.builder.Write(string(n.Literal))
		}
	case *ast.Math:
		if entering {
			content := string(n.Literal)
			r.builder.Write("$" + escapeString(content) + "$")
		}
	case *ast.Softbreak:
		if entering {
			r.builder.Write(" ")
		}
	case *ast.Hardbreak:
		if entering {
			r.builder.Write("\\ ")
		}
	default:
		// 기타 노드: 기본 순회 진행
	}

	return ast.GoToNext
}

// render child nodes of the given node
func (r *typRenderer) renderChildNodes(n ast.Node) string {
	var tempBuilder strings.Builder
	tempRenderer := *r
	tempRenderer.builder = NewIndentedBuilder("")
	for _, child := range n.GetChildren() {
		ast.Walk(child, &typVisitor{r: &tempRenderer})
	}
	return strings.TrimSuffix(tempBuilder.String()+tempRenderer.builder.String(), "\n")
}

// collect table content from the given table node
func (r *typRenderer) collectTableContent(table *ast.Table) string {
	var buf strings.Builder
	for _, child := range table.GetChildren() {
		tempRenderer := *r
		tempRenderer.builder = NewIndentedBuilder("")
		ast.Walk(child, &typVisitor{r: &tempRenderer})
		buf.WriteString(tempRenderer.builder.String())
	}
	return buf.String()
}

// type for table meta information (fill from tableMeta)
type TableData struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
	Rows      string
}

// function for rendering table content using template/table.tpl
func RenderTable(data TableData) (string, error) {
	tpl, err := template.ParseFiles("templates/table.tpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// type for image meta information (fill from imageMeta)
type FigureData struct {
	ImagePath string
	Caption   string
	Label     string
}

// function for rendering figure content using template/figure.tpl
func RenderFigure(data FigureData) (string, error) {
	tpl, err := template.ParseFiles("templates/figure.tpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
