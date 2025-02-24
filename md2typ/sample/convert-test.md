# 테스트 문서

이 문서는 여러 Markdown 기능을 시험하기 위한 예시입니다.

## 인라인 수식

수식 예시: $y = m x + b$

## 블록 수식

아래는 블록 수식 예시입니다: $y = m x + b$

## 목록

- 첫 번째 아이템
- 두 번째 아이템
- 세 번째 아이템

1. 순서가 있는 목록
2. 두 번째 아이템
3. 세 번째 아이템

## 블록 인용문

> 이것은 한 줄짜리 인용문입니다.
>
> 이것은 두 번째 줄짜리 인용문입니다.

## 코드 블록

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

## 이미지

여기에는 고양이 이미지를 삽입해봅시다.
![고양이](images/cat.png)

## 링크

여기는 [구글](https://www.google.com)로 가는 링크입니다.

## 테이블

아래는 간단한 테이블 예시입니다:

<!--typst-table
caption: "이것은 테이블 캡션 예시입니다"
placement: none
columns: (6em, auto, auto)
align: (center, center, right)
label: "tab:mytable"
-->

| 번호 | 이름   | 설명      |
| ---- | ------ | --------- |
| 1    | 사과   | 빨간 과일 |
| 2    | 바나나 | 노란 과일 |
| 3    | 키위   | 초록 과일 |

## Raw Typst 태그

<!--raw-typst
#box[이 문장은 Typst 문법으로 직접 작성되었습니다!]
-->

## 일부 구간 제외(Exclusion)

이 구간은 변환될 것입니다.

<!--typst-begin-exclude-->

이 구간은 `<!--typst-begin-exclude-->`부터
`<!--typst-end-exclude-->` 사이이므로
Typst 변환 시 출력되지 않습니다.

<!--typst-end-exclude-->

또한 이 구간 뒤쪽은 변환됩니다.

위 예시 문서를 이용해 변환 프로그램을 실행하면, 다음 기능들을 한꺼번에 확인할 수 있습니다:

1. **Heading 레벨 변환**
2. **인라인/블록 수식 처리**: 수식이 Typst 형식으로 변환되는지 확인
3. **목록(순서, 비순서) 변환**
4. **블록 인용문 처리** (옵션에 따라 Typst의 `#blockquote`로 변환 여부 확인)
5. **코드 블록**
6. **이미지 삽입** (alt 텍스트가 캡션으로 잘 들어가는지 확인)
7. **링크 변환**
8. **테이블 변환** (셀 정렬이 제대로 작동되는지 확인)
9. **Raw Typst 태그** (HTML 주석 안의 Typst 코드를 그대로 삽입하는 기능)
