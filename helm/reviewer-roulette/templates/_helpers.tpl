{{/*
Expand the name of the chart.
*/}}
{{- define "reviewer-roulette.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "reviewer-roulette.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "reviewer-roulette.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "reviewer-roulette.labels" -}}
helm.sh/chart: {{ include "reviewer-roulette.chart" . }}
{{ include "reviewer-roulette.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "reviewer-roulette.selectorLabels" -}}
app.kubernetes.io/name: {{ include "reviewer-roulette.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "reviewer-roulette.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "reviewer-roulette.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database URL construction
*/}}
{{- define "reviewer-roulette.databaseURL" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "postgresql://%s:%s@%s:%d/%s?sslmode=%s" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "reviewer-roulette.fullname" .) (.Values.postgresql.service.port | int) .Values.postgresql.auth.database .Values.config.postgres.sslMode }}
{{- else }}
{{- printf "postgresql://%s:%s@%s:%d/%s?sslmode=%s" .Values.config.postgres.user .Values.config.postgres.password .Values.config.postgres.host (.Values.config.postgres.port | int) .Values.config.postgres.database .Values.config.postgres.sslMode }}
{{- end }}
{{- end }}

{{/*
Redis URL construction
*/}}
{{- define "reviewer-roulette.redisURL" -}}
{{- if .Values.redis.enabled }}
{{- if .Values.redis.auth.enabled }}
{{- printf "redis://:%s@%s-redis-master:%d/%d" .Values.redis.auth.password (include "reviewer-roulette.fullname" .) (.Values.redis.service.port | int) (.Values.config.redis.db | int) }}
{{- else }}
{{- printf "redis://%s-redis-master:%d/%d" (include "reviewer-roulette.fullname" .) (.Values.redis.service.port | int) (.Values.config.redis.db | int) }}
{{- end }}
{{- else }}
{{- if .Values.config.redis.password }}
{{- printf "redis://:%s@%s:%d/%d" .Values.config.redis.password .Values.config.redis.host (.Values.config.redis.port | int) (.Values.config.redis.db | int) }}
{{- else }}
{{- printf "redis://%s:%d/%d" .Values.config.redis.host (.Values.config.redis.port | int) (.Values.config.redis.db | int) }}
{{- end }}
{{- end }}
{{- end }}
