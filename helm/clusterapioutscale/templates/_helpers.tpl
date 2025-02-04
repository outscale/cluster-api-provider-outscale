{{- define "clusterapioutscale.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.controllermanager" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-controller-manager" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.webhookservice" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-webhook-service" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.certificate" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-serving-cert" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.service" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-controller-manager-service" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.metricsClusterRole" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-metrics-reader" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.configmap" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-manager-config" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.proxyClusterRole" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-proxy-role" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.proxyClusterRoleBinding" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-proxy-rolebinding" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.managerClusterRole" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-manager-role" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.managerClusterRoleBinding" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-manager-rolebinding" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.leaderElectionRole" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-leader-election-role" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.leaderElectionRoleBinding" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-leader-election-rolebinding" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.serviceAccount" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-controller-manager" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.deployment" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-controller-manager" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.issuer" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-selfsigned-issuer" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.mutatingwebhook" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-mutating-webhook-configuration" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "clusterapioutscale.validatingwebhook" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-validating-webhook-configuration" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
