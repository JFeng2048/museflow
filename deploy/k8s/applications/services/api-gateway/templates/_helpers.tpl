{{/*
CORS 允许来源：由 applications/overlays/secrets.yaml 的 domain 派生，
格式为 <domain.scheme>://<domain.host>；额外来源见 apiGateway.extraAllowOrigins。
*/}}
{{- define "api-gateway.allowOrigins" -}}
{{- $domain := .Values.domain | default dict -}}
{{- $scheme := $domain.scheme | default "https" -}}
{{- $host := required "缺少 domain.host：请在 deploy/k8s/applications/overlays/secrets.yaml 中配置对外域名" $domain.host -}}
{{- $origin := printf "%s://%s" $scheme $host -}}
{{- $extra := .Values.apiGateway.extraAllowOrigins | default list -}}
{{- join "," (prepend $extra $origin) -}}
{{- end -}}
