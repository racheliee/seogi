package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// Typst 변환 시 사용할 옵션 정의
// iota를 사용해 각 상수를 2의 제곱 형태(비트 플래그)로 만듦
const (
	OptionBlockquote = 1 << iota // 1
	OptionRawTypst               // 2
	OptionMath                   // 4
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



// printAST는 Markdown 파싱 후 생성되는 AST(Abstract Syntax Tree) 구조를 콘솔에 출력하는 디버깅용 함수
func printAST(node ast.Node, depth int) {
	ast.Walk(node, &astVisitor{depth: depth})
}

// astVisitor는 AST를 순회하며 노드 정보를 콘솔에 출력하기 위한 구조체
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

// getFigureLabel은 이미지 파일 경로에서 파일명을 추출해 "fig:파일명" 형태의 레이블을 생성
func getFigureLabel(imagePath string) string {
	base := filepath.Base(imagePath)                         // 전체 경로에서 파일명만 추출
	label := strings.TrimSuffix(base, filepath.Ext(base))    // 확장자 제거
	return "fig:" + label
}

// typRenderer는 Markdown AST를 Typst 포맷으로 변환하기 위한 렌더러 구조체
type typRenderer struct {
	builder        *strings.Builder // 변환 결과를 담을 버퍼
	opts           Options          // 활성화된 옵션들
	h1Level        int              // Heading 레벨 보정값
	skipBlocks     bool             // 특정 구간(<!--typst-begin-exclude--> ~ <!--typst-end-exclude-->) 스킵 여부
	altTextBuffer  *strings.Builder // 이미지 대체 텍스트를 임시로 모으는 버퍼
	currentTableMeta *tableMeta     // 현재 테이블의 meta 정보
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
// opts를 통해 옵션을 설정하며, h1Level로 Heading 레벨의 시작값을 조정할 수 있음
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
	// <!--typst-begin-exclude-->가 등장하여 skipBlocks가 true가 된 상태에서는
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
			// 문단이 끝날 때는 빈 줄 두 개 삽입
			r.builder.WriteString("\n\n")
		}

	// ---------------------------------------------------------
	// BLOCKQUOTE
	// ---------------------------------------------------------
	case *ast.BlockQuote:
		// OptionBlockquote가 설정된 경우에만 Typst의 blockquote 구문을 사용
		// TODO: Typst 템플릿 파일에 blockquote 관련 추가 필요 (임시로 quote로 대체)
		if !r.hasOption(OptionBlockquote) {
			return ast.GoToNext
		}
		if entering {
			r.builder.WriteString("#quote[")
		} else {
			r.builder.WriteString("]\n\n")
		}

	// ---------------------------------------------------------
	// EM(italic) / STRONG(bold)
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
			// Typst에서는 #raw(block:true, lang:"언어", "코드") 형태를 사용
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
				// 기본값
				meta = tableMeta{
					Caption:   "Default Table Caption",
					Placement: "none",
					Columns:   "(auto, auto, auto)",
					Align:     "(start, start, start)",
				}
			}

			// #figure(...) 열기
			r.builder.WriteString("#figure(\n")

			// caption
			r.builder.WriteString(fmt.Sprintf("  caption: [%s],\n", meta.Caption))

			// placement
			r.builder.WriteString(fmt.Sprintf("  placement: %s,\n", meta.Placement))

			// table(
			r.builder.WriteString("  table(\n")

			// columns
			r.builder.WriteString(fmt.Sprintf("    columns: %s,\n", meta.Columns))

			// align
			r.builder.WriteString(fmt.Sprintf("    align: %s,\n", meta.Align))

			// report.template의 기본 스타일
			r.builder.WriteString(`    inset: (x: 8pt, y: 4pt),
		stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
		fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
	`)

			// 테이블 내용은 table.header[...] 혹은 바로 [row1],[row2],... 형태로 추가
			// 우선은 "table.header(" 구문만 여기서 찍고, 나머지는 기존 case *ast.TableHeader:에서 작성
		} else {
			// table(...) 닫고
			r.builder.WriteString("  )\n")

			// #figure(...) 닫고
			if r.currentTableMeta.Label != "" {
				r.builder.WriteString(fmt.Sprintf(") <%s>\n", r.currentTableMeta.Label))
				// 사용 후 초기화
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
    case *ast.TableBody, *ast.TableFooter:
        // You could similarly distinguish table.body(...) or table.footer(...) if desired.
        // For simplicity, we do not wrap them differently here, but you could do so:
        //   if entering { r.builder.WriteString("table.body(") } else { r.builder.WriteString("),") }
        // Or similarly for footer.
    case *ast.TableRow:
        // No explicit delimiter. The Rust code lumps them in the final result.
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
			// 먼저 alt 텍스트를 수집하기 위해 builder를 임시로 바꿈
			oldBuilder := r.builder
			r.altTextBuffer = &strings.Builder{}
			r.builder = r.altTextBuffer

			// 이미지 노드의 자식(주로 Text 노드) 순회
			for _, child := range n.Children {
				ast.Walk(child, &typVisitor{r})
			}

			// alt 텍스트 수집 후 원래 builder 복원
			r.builder = oldBuilder
			dest := string(n.Destination)
			altText := strings.TrimSpace(r.altTextBuffer.String())
			label := getFigureLabel(dest)

			// #figure( image("..."), caption: [...], ... ) 형식으로 작성
			r.builder.WriteString(fmt.Sprintf(
				"\n#figure(\n\tplacement: none,\n\timage(%q)",
				escapeString(dest),
			))

			// alt 텍스트가 있다면 캡션으로 사용
			if altText != "" {
				r.builder.WriteString(fmt.Sprintf(",\n\tcaption: [%s]", escapeString(altText)))
			}

			// 라벨(fig:파일명)
			r.builder.WriteString(fmt.Sprintf("\n) <%s>\n", label))

			// 이미지 노드 내부 텍스트(자식 노드)는 이미 altText로만 사용했으므로, 추가 탐색 불필요
			return ast.SkipChildren
		}

	// ---------------------------------------------------------
	// HTML 블록, HTML 스팬
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

			// OptionRawTypst가 꺼져 있으면 raw typst 처리 무시
			if !r.hasOption(OptionRawTypst) {
				return ast.GoToNext
			}

			if metaRaw, ok := isTableMetaComment(n); ok {
				m := parseTableMeta(metaRaw)
				r.currentTableMeta = &m
				// typst-table 코멘트 외 다른 처리(예: raw-typst 등)는 필요한 경우 추가
				return ast.GoToNext
			}

			// <!--raw-typst가 포함된 HTML이면 Typst 코드만 추출
			if strings.Contains(htmlContent, "<!--raw-typst") {
				extracted := extractRawTypst(htmlContent)
				r.builder.WriteString(extracted)
			}
		}

	// ---------------------------------------------------------
	// TEXT
	// ---------------------------------------------------------
	case *ast.Text:
		if entering {
			content := string(n.Literal)

			// 문서 내에 "![...](...)" 형태의 이미지를 텍스트로 포함하고 있다면 교체
			if strings.Contains(content, "![") && strings.Contains(content, "](") {
				imageRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
				converted := imageRe.ReplaceAllStringFunc(content, func(m string) string {
					matches := imageRe.FindStringSubmatch(m)
					if len(matches) == 3 {
						alt := matches[1]
						url := matches[2]
						return `#image("` + escapeString(url) + `", alt:"` + escapeString(alt) + `")`
					}
					return m
				})
				r.builder.WriteString(converted)
				return ast.GoToNext
			}

			// 일반 텍스트는 그대로 추가
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
	// 줄바꿈(Soft / Hard)
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

// // ---------------------------------------------------------
// // 테이블 정렬 관련 함수
// // ---------------------------------------------------------

// // gatherTableAlignments는 Table 노드의 Header 행에서 셀 정렬 정보를 추출해 반환
// func gatherTableAlignments(t *ast.Table) []string {
// 	var aligns []string

// 	for _, child := range t.GetChildren() {
// 		if header, ok := child.(*ast.TableHeader); ok {
// 			// 일반적으로 헤더에는 하나의 행(Row)만 존재한다고 가정
// 			if rowNode := header.GetChildren(); len(rowNode) > 0 {
// 				if row, ok := rowNode[0].(*ast.TableRow); ok {
// 					for _, cellNode := range row.GetChildren() {
// 						if cell, ok := cellNode.(*ast.TableCell); ok {
// 							aligns = append(aligns, cellAlignString(cell.Align))
// 						}
// 					}
// 				}
// 			}
// 			break
// 		}
// 	}
// 	return aligns
// }

// // cellAlignString은 ast.TableCell의 Align 값을 Typst에서 사용하는 정렬 문자열로 변환
// func cellAlignString(a ast.CellAlignFlags) string {
// 	switch a {
// 	case ast.TableAlignmentLeft:
// 		return "left"
// 	case ast.TableAlignmentCenter:
// 		return "center"
// 	case ast.TableAlignmentRight:
// 		return "right"
// 	default:
// 		return "left"
// 	}
// }

// ---------------------------------------------------------
// Exclusion Marker(일부 블록 제외) 관련 함수
// ---------------------------------------------------------

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

// ---------------------------------------------------------
// raw Typst 추출 함수
// ---------------------------------------------------------

// extractRawTypst는 "<!--raw-typst" 이후의 문자열을 간단히 추출
func extractRawTypst(s string) string {
	idx := strings.Index(s, "<!--raw-typst")
	if idx == -1 {
		return ""
	}
	rest := s[idx+len("<!--raw-typst"):]
	end := strings.Index(rest, "-->")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// ---------------------------------------------------------
// Stirng Escape 함수
// ---------------------------------------------------------

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

// ---------------------------------------------------------
// 메인 함수
// ---------------------------------------------------------

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

	// 옵션은 필요에 따라 조합해 사용 가능
	// 여기서는 blockquote, raw typst, math 옵션을 켬
	opts := Options(OptionBlockquote | OptionRawTypst | OptionMath)

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
