# PROPUESTA COMERCIAL — JarvisOS, su empleado digital

> **Jarvis — Cumple. No descansa. Rinde cuentas.**
> No vendemos IA. Vendemos un **empleado**: un colaborador digital por puesto
> de trabajo que recibe órdenes, las cumple hasta terminarlas, documenta cada
> paso y no se olvida de nada. Corre **local, en la PC de su empresa**, y los
> datos no salen de su infraestructura.

**Versión del producto:** 0.14.0 — Fases 1-4 completas, F5 (esta propuesta).
**Última actualización:** 2026-08-06.

---

## 1. El problema

Las tareas operativas repetitivas de una pyme consumen horas todos los días:

- Preparar informes, presupuestos y presentaciones a mano, cada semana.
- Redactar y enviar el mismo tipo de correos una y otra vez.
- Completar formularios web repetitivos (cargas, pedidos, planillas).
- Publicar contenido en redes sociales y olvidarse de darle seguimiento.
- Olvidar órdenes a mitad de camino ("lo dejo para después") sin rastro de
  qué se hizo, qué falta y qué falló.

El costo no es solo el tiempo: es la **falta de documentación y
trazabilidad**. Nadie sabe — ni puede auditar — qué se hizo, quién lo hizo y
cuándo.

## 2. La solución

JarvisOS es un **empleado digital por puesto** que vive en la PC de la empresa:

| Capacidad | Qué hace en la práctica |
|---|---|
| **Órdenes que no se abandonan** | "Jarvis, hacé X" → trabaja solo, retoma tras reinicios, y solo cierra cuando se verificó el resultado o el dueño lo confirmó. |
| **Control real de la PC** | Abre/cierra apps, organiza archivos, captura pantalla, diagnostica RAM/CPU/disco/red, todo por voz o web. |
| **Trabajo de oficina** | Email (enviar y leer por SMTP/IMAP), documentos Word/Excel/PowerPoint, agenda y calendario (local y Outlook), PDF. |
| **Redes sociales** | Publica en X y LinkedIn, **solo con aprobación del dueño** y con auditoría de todo. |
| **Formularios web** | Guarda plantillas de formularios repetitivos y las rellena en el navegador cuando se le pide. |
| **Memoria y aprendizaje** | Recuerda hechos, notas y recordatorios; aprende procedimientos que se repiten. |
| **Rinde cuentas** | Todo lo que hace queda en un registro de auditoría inmutable, visible en el panel del dueño. |

### Cómo trabaja

```
Dueño:   "Jarvis, prepará la presentación mensual, mandala por mail y
         publicá el resumen en X."
Jarvis:  [agenda una orden] → [trabaja solo] → [la presentación se genera]
         → [el mail se envía SOLO con su aprobación] → [el post se publica
         SOLO con su aprobación] → [reporta qué hizo, qué falta y qué falló]
```

Cada acción sensible (enviar, publicar, borrar, instalar) **se detiene y pide
aprobación** con el PIN del dueño o desde el panel. No hay sorpresas: el
dueño es quien da el "sí" final.

## 3. Privacidad como ventaja

Este es el argumento de venta principal, y no es marketing:

- **Los datos no salen de la infraestructura de la empresa.** El empleado
  corre 100% local en la PC del puesto. Sin nube, sin telemetría, sin que
  nadie más lea su información.
- **Reconocimiento de voz offline** (Vosk) — el audio tampoco sale de la PC.
- **IA agnóstica y opcional**: puede usar un modelo local (Ollama, gratis) o
  un proveedor Open AI-compatible que la empresa elija y controle.
- La única dependencia opcional de red es la que la empresa decida: enviar
  email, publicar en redes o buscar en internet. Todo el resto es local.

Para empresas con requisitos de confidencialidad (abogados, contadores,
salud, industria), este punto por sí solo diferencia el producto.

## 4. Demo guionizada (3 escenarios reales)

Demo en vivo de **~15 minutos** sobre la PC de un puesto real, con datos del
rubro del cliente. Cada escenario muestra una capacidad distinta y termina
con un resultado verificable en pantalla.

### Escenario 1 — Informe semanal (documentación y trazabilidad)

```
Dueño:  "Jarvis, agendá una orden: prepará el informe semanal de ventas."
Jarvis: "Orden #1 agendada. Voy a trabajar en ella y la retomo sola si
        se reinicia la PC."
        [genera el informe en Word/PDF con los datos reales]
        "Informe generado y guardado en su carpeta de trabajo. Lo dejo
        listo para que lo revise."
```

**Qué ve el cliente:** la orden nació, se trabajó y terminó con verificación;
el panel del dueño muestra la trazabilidad completa (qué se hizo, en qué
orden, con qué resultado) y el archivo generado está en disco.

