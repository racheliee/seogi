# md2typ

- `md2typ` is a Golang program that converts Markdown files into Typst format.
- It uses the [gomarkdown/markdown](https://github.com/gomarkdown/markdown) package to parse Markdown text into an Abstract Syntax Tree (AST), and then traverses the AST to convert each Markdown element into Typst syntax.

---

## Key Features

- **Markdown → Typst Conversion**:  
  Converts various Markdown syntax elements (headings, paragraphs, blockquotes, emphasis, code, lists, tables, images, links, etc.) into Typst format.

- **Heading Conversion**:  
  Converts Markdown's `# Heading` syntax into Typst's heading format using equals signs (`=`) (e.g., `# Heading` → `= Heading`).

- **Paragraphs and Line Breaks**:  
  Each paragraph is separated by two empty lines in the Typst output.

- **Blockquotes**:  
  When the `OptionBlockquote` option is enabled, Markdown blockquotes are converted into Typst's `#quote[...]` syntax.

  > Later, the `#blockquote` template needs to be modified.

- **Text Formatting**:

  - _Italics_: `#emph[...]`
  - **Bold**: `#strong[...]`
  - ~~Strikethrough~~: `#strike[...]`

- **Code**:

  - Inline code: `#raw(block:false, "code")`
  - Code blocks: `#raw(block:true, lang:"language", "code")`

- **Horizontal Rule**:  
  Markdown's horizontal rule is converted into Typst's `#line(length:100%)`.

- **Lists**:  
  Ordered lists are converted into `#enum(start:1, ... )`, and unordered lists are converted into `#list(...)`. Each list item is wrapped in square brackets (`[ ]`).

- **Tables**:  
  Markdown tables are converted into Typst's `#figure(...)` syntax with `table(...)` inside. Additionally, HTML comments (`<!--typst-table ... -->`) can be used to set table metadata (caption, position, column structure, alignment, label) (default values exist).

- **Images**:  
  Markdown image syntax (`![alt](url)`) is converted into Typst's `#figure(...)` syntax. The filename is extracted from the image file path to automatically generate a label in the form of `fig:filename`, and the alt text is used as the caption.

- **Links**:  
  Markdown links are converted into `#link("URL")[text]` format.

- **Math Expressions**:  
  Inline math expressions are converted into `$...$`, and block math expressions are converted into `$$...$$`.

  > For block math expressions, due to auto-formatting issues in Markdown files, consider using `<!-raw-typst-->` comments or other solutions.

- **Raw Typst Code and Exclusion Blocks**:
  - Code within HTML comments like `<!--raw-typst ... -->` is extracted as raw Typst code.
  - Blocks between `<!--typst-begin-exclude-->` and `<!--typst-end-exclude-->` are excluded from conversion.

## How to Run

### Build

Run the following command in the project directory to build the `md2typ` executable:

```bash
go build -o md2typ .
```

### Conversion Execution

To convert a Markdown file into Typst format, run the following command:

```bash
./md2typ <input.md> [output.typ]
```

- `<input.md>`: Path to the Markdown file to be converted.
- `[output.typ]`: (Optional) Path to the output Typst file. If not specified, a file with the same name as the input file but with a `.typ` extension will be created.

Example:

```bash
./md2typ ./sample/convert-test.md
```

The above command converts `./sample/convert-test.md` and saves it as `convert-test.typ`.

---

## Options and Settings

Currently, conversion options are hardcoded and can be combined using bit flags as follows:

- `OptionBlockquote`: Enables blockquote conversion.
- `OptionRawTypst`: Enables raw Typst code processing.
- `OptionMath`: Enables math expression conversion.

These options are combined and used within the `main` function and can be adjusted as needed.

---

## TODO

- Modify to handle subpar image comments and their corresponding adjustments.
- Improve to allow setting template metadata via YAML headers.
- Consider allowing raw-typst comments to be inserted separately above code blocks.
- Consider applying different options based on the template during conversion.
