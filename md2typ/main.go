package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	
	// validate arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.md> [output.typ]\n", os.Args[0])
		os.Exit(1)
	}

	// set input and output file paths
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

	// read input markdown file
	mdData, err := os.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	// extract YAML header
	meta, content, err := extractYAMLHeader(mdData)
	if err != nil {
		panic(err)
	}

	// generate Typst metadata header from YAML
	var header string
	if meta != nil {
		header, err = generateTypstHeader(meta)
		if err != nil {
			panic(err)
		}
	}

	// render markdown content to Typst
	// TODO: option should be set according to the template
	opts := Options(OptionDummy1 | OptionDummy2)
	typstBody, err := Render(content, opts, 1)
	if err != nil {
		panic(err)
	}

	// write to output file
	finalOutput := header + "\n" + typstBody
	err = os.WriteFile(outputFile, []byte(finalOutput), 0644)
	if err != nil {
		panic(err)
	}

	// print success message
	_, _ = io.WriteString(os.Stdout, fmt.Sprintf("success: %s -> %s\n", inputFile, outputFile))
}
