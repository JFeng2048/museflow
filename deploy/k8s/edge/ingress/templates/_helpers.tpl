{{/*
对外访问域名：唯一来源是 applications/overlays/secrets.yaml 的 domain.host。
不设默认值，缺失时直接报错，避免渲染出错误域名或静默回退到历史值。
*/}}
{{- define "ingress.domainHost" -}}
{{- $domain := .Values.domain | default dict -}}
{{- required "缺少 domain.host：请在 deploy/k8s/applications/overlays/secrets.yaml 中配置对外域名" $domain.host -}}
{{- end -}}
