{{/*
Expand the name of the chart.
*/}}
{{- define "proxynd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
release-name이라고 나오는거 없애기
*/}}
{{- define "proxynd.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- if eq .Release.Name "release-name" }}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "proxynd.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "proxynd.labels" -}}
helm.sh/chart: {{ include "proxynd.chart" . }}
{{ include "proxynd.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "proxynd.selectorLabels" -}}
app.kubernetes.io/name: {{ include "proxynd.name" . }}
app.kubernetes.io/instance: {{ include "proxynd.fullname" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "proxynd.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "proxynd.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
