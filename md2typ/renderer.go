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

type Options uint8

const (
	OptionDummy1 = 1 << iota
	OptionDummy2
)

// Render는 Markdown 데이터를 파싱하여 Typst 코드 문자열로 변환합니다.
func Render(md []byte, opts Options, h1Level int) (string, error) {
	extensions := parser.CommonExtensions | parser.Strikethrough | parser.Tables | parser.NoEmptyLineBeforeBlock | parser.Includes
	p := parser.NewWithExtensions(extensions)
	doc := markdown.Parse(md, p)
	r := NewTypRenderer(opts, h1Level)
	ast.Walk(doc, &typVisitor{r: r})
	return r.builder.String(), nil
}

// typVisitor는 ast.Walk 호출 시 typRenderer의 walker를 위임합니다.
type typVisitor struct {
	r *typRenderer
}

func (v *typVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	return v.r.walker(node, entering)
}

// typRenderer는 AST 순회를 통해 Typst 코드를 생성합니다.
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

func NewTypRenderer(opts Options, h1Level int) *typRenderer {
	return &typRenderer{
		builder: NewIndentedBuilder("  "),
		opts:    opts,
		h1Level: h1Level,
	}
}

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
			// 자식 노드를 렌더링한 후, 앞뒤 공백을 제거
			content := strings.TrimSpace(r.renderChildNodes(n))
			// 각 항목을 새 줄로 출력하도록 WriteLine을 사용
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
			// 기존: 하드코딩 대신, meta.Columns 값이 비어있으면 헤더의 TableCell 개수에 따라 자동 생성
			if meta.Columns == "" {
				// headerFinderVisitor를 사용하여 첫 번째 TableHeader를 찾고 셀 개수를 계산
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
			tableContent := r.collectTableContent(n)
			tableData := TableData{
				Caption:   meta.Caption,
				Placement: meta.Placement,
				Columns:   meta.Columns,
				Align:     meta.Align,
				Label:     meta.Label,
				Rows:      tableContent,
			}
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
		if entering {
			oldBuilder := r.builder
			r.altTextBuffer = &strings.Builder{}
			r.builder = NewIndentedBuilder("")
			for _, child := range n.Children {
				ast.Walk(child, &typVisitor{r: r})
			}
			altText := strings.TrimSpace(r.builder.String())
			r.builder = oldBuilder
			dest := string(n.Destination)
			var label string
			if r.currentImageMeta != nil && r.currentImageMeta.Label != "" {
				label = r.currentImageMeta.Label
				r.currentImageMeta = nil
			}
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
			if isBeginExclude(htmlContent) {
				r.skipBlocks = true
				return ast.GoToNext
			}
			if isEndExclude(n) {
				r.skipBlocks = false
				return ast.GoToNext
			}
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
			if strings.Contains(htmlContent, "<!--raw-typst") {
				r.rawTypstNext = true
				return ast.GoToNext
			}
		}
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

func (r *typRenderer) renderChildNodes(n ast.Node) string {
	var tempBuilder strings.Builder
	tempRenderer := *r
	tempRenderer.builder = NewIndentedBuilder("")
	for _, child := range n.GetChildren() {
		ast.Walk(child, &typVisitor{r: &tempRenderer})
	}
	return strings.TrimSuffix(tempBuilder.String()+tempRenderer.builder.String(), "\n")
}

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

type FigureData struct {
	ImagePath string
	Caption   string
	Label     string
}

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

type TableData struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
	Rows      string
}

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
