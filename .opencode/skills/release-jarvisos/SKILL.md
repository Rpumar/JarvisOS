---
name: release-jarvisos
description: Proceso de build y empaquetado de JarvisOS. Use when compiling the release executable, packaging, running build.ps1, bumping the version, or installing.
---

# Release de JarvisOS

## Build

Desde la raíz del repo:

```powershell
.\build.ps1 -Config release
```

Produce `JarvisOS.exe` (con `-ldflags="-s -w"`). Para debug:

```powershell
.\build.ps1 -Config debug
```

## Verificación antes del release

1. `go build ./...` compila limpio.
2. `go vet ./...` sin hallazgos.
3. `go test ./...` todo en verde.
4. El `.exe` se genera sin errores.

## Versión

- La versión vive en `config/config.go` (default `config.Load()`) y en `PROGRESO_JARVISOS.md`.
- No cambies el modelo IA por defecto (`qwen2.5-coder:7b`) sin pedir confirmación.

## Instalación

- `instalar.ps1` instala el asistente.
- `install_ollama.ps1` instala/configura Ollama con el modelo.
- Datos del usuario van a `%USERPROFILE%\JarvisOS-datos\` — nunca al repo.

## CI

- `.github/workflows/test.yml` corre `go test -mod=vendor -v ./...` en Windows.
- Antes de mergear, el CI debe pasar.
