# Changelog

Todos los cambios notables de JarvisOS se documentan aquí.
Formato: [Keep a Changelog](https://keepachangelog.com/es/1.1.0/),
versionado: [SemVer](https://semver.org/lang/es/).

El software en sí no sigue estrictamente SemVer (versión 0.x), pero los
cambios se agrupan por versión mayor, con la versión actual reflejada en
`config.json` (`version`) y en el banner de arranque.

## [0.14.1] — En curso

### Agregado
- Plano de control en la nube (decisión 1, F5+): paquete `control/` con
  servidor HTTP (`JarvisOS.exe --control`) que emite y gestiona licencias
  online, registra instalaciones (heartbeat) y controla puestos en uso.
  Persistencia JSON con escritura atómica, solo librería estándar.
  - API: `emitir`/`suspender`/`reactivar` (token maestra vía env
    `JARVISOS_CONTROL_TOKEN`), `activar` y `heartbeat` (el agente se
    autentica con su licencia), `estado` (panel admin).
  - El agente reporta al plano de control cuando `control_url` está
    configurado: activa la instalación al arranque (`id_instalacion`
    autogenerado y persistido) y hace heartbeat con puestos en uso cada
    5 min. Todo best-effort: si el servidor no responde, el agente sigue
    funcionando 100% local.
  - Comando por voz: "estado del plano de control" / "sincronizá la
    licencia".
  - Actualizaciones (F5+): el servidor publica la última versión
    (`POST /api/v1/version` con token admin, `GET /api/v1/version` público)
    y el heartbeat la entrega al agente; por voz se avisa cuando hay una
    actualización disponible ("Hay una actualización disponible: versión X").
  - Panel del dueño en la nube: `http://<servidor>:8443/panel` es un
    dashboard web embebido en el binario (sin dependencias externas) que
    muestra licencias, instalaciones, puestos y última versión; permite
    emitir licencias, suspender/reactivar y publicar versiones. La página
    es pública (HTML) pero todas sus acciones van por la API con token
    maestra, que nunca vive en el navegador.
  - Tests: emisión, activación, límite de puestos, suspensión/reactivación,
    persistencia, flujo HTTP completo, cliente real contra el servidor,
    publicación/consulta de versión y panel (HTML servido, redirección de
    raíz y autenticación exigida en la API).
- Documentación legal y de privacidad (decisión 5): `EULA.md` (licencia por
  plan/puesto, restricciones, garantía, limitación de responsabilidad) y
  `PRIVACIDAD.md` (datos 100% locales, qué puede salir de la PC y bajo qué
  condiciones, seguridad, retención y derechos del responsable).
- Clave de licencia por instalación con límite de puestos (decisión 4.1):
  `core/licencia.go` genera y valida claves `JARVIS-PLAN-PUESTOS-NONCE-FIRMA`
  firmadas con HMAC-SHA256 (plan lite 1 puesto / pro 5 / empresa 50). Se valida
  al arrancar (banner `[LICENCIA]`), por voz ("activá la licencia ..." /
  "qué licencia tengo") y desde el panel web. Sin clave = modo piloto sin
  límite; con clave, registrar usuarios por voz o panel respeta el tope de
  puestos.
- Backups automáticos de `JarvisOS-datos` con rotación a 7 (`core/backups.go`).
  - Se respalda al arrancar (consola y modo servicio) de forma best-effort.
  - Comando por voz: "hacé un respaldo" crea uno, "qué respaldos tengo" lista.
  - Tests: copia de árbol, no-recursión, rotación, carpetas vacías.

### Corregido
- Una orden con procedimiento conocido ya **no se bloquea** cuando la IA
  falla (error de conexión o respuestas que no son JSON): si los pasos del
  procedimiento se ejecutaron, la orden se marca cumplida. El bloqueo queda
  solo para órdenes sin procedimiento.
- `demo.ps1`: `procedimientos.json` se sembraba como `{"procedimientos": [...]}`
  (formato que el gestor no lee) y `ConvertTo-Json` en PowerShell 5.1 convierte
  listas de 1 elemento en objeto suelto. Se sembran ahora listas planas
  (`EscribirJSONLista`) en `ordenes.json`, `tareas.json`, `agenda.json` y
  `procedimientos.json`; el escenario 1 de la demo se cumple de verdad.
- Confirmar una acción peligrosa **no implementada** ahora da una guía
  ("aprendé que para hacer X: paso 1, paso 2") en vez del marcador crudo
  `__NO_RECONOCIDO__`.
- "Rellenar formulario" se audita como acción externa aprobable y se ejecuta
  de verdad al confirmar (se eliminó la rama que siempre devolvía `true`).
- CI (`test.yml`) no se disparaba: el repo usa `master` y el workflow solo
  escuchaba `main`.

## [0.14.0] — 2025

### Agregado
- F1–F4 completas y F5 en curso (ver `PLAN-EMPRESA.md`).
- Órdenes que no se abandonan (`core/ordenes.go`): persistencia atómica,
  historial de acciones, reporte, retomar tras reinicio y cada 5 min
  (`vigilarOrdenes`).
- Bucle de agente con IA en JSON estricto: decide acciones, ejecuta, observa,
  ajusta y aprende procedimientos; las sensibles esperan aprobación.
- Seguridad y auditoría (`core/security`, `core/audit`): clasificación de
  riesgo, registro inmutable JSONL con rotación a 10 MB, aprobación por PIN
  con expiración por timeout y watchdog.
- Roles y permisos (dueño/admin/empleado), contraseña de acceso local y
  panel del dueño en la WebUI con aprobaciones y visor de auditoría.
- Integraciones de oficina (F3): email SMTP + lectura IMAP con cliente propio,
  Office por COM (Word/Excel/PPT), Outlook, PDF (generador puro en Go +
  conversión por COM), agenda local con sync a Outlook, redes sociales
  (X OAuth 1.0a + LinkedIn), formularios web con autocompletado.
- Perfil de empresa estructurado (`empresa.json`), onboarding guiado en el
  primer arranque y modo asistente corporativo (traductor IA con whitelist).
- Quitado el subsistema de desarrollo (CoderAgent, proyectos web); el modelo
  IA default vuelve a `mistral:latest`.

### Corregido
- EOF de stdin ya no spamea avisos en loop; "mis tareas" lista tareas en vez
  de caer al fallback IA; "creá una tarea X" agenda directo.

## [0.13.0] — 2025

### Agregado
- F1–F3: órdenes, auditoría/RBAC/aprobaciones y primeras integraciones.
- Conector IA agnóstico (Ollama/Groq/OpenRouter), roles integrados, tareas y
  aprendizaje de procedimientos con consulta.
- Empleado digital "lite": limpieza del bloat personal/diversión.
- Calidad: `golangci-lint` config, corrección de ~45 errores sin chequear,
  código muerto eliminado, `govulncheck` sin vulnerabilidades.

## [0.12.x] — 2025

### Agregado
- Skills + ciclo iterativo de mejora de proyectos (editar-verificar-corregir).
- Desarrollador fullstack por voz (retirado en 0.14.0).
- Estabilidad web: recuperación de panics, puerto automático, timeout en
  PowerShell y banner de reconexión.

## [0.11.x] — 2025

### Agregado
- Ingeniero: diagnóstico profundo con puntaje de salud, mantenimiento,
  integridad, servicios, eventos y modo vigilante con alertas.
- Plan de acción priorizado por gravedad con ejecución automática de
  acciones seguras.

## [0.10.x] — 2025

### Agregado
- Gestión de archivos por voz (crear, buscar, ubicar, borrar a papelera) y
  notas; notas exentas de confirmación.
- WebUI: panel de mando con estado en vivo, botones rápidos e historial
  persistente; TTS configurable y dictado por micrófono en la web; modo
  presentación fullscreen con HUD.

## [0.9.x] — 2025

### Agregado
- Armas (manos de sistema): ping, velocidad de internet, escaneo de red,
  flush DNS, adaptadores, RAM, plan de energía, limpieza de temporales,
  organizar descargas, grabar pantalla, modo oscuro, firewall, puertos,
  procesos en red, sesiones y rutinas, notificaciones, comprimir/descomprimir,
  expulsar USB, mantener despierta, audio, cámaras, informe del sistema y
  portapapeles.

## [0.8.x] — 2025

### Agregado
- Preferencias persistentes (apps, comandos, nombre y saludo personalizado).
- Event bus y clasificador de intenciones (fuzzy).
- Apps configurables, clima/news con API key, apagar PC, SQLite en memoria.
- Push-to-talk: micrófono solo al activar con "jarvis".
- Modelo configurable y agente con Ollama; plan persistente y resumen
  automático; aprendizaje de comandos.

## [0.7.0] — 2025

### Agregado
- SQLite, script de build, CI, vendor e instalador.
- Voz: reconocimiento offline (Vosk) con respaldo conversacional por IA.
