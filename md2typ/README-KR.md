# md2typ

- `md2typ`는 Markdown 파일을 Typst 형식으로 변환하는 golang program
- [gomarkdown/markdown](https://github.com/gomarkdown/markdown) 패키지를 사용하여 Markdown 텍스트를 AST(Abstract Syntax Tree)로 파싱한 후, AST를 순회하면서 각 Markdown 요소를 Typst 문법으로 변환

---

## 주요 기능

- **Markdown → Typst 변환**:  
  Markdown의 다양한 구문(헤딩, 단락, 인용구, 강조, 코드, 리스트, 테이블, 이미지, 링크 등)을 Typst 형식으로 변환

- **헤딩 변환**:  
  Markdown의 `# Heading` 구문을 Typst의 등호(`=`)를 사용한 헤딩 형식으로 변환 (예: `# Heading` → `= Heading`)

- **인용구**:  
  Markdown의 인용구를 Typst의 `#quote(block: true, "...")` 구문으로 변환

- **텍스트 서식**:

  - _이탤릭체_: `#emph[...]`
  - **볼드체**: `#strong[...]`
  - ~~취소선~~: `#strike[...]`

- **코드**:

  - 인라인 코드: `#raw(block:false, "코드")`
  - 코드 블록: `#raw(block:true, lang:"언어", "코드")`

- **수평 구분선**:  
  Markdown의 수평 구분선은 Typst의 `#line(length:100%)`으로 변환됨

- **리스트**:  
  순서가 있는 리스트는 `#enum(start:1, ... )`, 순서 없는 리스트는 `#list(...)`로 변환되며, 각 리스트 항목은 대괄호(`[ ]`)로 감싸짐

- **테이블**:  
  Markdown 테이블은 Typst의 `#figure(...)` 내부의 `table(...)` 구문으로 변환됨
  추가적으로 HTML 주석(`<!--typst-table ... -->`)을 통해 테이블의 메타데이터(캡션, 위치, 열 구성, 정렬, 라벨)를 설정 (default 값 존재)

- **이미지**:  
  Markdown 이미지 문법(`![alt](url)`)은 Typst의 `#figure(...)` 구문으로 변환
  추가적으로 HTML 주석(`<!--typst-image ... -->`)을 통해 이미지의 메타데이터(라벨)를 설정 (default 없음)

- **링크**:  
  Markdown의 링크는 `#link("URL")[텍스트]` 형식으로 변환됨

- **수식 표현**:  
  인라인 수식은 `$...$`, 블록 수식은 `$$...$$` 형태로 변환됨

  > 블록 수식의 경우 md 파일의 auto formatting으로 인한 문제로 <!-raw-typst--> 주석을 이용

- **Raw Typst 코드 및 제외 블록**:
  - HTML 주석 `<!--raw-typst-->` 이후의 코드 블록은 Typst 코드로 그대로 추출
  - `<!--typst-begin-exclude-->`와 `<!--typst-end-exclude-->` 사이의 블록은 변환 대상에서 제외

## 실행 방법

### 빌드

프로젝트 디렉토리에서 다음 명령어를 실행하여 `md2typ` 실행 파일을 생성

```bash
go build -o md2typ .
```

### 변환 실행

Markdown 파일을 Typst 형식으로 변환하려면 아래와 같이 실행

```bash
./md2typ <input.md> [output.typ]
```

- `<input.md>`: 변환할 Markdown 파일의 경로
- `[output.typ]`: (선택 사항) 출력할 Typst 파일 경로. 지정하지 않을 경우, 입력 파일과 동일한 이름에 `.typ` 확장자가 붙은 파일이 생성

예시:

```bash
./md2typ ./sample/convert-test.md
```

위 명령어는 `./sample/convert-test.md` 파일을 변환하여 `convert-test.typ` 파일로 저장

---

## 옵션 및 설정

template에 따라 변환 로직을 다르게 하기 위해 두었으며, 현재는 dummy option

---

## TODO

- subpar image 주석과 대응하도록 수정 필요
- YAML 헤더를 통해 템플릿의 메타데이터 설정 가능하도록 개선
