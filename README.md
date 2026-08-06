# JarvisOS

Asistente de voz de escritorio para Windows, en español, con reconocimiento
**offline** (Vosk) y respaldo conversacional opcional por IA. Maneja la PC
con la voz: apps, búsquedas, memoria, recordatorios, tareas, órdenes,
email, documentos de Office, agenda y PDF — todo sin librerías externas
más allá de la estándar y PowerShell.

**Versión:** 0.14.0+ (F1–F4 completas, F5 en curso).
**Ver `PROGRESO_JARVISOS.md`** para el historial técnico detallado y
**`INSTALACION.md`** para la guía de instalación.

## 1. Funcionalidades

1. **Gestor de Órdenes** (`core/ordenes.go`): orden = {objetivo, fecha, estado, historial de acciones, reporte}. Estados: `pendiente → en_progreso → verificando → terminada`, más `bloqueada` y `cancelada`.
2. **El empleado no abandona**: las órdenes persisten en disco; al arrancar, Jarvis anuncia las pendientes y las retoma con "retomá las órdenes".
3. **Reconocimiento de voz offline** (Vosk, español) y síntesis con voz de Windows.
4. **Comandos de sistema**: abrir/cerrar apps, volumen real por `keybd_event`, play/pausa/pista, captura de pantalla, bloqueo de pantalla, minimizar, red (IP, ping, velocidad, escaneo), RAM/CPU/disco, plan de energía, batería, temporales, organización de Descargas.
5. **Búsquedas y navegación**: Google, YouTube, Wikipedia, "ir a [sitio]" con URL directa.
6. **Memoria de sesión** (última app/búsqueda, historial IA) y **memoria persistente** en JSON (`%USERPROFILE%\JarvisOS-datos\memoria.json`): hechos ("recordá que me llamo X"), notas libres, y comandos "qué recordás", "cuál es mi nombre".
7. **Recordatorios y timers**: "recordame llamar a mamá a las 5 de la tarde" (Jarvis habla solo a la hora), "poné un timer de 5 minutos", listar y cancelar.
8. **Email (F3)**: envío por **SMTP con la librería estándar** (Gmail/Outlook con contraseña de aplicación; campos `email_smtp_*` en `config.json`) y **lectura de bandeja por IMAP** (cliente mínimo propio sobre stdlib, `email_imap_host`/`port`/`max`). Comando: "enviá un email a persona@dominio.com con asunto ... y el texto ..." y "leé los últimos 5 correos". El envío es una **acción externa**: pasa por **aprobación** del dueño (PIN/panel) antes de salir y queda en la auditoría. ✅ **Probado en vivo con cuenta real** (Gmail, contraseña de aplicación): SMTP envió un correo real y IMAP lo leyó de vuelta de la bandeja (smoke test con gate `JARVIS_EMAIL_SMOKE=1`).
9. **Trato personalizado**: si le decís tu nombre, te llama por él en todas las respuestas.
10. **Seguridad (F2)**: paquetes `core/security` (clasificación de riesgo) y `core/audit` (registro inmutable); acciones sensibles requieren **aprobación por PIN/panel** con reanudación automática; procesos críticos de Windows protegidos contra `cerrarApp`; los scripts generados por IA bloquean patrones peligrosos y nunca se ejecutan sin confirmación explícita.
11. **Office por COM (F3)**: crear documentos de **Word/Excel/PowerPoint** vía PowerShell COM (`core/manos_office.go`), sin librerías externas. Comandos: "creá un documento word llamado informe", "crear una planilla de excel en presupuesto", "hacé un powerpoint". ✅ **Probado en vivo** con Office 16: los tres formatos se generan y guardan en `workspace_root` (smoke test con gate `JARVIS_SMOKE=1`).
12. **Calendario/agenda local (F3)**: `core/agenda.go` — eventos con título, inicio y opcional ubicación, persistidos en **JSON** (`JarvisOS-datos/agenda.json`), **offline y sin credenciales**. Reutiliza los parsers de fecha/hora de los recordatorios ("mañana", "el martes", "el 5 de agosto", "a las 15"). Comandos: "agendá una reunión mañana a las 15", "qué tengo hoy", "qué tengo mañana", "próximos eventos", "cancelá el evento X".
13. **Sincronización con Outlook (F3)**: `core/manos_outlook.go` — exporta los eventos de la agenda local como citas en el **calendario de Outlook por COM** (`outlook.application`, sin librerías externas) y los marca para no duplicarlos; también lee los próximos turnos. Comandos: "sincronizá la agenda con outlook", "leé mis próximos eventos de outlook". Requiere Outlook instalado y una cuenta configurada (si no hay cuenta, responde con un error claro). Smoke test con gate `JARVIS_SMOKE=1`.
14. **PDF (F3)**: `core/manos_pdf.go` — dos vías sin librerías externas: (a) **generador de PDF puro en Go** (formato mínimo: catálogo, páginas, stream de texto Helvetica, xref validado en tests; codificación WinAnsi para acentos del español), para "exportá mis notas a pdf"; (b) **conversión de Office → PDF por COM** (Word `SaveAs2(...17)`, Excel `ExportAsFixedFormat(0,...)`, PowerPoint `SaveAs(...32)`) para "convertí informe.docx a pdf". Los PDF generados guardan en `workspace_root`.
15. **Redes sociales (F3)**: `core/manos_redes.go` — publicar en **X** (API v2 `https://api.twitter.com/2/tweets` con firma **OAuth 1.0a HMAC-SHA1** implementada a mano sobre la librería estándar, RFC 5849, validada contra el vector canónico del RFC) y en **LinkedIn** (`https://api.linkedin.com/v2/ugcPosts` con Bearer). Comandos: "publicá en x que estoy trabajando en Jarvis", "publicá en linkedin un aviso de la promo", "twitteá un tuit". Publicar es una **acción externa visible**: pasa por **aprobación** del dueño (PIN/panel) antes de salir y queda en la auditoría. Credenciales en `config.json`: `x_api_key`, `x_api_secret`, `x_access_token`, `x_access_secret`, `linkedin_token`, `linkedin_author`. Los tests apuntan a servidores locales (`httptest`) para no tocar la red real.
16. **Web UI** (`webui/`): panel de control con **roles** (Operador vs Admin), login con contraseña, sesiones, **aprobación de acciones sensibles por PIN o botón** y **visor de auditoría**.

