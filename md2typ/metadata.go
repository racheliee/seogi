package main

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

// Metadata는 YAML 프론트매터의 내용을 담습니다.
type Metadata struct {
	Title        string `yaml:"title"`
	Course       string `yaml:"course"`
	Date         string `yaml:"date"`
	Authors      []struct {
		Name         string `yaml:"name"`
		Department   string `yaml:"department"`
		Organization string `yaml:"organization"`
		Email        string `yaml:"email"`
	} `yaml:"authors"`
	Bibliography string `yaml:"bibliography"`
	Toc          bool   `yaml:"toc"`
}

// extractYAMLHeader는 Markdown 파일의 시작부분 YAML 헤더를 추출한 후,
// 메타데이터와 YAML 헤더를 제거한 Markdown 본문을 반환합니다.
func extractYAMLHeader(mdData []byte) (*Metadata, []byte, error) {
	mdStr := string(mdData)
	lines := strings.Split(mdStr, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		var yamlLines []string
		var i int
		for i = 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				break
			}
			yamlLines = append(yamlLines, lines[i])
		}
		yamlContent := strings.Join(yamlLines, "\n")
		var meta Metadata
		if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
			return nil, mdData, err
		}
		content := strings.Join(lines[i+1:], "\n")
		return &meta, []byte(content), nil
	}
	return nil, mdData, nil
}

// generateTypstHeader는 templates/typst_header.tpl 파일을 이용해 Typst 헤더를 생성합니다.
func generateTypstHeader(meta *Metadata) (string, error) {
	tpl, err := template.ParseFiles("templates/typst_header.tpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, meta); err != nil {
		return "", err
	}
	return buf.String(), nil
}
