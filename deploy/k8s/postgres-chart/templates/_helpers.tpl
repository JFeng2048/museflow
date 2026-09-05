{{/*
============================================================
_helpers.tpl - 公共模板辅助函数
============================================================
*/}}

{{/*
展开 chart 名称与版本
*/}}
{{- define "postgres-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
资源全名（兼容 nameOverride / fullnameOverride）
*/}}
{{- define "postgres-chart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
通用标签
*/}}
{{- define "postgres-chart.labels" -}}
helm.sh/chart: {{ include "postgres-chart.chart" . }}
{{ include "postgres-chart.selectorLabels" . }}
app.kubernetes.io/part-of: infrastructure
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
选择器标签（StatefulSet selector 必须固定，不能随 release 变化）
*/}}
{{- define "postgres-chart.selectorLabels" -}}
app: postgres
{{- end }}

{{/*
资源命名空间
*/}}
{{- define "postgres-chart.namespace" -}}
{{- default .Release.Namespace .Values.namespace }}
{{- end }}

{{/*
Secret 名称：优先使用已存在的 Secret
*/}}
{{- define "postgres-chart.secretName" -}}
{{- if .Values.credentials.existingSecret }}
{{- .Values.credentials.existingSecret }}
{{- else }}
{{- include "postgres-chart.fullname" . }}-secret
{{- end }}
{{- end }}

{{/*
ConfigMap 名称
*/}}
{{- define "postgres-chart.configName" -}}
{{- include "postgres-chart.fullname" . }}-config
{{- end }}

{{/*
初始化脚本 ConfigMap 名称
*/}}
{{- define "postgres-chart.initScriptsName" -}}
{{- include "postgres-chart.fullname" . }}-init-scripts
{{- end }}