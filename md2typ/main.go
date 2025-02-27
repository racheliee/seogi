package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.md> [output.typ]\n", os.Args[0])
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

	// YAML 프론트매터 추출 및 Typst 헤더 생성
	meta, content, err := extractYAMLHeader(mdData)
	if err != nil {
		panic(err)
	}

	var header string
	if meta != nil {
		header, err = generateTypstHeader(meta)
		if err != nil {
			panic(err)
		}
	}

	// Markdown 본문을 Typst 코드로 변환
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

	_, _ = io.WriteString(os.Stdout, fmt.Sprintf("success: %s -> %s\n", inputFile, outputFile))
}
