#import "../../typst-templates/report/report.typ":*

#show: report.with(
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
      department: [{{ .Department }}],
      organization: [{{ .Organization }}],
      email: "{{ .Email }}"
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
