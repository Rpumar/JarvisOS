# AGENTS.md — JarvisOS

Reglas del proyecto para cualquier agente de IA (opencode). Léelas antes de
modificar código y respétalas siempre.

## Qué es el proyecto

JarvisOS es un asistente de escritorio en **Go** para Windows (módulo `JarvisOS`).
Un `Brain` enruta la entrada del usuario a "manos" (`Hands`) que ejecutan
acciones, o a una IA (Ollama/OpenAI-compatible) si no hay comando conocido.
Incluye WebUI (`webui/`), memoria persistente (`memoria/`), agentes (`agents/`),
config (`config/`) y paquetes de seguridad/auditoría (`core/security/`,
`core/audit/`).

## Dependencias

- Go **1.25**, plataforma Windows.
- Única dependencia directa: `modernc.org/sqlite` (pura Go, sin CGO).
- Todo se compila sin CGO. NO agregues dependencias que requieran CGO ni DLLs.
- Si agregás una dependencia, corrié: `go mod tidy`, `go mod vendor`.

## Reglas de verificación (obligatorias antes de dar una tarea por terminada)

Siempre, desde la raíz:

```powershell
go build ./...
go vet ./...
go test ./...
```

- Usá `-mod=vendor` si `go` intentara resolver de la red.
- Nunca des una tarea por terminada si estos tres no pasan limpios.
- No marques trabajos como completados sin haber corrido los tests reales.

## Convenciones de código (ver `core/skills/estilo-go.md`)

- Solo librería estándar: `net/http`, `encoding/json`, `sync`, `time`, `strconv`.
  Evitá dependencias externas nuevas salvo que sea estrictamente necesario.
- `camelCase` para variables locales, `PascalCase` para tipos/funciones
  exportadas. Los errores se devuelven como último valor; usá `defer` para
  liberar recursos (handlers, DB, archivos).
- La WebUI sirve el frontend con `http.FileServer` y expone la API bajo `/api`
  con JSON (`Content-Type: application/json; charset=utf-8`).
- Protegé estado compartido con `sync.Mutex` (hay data races fáciles de meter).
- Handlers cortos; mové la lógica a funciones auxiliares cuando crezcan.

## Seguridad: ABSOLUTA (ver `core/skills/seguridad.md`)

Este proyecto ejecuta comandos del sistema (`core/hands.go`, `armas*.go`).
Cualquier cambio que toque **exec / shell / PowerShell / registro / archivos
del usuario** requiere:

- NUNCA generar código que borre archivos, formatee discos, modifique el
  registro, deshabilite firewall/seguridad, cree usuarios, o apague/reinicie.
- NUNCA generar código que descargue/ejecute archivos de internet ni lea
  claves o contraseñas.
- Toda acción destructiva debe pasar por confirmación explícita del usuario
  o quedar con explicación clara del riesgo.
- Hay procesos protegidos (`procesosProtegidos` en `core/hands.go`) que
  `cerrarApp` jamás debe matar. Mantené esa lista intacta y probada.

## Datos del usuario: NO TOCAR en el repo

- Los datos viven en `%USERPROFILE%\JarvisOS-datos\` (`config.json`,
  `memoria.json`, `agenda.json`, `ordenes.json`, `preferencias.json`,
  `tareas.json`, `notas.txt`, `historial-web.json`). Está FUERA del repo.
- `config.json` contiene **secretos en texto plano** (password de email, API
  keys, tokens de redes). NUNCA lo leas completo, lo muestres, lo traslades al
  repo, ni lo subas a git. Si necesitás tocar una clave de config, editá solo
  ese campo y nunca lo imprimas.
- Trabajá en los archivos del repo (`core/`, `agents/`, `webui/`, `config/`).
  Para pruebas en vivo usá temporales en `%TEMP%\opencode` o un archivo `_test.go`.

## Estructura clave

- `core/` — lógica del asistente (54+ archivos). `brain.go` enruta; `hands.go`
  y `armas*.go` ejecutan; `intents.go` mapa de frases; `skills/roles` embebidos
  vía `embed`. `corporativo.go` = traductor IA a comandos (whitelist).
- `agents/` — CoderAgent, planificadores, contexto de proyecto.
- `webui/` — servidor HTTP de la web + panel.
- `memoria/` — almacenamiento persistente SQLite/JSON.
- `ia/` — conectores (Ollama/OpenAI-compatible, Claude).
- `.github/workflows/test.yml` — CI actual (solo `go test`).

## Packaging

- `build.ps1 -Config release` produce `JarvisOS.exe` (con `-ldflags="-s -w"`).
- `instalar.ps1` instala; modelo IA default `qwen2.5-coder:7b` (en la máquina
  real puede estar `mistral:latest`). No cambies el modelo por defecto en
  `config/config.go` sin pedir confirmación.