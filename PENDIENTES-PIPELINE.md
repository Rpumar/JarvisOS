# PENDIENTES-PIPELINE.md — Backlog listo para /pipeline

Tareas reales de JarvisOS para cerrar la ronda v0.15. Cada tarea está
redactada para ejecutarse como una sola invocación de `/pipeline`
(planificación → diseño → implementación → verificación → revisión).

Regla base: toda tarea debe terminar con `go build ./...`, `go vet ./...`
y `go test ./...` en verde.

---

## P0 — Correctos de calidad (bajo riesgo, alto valor)

### T1. Corregir falso positivo del matcheo de "hora" — [hecho]
- Origen: v0.5.0 documenta que el comando "hora" usa `strings.Contains`, así
  que cualquier frase con "ahora" como subcadena lo dispara de más.
- Alcance sugerido: `core/brain.go` / donde se enruta "hora"; usar chequeo de
  palabra completa o el clasificador de intenciones existente.
- Verificación: agregar tests de frases que NO deben disparar
  ("ahora mismo abrí chrome", "¿qué hora era cuando terminó?").

### T2. Recordatorios con fecha (no solo hora de hoy/mañana) — [hecho]
- Origen: PROGRESO "próximos pasos sugeridos"; v0.11 limitó el parseo a
  "a las H" de forma deliberada.
- Alcance: parsear "el [día de la semana]", "el [día] de [mes]"; ajustar al
  próximo día si ya pasó.
- Verificación: `recordatorios_test.go` con casos válidos (día pasado → se
  agenda el próximo), inválidos (debe dar error explícito, no adivinar).

### T3. Buscar dentro de las notas guardadas — [hecho]
- Origen: PROGRESO "próximos pasos sugeridos".
- Alcance: comando "tengo alguna nota sobre [tema]" que busque en las notas
  de `memoria/` y devuelva coincidencias.
- Verificación: tests de búsqueda con/sin coincidencias, mayúsculas
  minúsculas, y que no toque código de ejecución de comandos.

---

## P1 — Robustez (más valor, más cuidado)

### T4. Recordatorios recurrentes — [hecho]
- Origen: PROGRESO "próximos pasos sugeridos".
- Alcance: "todos los días a las 8, recordame la pastilla" — recordatorio que
  al cumplirse se reprograma solo.
- Verificación: test de que marcar cumplido un recordatorio recurrente genera
  el siguiente; sin duplicados en persistencia.

### T5. Lógica de "confirmación de acción peligrosa" auditada
- Origen: PROGRESO/listas de patrones bloqueados documentadas como "defensa
  por substring, no garantía formal".
- Alcance: auditar (`core/hands.go`, `armas*.go`, `core/seguridad*`) que toda
  acción destructiva pase por confirmación y que ninguna lista nueva de
  procesos protegidos quede incompleta.
- Verificación: revisar tests de `TestEsProcesoProtegido` y patrones
  bloqueados; agregar casos si faltan.

---

## P2 — Deuda de verificación (requiere máquina real, NO es tarea de /pipeline)

- Probar de punta a punta en la PC real lo acumulado desde v0.9 (memoria,
  recordatorios, trato personalizado, plano de control, auto-update).
- Medir ruido del micrófono y evaluar modelo Vosk más grande si hace falta.
- Decidir si se publica un ejecutable standalone de esta ronda
  (`build.ps1 -Config release`).

---

## Cómo usar esto

En una terminal de JarvisOS (opencode o Claude Code), parado en la raíz:

```
/pipeline resolver la tarea T1 del archivo PENDIENTES-PIPELINE.md:
corregir el falso positivo del matcheo de "hora" (strings.Contains)
```

Una tarea por invocación. Al terminar, editar este archivo marcando la tarea
como `[hecho]` y actualizar `CHANGELOG.md`.