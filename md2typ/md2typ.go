// go build -o md2typ .	: 실행 파일 생성
// ./md2typ ./sample/convert-test.md : 테스트 실행

// NOTE: trailing newline char는 tempbuiler를 이용하여 렌더링 후 제거하는 야매 방식으로 처리 -> 추후 수정 필요
// NOTE: 동적 table column 계산을 위한 방법도 visitor의 다수 선언을 이용한 야매 방식으로 처리 -> 추후 수정 필요

// TODO: subpar image 주석과 대응하게 사용할 수 있도록 수정

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
	"gopkg.in/yaml.v2"
)

// -----------------------------------------------------------------------------
//  Util Types & Constants
// -----------------------------------------------------------------------------

// 템플릿용 더미 옵션 2개
const (
	OptionDummy1 = 1 << iota // 1
	OptionDummy2             // 2
)

// 여러 옵션을 동시에 담기 위한 비트 플래그 타입
type Options uint8

// report yaml header 구조체
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

// AST를 순회하며 노드 정보를 콘솔에 출력하는 디버깅용 구조체
type astVisitor struct {
	depth int
}

// TableHeader를 찾기 위한 visitor 구조체
type headerFinderVisitor struct {
	header ast.Node
}

// TableCell의 개수를 세기 위한  visitor 구조체
type cellCounterVisitor struct {
	count int
}

// node의 자식 노드를 가져오기 위한 인터페이스
type childNodes interface {
	GetChildren() []ast.Node
}

// -----------------------------------------------------------------------------
// Debugging & AST Traversal Functions
// -----------------------------------------------------------------------------

// Markdown 파싱 후 생성되는 AST(Abstract Syntax Tree) 구조를 콘솔에 출력하는 디버깅용 함수
func printAST(node ast.Node, depth int) {
	ast.Walk(node, &astVisitor{depth: depth})
}

// ast.Walk에 의해 호출되어, AST의 각 노드를 방문할 때 실행됨
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

// node를 순회하며 첫 번째 TableHeader를 찾아 저장
func (v *headerFinderVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableHeader); ok {
			v.header = n
			return ast.Terminate // 첫 번째 TableHeader를 찾았으므로 중단
		}
	}
	return ast.GoToNext
}

// node를 순회하며 TableCell을 찾아 개수를 세고 cellCounterVisitor.count에 저장
func (v *cellCounterVisitor) Visit(n ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if _, ok := n.(*ast.TableCell); ok {
			v.count++
			return ast.SkipChildren // TableCell 내부는 더 이상 탐색할 필요 없음
		}
	}
	return ast.GoToNext
}

// -----------------------------------------------------------------------------
// Utility Functions (Table Meta, Image Meta, Exclusion, Raw Typst, String Escape)
// -----------------------------------------------------------------------------

// HTML 블록 혹은 스팬 내에 `<!--typst-table` 라인이 있으면 해당 문자열을 반환
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

