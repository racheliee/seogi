#import "report.typ":*

#show: report.with(
  title: [Test Report Title for Typst],
  course: [Introduction to Test  (SWE3003)],
  authors: (
    (
      name: "20xxxxxx  Hyungjun Shon",
      department: [Dept. of Computer Science and Engineering],
      organization: [Sungkyunkwan University],
      email: "example@gmail.com"
    ),
    (
      name: "20xxxxxxxx  Rachel Park",
      department: [Dept. of Computer Science and Engineering],
      organization: [Sungkyunkwan University],
      email: "example@gmail.com"
    ),

  ),
  bibliography: bibliography("refs.bib"),
)

> asdfs

// table of contents
#v(6mm)
#outline()
#pagebreak()

= Introduction

#quote(
  block: true,
  lorem(50)
)

#quote(block:true,"
This is a single-line quote.
This is a second-line quote.
")



Scientific writing is a crucial part of the research process, allowing researchers to share their findings with the wider scientific community. However, the process of typesetting scientific documents can often be a frustrating and time-consuming affair, particularly when using outdated tools such as LaTeX. Despite being over 30 years old, it remains a popular choice for scientific writing due to its power and flexibility. However, it also comes with a steep learning curve, complex syntax, and long compile times, leading to frustration and despair for many researchers @netwok2020 @netwok2022.

== Paper overview
In this paper we introduce Typst, a new typesetting system designed to streamline the scientific writing process and provide researchers with a fast, efficient, and easy-to-use alternative to existing systems. Our goal is to shake up the status quo and offer researchers a better way to approach scientific writing.

// 코드
```typescript
export const NOTICE_TABLE_HEADERS: DataTableHeaderProps[] = [
  { label: "순번", widthPercentage: 7, sort: true, selector: "id" },
  { label: "제목", widthPercentage: 20, sort: true, selector: "title" },
  { label: "작성일", widthPercentage: 10, sort: true, selector: "createdAt" },
  { label: "관리", widthPercentage: 7, sort: false },
];

export const GALLERY_TABLE_HEADERS: DataTableHeaderProps[] = [
  { label: "순번", widthPercentage: 7, sort: true, selector: "id" },
  { label: "제목", widthPercentage: 20, sort: true, selector: "title" },
  { label: "게시년도", widthPercentage: 10, sort: true, selector: "year" },
  { label: "게시월", widthPercentage: 10, sort: true, selector: "month" },
  { label: "작성일", widthPercentage: 10, sort: true, selector: "createdAt" },
  { label: "관리", widthPercentage: 7, sort: false },
];
``` 

this is a test code for typescript

=== test level 3 heading
By leveraging advanced algorithms and a user-friendly interface, Typst offers several advantages over existing typesetting systems, including faster document creation, simplified syntax, and increased ease-of-use.

=== test level 3 heading
By leveraging advanced algorithms and a user-friendly interface, Typst offers several advantages over existing typesetting systems, including faster document creation, simplified syntax, and increased ease-of-use.

To demonstrate the potential of Typst, we conducted a series of experiments comparing it to other popular typesetting systems, including LaTeX. Our findings suggest that Typst offers several benefits for scientific writing, particularly for novice users who may struggle with the complexities of LaTeX. Additionally, we demonstrate that Typst offers advanced features for experienced users, allowing for greater customization and flexibility in document creation.

Overall, we believe that Typst represents a significant step forward in the field of scientific writing and typesetting, providing researchers with a valuable tool to streamline their workflow and focus on what really matters: their research. In the following sections, we will introduce Typst in more detail and provide evidence for its superiority over other typesetting systems in a variety of scenarios.

= Methods <sec:methods> // add label in section
#lorem(45)

// 수식
$ a + b = gamma $ <eq:gamma> // equation with label
$ sum_(i in NN) 1 + i $


#lorem(80)

// 사진
#figure(
  placement: none,
  image("../../md2typ/sample/cat.png"),
  caption: [A circle representing the Sun.]
) <fig:sun> // figure with label

In @fig:sun you can see a common representation of the Sun, which is a star that is located at the center of the solar system.


#lorem(120)


#figure(
  caption: [이것은 테이블 캡션 예시입니다],
  placement: none,
  table(
    columns: (6em, auto, auto),
    align: (center, center, right),
    inset: (x: 8pt, y: 4pt),
		stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
		fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
	table.header([번호],[이름],[설명],),[1],[사과],[빨간 과일],[2],[바나나],[노란 과일],[3],[키위],[초록 과일],  )
) <tab:mytable>


In @tab:mytable, you see the planets of the solar system and their average distance from the Sun.
The distances were calculated with @eq:gamma that we presented in @sec:methods.

#lorem(240)

