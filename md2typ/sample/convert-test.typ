
= Test Document
This document is an example for testing various Markdown features.


== Inline Math
Example equation: $y = m x + b$


== Block Math
due to auto formatting of markdown files,

use raw-typst comments

$ sum_(k=0)^n k
    &= 1 + ... + n \
    &= (n(n+1)) / 2 $


== Lists
#list([First item],
[Second item],
[Third item],
)
#enum(start:1,[Ordered list item],
[Second item],
[Third item],
)

== Block Quotes

#quote(block:true,"This is a single-line quote.
This is a second-line quote.")

== Code Blocks
#raw(block:true,lang:"go","package main

import \"fmt\"

func main() {
    fmt.Println(\"Hello, world!\")
}
")

== Images
Let's insert a cat image here.


#figure(
	placement: none,
	image("./cat.png"),
	caption: [Cat]
) <fig:test-label>



== Links
Here is a link to #link("https://www.google.com")[Google].


== Table
Below is a simple table example:

#figure(
  caption: [This is an example of a table caption],
  placement: none,
  table(
    columns: (6em, auto, auto),
    align: (center, center, right),
    inset: (x: 8pt, y: 4pt),
			stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
			fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
	table.header([No.],[Name],[Description],),[1],[Apple],[Red fruit],[2],[Banana],[Yellow fruit],[3],[Kiwi],[Green fruit],  )
) <tab:mytable>

== Raw Typst Tags
#box[This sentence is written directly in Typst syntax!]


== Exclusion of Certain Sections
This section will be converted.

Additionally, the content after this exclusion will be converted.

By running the conversion program with this example document, you can verify the following features:

#enum(start:1,[#strong[Heading level conversion]],
[#strong[Inline/block math processing]: Check if equations are properly converted to Typst format.],
[#strong[List conversion (ordered and unordered)]],
[#strong[Block quote handling] (Optional: Verify if it converts to Typst's #raw(block:false,"#blockquote"))],
[#strong[Code blocks]],
[#strong[Image insertion] (Check if alt text correctly appears as captions)],
[#strong[Link conversion]],
[#strong[Table conversion] (Ensure correct alignment of cells)],
[#strong[Raw Typst tags] (Preserve Typst syntax inside HTML comments)],
)
