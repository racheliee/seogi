// go build -o md2typ .	: 실행 파일 생성
// ./md2typ ./sample/convert-test.md : 테스트 실행

// TODO: subpar image 주석과 대응하게 사용할 수 있도록 수정
// TODO: yaml header를 통해 template의 메타데이터 설정 가능하도록 수정

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// -----------------------------------------------------------------------------
// Constants & Basic Types
// -----------------------------------------------------------------------------

// 템플릿용 더미 옵션 2개
const (
	OptionDummy1 = 1 << iota // 1
	OptionDummy2             // 2
)

// Options는 여러 옵션을 동시에 담기 위한 비트 플래그 타입
type Options uint8

// table의 meta 정보를 담는 구조체
type tableMeta struct {
	Caption   string
	Placement string
	Columns   string
	Align     string
	Label     string
}

// image의 meta 정보를 담는 구조체
type imageMeta struct {
	Label     string
}

// -----------------------------------------------------------------------------
// Debugging & AST Traversal Types / Functions
// -----------------------------------------------------------------------------

// printAST는 Markdown 파싱 후 생성되는 AST(Abstract Syntax Tree) 구조를 콘솔에 출력하는 디버깅용 함수
func printAST(node ast.Node, depth int) {
	ast.Walk(node, &astVisitor{depth: depth})
}

// astVisitor는 AST를 순회하며 노드 정보를 콘솔에 출력하는 디버깅용 구조체
type astVisitor struct {
	depth int
}

// Visit 함수는 ast.Walk에 의해 호출되어, AST의 각 노드를 방문할 때 실행됨
func (v *astVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	if node == nil {
		return ast.GoToNext
	}

	// 노드 깊이에 따른 들여쓰기 설정 (AST 트리 구조 시각화)
	indent := strings.Repeat("  ", v.depth)
	fmt.Printf("%s- %T\n", indent, node)

	// 들어갈 때(depth 증가) / 빠져나올 때(depth 감소) 조정
	if entering {
		v.depth++
	} else {
		v.depth--
	}

	return ast.GoToNext
}

// -----------------------------------------------------------------------------
// Utility Functions (Table Meta, Image Meta, Exclusion, Raw Typst, String Escape)
// -----------------------------------------------------------------------------

// isTableMetaComment: HTML 블록 혹은 스팬 내에 `<!--typst-table` 라인이 있으면 해당 문자열을 반환
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
	// 주석 닫힘 `-->`까지 포함
	end := strings.Index(literal, "-->")
	if end == -1 {
		// 닫힘 표시가 없으면 남은 문자열 전부
		end = len(literal)
	}

	// <!--typst-table 부터 --> 직전까지 추출
	metaRaw := literal[idx+len("<!--typst-table") : end]
	return metaRaw, true
}

// parseTableMeta: 주석으로부터 table meta 정보를 파싱하여 tableMeta 구조체로 반환
func parseTableMeta(metaRaw string) tableMeta {
	// 기본값 설정
	tm := tableMeta{
		Caption:   "Default Table Caption", // 기본 캡션
		Placement: "none",                  // 기본 위치 지정 없음
		Columns:   "(auto, auto, auto)",    // 기본 3열
		Align:     "(start, start, start)", // 기본 왼쪽 정렬
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
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")
		// 각 필드에 값이 있는 경우에만 기본값 덮어쓰기
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

// isImageMetaComment: HTML 내에 `<!--typst-image` 라인이 있으면 해당 문자열 반환
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

// parseImageMeta: 주석으로부터 image meta 정보를 파싱 후 imageMeta 구조체로 반환
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
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")
		if key == "label" {
			im.Label = val
		}
	}
	return im
}

// isBeginExclude는 HTML에 <!--typst-begin-exclude가 포함되었는지 확인
func isBeginExclude(html string) bool {
	return strings.Contains(html, "<!--typst-begin-exclude")
}

// isEndExclude는 현재 노드가 <!--typst-end-exclude-->인지 확인
func isEndExclude(node ast.Node) bool {
	if htmlSpan, ok := node.(*ast.HTMLSpan); ok {
		return strings.Contains(string(htmlSpan.Literal), "<!--typst-end-exclude")
	}
	if htmlBlock, ok := node.(*ast.HTMLBlock); ok {
		return strings.Contains(string(htmlBlock.Literal), "<!--typst-end-exclude")
	}
	return false
}

// escapeString은 Typst 코드 내에서 필요한 백슬래시, 따옴표 등을 이스케이프 처리
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

