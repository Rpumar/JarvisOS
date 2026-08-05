---
name: go-conventions
description: Convenciones y verificación obligatoria del proyecto Go JarvisOS. Use when editing Go code, adding dependencies, running tests, or finishing any coding task in this repo. Triggers on words like build, vet, test, mod, vendor, dependencia, convencion.
---

# Convenciones de Go de JarvisOS

Este proyecto es un asistente de escritorio en Go para Windows. Respetá estas reglas en todo cambio de código.

## Antes de dar una tarea por terminada (OBLIGATORIO)

Desde la raíz del repo, en orden:

```powershell
go build ./...
go vet ./...
go test ./...
```

- Usá `-mod=vendor` si `go` intentara resolver de la red.
- No des la tarea por terminada si estos tres no pasan limpios.
- Corré los tests reales; no supongas que pasan.

## Dependencias

- Única dependencia directa: `modernc.org/sqlite` (pura Go, sin CGO).
- Todo compila sin CGO. NUNCA agregues dependencias que requieran CGO ni DLLs.
- Si agregás una dependencia: `go mod tidy` y `go mod vendor`.

## Estilo (solo librería estándar)

- Librería estándar: `net/http`, `encoding/json`, `sync`, `time`, `strconv`.
- `camelCase` variables locales, `PascalCase` tipos/funciones exportadas.
- Errores como último valor de retorno; `defer` para liberar recursos.
- WebUI: `http.FileServer` para el frontend, API bajo `/api` con JSON.
- Protegé estado compartido con `sync.Mutex`.
- Handlers cortos; lógica a funciones auxiliares.

## Estructura

- `core/brain.go` enruta; `hands.go` y `armas*.go` ejecutan; `intents.go` frases.
- `core/roles/` y `core/skills/` son archivos `.md` embebidos vía `embed` con front-matter (`nombre`, `prioridad`, `descripcion`, `activar`). Al crearlos, sumá el archivo y actualizá los tests de defaults (`roles_test.go`, `skills_test.go`).
- `core/corporativo.go` = traductor IA a comandos con whitelist de prefijos.
- Los tests del repo son la fuente de verdad para el comportamiento.
