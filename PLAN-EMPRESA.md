# PLAN: De JarvisOS a EMPLEADO EMPRESARIAL TOTAL

> No vendemos IA. Vendemos un **empleado**.
> El dueño de la empresa da una orden; Jarvis la cumple y **no la cierra hasta terminarla**.

---

## Principios rectores

1. **Un empleado, no un software.** El producto es Jarvis: un colaborador digital por puesto de trabajo. La IA es una herramienta interna, no el producto.
2. **Persistencia de órdenes.** Una orden no se da por terminada hasta que se ejecutó y se verificó el resultado (o el dueño la canceló).
3. **Local y privado por defecto.** Los datos no salen de la PC de la empresa. Es el argumento de venta principal.
4. **Lite y funcional.** Solo lo que trabaja. Nada de ocio; módulos de desarrollo fuera del core.
5. **Auditable y controlado.** Todo lo que hace queda registrado y requiere aprobación cuando corresponde.

---

## DECISIONES CLAVE (negocio serio, estable y escalable)

1. **Despliegue: híbrido agente-local + plano de control en la nube.**
   - El agente corre en la PC de la empresa (privacidad local = argumento de venta).
   - Un plano de control en la nube (agregable en F5+) gestiona licencias, puestos, actualizaciones y métricas opcionales.
   - **Ahora**: activación por clave de licencia local (sin servidor). El diseño ya separa datos de empresa (`JarvisOS-datos`) para migrarlos intactos.
2. **Monetización: suscripción por puesto/mes, 3 planes** (Lite / Pro / Empresa). Precio objetivo inicial: USD 50-150 por puesto/mes según plan.
3. **Arquitectura estable:**
   - Paquetes desacoplados: `core` (empleado), `auditoria`, `integraciones`, `web`, `control`.
   - Escritura de datos atómica (temp + rename) para que un crash no corrompa memoria/tareas/órdenes.
   - Backups automáticos de `JarvisOS-datos` con rotación (al arrancar). **Hecho**: `core/backups.go`, se respalda al arrancar (consola y servicio) y por voz (`hacé un respaldo`, `qué respaldos tengo`); rota a 7 backups.
   - Versionado semántico + CHANGELOG. **Hecho**: `CHANGELOG.md` (formato Keep a Changelog) con el historial v0.7 → v0.14.
   - Recuperación ante crash: órdenes pendientes se retoman (nunca se abandonan).
4. **Escalabilidad comercial:**
   - Clave de licencia por instalación con límite de puestos. **Hecho**: `core/licencia.go` (claves `JARVIS-PLAN-PUESTOS-NONCE-FIRMA` firmadas con HMAC, planes lite/pro/empresa, validación al arranque, comando por voz "activá la licencia ..." / "qué licencia tengo", límite de puestos al registrar usuarios desde voz y panel).
   - Distribución por **canales** (consultoras IT que instalan/mantienen) + pilotos directos.
   - Material de venta (propuesta + demo guionizada) en F5.
5. **Legal y soporte:** EULA + política de privacidad (datos locales) + facturación definida con el dueño; canal de soporte en plan Empresa.

---

## FASE 0 — Base (estado actual)

Ya disponible y verificado:
- Control real de la PC: apps, archivos, red, diagnóstico, mantenimiento, capturas, voz.
- Memoria persistente, notas, recordatorios, listas, rutinas.
- Tareas (agendar/listar/marcar/borrar), procedimientos aprendidos con consulta cuando no sabe.
- 4 roles (CEO, marketing, humano, asistente corporativo) + contexto de empresa.
- IA agnóstica y gratis: Ollama, Groq, OpenRouter, LM Studio.
- Web UI con chat, TTS, diagnóstico y botones rápidos.
- Suite de tests verde, build/vet limpios, commit base `60c9d17`.

**Cierre de fase:** arrancar y operar por voz/web todas las capacidades sin errores.

---

## FASE 1 — Núcleo del EMPLEADO (órdenes que no se abandonan)

Objetivo: pasar de "catálogo de comandos" a **ejecutor persistente de órdenes**.

1. **Gestor de Órdenes** (`core/ordenes.go`):
   - Orden = { objetivo, fecha, estado (pendiente/en_progreso/verificando/terminada/bloqueada), historial de acciones, reporte final }.
   - La orden queda en `pendiente` y el sistema la retoma sola (bucle cada X minutos y al arrancar).
   - Solo pasa a `terminada` cuando se **verifica** el resultado o el **dueño confirma**.
2. **Bucle de trabajo (agent loop)**:
   - La IA recibe la orden y el catálogo de herramientas → decide la siguiente acción → la ejecuta → observa el resultado → ajusta → reporta.
   - Si se bloquea (falla repetida), pasa a `bloqueada` y le pregunta al dueño cómo seguir (consulta, no adivina).
3. **Procedimientos reales**: si una orden requiere un paso que no es un comando soportado, el bucle lo resuelve con IA/herramientas y **lo aprende** para la próxima.
4. **Recuperación**: al arrancar, Jarvis retoma todas las órdenes en `pendiente/en_progreso` y las sigue (no se olvida).
5. **Reporte por orden**: al terminar (o al pedirlo), Jarvis entrega un resumen: qué hizo, en qué orden, qué falló, qué necesita.

