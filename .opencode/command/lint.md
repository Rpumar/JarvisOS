---
description: Revisa el proyecto con golangci-lint y govulncheck (si están instalados). Si falta una herramienta, indica cómo instalarla.
---
Revisá el proyecto Go con linters y verificación de vulnerabilidades:

1. `golangci-lint run` (si `golangci-lint` está instalado). Si no:
   `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
2. `govulncheck ./...` (si `govulncheck` está instalado). Si no:
   `go install golang.org/x/vuln/cmd/govulncheck@latest`

Ejecutá con `-mod=vendor`. Reportá los hallazgos y corregí los que apliquen.
