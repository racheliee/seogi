#figure(
  {{- if .Caption }}
  caption: [{{ .Caption }}],
  {{- end }}
  placement: {{ .Placement }},
  table(
    columns: {{ .Columns }},
    {{- if .Align }}
    align: {{ .Align }},
    {{- end }}
    inset: (x: 8pt, y: 4pt),
		stroke: (x, y) => if y <= 1 { (top: 0.5pt) },
		fill: (x, y) => if y > 0 and calc.rem(y, 2) == 0  { rgb("#efefef") },
    {{- if .Rows }}
    {{ .Rows }}
    {{- end }}
  )
) {{- if .Label }} <tab:{{ .Label }}> {{- end }}