**Criterio de salida:** "Dueño: hacé X" → Jarvis trabaja solo, retoma tras reinicios, y solo cierra con verificación o confirmación. Demo con 3 órdenes reales.

---

## FASE 2 — Confiabilidad y control B2B

Objetivo: lo que una empresa exige antes de comprar.

1. **Log de auditoría** (`core/auditoria.go`, JSON + exportable):
   - Quién dio la orden, qué orden, cada acción ejecutada, con timestamp y resultado.
   - Acciones sensibles marcadas; visor en la web (solo rol dueño/admin).
2. **Roles y permisos por usuario**:
   - Perfiles: `dueño` (todo, aprueba), `admin` (opera + configura), `empleado` (da órdenes de su área, no borra auditoría).
   - Autenticación local (PIN/contraseña de la instalación) + selección de perfil.
3. **Aprobaciones obligatorias**: borrar/instalar/formatear/enviar por email/redes → piden confirmación con PIN del dueño (o aprobación desde el panel).
4. **Panel del dueño** en la web: órdenes activas, pendientes de hoy, actividad reciente, alertas, informe diario automático.

**Criterio de salida:** un tercero puede auditar qué hizo Jarvis el día X, y ninguna acción sensible ocurre sin aprobación.

---

## FASE 3 — Integraciones de oficina

Objetivo: que el empleado haga el trabajo de oficina real.

1. **Email**: Gmail + Outlook (API oficial o SMTP/IMAP con credenciales locales). Leer/resumir/redactar/enviar con aprobación.
2. **Office**: Word, Excel y PowerPoint por COM (automation). Generar informes, presupuestos, presentaciones y datos en tablas.
3. **Calendario**: agenda del dueño, agendar/reuniones, recordatorios con antelación.
4. **PDF/archivos**: unir, separar, convertir a texto, generar PDFs desde documentos.
5. **Redes sociales**: programar/publicar contenido (con aprobación del dueño) en las plataformas que la empresa use.
6. **Formularios/web**: rellenar procesos web repetitivos con la navegación automatizada.

**Criterio de salida:** "Jarvis, prepará la presentación mensual, mandala por mail y publicá el resumen" → lo hace y reporta (con aprobaciones donde corresponda).

---

## FASE 4 — Producto y presencia B2B

1. **Marca**: "Jarvis — su empleado digital. Cumple, no descansa, rinde cuentas." Eliminar la estética de asistente personal.
2. **Web UI corporativa**: dashboard del dueño (órdenes, pendientes, actividad, aprobaciones), onboarding guiado (primera configuración: perfil de empresa, dueño, aprobaciones).
3. **Perfil de empresa ampliado**: la `empresa.md` actual pasa a ser un perfil estructurado (rubro, productos, clientes, tono, redes, datos de contacto).
4. **Vigilante**: el modo de vigilancia de la PC queda como paquete opcional, fuera del perfil oficina (lite).

**Criterio de salida:** una empresa nueva puede instalar, configurar a su dueño y dar su primera orden en menos de 15 minutos.

**Estado:** ✅ completa. Marca corporativa aplicada (webui, onboarding en primer
arranque), dashboard del dueño con aprobaciones y auditoría, perfil de empresa
estructurado (`empresa.json`) y modo vigilante disponible como intención aparte.

---

## FASE 5 — Propuesta comercial para venderle a la empresa

Documento de venta (`PROPUESTA-COMERCIAL.md`):
1. **Problema**: tareas operativas repetitivas consumen horas; falta documentación y trazabilidad.
2. **Solución**: un empleado digital por puesto que recibe órdenes, cumple, documenta y rinde cuentas; local y privado.
3. **Demo guionizada**: 3 escenarios reales del rubro del cliente (informe, email, presentación).
4. **Piloto**: 1 puesto, 30 días, métricas de ROI (horas ahorradas por semana, tareas terminadas).
5. **Precios**: Lite (por puesto) / Pro (integraciones de oficina) / Empresa (multiusuario + auditoría + soporte).
6. **Privacidad como ventaja**: nada sale de la infraestructura de la empresa.
7. **Soporte**: instalación guiada, capacitación al dueño, canal de soporte.

**Criterio de salida:** documento listo para presentar + demo funcional en vivo.

**Estado:** ✅ en curso. `PROPUESTA-COMERCIAL.md` completo (problema, solución,
privacidad, demo guionizada de 3 escenarios, piloto con ROI, precios y soporte).
Demo funcional en vivo: son las capacidades ya implementadas (F1-F4).
Completado: `demo.ps1` (demo guiada con sandbox de datos vía `JARVISOS_DATOS`)
y el **informe de cierre de piloto** por voz ("informe del piloto") con horas
ahorradas y aprobadas/denegadas/expiradas.

---

## Orden de trabajo sugerido

1. Fase 1 (núcleo empleado) — el cambio de producto más importante.
2. Fase 2 (auditoría + permisos) — condición para vender.
3. Fase 3 (integraciones) — diferenciación real.
4. Fase 4 y 5 (marca + propuesta) — comercialización.

Cada fase termina con build + tests verdes y una demo operable.