// 주석으로부터 table meta 정보를 파싱하여 tableMeta 구조체로 반환
func parseTableMeta(metaRaw string) tableMeta {
	// 기본값 설정
	tm := tableMeta{
		Caption:   "",
		Placement: "none",
		Columns:   "", // 빈 문자열이면 이후 동적 계산 진행
		Align:     "", // 빈 문자열이면 align 옵션은 출력하지 않음
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

// 주어진 노드 내부의 TableCell 개수를 반환
func headerCellsCount(node ast.Node) int {
	visitor := &cellCounterVisitor{}
	ast.Walk(node, visitor)
	return visitor.count
}

// HTML 내에 `<!--typst-image` 라인이 있으면 해당 문자열 반환
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

// 주석으로부터 image meta 정보를 파싱 후 imageMeta 구조체로 반환
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

// 주어진 노드의 자식들을 임시 버퍼에 렌더링한 결과를 반환
func (r *typRenderer) renderChildNodes(n childNodes) string {
	var tempBuilder strings.Builder
	// r의 얕은 복사본을 생성하고 builder를 임시 버퍼로 교체
	tempRenderer := *r
	tempRenderer.builder = &tempBuilder

	// n의 자식 노드를 순회하며 렌더링
	for _, child := range n.GetChildren() {
		ast.Walk(child, &typVisitor{r: &tempRenderer})
	}
	// 마지막 2개의 newline을 제거한 후 반환 (paragraph의 경우 두 번의 newline이 추가되므로)
	return strings.TrimSuffix(tempBuilder.String(), "\n\n")
}

// HTML에 <!--typst-begin-exclude가 포함되었는지 확인
func isBeginExclude(html string) bool {
	return strings.Contains(html, "<!--typst-begin-exclude")
}

// 현재 노드가 <!--typst-end-exclude-->인지 확인
func isEndExclude(node ast.Node) bool {
	if htmlSpan, ok := node.(*ast.HTMLSpan); ok {
		return strings.Contains(string(htmlSpan.Literal), "<!--typst-end-exclude")
	}
	if htmlBlock, ok := node.(*ast.HTMLBlock); ok {
		return strings.Contains(string(htmlBlock.Literal), "<!--typst-end-exclude")
	}
	return false
}

// Typst 코드 내에서 필요한 백슬래시, 따옴표 등을 이스케이프 처리
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

// Markdown AST를 Typst 포맷으로 변환하기 위한 렌더러 구조체
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

// ast.Walk 시 실제 노드 방문 로직을 typRenderer에 위임하기 위한 구조체
type typVisitor struct {
	r *typRenderer
}

// Markdown AST를 순회하면서 각 노드를 typRenderer.walker에 전달
func (v *typVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	return v.r.walker(node, entering)
}

// typRenderer 구조체 인스턴스를 생성해 반환
func newTypRenderer(opts Options, h1Level int) *typRenderer {
	return &typRenderer{
		builder: &strings.Builder{},
		opts:    opts,
		h1Level: h1Level,
	}
}

// Markdown 텍스트를 AST로 파싱한 뒤, Typst 문자열로 변환해 반환
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
	// fmt.Println("==== AST 구조 출력 ====")
	// printAST(doc, 0)
	// fmt.Println("========================")

	// 생성한 AST를 기반으로 Typst 변환기를 생성 후 순회하며 변환 수행
	r := newTypRenderer(opts, h1Level)
	ast.Walk(doc, &typVisitor{r})
	return r.builder.String(), nil
}

// AST를 순회하며, 노드 종류와 상태에 따라 Typst 코드를 생성
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
		if !entering {
			r.builder.WriteString("\n\n")
		}

	// ---------------------------------------------------------
	// BLOCKQUOTE
	// ---------------------------------------------------------
	case *ast.BlockQuote:
		if entering {
			// 자식 노드를 별도의 임시 버퍼에 렌더링 후, 결과를 받아옴
			content := r.renderChildNodes(n)
			// blockquote 시작 마커와 함께 출력
			r.builder.WriteString("\n#quote(block:true,\"")
			r.builder.WriteString(content)
			r.builder.WriteString("\")\n")
			// 자식 노드 처리를 이미 했으므로 더 이상 순회하지 않음
			return ast.SkipChildren
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
			// 리스트 아이템 시작 시, 헬퍼 함수를 호출해 자식 내용을 캡처
			content := r.renderChildNodes(n)
			r.builder.WriteString("[")
			r.builder.WriteString(content)
			r.builder.WriteString("],\n")
			// 자식 노드를 이미 처리했으므로 더 이상 순회하지 않도록 SkipChildren 반환
			return ast.SkipChildren
		}
	
	

	// ---------------------------------------------------------
	// TABLES
	// ---------------------------------------------------------
	case *ast.Table:
		if entering {
			// meta 주석이 있으면 사용, 없으면 기본 meta 생성
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

			// 테이블 내에서 첫 번째 TableHeader를 찾음
			hfv := &headerFinderVisitor{}
			ast.Walk(n, hfv)
			headerCells := 0
			if hfv.header != nil {
				headerCells = headerCellsCount(hfv.header)
			}

			// columns 값이 빈 경우, 헤더 셀 수에 따라 자동 생성
			if meta.Columns == "" {
				cols := make([]string, headerCells)
				for i := 0; i < headerCells; i++ {
					cols[i] = "auto"
				}
				meta.Columns = "(" + strings.Join(cols, ", ") + ")"
			}

			r.builder.WriteString("#figure(\n")
			if meta.Caption != "" {
				r.builder.WriteString(fmt.Sprintf("  caption: [%s],\n", meta.Caption))
			}
			r.builder.WriteString(fmt.Sprintf("  placement: %s,\n", meta.Placement))
			r.builder.WriteString("  table(\n")
			r.builder.WriteString(fmt.Sprintf("    columns: %s,\n", meta.Columns))
			// align 값이 비어있으면 해당 옵션은 출력하지 않음
			if meta.Align != "" {
				r.builder.WriteString(fmt.Sprintf("    align: %s,\n", meta.Align))
			}
			r.builder.WriteString(`    inset: (x: 8pt, y: 4pt),
			stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
			fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
	`)
		} else {
			r.builder.WriteString("  )\n")
			if r.currentTableMeta != nil && r.currentTableMeta.Label != "" {
				r.builder.WriteString(fmt.Sprintf(") <tab:%s>\n", r.currentTableMeta.Label))
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
	// case *ast.MathBlock:
	// 	if entering {
	// 		content := string(n.Literal)
	// 		r.builder.WriteString(`$$`)
	// 		r.builder.WriteString(escapeString(content))
	// 		r.builder.WriteString("$$\n\n")
	// 	}

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
// YAML 프론트매터 추출 및 Typst 헤더 생성 함수
// -----------------------------------------------------------------------------

// extractYAMLHeader는 Markdown 파일의 시작부분 YAML 헤더를 추출하고 파싱한 후,
// 파싱된 메타데이터와 YAML 헤더를 제거한 Markdown 본문을 반환합니다.
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

// generateTypstHeader는 파싱된 Metadata를 기반으로 Typst 문법의 헤더 문자열을 생성합니다.
// 빈 값인 필드는 아예 출력하지 않습니다.
func generateTypstHeader(meta *Metadata) string {
	var b strings.Builder
	b.WriteString(`#import "../../typst-templates/report/report.typ":*

#show: report.with(
`)
	// title 필드는 항상 포함 (비어있으면 default 처리 가능)
	if meta.Title != "" {
		b.WriteString("  title: [")
		b.WriteString(meta.Title)
		b.WriteString("],\n")
	}
	if meta.Course != "" {
		b.WriteString("  course: [")
		b.WriteString(meta.Course)
		b.WriteString("],\n")
	}
	if meta.Date != "" {
		b.WriteString("  date: [")
		b.WriteString(meta.Date)
		b.WriteString("],\n")
	}
	if len(meta.Authors) > 0 {
		b.WriteString("  authors: (")
		for _, author := range meta.Authors {
			b.WriteString(`
    (
      name: "`)
			b.WriteString(author.Name)
			b.WriteString(`",
      department: [`)
			b.WriteString(author.Department)
			b.WriteString(`],
      organization: [`)
			b.WriteString(author.Organization)
			b.WriteString(`],
	  	email: "`)
			b.WriteString(author.Email)
			b.WriteString(`"),`)
		}
		b.WriteString("),\n")
	}
	if meta.Bibliography != "" {
		b.WriteString("  bibliography: bibliography(\"")
		b.WriteString(meta.Bibliography)
		b.WriteString("\"),\n")
	}
	b.WriteString(")\n")
	if meta.Toc {
		b.WriteString(`
// table of contents
#v(6mm)
#outline()
#pagebreak()
`)
	}
	return b.String()
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

	// YAML 프론트매터 추출: 메타데이터와 Markdown 본문 분리
	meta, content, err := extractYAMLHeader(mdData)
	if err != nil {
		panic(err)
	}

	var header string
	if meta != nil {
		header = generateTypstHeader(meta)
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

	_, _ = io.WriteString(os.Stdout,
		fmt.Sprintf("success: %s -> %s\n", inputFile, outputFile))
}