# Política de Privacidad — JarvisOS

> Versión 1.0 · Fecha de vigencia: {FECHA} · Jurisdicción: {PAÍS}
> Plantilla para el dueño del proyecto: completá los campos `{...}`.

## 1. Principio central

JarvisOS se diseñó con una regla: **los datos no salen de la PC de la
empresa salvo que el cliente lo pida y lo apruebe expresamente**. Este
documento explica, con precisión técnica, qué se procesa, dónde vive y
qué puede salir — y en qué condiciones.

## 2. Qué datos procesa y dónde viven

Todo se almacena **en la PC del puesto**, en la carpeta
`%USERPROFILE%\JarvisOS-datos`:

| Dato | Archivo | Ejemplos |
|---|---|---|
| Configuración y credenciales | `config.json` | claves de email, APIs, PIN/contraseña (hash), clave de licencia |
| Memoria persistente | `memoria.json` | hechos recordados por el usuario |
| Tareas | `tareas.json` | pendientes y completadas |
| Agenda | `agenda.json` | reuniones y eventos |
| Órdenes | `ordenes.json` | objetivos, historial de acciones, reportes |
| Preferencias | `preferencias.json` | nombre, apps, saludo |
| Notas | `notas.txt` | notas tomadas por voz |
| Auditoría | `auditoria.jsonl` | quién, qué y cuándo se ejecutó (inmutable) |
| Perfiles de usuario | `perfil.json` | nombres, áreas y roles de puestos |
| Empresa | `empresa.json` / `empresa.md` | perfil del negocio |
| Formularios | `formularios.json` | plantillas de cargas repetitivas |
| Historial web | `historial-web.json` | conversaciones del panel web |
| Backups locales | `backups/` | copias automáticas con rotación |

Los archivos se escriben con permisos restringidos (solo el usuario del
sistema). El Software **no transmite** esta carpeta a ningún servidor.

## 3. Qué puede salir de la PC (solo con acción y aprobación del usuario)

El Software está **desconectado por defecto**. Solo se produce tráfico de
red cuando el usuario lo configura y lo aprueba:

- **Email**: enviar/leer por SMTP/IMAP con las credenciales de la propia
  cuenta del usuario (los mensajes viajan a su proveedor de correo).
- **Redes sociales**: publicar en X/LinkedIn con las cuentas del usuario,
  **siempre con aprobación** (PIN del dueño o panel).
- **IA externa (opcional)**: si el usuario configura un proveedor de IA
  OpenAI-compatible (p. ej. Groq, OpenRouter), las preguntas que se le hagan
  y el contexto de la conversación pueden enviarse a ese proveedor. Es una
  decisión de configuración del usuario; por defecto el Software intenta usar
  un modelo local (Ollama) y no envía nada.
- **Internet bajo demanda**: búsquedas web o apertura de URLs que el usuario
  pida explícitamente.

Nada de esto ocurre automáticamente ni sin configuración o aprobación.

## 4. Reconocimiento de voz

El reconocimiento de voz es **offline** (local, Vosk). El audio no se
transcribe en ningún servidor y no se sube a la nube.

## 5. Seguridad

- Contraseña de acceso al panel con hash (SHA-256) y sesiones web con cookie
  HttpOnly.
- Aprobación obligatoria (PIN del dueño o panel) para acciones sensibles:
  enviar, publicar, borrar, instalar, formatear, apagar.
- Registro de auditoría inmutable y exportable para el dueño.
- Backups automáticos locales con rotación.
- Roles: `dueño` / `admin` / `empleado` controlan quién aprueba y quién ve
  la auditoría.

## 6. Retención y borrado

- Los datos se conservan mientras el usuario no los borre. El Software no
  establece plazos de retención propios.
- Borrar la carpeta `JarvisOS-datos` elimina los datos de forma local; se
  recomienda usar los backups para recuperaciones accidentales.
- Para eliminar datos personales definitivamente: borrar la carpeta de datos
  y, si corresponde, vaciar la papelera del sistema.

## 7. Derechos del responsable de datos

El **cliente (empresa) es el responsable** de los datos que ingresa al
Software. Puede acceder, corregir, exportar y eliminar sus datos en cualquier
momento porque residen en su PC. El proveedor de JarvisOS no tiene acceso
remoto a esos datos.

## 8. Contacto

Para consultas de privacidad: {EMAIL / CANAL}. Respuesta en {PLAZO}.

---

**Cambios a esta política:** la versión vigente estará siempre disponible
en el repositorio/documentación del Software. Los cambios se comunicarán a
los clientes con suscripción activa.
