#figure(
  placement: none,
  image("{{ .ImagePath }}")
  {{- if .Caption }}, 
  caption: [{{ .Caption }}]{{- end }}
){{- if .Label }} <fig:{{ .Label }}> {{- end }}