// -----------------------------------------------------------------------------
// Typst 변환 관련 Types & Functions
// -----------------------------------------------------------------------------

// typRenderer는 Markdown AST를 Typst 포맷으로 변환하기 위한 렌더러 구조체
type typRenderer struct {
	builder           *strings.Builder // 변환 결과를 담을 버퍼
	opts              Options          // 활성화된 옵션 (현재는 더미 옵션)
	h1Level           int              // Heading 레벨 보정값
	skipBlocks        bool             // 특정 구간(<!--typst-begin-exclude--> ~ <!--typst-end-exclude>) 스킵 여부
	altTextBuffer     *strings.Builder // 이미지 대체 텍스트를 임시로 모으는 버퍼
	currentTableMeta  *tableMeta       // 현재 테이블의 meta 정보
	currentImageMeta  *imageMeta       // 현재 이미지의 meta 정보 (label 등)
	rawTypstNext      bool             // raw-typst 주석 후 다음 code block을 typst 코드 그대로 삽입하기 위한 플래그
}

// typVisitor는 ast.Walk 시 실제 노드 방문 로직을 typRenderer에 위임하기 위한 구조체
type typVisitor struct {
	r *typRenderer
}

// Visit는 Markdown AST를 순회하면서 각 노드를 typRenderer.walker에 전달
func (v *typVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	return v.r.walker(node, entering)
}

// newTypRenderer는 typRenderer 구조체 인스턴스를 생성해 반환
func newTypRenderer(opts Options, h1Level int) *typRenderer {
	return &typRenderer{
		builder: &strings.Builder{},
		opts:    opts,
		h1Level: h1Level,
	}
}

// Render 함수는 Markdown 텍스트를 AST로 파싱한 뒤, Typst 문자열로 변환해 반환
func Render(md []byte, opts Options, h1Level int) (string, error) {
	// Markdown 파서에 사용할 확장 기능들 설정
	extensions := parser.CommonExtensions |
		parser.Strikethrough |
		parser.Tables |
		parser.NoEmptyLineBeforeBlock |
		parser.Includes
	// 지정된 확장 기능을 가진 파서 생성
	p := parser.NewWithExtensions(extensions)
	// Markdown 텍스트를 AST로 파싱
	doc := markdown.Parse(md, p)

	// 디버그용 AST 출력
	fmt.Println("==== AST 구조 출력 ====")
	printAST(doc, 0)
	fmt.Println("========================")

	// 생성한 AST를 기반으로 Typst 변환기를 생성 후 순회하며 변환 수행
	r := newTypRenderer(opts, h1Level)
	ast.Walk(doc, &typVisitor{r})
	return r.builder.String(), nil
}