## 2. Arquitectura

```
JarvisOS/
├── main.go               # arranque, loop de escucha, vigía de recordatorios
├── core/                 # lógica central (hands, brain, intents, agenda,
│                         #   outlook, office, pdf, tareas, órdenes, rutinas,
│                         #   recordatorios, memoria de sesión, seguridad, audit)
│   ├── hands.go          # ejecuta acciones en Windows (apps, volumen, etc.)
│   ├── brain.go          # orquestador: comando local → IA de respaldo
│   ├── ears.go           # micrófono (PortAudio) + STT (Vosk)
│   └── ...               # ver funcionalidades
├── config/               # configuración centralizada (config.json)
├── ia/                   # conector de respaldo a IA (OpenAI-compatible)
├── memoria/              # memoria persistente en JSON (SQLite para listas)
├── webui/                # panel web con roles y aprobación
└── vendor/               # dependencias vendorizadas (sin red en build)
```

- Los comandos se enrutan por **intents** (`core/intents.go`): frases + palabras clave, con prioridad por orden (los más específicos primero).
- Todo lo externo (Office, Outlook, PowerShell, red) va por `ejecutarConTimeout` con timeout de 30 s por defecto.
- **Sin librerías externas para F3**: Office/Outlook/PDF usan la librería estándar de Go + PowerShell COM.

## 3. Suite de tests

Suite verde actual: **core** (~22 s, incluye enrutamiento completo con una IA falsa de prueba, diagnóstico, email, Office, Outlook, PDF y agenda), **ia**, **webui** (RBAC, login, sesiones y todos los handlers de la API), **memoria** (incluye el CRUD de listas que destapó un bug real de queries anidadas en `ObtenerListas`, corregido) y **config** (Load/Save, defaults, archivo ausente/corrupto). También hay smoke tests reales por web (agendar → aprender → cumplir; persistencia tras reinicio) y smoke tests con gate de ambiente para **email real** (`JARVIS_EMAIL_SMOKE=1`), **Office real** (`JARVIS_SMOKE=1`) y **Outlook** (`JARVIS_SMOKE=1`).

```
go build ./...           # compila todo
go vet ./...             # análisis estático
go test ./...            # todos los tests
```

