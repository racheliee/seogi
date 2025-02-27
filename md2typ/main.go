package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v2"
)

// -----------------------------------------------------------------------------
// 기본 상수 및 타입
// -----------------------------------------------------------------------------

// Options 및 관련 상수 (필요에 따라 확장 가능)
type Options uint8

const (
	OptionDummy1 = 1 << iota
	OptionDummy2
)

// YAML 프론트매터의 내용을 담을 Metadata 구조체
type Metadata struct {
	Title        string `yaml:"title"`
	Course       string `yaml:"course"`
	Date         string `yaml:"date"`
	Authors      []struct {
		Name         string `yaml:"name"`
		Department   string `yaml:"department"`
		Organization string `yaml:"organization"`
		Email        string `yaml:"email"`
	} `yaml:"authors"`
	Bibliography string `yaml:"bibliography"`
	Toc          bool   `yaml:"toc"`
}

// 테이블 메타 정보를 위한 구조체 (HTML 주석 처리 등에서 사용)
type tableMeta struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
}

// 이미지 메타 정보를 위한 구조체
type imageMeta struct {
	Label string
}

// -----------------------------------------------------------------------------
// 인덴테이션 헬퍼: IndentedBuilder
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// YAML 프론트매터 추출 및 Typst 헤더 생성
// -----------------------------------------------------------------------------

// extractYAMLHeader는 Markdown 파일의 시작부분 YAML 헤더를 추출한 후,
// 메타데이터와 YAML 헤더를 제거한 Markdown 본문을 반환합니다.
func extractYAMLHeader(mdData []byte) (*Metadata, []byte, error) {
	mdStr := string(mdData)
	lines := strings.Split(mdStr, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		var yamlLines []string
		var i int
		for i = 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				break
			}
			yamlLines = append(yamlLines, lines[i])
		}
		yamlContent := strings.Join(yamlLines, "\n")
		var meta Metadata
		if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
			return nil, mdData, err
		}
		content := strings.Join(lines[i+1:], "\n")
		return &meta, []byte(content), nil
	}
	return nil, mdData, nil
}

// generateTypstHeader는 templates/typst_header.tpl 파일을 읽어 YAML 메타데이터 기반 Typst 헤더를 생성합니다.
func generateTypstHeader(meta *Metadata) (string, error) {
	tpl, err := template.ParseFiles("templates/typst_header.tpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, meta); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// -----------------------------------------------------------------------------
// AST 변환: Markdown → Typst
// -----------------------------------------------------------------------------

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
	builder           *IndentedBuilder // 인덴테이션 헬퍼 사용
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

// walker는 AST 노드를 순회하며 각 노드 유형에 따라 Typst 코드를 생성합니다.
func (r *typRenderer) walker(node ast.Node, entering bool) ast.WalkStatus {
	// Exclusion 처리: <!--typst-begin-exclude--> ~ <!--typst-end-exclude-->
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
			r.builder.WriteLine("")
		}

	// ---------- BlockQuote ----------
	case *ast.BlockQuote:
		if entering {
			content := r.renderChildNodes(n)
			// 예시: blockquote를 #quote 함수로 변환
			r.builder.WriteLine("")
			r.builder.Write(fmt.Sprintf("#quote(block:true, \"%s\")", content))
			return ast.SkipChildren
		}

	// ---------- Emphasis & Strong & Strikethrough ----------
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
			r.builder.WriteLine(")")
		}
	case *ast.ListItem:
		if entering {
			content := r.renderChildNodes(n)
			r.builder.Write("[")
			r.builder.Write(content)
			r.builder.WriteLine("],")
			return ast.SkipChildren
		}

	// ---------- Table ----------
	case *ast.Table:
		if entering {
			// meta 정보 처리 (HTML 주석 등에서 이미 설정된 경우 사용)
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
			// Columns 값이 비어있으면 기본값 지정
			if meta.Columns == "" {
				meta.Columns = "(6em, auto, auto)"
			}
			// 테이블의 자식 노드(헤더, 행, 셀) 내용을 수집
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
			// alt 텍스트 수집
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

	// ---------- HTML Blocks / Spans (메타, exclude 등) ----------
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

	// ---------- Text ----------
	case *ast.Text:
		if entering {
			r.builder.Write(string(n.Literal))
		}

	// ---------- Math ----------
	case *ast.Math:
		if entering {
			content := string(n.Literal)
			r.builder.Write("$" + escapeString(content) + "$")
		}

	// ---------- Softbreak / Hardbreak ----------
	case *ast.Softbreak:
		if entering {
			r.builder.Write(" ")
		}
	case *ast.Hardbreak:
		if entering {
			r.builder.Write("\\ ")
		}

	default:
		// 기타 노드는 기본 순회 진행
	}

	return ast.GoToNext
}

