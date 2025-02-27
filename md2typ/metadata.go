package main

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

// metadata structure for YAML header in Markdown files (for report template)
type Metadata struct {
	DocumentType string `yaml:"type"`
	TemplatePath string
	Title        string `yaml:"title"`
	Course       string `yaml:"course"`
	Date         string `yaml:"date"`
	Authors      []struct {
		Name         string `yaml:"name"`
		StudentNo    string `yaml:"student-no"`
		Department   string `yaml:"department"`
		Organization string `yaml:"organization"`
		Email        string `yaml:"email"`
	} `yaml:"authors"`
	Bibliography string `yaml:"bibliography"`
	Toc          bool   `yaml:"toc"`
}

// extract YAML header from Markdown data and return metadata and content
func extractYAMLHeader(mdData []byte) (*Metadata, []byte, error) {
	mdStr := string(mdData)
	lines := strings.Split(mdStr, "\n")

	// extract YAML header area
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

		// handle document type
		if meta.DocumentType == "" {
			meta.DocumentType = "report"
			meta.TemplatePath = "../../typst-templates/report/report.typ"
		} else if meta.DocumentType == "report" {
			meta.TemplatePath = "../../typst-templates/report/report.typ"
		} else if meta.DocumentType == "assignment" {
			meta.TemplatePath = "../../typst-templates/assignment/lib.typ"
		}

		// extract content area
		content := strings.Join(lines[i+1:], "\n")
		return &meta, []byte(content), nil
	}
	return nil, mdData, nil
}

// generate Typst metadata header from YAML metadata using template/typst_header.tpl
func generateTypstHeader(meta *Metadata) (string, error) {

	// parse header template
	tpl, err := template.ParseFiles("templates/typst_header.tpl")
	if err != nil {
		return "", err
	}

	// execute template with metadata
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, meta); err != nil {
		return "", err
	}

	// return rendered header
	return buf.String(), nil
}