Cobertura aproximada: **security 100%**, **audit 96%**, **config 85%**, **webui 77%**, **memoria 57%**, **core ~50%**, **ia 31%**.

## 4. Roadmap

- **F1 — Base**: ✅ completa (personalidad, micrófono, comandos de sistema, IA de respaldo).
- **F2 — Confiabilidad B2B** ✅ *completa*: paquetes `security` (clasificación de riesgo) y `audit` (registro inmutable); **aprobación por PIN/panel con reanudación automática** para acciones sensibles; **timeout de aprobación de 5 min** (la orden expira sola) y **rotación automática de la auditoría** a 10 MB; **timeout de ejecución de comandos externos** de 30 s; **acceso con roles** (Operador vs Admin) con **contraseña de acceso**, sesión web, **visor de auditoría** en el panel y aprobación solo para Admin. Decisión: la auditoría se consume por pantalla, no por voz (la voz no puede autenticar quién pide el dato).
- **F3 — Integraciones de oficina** *(en curso)*: **email completo (envío SMTP + lectura IMAP)** ✅ probado en vivo con cuenta real; **Office por COM (Word/Excel/PPT)** ✅ probado en vivo con Office 16; **calendario/agenda local** ✅ (eventos en `agenda.json` bajo `JarvisOS-datos`, offline, sin credenciales; comandos "agendá una reunión mañana a las 15", "qué tengo hoy", "cancelá el evento X"); **sync con calendario de Outlook por COM** ✅ (exporta la agenda local como citas, sin duplicados; requiere cuenta configurada); **PDF** ✅ (generador puro en Go + conversión Office→PDF por COM); **redes sociales** ✅ (X con OAuth 1.0a sobre stdlib + LinkedIn con Bearer, ambos exigiendo aprobación; validados con vector canónico del RFC 5849 y servidores de prueba; falta smoke real con credenciales del cliente).
- **F4 — Producto y presencia B2B** ✅ *completa*: marca corporativa ("Jarvis — su empleado digital. Cumple, no descansa, rinde cuentas"), dashboard del dueño con aprobaciones y auditoría, **onboarding guiado de primer arranque** (empresa + dueño + PIN + contraseña en un paso, < 15 min), perfil de empresa estructurado y modo vigilante aparte.
- **F5 — Propuesta comercial** *(en curso)*: `PROPUESTA-COMERCIAL.md` ✅ (problema, solución, privacidad como ventaja, demo guionizada de 3 escenarios, piloto con métricas de ROI, precios Lite/Pro/Empresa y soporte); demo funcional en vivo = capacidades ya implementadas.

## 5. Instalación

Ver **`INSTALACION.md`** — guía paso a paso (MSYS2, PortAudio, Vosk, variables de entorno, y troubleshooting con el error `-mthreads` ya resuelto de antemano con `CGO_CFLAGS_ALLOW`).

## 6. Puntos abiertos para una opinión externa

1. **Seguridad**: ✅ acciones sensibles ya no bloquean en seco: pasan a **`esperando_aprobacion`**, alerta en la Web UI y aprobación por **PIN del dueño** o botón del panel, con reanudación automática del bucle. ✅ **Roles**: el panel distingue **Operador** (consulta) de **Admin** (aprueba, ve auditoría) con contraseña de acceso y sesión web. Pendiente: doble factor más estricto.
2. **F3**: email y Office probados en vivo; Outlook requiere cuenta configurada en la máquina del cliente; redes sociales implementadas con tests contra el vector canónico del RFC 5849 y servidores `httptest`, pendientes de smoke real con las API keys del cliente (X: developer.twitter.com; LinkedIn: developer.linkedin.com).
3. **Cobertura**: ✅ config (85%) y webui (77%) ahora tienen tests propios. Los tests de config destaparon y corrigieron un bug real: un `config.json` sin `require_approval` **desactivaba silenciosamente la aprobación de acciones sensibles** (el default es `true`); ahora solo se pisa el campo cuando el archivo lo define. Pendiente: ia y parte de core.
4. **PDF**: el generador puro es texto simple (sin tablas/imágenes); para documentos ricos se recomienda el camino Office→PDF.
5. **Arquitectura**: ✅ paquetes `core/security` y `core/audit` creados y desacoplados de `core`. ✅ memoria + IA con cobertura real. Pendiente: evaluar separar `integraciones`/`control` en paquetes propios al llegar a F3.