// walker 함수는 AST를 순회하며, 노드 종류와 상태에 따라 Typst 코드를 생성
func (r *typRenderer) walker(node ast.Node, entering bool) ast.WalkStatus {
	// <!--typst-begin-exclude-->가 등장하여 skipBlocks가 true인 경우,
	// <!--typst-end-exclude--> 노드가 나올 때까지 모든 블록을 건너뜀
	if r.skipBlocks && !isEndExclude(node) {
		return ast.GoToNext
	}

	switch n := node.(type) {

	// ---------------------------------------------------------
	// HEADINGS
	// ---------------------------------------------------------
	case *ast.Heading:
		if entering {
			r.builder.WriteByte('\n')
			eqCount := int(n.Level-1) + r.h1Level
			for i := 0; i < eqCount; i++ {
				r.builder.WriteByte('=')
			}
			r.builder.WriteByte(' ')
		} else {
			r.builder.WriteString("\n")
		}

	// ---------------------------------------------------------
	// PARAGRAPH
	// ---------------------------------------------------------
	case *ast.Paragraph:
		r.builder.WriteString("\n")

	// ---------------------------------------------------------
	// BLOCKQUOTE
	// ---------------------------------------------------------
	case *ast.BlockQuote:
		if entering {
			r.builder.WriteString("#quote(block:true,\"")
		} else {
			r.builder.WriteString("\")\n\n")
		}

	// ---------------------------------------------------------
	// EM (italic) / STRONG (bold)
	// ---------------------------------------------------------
	case *ast.Emph:
		if entering {
			r.builder.WriteString("#emph[")
		} else {
			r.builder.WriteByte(']')
		}
	case *ast.Strong:
		if entering {
			r.builder.WriteString("#strong[")
		} else {
			r.builder.WriteByte(']')
		}

	// ---------------------------------------------------------
	// STRIKETHROUGH
	// ---------------------------------------------------------
	case *ast.Del:
		if entering {
			r.builder.WriteString("#strike[")
		} else {
			r.builder.WriteByte(']')
		}

	// ---------------------------------------------------------
	// CODE (INLINE)
	// ---------------------------------------------------------
	case *ast.Code:
		if entering {
			r.builder.WriteString(`#raw(block:false,"`)
			r.builder.WriteString(escapeString(string(n.Literal)))
			r.builder.WriteString(`")`)
		}

	// ---------------------------------------------------------
	// CODE BLOCK
	// ---------------------------------------------------------
	case *ast.CodeBlock:
		if entering {
			// raw-Typst 주석이 있었으면 typst 코드를 그대로 삽입
			if r.rawTypstNext {
				r.builder.WriteString(string(n.Literal))
				r.builder.WriteString("\n")
				r.rawTypstNext = false
			} else {
				r.builder.WriteString("#raw(block:true,")
				if len(n.Info) > 0 {
					tokens := strings.Fields(string(n.Info))
					langOnly := tokens[0]
					r.builder.WriteString(`lang:"`)
					r.builder.WriteString(escapeString(langOnly))
					r.builder.WriteString(`",`)
				}
				r.builder.WriteByte('"')
				r.builder.WriteString(escapeString(string(n.Literal)))
				r.builder.WriteString("\")\n")
			}
		}

	// ---------------------------------------------------------
	// 수평 구분선
	// ---------------------------------------------------------
	case *ast.HorizontalRule:
		if entering {
			r.builder.WriteString("#line(length:100%)\n")
		}

	// ---------------------------------------------------------
	// LIST / LIST ITEM
	// ---------------------------------------------------------
	case *ast.List:
		if entering {
			if (n.ListFlags & ast.ListTypeOrdered) != 0 {
				r.builder.WriteString("#enum(start:1,")
			} else {
				r.builder.WriteString("#list(")
			}
		} else {
			r.builder.WriteString(")\n")
		}
	case *ast.ListItem:
		if entering {
			r.builder.WriteByte('[')
		} else {
			r.builder.WriteString("],")
		}

	// ---------------------------------------------------------
	// TABLES
	// ---------------------------------------------------------
	case *ast.Table:
		if entering {
			// 테이블 메타정보가 있다면 사용, 없으면 기본값 사용
			var meta tableMeta
			if r.currentTableMeta != nil {
				meta = *r.currentTableMeta
			} else {
				meta = tableMeta{
					Caption:   "Default Table Caption",
					Placement: "none",
					Columns:   "(auto, auto, auto)",
					Align:     "(start, start, start)",
				}
			}

			r.builder.WriteString("#figure(\n")
			r.builder.WriteString(fmt.Sprintf("  caption: [%s],\n", meta.Caption))
			r.builder.WriteString(fmt.Sprintf("  placement: %s,\n", meta.Placement))
			r.builder.WriteString("  table(\n")
			r.builder.WriteString(fmt.Sprintf("    columns: %s,\n", meta.Columns))
			r.builder.WriteString(fmt.Sprintf("    align: %s,\n", meta.Align))
			r.builder.WriteString(`    inset: (x: 8pt, y: 4pt),
		stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
		fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
`)
		} else {
			r.builder.WriteString("  )\n")
			if r.currentTableMeta != nil && r.currentTableMeta.Label != "" {
				r.builder.WriteString(fmt.Sprintf(") <%s>\n", r.currentTableMeta.Label))
				r.currentTableMeta = nil
			} else {
				r.builder.WriteString(")\n")
			}
		}
	case *ast.TableHeader:
		if entering {
			r.builder.WriteString("table.header(")
		} else {
			r.builder.WriteString("),")
		}
	case *ast.TableCell:
		if entering {
			r.builder.WriteByte('[')
		} else {
			r.builder.WriteString("],")
		}

	// ---------------------------------------------------------
	// LINKS
	// ---------------------------------------------------------
	case *ast.Link:
		if entering {
			dest := string(n.Destination)
			r.builder.WriteString(`#link("`)
			r.builder.WriteString(escapeString(dest))
			r.builder.WriteString(`")[`)
		} else {
			r.builder.WriteByte(']')
		}

	// ---------------------------------------------------------
	// IMAGES ( #figure 로 변환 )
	// ---------------------------------------------------------
	case *ast.Image:
		if entering {
			// alt 텍스트 수집을 위해 임시 버퍼 사용
			oldBuilder := r.builder
			r.altTextBuffer = &strings.Builder{}
			r.builder = r.altTextBuffer
			for _, child := range n.Children {
				ast.Walk(child, &typVisitor{r})
			}
			r.builder = oldBuilder
			dest := string(n.Destination)
			altText := strings.TrimSpace(r.altTextBuffer.String())
			var label string
			// HTML 주석으로 전달된 image meta에서 label이 있으면 사용, 없으면 label 출력 안함.
			if r.currentImageMeta != nil && r.currentImageMeta.Label != "" {
				label = r.currentImageMeta.Label
				r.currentImageMeta = nil
			}
			r.builder.WriteString(fmt.Sprintf(
				"\n#figure(\n\tplacement: none,\n\timage(%q)",
				escapeString(dest),
			))
			// alt 텍스트가 있다면 캡션으로 사용
			if altText != "" {
				r.builder.WriteString(fmt.Sprintf(",\n\tcaption: [%s]", escapeString(altText)))
			}
			if label != "" {
				r.builder.WriteString(fmt.Sprintf("\n) <fig:%s>\n", label))
			} else {
				r.builder.WriteString("\n)\n")
			}
			return ast.SkipChildren
		}

	// ---------------------------------------------------------
	// HTML 블록, HTML 스팬 처리
	// ---------------------------------------------------------
	case *ast.HTMLSpan, *ast.HTMLBlock:
		if entering {
			htmlContent := ""
			switch x := n.(type) {
			case *ast.HTMLSpan:
				htmlContent = string(x.Literal)
			case *ast.HTMLBlock:
				htmlContent = string(x.Literal)
			}
			// <!--typst-begin-exclude-->인지 확인
			if isBeginExclude(htmlContent) {
				r.skipBlocks = true
				return ast.GoToNext
			}
			// <!--typst-end-exclude-->인지 확인
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
			// raw-typst 주석이 발견되면 다음 code block에 대해 typst 코드를 그대로 삽입하도록 플래그 설정
			if strings.Contains(htmlContent, "<!--raw-typst") {
				r.rawTypstNext = true
				return ast.GoToNext
			}
		}

	// ---------------------------------------------------------
	// TEXT
	// ---------------------------------------------------------
	case *ast.Text:
		if entering {
			content := string(n.Literal)
			r.builder.WriteString(content)
		}

	// ---------------------------------------------------------
	// MATH (인라인 / 블록)
	// ---------------------------------------------------------
	case *ast.Math:
		if entering {
			content := string(n.Literal)
			r.builder.WriteString(`$`)
			r.builder.WriteString(escapeString(content))
			r.builder.WriteString(`$`)
		}
	case *ast.MathBlock:
		if entering {
			content := string(n.Literal)
			r.builder.WriteString(`$$`)
			r.builder.WriteString(escapeString(content))
			r.builder.WriteString("$$\n\n")
		}

	// ---------------------------------------------------------
	// 줄바꿈 (Soft / Hard)
	// ---------------------------------------------------------
	case *ast.Softbreak:
		if entering {
			r.builder.WriteString(" ")
		}
	case *ast.Hardbreak:
		if entering {
			r.builder.WriteString("\\ ")
		}
	}

	return ast.GoToNext
}
// hasOption은 특정 옵션 비트가 켜져 있는지 확인하는 헬퍼 함수
func (r *typRenderer) hasOption(opt Options) bool {
	return (r.opts & opt) != 0
}
// -----------------------------------------------------------------------------
// 메인 함수
// -----------------------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "사용법: %s <input.md> [output.typ]\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]

	// 출력 파일 경로 결정
	// 1) 두 번째 인자가 있으면 사용
	// 2) 없으면 입력 파일 확장자를 .typ로 변경
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

	// 더미 옵션 2개 사용 (template 용도)
	opts := Options(OptionDummy1 | OptionDummy2)

	// h1Level=1로 설정하여 Markdown의 Heading 레벨에 1만큼 더함
	typstData, err := Render(mdData, opts, 1)
	if err != nil {
		panic(err)
	}

	// 결과 Typst 파일로 출력
	err = os.WriteFile(outputFile, []byte(typstData), 0644)
	if err != nil {
		panic(err)
	}

	// 변환 완료 메시지
	_, _ = io.WriteString(os.Stdout,
		fmt.Sprintf("성공적: %s -> %s\n", inputFile, outputFile))
}
