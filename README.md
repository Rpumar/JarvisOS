# JarvisOS — Empleado Digital Empresarial

> No vendemos IA. Vendemos un **empleado**.
> El dueño de la empresa da una orden; Jarvis la cumple y **no la cierra hasta terminarla**.

## 1. Qué es

JarvisOS es un asistente de escritorio para Windows, escrito en **Go**, que se está convirtiendo en un **empleado digital B2B**: un colaborador que vive en la PC de la empresa, recibe órdenes por voz o chat, las ejecuta paso a paso, documenta cada acción, se recupera tras reinicios y **solo da una orden por terminada cuando el resultado fue verificado** (o el dueño la confirma/cancela).

La propuesta de negocio (detallada en `PLAN-EMPRESA.md`):
- **Privacidad local como argumento de venta**: los datos no salen de la PC de la empresa.
- **Suscripción por puesto/mes** (planes Lite/Pro/Empresa, USD 50–150).
- Despliegue híbrido: agente local ahora + plano de control en la nube en fases futuras.
- Distribución por canales (consultoras IT) y pilotos directos.

## 2. Estado actual (Fases 0 y 1 completas)

### Fase 0 — Base verificada
- Control real de la PC: abrir/cerrar apps, archivos, red, diagnóstico, mantenimiento, capturas, voz, multimedia.
- Memoria persistente, notas, recordatorios, listas, rutinas.
- **5 roles** (ingeniero, desarrollador, CEO, marketing, humano) + contexto de empresa inyectado en la IA.
- **IA agnóstica y gratuita**: Ollama, Groq, OpenRouter, LM Studio (probe de disponibilidad de 5 s).
- Web UI con chat, TTS, diagnóstico y botones rápidos.
- Bloat eliminado por decisión del dueño: nada de ocio/apps personales; solo lo que trabaja.

### Fase 1 — Núcleo del EMPLEADO (lo nuevo)
1. **Gestor de Órdenes** (`core/ordenes.go`): orden = {objetivo, fecha, estado, historial de acciones, reporte}. Estados: `pendiente → en_progreso → verificando → terminada`, más `bloqueada` y `cancelada`.
2. **El empleado no abandona**: las órdenes persisten en disco; al arrancar, Jarvis anuncia las pendientes y las retoma con "retomá las órdenes".
3. **Bucle de agente con IA** (`core/agente.go`): la IA recibe la orden + un **catálogo de herramientas** y responde con **JSON estricto** (`{"accion":"<comando>","razon":"<por qué>"}` para ejecutar un paso, o `{"fin":"<resumen>"}` para cerrar). El bucle ejecuta, observa el resultado, ajusta (hasta 10 iteraciones) y, al cumplirse, **aprende el procedimiento** para la próxima vez. Si la IA responde JSON malformado, se le re-pide mostrando el error (hasta 3 veces) antes de bloquear la orden.
4. **Procedimientos** (`core/empleado.go`): se aprenden en 1 o 2 turnos ("aprendé que para hacer X: paso 1, paso 2"), se ejecutan, se consultan y se inyectan en el prompt de la IA.
5. **Acciones sensibles rechazadas**: el bucle bloquea y pide aprobación si la IA propone instalar/borrar/formatear/comprar (ej.: "borrar la carpeta de respaldos").
6. **Escritura atómica** (temp + rename) para que un crash no corrompa órdenes/tareas/procedimientos.
7. **Reporte por orden**: qué hizo, en qué orden, qué falló, qué necesita.

## 3. Arquitectura

```
main.go            → 3 modos: consola normal, servicio, web (http://127.0.0.1:8080)
core/              → empleado: ordenes.go, agente.go, empleado.go, tareas, rutinas, memoria,
                     intents.go (clasificador), brain.go, hands.go (catálogo de herramientas), roles, skills
ia/                → conector agnóstico (Ollama/Groq/OpenRouter/LM Studio)
agents/            → subagentes (coder, planificador) — módulo dev, fuera del perfil oficina
config/            → configuración local
webui/             → interfaz web (chat, TTS, botones)
vendor/            → dependencias (sqlite, portaudio, uuid, etc.)
JarvisOS-datos/    → datos del empleado en %USERPROFILE%: config.json, ordenes.json, tareas.json,
                     procedimientos.json, rutinas.json, memoria.json, preferencias.json, empresa.md
```

Reglas del código: paquetes desacoplados, tests por paquete, build + `go vet` + suite de tests verdes en cada checkpoint, datos de la empresa separados de la instalación.

## 4. Tecnología

- Go 1.26, Windows (Win32/COM para control del sistema).
- SQLite (modernc.org), PortAudio (voz), Web UI sin framework pesado.
- Voz: entrada (portaudio) y salida TTS (voces de Windows).
- Persistencia en JSON local (escritura atómica).

## 5. Cómo se prueba

```
go build -o JarvisOS.exe .
go vet ./core/ ./ia/ ./config/ ./webui/ ./agents/
go test ./core/ ./ia/ ./config/ ./webui/ ./agents/
```

Suite verde actual: **core** (17 s, incluye el bucle del agente con una IA falsa de prueba), **ia**, **agents**; config/webui sin tests. También hay smoke tests reales por web (agendar → aprender → cumplir; persistencia tras reinicio).

Ejemplo del bucle con IA (probado con IA falsa):
1. Dueño: "agendá una orden preparar la presentación".
2. Jarvis: Orden #1 registrada, la cumplo y no la cierro hasta terminarla.
3. Bucle: `{"accion":"eco hola","razon":"saludar"}` → resultado → `{"accion":"eco listo"}` → `{"fin":"presentación preparada y verificada"}`.
4. Resultado: orden terminada con reporte + **procedimiento aprendido** (2 pasos) para la próxima.

## 6. Qué sigue (plan por fases)

- **F2 — Confiabilidad B2B**: log de auditoría, roles/permisos por usuario, aprobaciones obligatorias con PIN, panel del dueño en la web. *(condición para vender)*
- **F3 — Integraciones de oficina**: email (Gmail/Outlook), Office por COM (Word/Excel/PPT), calendario, PDF, redes sociales.
- **F4 — Producto y presencia B2B**: marca corporativa, dashboard, onboarding en <15 min.
- **F5 — Propuesta comercial**: documento de venta + demo guionizada + pilotos.

## 7. Puntos abiertos para una opinión externa

1. **Diseño del agent loop**: ✅ migrado a **JSON estricto** (`{"accion":..., "razon":...}` / `{"fin":...}`) con reintento automático ante JSON malformado (hasta 3 veces antes de bloquear). Pendiente de evaluar: tool calling nativo de las APIs y verificación más estricta del resultado.
2. **Seguridad**: el bucle bloquea acciones sensibles por palabra clave; ¿basta o hace falta un modelo de aprobación con PIN por acción (F2)?
3. **Persistencia**: JSON local con escritura atómica vs SQLite para órdenes/historial con volúmenes reales.
4. **IA agnóstica gratuita**: ¿es viable como producto B2B con modelos open-weight locales, o limita la calidad del bucle?
5. **Precios y fases**: ¿es realista la ruta F0→F5 y la monetización por puesto/mes para un MVP?
6. **Arquitectura**: ¿separar `auditoria`, `integraciones`, `web`, `control` en paquetes propios ahora o al llegar F2/F3?
