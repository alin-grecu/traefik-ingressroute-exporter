{{- define "release.labels" -}}
app.kubernetes.io/name: {{ .Release.Name }}
{{- end -}}