### Escenario 2 — Email con aprobación (control total)

```
Dueño:  "Jarvis, enviá un email a cliente@empresa.com con asunto
        'Presupuesto' y el texto 'Adjunto el presupuesto solicitado'."
Jarvis: "Esto va a salir de su cuenta a un destinatario externo.
        ¿Lo aprueba, señor? [PIN / botón en el panel]"
Dueño:  [aprueba con su PIN]
Jarvis: "Email enviado. Quedó registrado en la auditoría."
```

**Qué ve el cliente:** ninguna acción externa ocurre sin su aprobación
explícita; el envío queda registrado con fecha, comando y resultado en el
visor de auditoría del panel.

### Escenario 3 — Formulario web repetitivo (plantillas)

```
Dueño:  "Jarvis, creá un formulario factura para https://miplanilla.com."
Jarvis: "Formulario 'factura' creado. Diga 'agregá el campo email con ...'."
Dueño:  "Agregá el campo email con clientes@miempresa.com a factura."
        [...
Dueño:  "Rellená el formulario factura."
Jarvis: "Abrí la página y completé los campos cargados. Verifique y
        apruebe el envío."
```

**Qué ve el cliente:** la carga repetitiva que le toma 10 minutos cada vez se
convierte en dos órdenes de voz; la plantilla queda guardada para siempre.

**Cierre de la demo (1 min):** el panel del dueño muestra, en una sola
pantalla, órdenes activas, actividad de hoy y el informe diario automático —
todo con la auditoría al alcance.

## 5. Piloto

- **Alcance:** 1 puesto, 30 días, con acompañamiento del instalador.
- **Objetivo:** que el dueño dé sus primeras órdenes reales y mida el ahorro.
- **Métricas de ROI que se capturan desde el día 1:**
  - Horas ahorradas por semana (tareas que antes hacía a mano y ahora Jarvis).
  - Tareas terminadas por semana (contador real de órdenes cerradas).
  - Acciones aprobadas/rechazadas (control del dueño sobre lo sensible).
  - Informes y correos generados con trazabilidad completa.
- **Salida del piloto:** informe de 30 días con los números; decisión de
  ampliar a más puestos con datos, no con promesas.

## 6. Precios (por puesto / mes)

| Plan | Para quién | Incluye | Precio objetivo |
|---|---|---|---|
| **Lite** | El dueño que trabaja solo | Órdenes persistentes, control de PC, memoria, agenda, formularios web | USD 50 |
| **Pro** | Puesto de oficina | Todo lo de Lite + email, Office (Word/Excel/PPT), PDF, Outlook | USD 90 |
| **Empresa** | Equipos multi-puesto | Todo lo de Pro + multiusuario con perfiles (dueño/admin/empleado), auditoría completa, soporte con canal dedicado | USD 150 |

*Precios orientativos por puesto/mes para el plan base; ajustar a la
estrategia del canal. Todos los planes incluyen la primera instalación
guiada y la configuración inicial (onboarding en menos de 15 minutos).*

## 7. Soporte

- **Instalación guiada:** un instalador (canal/consultora) instala, configura
  el perfil de la empresa y al dueño, y valida la primera orden en la
  primera visita.
- **Capacitación al dueño:** 30-60 minutos para dominar las órdenes
  principales (órdenes, email, office, formularios, panel del dueño).
- **Canal de soporte:** incluido en el plan Empresa (email/WhatsApp con
  SLA definido); respuestas de uso común para Lite y Pro vía documentación.

---

## Anexo — Lo que JarvisOS ya hace hoy (verificado)

Todo lo de esta propuesta corresponde a capacidades **ya implementadas y
testeadas** en el producto (v0.14.0), no a promesas de hoja de ruta:

- ✅ Órdenes persistentes con retoma automática y estados verificables.
- ✅ Email real probado en vivo (SMTP envía, IMAP lee, con cuenta real).
- ✅ Office real probado en vivo (Word/Excel/PowerPoint por COM).
- ✅ Agenda local + sincronización con Outlook.
- ✅ PDF (generador puro en Go + conversión Office→PDF).
- ✅ Redes sociales (X con OAuth 1.0a sobre stdlib, LinkedIn) con aprobación.
- ✅ Formularios web con plantillas y autocompletado.
- ✅ Perfiles de usuario (dueño/admin/empleado) y roles en el panel.
- ✅ Auditoría inmutable + aprobaciones obligatorias con PIN/panel.
- ✅ Panel del dueño en la web + informe diario automático.
- ✅ Onboarding guiado de primer arranque (< 15 minutos).
- ✅ Suite de tests verde: build/vet/test limpios.

**Próximo paso:** coordinar la demo guionizada con datos del rubro del
cliente. Preparación: 1 día; duración: 15 minutos en vivo.