// renderChildNodes는 주어진 노드의 자식들을 임시 버퍼에 렌더링하여 문자열로 반환합니다.
func (r *typRenderer) renderChildNodes(n ast.Node) string {
	var tempBuilder strings.Builder
	tempRenderer := *r
	// 임시 버퍼로 교체
	tempRenderer.builder = NewIndentedBuilder("")
	for _, child := range n.GetChildren() {
		ast.Walk(child, &typVisitor{r: &tempRenderer})
	}
	return strings.TrimSuffix(tempBuilder.String()+tempRenderer.builder.String(), "\n")
}

// -----------------------------------------------------------------------------
// 유틸리티 함수
// -----------------------------------------------------------------------------

// escapeString은 Typst 코드 내에 사용될 때, 백슬래시나 따옴표 등을 이스케이프 처리합니다.
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

// isBeginExclude는 HTML에 <!--typst-begin-exclude-->가 포함되었는지 확인합니다.
func isBeginExclude(html string) bool {
	return strings.Contains(html, "<!--typst-begin-exclude")
}

// isEndExclude는 현재 노드가 <!--typst-end-exclude-->를 포함하는지 확인합니다.
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

// isTableMetaComment는 HTML 블록/스팬 내에 <!--typst-table ...--> 주석이 있는지 확인합니다.
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

// parseTableMeta는 주어진 meta 문자열을 분석해 tableMeta를 반환합니다.
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

func (r *typRenderer) collectTableContent(table *ast.Table) string {
	var buf strings.Builder
	// table의 모든 자식 노드를 순회합니다.
	for _, child := range table.GetChildren() {
		// 임시 렌더러를 사용하여 해당 자식의 Typst 출력을 수집
		tempRenderer := *r
		tempRenderer.builder = NewIndentedBuilder("") // 새로운 빌더 사용
		ast.Walk(child, &typVisitor{r: &tempRenderer})
		buf.WriteString(tempRenderer.builder.String())
	}
	return buf.String()
}

// isImageMetaComment는 HTML 블록/스팬 내에 <!--typst-image ...--> 주석이 있는지 확인합니다.
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

// parseImageMeta는 주어진 meta 문자열을 분석해 imageMeta를 반환합니다.
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

// -----------------------------------------------------------------------------
// Figure 및 Table 템플릿 처리
// -----------------------------------------------------------------------------

// FigureData는 이미지 처리를 위한 데이터를 담는 구조체
type FigureData struct {
	ImagePath string
	Caption   string
	Label     string
}

// RenderFigure는 templates/figure.tpl 파일을 이용해 이미지 관련 Typst 코드를 생성합니다.
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

// TableData는 테이블 처리를 위한 데이터를 담는 구조체
type TableData struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
	Rows      string // 추가: 테이블 행/셀 내용
}

// RenderTable는 templates/table.tpl 파일을 이용해 테이블 관련 Typst 코드를 생성합니다.
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

// -----------------------------------------------------------------------------
// 메인 함수: 전체 파이프라인 실행
// -----------------------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "사용법: %s <input.md> [output.typ]\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	var outputFile string
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	} else {
		ext := filepath.Ext(inputFile)
		if strings.EqualFold(ext, ".md") {
			base := strings.TrimSuffix(inputFile, ext)
			outputFile = base + ".typ"
		} else {
			outputFile = inputFile + ".typ"
		}
	}

	// 입력 Markdown 파일 읽기
	mdData, err := os.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	// YAML 프론트매터 추출
	meta, content, err := extractYAMLHeader(mdData)
	if err != nil {
		panic(err)
	}

	var header string
	if meta != nil {
		header, err = generateTypstHeader(meta)
		if err != nil {
			panic(err)
		}
	}

	opts := Options(OptionDummy1 | OptionDummy2)
	typstBody, err := Render(content, opts, 1)
	if err != nil {
		panic(err)
	}

	finalOutput := header + "\n" + typstBody
	err = os.WriteFile(outputFile, []byte(finalOutput), 0644)
	if err != nil {
		panic(err)
	}

	_, _ = io.WriteString(os.Stdout, fmt.Sprintf("success: %s -> %s\n", inputFile, outputFile))
}
