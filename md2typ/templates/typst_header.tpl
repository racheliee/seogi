#import "{{ .TemplatePath}}":*

#show: {{ .DocumentType }}.with(
	{{- if .Title }}
  title: [{{ .Title }}],
	{{- end }}
	{{- if .Course }}
  course: [{{ .Course }}],
	{{- end }}
	{{- if .Date }}
  date: [{{ .Date }}],
	{{- end }}
	{{- if .Authors }}
  authors: (
      {{- range .Authors }}
    (
      name: "{{ .Name }}",
      {{- if .StudentNo }}
      student-no: "{{ .StudentNo }}",
      {{- end }}
      {{- if .Department }}
      department: [{{ .Department }}],
      {{- end }}
      {{- if .Organization }}
      organization: [{{ .Organization }}],
      {{- end }}
      {{- if .Email }}
      email: "{{ .Email }}"
      {{- end }}
    ),
      {{- end }}
  ),
	{{- end }}
	{{- if .Bibliography }}
  bibliography: bibliography("{{ .Bibliography }}"),
	{{- end }}
)
{{ if .Toc }}
// table of contents
#v(6mm)
#outline()
#pagebreak()
{{ end }}
