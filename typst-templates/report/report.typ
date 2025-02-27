// report template based on charge ieee template

#let report(
  // The paper's title.
  title: [Paper Title],

  // An array of authors. For each author you can specify a name,
  // department, organization, location, and email. Everything but
  // but the name is optional.
  authors: (),


  // course name for header
  course: [course name],

  // The paper's abstract. Can be omitted if you don't have one.
  abstract: none,

  // A list of index terms to display after the abstract.
  index-terms: (),

  // The article's paper size. Also affects the margins.
  paper-size: "a4",

  // The result of a call to the `bibliography` function or `none`.
  bibliography: none,

  // How figures are referred to from within the text.
  figure-supplement: [Fig.],

  // The paper's content.
  body
) = {

  // if authors is empty, assign default author before using it
if authors == () {
  authors = (
    (name: "Author Name", 
    department: [Dept. of Computer Science and Engineering], 
    organization: [Sungkyunkwan University], 
    email: "example@gmail.com"),
  )
}

  
  // Set document metadata.
  set document(title: title, author: authors.map(author => author.name))

  // Set the body font. (korean font added)
  set text(font: ("Libertinus Serif","UnBatangOTF"), size: 10pt, spacing: .35em)

  // Enums numbering
  set enum(numbering: "1.a.i.")


  // Tables & figures
  show figure: set block(spacing: 15.5pt)
  show figure: set place(clearance: 15.5pt)
  show figure.where(kind: table): set figure.caption(position: top)
  show figure.where(kind: table): set text(size: 8pt )
  show figure.where(kind: table): set figure(numbering: "1")
  show figure.where(kind: image): set figure(supplement: figure-supplement, numbering: "1")
  show figure.caption: set text(size: 8pt)
  show figure.caption: set align(start)
  show figure.caption.where(kind: table): set align(center)
  
  
  // add custom annotation for table and figure
//   show ref: fig => {
//   if fig.element != none and fig.element.func() == figure {
//     let label = if fig.element.kind == table { "Table. " } else { "Fig. " }
//     let numbers = numbering(fig.element.numbering, ..fig.element.counter.at(fig.location()))
//     label + numbers  
//   } else {
//     it 
//   }
// }


  // Adapt supplement in caption independently from supplement used for references.
  show figure: fig => {
    let prefix = (
      if fig.kind == table [TABLE]
      else if fig.kind == image [Fig.]
      else [#fig.supplement]
    )
    let numbers = numbering(fig.numbering, ..fig.counter.at(fig.location()))
    show figure.caption: it => [#prefix~#numbers: #it.body]
    show figure.caption: set align(center)
    fig
  }

  // Code blocks with JetBrains Mono font and separated by lines
  show raw: set text(
    font: "JetBrains Mono",
    ligatures: false,
    size: 0.8em / 0.8,
    spacing: 100%,
  )
  
  show raw: it => {
    if it.block == true {
      v(1em)
      line(length: 100%, stroke: 0.3pt)
      v(1em)
      it
      v(1em)
      line(length: 100%, stroke: 0.3pt)
      v(1em)
    } else {
      it
    }
  }

  // quote style with italic font and indent and line infront 
 show quote.where(block: true): block.with(stroke: (left:1.5pt + gray, rest: none))
  

  // Configure the page and multi-column properties. (set to 1 in default)
  // toc looks ugly when set to 2
  set columns(gutter: 12pt)
  set page(
    columns: 1,
    paper: paper-size,

    header: context {
    if counter(page).get().first() > 1 [
      #set text(9pt)
      #course
      #h(1fr)
      #counter(page).display()
      #line(length: 100%, stroke: 0.5pt)
      #v(5pt)
    ]
  },
    
    // The margins depend on the paper size.
    margin: if paper-size == "a4" {
      (x: 75.5pt, top: 90.51pt, bottom: 89.51pt)
    } else {
      (
        x: (50pt / 216mm) * 100%,
        top: (85pt / 279mm) * 100%,
        bottom: (64pt / 279mm) * 100%,
      )
    }
  )
  
  // configure outline
  set outline(indent: auto)
  show outline.entry.where(level: 1): it => {
    v(12pt, weak: true)
    strong(it)
  }

  // Configure equation numbering and spacing.
  set math.equation(numbering: "(1)")
  show math.equation: set block(spacing: 0.65em)

  // Configure appearance of equation references
  show ref: it => {
    if it.element != none and it.element.func() == math.equation {
      // Override equation references.
      link(it.element.location(), numbering(
        it.element.numbering,
        ..counter(math.equation).at(it.element.location())
      ))
    } else {
      // Other references as usual.
      it
    }
  }

  // Configure lists.
  set enum(indent: 10pt, body-indent: 9pt)
  set list(indent: 10pt, body-indent: 9pt)

  // Configure headings.
  set heading(numbering: "1.1.1  ")
  show heading: it => {
    // Get all heading counter levels
    let levels = counter(heading).get()

    set text(10pt, weight: 400)
    if it.level == 1 {
      // First-level headings are left-aligned and bold.
      // We don't want to number the acknowledgment section.
      let is-ack = it.body in ([Acknowledgment], [Acknowledgement], [Acknowledgments], [Acknowledgements])
      set align(left)
      set text(if is-ack { 14pt } else { 14pt }, weight: "bold")
      show: block.with(above: 25pt, below: 13.75pt, sticky: true)
      if it.numbering != none and not is-ack {
        numbering("1.", ..levels)
        h(7pt, weak: true)
      }
      it.body
    } else if it.level == 2 {
      // Second-level headings are run-ins.
      set par(first-line-indent: 0pt)
      set text( weight: "bold", size: 11pt)
      show: block.with(spacing: 11pt, sticky: true)
      if it.numbering != none {
        numbering("1.1.", ..levels)
        h(7pt, weak: true)
      }
      it.body
    } else if it.level == 3 {
      // Third-level headings are run-ins.
      set par(first-line-indent: 0pt)
      set text(weight: "bold", size: 9pt)
      show: block.with(spacing: 11pt, sticky: true)
      if it.numbering != none {
        numbering("1.1.1.", ..levels)
        h(7pt, weak: true)
      }
      it.body
    }
  }

  // Style bibliography.
  show std.bibliography: set text(8pt)
  show std.bibliography: set block(spacing: 0.5em)
  set std.bibliography(title: text(10pt)[References], style: "ieee")

  // Display the paper's title and authors at the top of the page,
  place(
    top,
    float: true,
    scope: "parent",
    clearance: 30pt,
    {

      // gap for the first page
      v(20mm)

      // Display the title.
      v(3pt, weak: true)
      align(center, par(leading: 0.5em, text(size:18pt, weight: "bold", title)))
      v(13.35mm, weak: true)

      // Display the authors list.
      set par(leading: 0.6em)
      for i in range(calc.ceil(authors.len() / 3)) {
        let end = calc.min((i + 1) * 3, authors.len())
        let is-last = authors.len() == end
        let slice = authors.slice(i * 3, end)
        grid(
          columns: slice.len() * (1fr,),
          gutter: 12pt,
          ..slice.map(author => align(center, {
            if "student-no" in author [
              \ #text(size: 11pt, author.student-no + "  " + author.name)
            ] else [
              \ #text(size: 11pt, author.name)
            ]
            if "department" in author [
              \ #emph(author.department)
            ]
            if "organization" in author [
              \ #emph(author.organization)
            ]
            if "location" in author [
              \ #author.location
            ]
            if "email" in author {
              if type(author.email) == str [
                \ #link("mailto:" + author.email)
              ] else [
                \ #author.email
              ]
            }
          }))
        )

        if not is-last {
          v(16pt, weak: true)
        }
      }

      // Display todays date
      v(18mm, weak: true)
      let today = datetime.today()
      align(center, today.display("[month repr:long] [day], [year]"))
    }

  )

  // Configure paragraph properties.
  set par(spacing: 1.2em, justify: true, first-line-indent: 0em, leading: 0.45em)

  // Display abstract and index terms.
  if abstract != none [
    #set text(9pt, weight: 700, spacing: 150%)
    #h(1em) _Abstract_---#h(weak: true, 0pt)#abstract

    #if index-terms != () [
      #h(.3em)_Index Terms_---#h(weak: true, 0pt)#index-terms.join(", ")
    ]
    #v(2pt)
  ]

  // Display the paper's contents.
  set par(leading: 0.5em)
  body

  // Display bibliography.
  bibliography
}