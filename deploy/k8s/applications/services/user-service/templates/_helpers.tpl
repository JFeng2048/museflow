{{/*
Turnstile 允许的 hostname：对外域名取自 applications/overlays/secrets.yaml 的
domain.host，本地调试域名见 userService.extraTurnstileHostnames。
*/}}
{{- define "user-service.turnstileHostnames" -}}
{{- $domain := .Values.domain | default dict -}}
{{- $host := required "缺少 domain.host：请在 deploy/k8s/applications/overlays/secrets.yaml 中配置对外域名" $domain.host -}}
{{- $extra := .Values.userService.extraTurnstileHostnames | default list -}}
{{- join "," (prepend $extra $host) -}}
{{- end -}}
