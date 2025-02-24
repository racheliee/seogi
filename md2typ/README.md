# md2typ

- `md2typ` is a Go program that converts Markdown files into Typst format.
- It utilizes the [gomarkdown/markdown](https://github.com/gomarkdown/markdown) package to parse Markdown text into an Abstract Syntax Tree (AST) and then traverses the AST to convert each Markdown element into Typst syntax.

---

## Key Features

- **Markdown → Typst Conversion**:  
  Supports conversion of various Markdown elements (headings, paragraphs, blockquotes, emphasis, code, lists, tables, images, links, etc.) into Typst format.

- **Heading Conversion**:  
  Markdown headings (e.g., `# Heading`) are converted into Typst's heading format using equals signs (`=`).  
  Example: `# Heading` → `= Heading`

- **Blockquotes**:  
  Markdown blockquotes are converted into Typst’s `#quote(block: true, "...")` syntax.

- **Text Formatting**:

  - _Italic_: `#emph[...]`
  - **Bold**: `#strong[...]`
  - ~~Strikethrough~~: `#strike[...]`

- **Code**:

  - Inline code: `#raw(block:false, "code")`
  - Code blocks: `#raw(block:true, lang:"language", "code")`

- **Horizontal Rules**:  
  Markdown horizontal rules are converted to Typst’s `#line(length:100%)`.

- **Lists**:

  - Ordered lists are converted to `#enum(start:1, ... )`
  - Unordered lists are converted to `#list(...)`  
    Each list item is enclosed in square brackets (`[ ]`).

- **Tables**:

  - Markdown tables are converted into Typst’s `#figure(...)` containing a `table(...)` block.
  - HTML comments (`<!--typst-table ... -->`) are used to define metadata such as captions, positioning, column configuration, alignment, and labels (with default values provided).

- **Images**:

  - Markdown image syntax (`![alt](url)`) is converted into Typst’s `#figure(...)` syntax.
  - HTML comments (`<!--typst-image ... -->`) are used for metadata configuration (such as labels, with no default values).

- **Links**:  
  Markdown links are converted to Typst’s `#link("URL")[text]` format.

- **Mathematical Expressions**:

  - Inline equations are converted as `$...$`.
  - Block equations are converted as `$$...$$`.

  > Block equations require the `<!--raw-typst-->` comment to prevent auto-formatting issues in Markdown.

- **Raw Typst Code & Exclusion Blocks**:
  - Code blocks following the HTML comment `<!--raw-typst-->` are extracted as raw Typst code.
  - Blocks enclosed between `<!--typst-begin-exclude-->` and `<!--typst-end-exclude-->` are excluded from conversion.

---

## Usage

### Build

Run the following command in the project directory to generate the `md2typ` executable:

```bash
go build -o md2typ .
```

### Convert Markdown to Typst

To convert a Markdown file into Typst format, run:

```bash
./md2typ <input.md> [output.typ]
```

- `<input.md>`: Path to the Markdown file to be converted.
- `[output.typ]` (optional): Path to the output Typst file. If not specified, a file with the same name as the input file but with the `.typ` extension will be created.

Example:

```bash
./md2typ ./sample/convert-test.md
```

This command converts `./sample/convert-test.md` into `convert-test.typ`.

---

## Options & Settings

A template-based approach allows for different conversion logic. Currently, it is a placeholder option.

---

## TODO

- Modify the image metadata parsing to align with Typst comments.
- Improve metadata handling by allowing YAML headers for template configuration.
