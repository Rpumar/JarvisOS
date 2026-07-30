# JarvisOS — Progreso

**Versión actual:** 0.13.0
**Estado:** ✅ Confirmado funcionando en Windows real (detalle abajo, ver
"Validación real" — corresponde a v0.8.0; **todo de v0.9.0 en adelante
(memoria, recordatorios, trato personalizado, seguridad, y esta ronda de
comandos nuevos) todavía no se probó en la máquina real** — a pedido
explícito del usuario, se construyó todo junto y se prueba en una sola
tanda cuando llegue a su máquina, no en cada paso intermedio).

## Roadmap de mejoras (pedido del usuario, en fases)

Las 4 fases originales están completas (Personalidad, Memoria, Funciones,
Seguridad). Esta versión es una ronda adicional de comandos, pedida aparte.

## v0.12.0 → v0.13.0: más comandos y funciones

**Recordatorios, ampliado:**
- `poné un timer de [X] minutos/segundos/horas` — duración relativa
  (`core/recordatorios.go`, `extraerTimer`), reutiliza la misma
  infraestructura de recordatorios de la Fase 3 (mismo vigía en segundo
  plano, mismo `Almacen`)
- `qué recordatorios tengo` — lista los pendientes con su hora (distinto de
  "qué recordás", que lista notas)
- `cancelá el recordatorio de [algo]` / `cancelá todos los recordatorios`

**Navegación:**
- `ir a [sitio]` / `andá a [sitio]` — abre una URL directa, sin pasar por
  el buscador (completa ".com" y "https://" si hace falta)

**Apps:** Word, Excel, PowerPoint sumados al catálogo de `abrirApp`.

**Conversación:** `qué dijiste` / `repetí eso` — repite la última respuesta
hablada de la sesión. Nuevo campo `ultimaRespuesta` en `Brain`, actualizado
en `Process()` después de la personalización (así repite exactamente lo
que se dijo, no el texto sin personalizar).

**Memoria persistente, dos métodos nuevos** (`memoria/memoria.go`):
`ObtenerRecordatoriosPendientesTexto` (todos los pendientes, no solo los
vencidos como `RecordatoriosPendientes`) y `CancelarRecordatorios` (por
texto de búsqueda, o todos si viene vacío). Ambos se sumaron a la interfaz
`core.MemoriaPersistente`.

**Orden de enrutamiento, por qué importa:** timer y "cancelar recordatorio"
se chequean ANTES que el recordatorio-con-hora y el "recordar" genérico,
porque comparten prefijos ("avisame en..." vs "avisame a las...") y sin ese
orden terminarían mal clasificados. Documentado en los comentarios de
`core/memoria_sesion.go`, no es obvio con solo mirar el código.

**Tests:** ampliación de `memoria_test.go`, `recordatorios_test.go`
(`extraerTimer`, casos válidos e inválidos) y `brain_test.go` (timer con
prioridad sobre recordatorio-con-hora, cancelar específico y "todos",
listar pendientes, "qué dijiste" con y sin historia previa).

## v0.11.0 → v0.12.0: Fase 4, repaso de seguridad

Repaso real del código exec/PowerShell existente, no un trámite — esto es
lo que salió:

**1. Hallazgo y arreglo: `cerrarApp` sin ninguna protección.**
Antes de esta ronda, "cerrar [lo que sea]" le pasaba el nombre directo a
`taskkill /F`, sin distinguir entre Chrome y un proceso crítico de Windows.
Un "cerrar explorer" mal reconocido por voz (o pedido literal sin pensarlo)
hacía exactamente lo mismo que cerrar cualquier app. Se agregó
`procesosProtegidos` en `core/hands.go`: una lista de procesos del sistema
(`winlogon.exe`, `csrss.exe`, `lsass.exe`, `explorer.exe`, etc.) que
`cerrarApp` se niega a matar, con la razón explicada al usuario en vez de
fallar en silencio. Con test dedicado (`TestEsProcesoProtegido`).

**2. Mismo hallazgo, mismo arreglo, en CoderAgent.**
El mismo riesgo existía por otra puerta: un script generado por IA podría
matar esos mismos procesos vía `Stop-Process` o `taskkill` sin pasar por
`Hands` en absoluto. Se agregaron los patrones equivalentes a
`patronesBloqueados` en `agents/coder_agent.go`, con tests que confirman
tanto que se bloquean los procesos críticos como que **no** se bloquea de
más (`Stop-Process -Name notepad` sigue permitido).

**3. Robustez: el vigía de recordatorios ya no puede tumbar todo el proceso.**
En Go, un panic sin recuperar en cualquier goroutine mata TODO el proceso,
no solo esa goroutine — un error inesperado en `vigilarRecordatorios`
(Fase 3) se hubiera llevado puesto el micrófono y la voz también. Se
agregó `recover()` en `main.go`: como mucho, el vigía deja de revisar
recordatorios; el resto de JarvisOS sigue andando.

**4. Verificado (sin necesidad de cambios) — queda documentado por qué:**
- `Hablar` y `copiarAlPortapapeles` escapan comillas simples doblándolas
  (`''`) antes de insertarlas en un string de PowerShell con comillas
  simples — es el escape correcto para ese contexto, y dentro de un string
  de comillas simples de PowerShell casi todo lo demás se trata literal.
- Las búsquedas (`buscarEnGoogle`/`Youtube`/`Wikipedia`) pasan la consulta
  por `url.QueryEscape` antes de armar la URL — los caracteres especiales
  de shell quedan codificados como `%XX` antes de llegar a `cmd /C start`,
  no como texto crudo.
- `abrirApp`/`abrirCarpeta` solo usan valores de un mapa fijo en el código
  o nombres de carpeta hardcodeados — el texto que dice el usuario nunca
  llega directo a la línea de comandos en esos casos.
- La decisión de guardar la memoria persistente sin cifrar sigue en pie:
  ahora guarda más datos (cumpleaños, trabajo, recordatorios) que en la v0.10.0
  original, pero sigue sin ser información sensible en el sentido que
  importa (passwords, datos financieros). Se re-confirma acá explícitamente,
  no se da por sentado sin más.

**Lo que esta revisión NO cubrió** (para ser honesto sobre el alcance):
no se auditó `ears.go`/`tts` en profundidad (no hay superficie de
inyección ahí, es solo captura/síntesis de audio), y las listas de
patrones bloqueados siguen siendo defensa en profundidad por substring,
no una garantía formal — coincidencia textual, no análisis del AST del
script generado.

## v0.10.0 → v0.11.0: Fase 3, tres piezas

### 1. Trato personalizado
Si Jarvis sabe tu nombre (por "me llamo X" o "llamame X"), te llama por tu
nombre en TODAS las respuestas en vez de "señor" — no solo el saludo.

**Cómo se implementó sin tocar las decenas de strings que dicen "señor" en
`hands.go`/`personalidad.go`:** un solo reemplazo centralizado. `Process()`
se separó en `procesarInterno()` (la lógica de siempre, sin tocar) +
`Process()` (wrapper que le aplica `personalizarRespuesta`, un
`strings.ReplaceAll("señor", nombre)` al final). `Saludar()`/`Despedirse()`
son métodos nuevos de `Brain` que hacen lo mismo con `Saludo()`/`Despedida()`
de la Fase 1; `main.go` ahora los llama a través de `Brain` en vez de usar
`core.Saludo()`/`core.Despedida()` directo, para que también se beneficien.

### 2. Más hechos reconocidos
Se sumaron `cumpleaños` y `trabajo` a los patrones de `me llamo`/`vivo en`
que ya existían (`core/memoria_sesion.go`), y `llamame X`/`decime X` como
formas adicionales de fijar el nombre (útil para elegir un apodo, no solo
tu nombre real). Preguntas nuevas: `cuándo es mi cumpleaños`, `dónde trabajo`.

### 3. Recordatorios con hora real (la pieza más grande)
`recordame llamar a mamá a las 5 de la tarde` programa un aviso; JARVIS
**habla solo**, sin que se le pregunte, cuando llega la hora. Es la primera
vez que JarvisOS deja de ser puramente reactivo.

- **`core/recordatorios.go`** (nuevo): parseo de "a las H", "a las H:MM",
  "de la mañana/tarde/noche". Alcance limitado a propósito: NO entiende
  fechas ("el jueves") ni tiempos relativos ("en una hora") — si no
  reconoce el patrón, falla explícito en vez de adivinar mal.
- **`memoria/memoria.go`**: se agregó `Recordatorio` (con ID único) y
  `AgregarRecordatorio`/`RecordatoriosPendientes`/`MarcarCumplido`.
- **Concurrencia real, no teórica:** a partir de acá, dos goroutines tocan
  `Almacen` al mismo tiempo (el flujo normal de comandos + el vigía en
  segundo plano de `main.go`). El comentario viejo que decía "no hace
  falta mutex, todo pasa en una sola goroutine" dejó de ser cierto, así que
  se agregó un `sync.Mutex` real protegiendo todos los métodos, no solo se
  actualizó el comentario.
- **`main.go`**: nueva goroutine `vigilarRecordatorios`, un ticker cada 30
  segundos. Es la única parte de JarvisOS que le habla al usuario sin que
  lo hayan invocado primero. No tiene apagado prolijo (no escucha
  cancelación) — muere con el proceso al cerrar, y para esta versión
  alcanza con eso.

**Tests:** ampliación de `memoria_test.go` (incluye un test de acceso
concurrente real con goroutines, pensado para correr con `go test -race`),
`core/recordatorios_test.go` nuevo (parseo de hora, casos válidos e
inválidos, ajuste al día siguiente), y ampliación de `brain_test.go` (trato
personalizado, hechos nuevos, prioridad de recordatorio-con-hora sobre
nota-libre).

## v0.9.0 → v0.10.0: Fase 2, memoria

**Dos tipos de memoria, con roles distintos:**

**A. Memoria de sesión** (en RAM, se resetea al reiniciar) — vive como
campos de `Brain` (`core/memoria_sesion.go`):
- `ultimaApp` / `ultimaBusqueda`: permiten "cerralo" (cierra la última app
  abierta) y "lo mismo en youtube" (repite la última búsqueda, pero ahí)
- `historialIA`: últimos 5 turnos con el respaldo conversacional, para que
  "¿y qué más hizo?" tenga sentido después de "¿quién fue Einstein?"

**B. Memoria persistente** (sobrevive a reiniciar) — paquete nuevo
`memoria/`, un archivo JSON en `%USERPROFILE%\JarvisOS-datos\memoria.json`:
- Hechos estructurados (`nombre`, `ciudad`) reconocidos por patrón:
  "recordá que me llamo X" / "vivo en X"
- Notas libres para cualquier otra cosa: "recordá que tengo reunión el jueves"
- Comandos nuevos: `recordá que...`, `cuál es mi nombre`, `dónde vivo`,
  `qué recordás`

**Refactor que acompañó esto:** `NewBrain` iba a quedar con 4 parámetros
posicionales (Hands, IA, Coder, Memoria). Se reemplazó por un struct
`BrainOpciones{IA, Coder, Memoria}`, para no seguir creciendo la firma en
cada fase futura. También se separaron las interfaces de `brain.go` a
`core/interfaces.go` (Brain ya hacía demasiadas cosas en un solo archivo).

**Decisión de seguridad, ya conversada y aceptada:** el JSON de memoria
persistente queda en texto plano, sin cifrar. Documentado explícitamente en
`memoria/memoria.go` como decisión consciente, no como descuido — no es el
lugar para nada sensible si en algún momento se quisiera guardar algo así.

**Tests:** `memoria/memoria_test.go` (incluye el test más importante de
este paquete: que los datos sobrevivan de verdad a crear un `Almacen` nuevo
apuntando al mismo archivo, no solo que funcionen en memoria durante el
test) + ampliación de `core/brain_test.go` con mocks de memoria y casos de
resolución de pronombres.

## Validación real (en la máquina del usuario, fuera de esta sesión)

Camino recorrido hasta que anduvo, para que quede como referencia:
1. `go run go main .` (typo, no es sintaxis válida) → corregido a `go run .`
2. `core/hands.go` local desactualizado (faltaban `formatearHora`,
   `formatearFecha`, `ComandoNoReconocido`) → resuelto reemplazando
   `core/hands.go`, `core/brain.go`, `core/ears.go` con la versión vigente
3. Dependencias no descargadas (`no required module provides package...`)
   → resuelto con `go mod tidy`
4. A partir de ahí: compiló y corrió. Micrófono, voz, y comandos
   funcionando de verdad, no en teoría.

No hubo reporte del error de `-mthreads` anticipado en `INSTALACION.md`
(Paso 4) — puede que no haya aparecido en esta combinación específica de
versiones. Se deja la guía como está, ya que anticiparlo no hizo daño y
puede ayudar a alguien más.

**Esta validación es de v0.8.0.** Todo lo de v0.9.0 en adelante (personalidad,
memoria, recordatorios) está escrito y revisado a mano, pero no probado en
la máquina real todavía — queda pendiente para cuando el usuario decida
probar todo junto.

**Qué cambió:** las respuestas de los momentos más frecuentes dejaron de
ser una única línea fija y ahora se eligen al azar entre variantes con más
carácter — tono seguro, cálido, con ingenio seco (inspirado en el
*arquetipo* de mayordomo-IA competente, sin copiar líneas de ninguna
película puntual).

**Archivo nuevo:** `core/personalidad.go` con 4 bancos de frases:
`Saludo()`, `Despedida()`, `RespuestaConfusion()`, `ConfirmacionGenerica()`.
Se conectaron en los puntos de mayor frecuencia real de uso:
- Saludo al activarse con "jarvis" (`main.go`)
- Despedida al decir "apagar" (`main.go`)
- Cuando ningún comando coincide (`core/brain.go`)
- Confirmación de `abrir [app]`, el comando más repetido en la práctica
  (`core/hands.go`)

**Por qué no se tocó TODO `hands.go`:** se buscó el mayor impacto percibido
con el menor riesgo — variar los 4 puntos de mayor frecuencia, no las ~20
funciones de `hands.go` una por una. Eso queda como posible ajuste menor a
futuro si se quiere, pero no es necesario para que el cambio se note.

**Respaldo de IA:** el prompt de sistema en `ia/conector.go` también se
actualizó para pedir explícitamente ese mismo tono cuando responde de forma
conversacional (solo aplica si hay `OPENAI_API_KEY` configurada).

**Tests:** `core/personalidad_test.go` — verifica que cada función siempre
devuelva una frase perteneciente a su banco (corrido 50 veces por función,
por ser aleatorio).

## v0.7.0 → v0.8.0: guía de instalación
GCC (para CGO), PortAudio vía `pacman`, descarga manual de Vosk + modelo en
español, variables de entorno, compilación, y una tabla de troubleshooting.

Vale la pena destacar un hallazgo concreto de la investigación: hay un error
**conocido y documentado** (`invalid flag in pkg-config --cflags: -mthreads`)
que aparece casi siempre al combinar `gordonklaus/portaudio` con MSYS2 en
Windows — es un choque entre lo que devuelve `pkg-config` y la lista de
banderas que Go permite por seguridad. Se lo incluí ya resuelto en la guía
(variable `CGO_CFLAGS_ALLOW`) para que no te encuentres con eso a ciegas.

## v0.6.0 → v0.7.0: tests

**Cobertura, con honestidad sobre los límites:**

| Archivo | ¿Testeado? | Por qué |
|---|---|---|
| `core/brain_test.go` | Sí, a fondo | Enrutamiento completo de `Process()` con mocks (comando local, respaldo IA, IA no disponible/con error, disparo de CoderAgent, propuesta pendiente confirmar/cancelar/ambigua), más `esPeticionDeCodigo` |
| `agents/coder_agent_test.go` | Sí, a fondo | El flujo Proponer/Confirmar/Cancelar completo con un generador falso, y sobre todo `patronPeligrosoEn` — la pieza de seguridad más crítica del proyecto |
| `ia/conector_test.go` | Sí | `parsearRespuestaCodigo` (todos los casos borde) y `Disponible()`/timeouts, sin llamadas de red reales |
| `core/hands_test.go` | Parcial | Solo `formatearHora`/`formatearFecha` (lógica pura, separada de `time.Now()` para poder testearla) |
| `core/hands.go` (el resto) | **No** | `abrirApp`, `cerrarApp`, `volumen`, `capturarPantalla`, `buscarEnGoogle`, etc. llaman directo a `exec.Command`/PowerShell. Son wrappers finos sobre Windows — un test unitario no prueba nada real ahí, hace falta Windows de verdad |
| `core/ears.go` | **No** | Requiere micrófono y PortAudio/Vosk reales |
| `main.go` | **No** | Es el punto de entrada que conecta todo; se prueba de forma más natural corriendo el programa |

**Refactor chico para permitir los tests de Brain:** `Brain` dependía de
`*Hands` (tipo concreto). Ahora depende de una interfaz nueva,
`EjecutorComandos` (`RunCommand` + `Hablar`), así que en los tests se le
puede inyectar una implementación falsa sin tocar Windows. `*Hands` ya
cumple esa interfaz sin ningún cambio — mismo patrón que ya se usaba con
`ConectorIA` y `AgenteDeCodigo`. `main.go` no necesitó cambios.

**Cómo correrlos** (necesita Go instalado, cosa que yo no tengo en este
entorno — ver Limitaciones):
```
go test ./...          # todos los tests
go test ./... -v       # con detalle de cada caso
go test ./agents/...   # solo el paquete de CoderAgent
```

**Por qué vale la pena esta cobertura en particular:** no es "cobertura por
cobertura". `TestPatronPeligrosoEn` es probablemente el test más importante
de todo el proyecto: si alguien toca `patronesBloqueados` en el futuro y
rompe algo sin querer, este test lo va a agarrar antes de que llegue a
ejecutarse un script real. Lo mismo con el enrutamiento de `Brain`: es la
pieza que decide qué se ejecuta y qué no.

## v0.5.0 → v0.6.0: CoderAgent implementado

`agents/coder_agent.go` pasó de ser un cimiento vacío a un agente funcional
que genera y ejecuta scripts de PowerShell a pedido, vía IA.

**Flujo de voz:**
```
Usuario:  "Jarvis, escribí un script que liste los archivos más grandes
           de mi carpeta de descargas"
JARVIS:   "El script lista los 10 archivos más grandes de Descargas
           ordenados por tamaño. Lo guardé en
           C:\Users\...\JarvisOS-scripts\script_1234567890.ps1 por si
           quiere revisarlo antes. ¿Confirmo y lo ejecuto, señor?
           Diga 'confirmar' o 'cancelar'."
Usuario:  "confirmar"
JARVIS:   "Listo, señor. Resultado: ..."
```

**Diseño de seguridad (no es un extra, es el núcleo del diseño):**
1. **Nunca ejecución automática.** `Proponer()` genera el script y lo
   guarda en disco, pero jamás lo corre. Solo `Confirmar()` ejecuta, y
   requiere una frase de voz explícita en un turno aparte.
2. **Mientras hay una propuesta pendiente, `Brain` no procesa otra cosa** —
   todo se interpreta como confirmar/cancelar hasta resolverla. Evita dejar
   una ejecución ambigua "en el aire".
3. **Bloqueo de patrones peligrosos** (`agents/coder_agent.go`,
   `patronesBloqueados`): si el script generado incluye formatear un disco,
   apagar el equipo, borrar carpetas amplias del sistema, deshabilitar
   Windows Defender, o descargar/ejecutar algo de internet, JarvisOS se
   niega a proponerlo — ni siquiera pide confirmación.
4. **El prompt al modelo** (`ia/conector.go`, `promptSistemaCodigo`) le
   ordena negarse (devolver código vacío) ante pedidos riesgosos o
   ambiguos, en vez de intentar cumplirlos "de forma más segura" por su
   cuenta.
5. **`config.RequireApproval` NO controla nada de esto.** La confirmación
   es obligatoria siempre, fija en el código, no una opción que se pueda
   desactivar. Se documentó así explícitamente en `config/config.go` para
   que no quede la falsa impresión de que ese flag es un interruptor.

**Honestidad sobre los límites de esto:** el bloqueo de patrones es
defensa en profundidad, no una garantía completa — coincide por substring,
así que una frase suficientemente distinta podría evadirlo. La protección
real es la confirmación humana. Tampoco hay timeout en la ejecución: un
script generado que se cuelgue bloquearía a JARVIS hasta matarlo manualmente
desde el Administrador de tareas (queda como TODO).

Los scripts, ejecutados o cancelados, quedan guardados en
`%USERPROFILE%\JarvisOS-scripts\` como rastro de auditoría.

**Nuevos comandos de voz:** `escribe un script que...` / `escribime un
script que...` / `crea un script que...` / `generame un script que...` /
`hazme un script que...`, más `confirmar` / `cancelar` cuando hay una
propuesta pendiente.

**Cambio de firma:** `core.NewBrain` ahora recibe un tercer parámetro
(`coder AgenteDeCodigo`). Ya actualizado en `main.go`.

## v0.4.0 → v0.5.0: más comandos + bug fix

**Bug encontrado y corregido:** el control de volumen (`subir/bajar volumen`,
`silencio`) usaba `WScript.Shell.SendKeys('{volume_up}')`. Ese código no es
válido — el lenguaje de SendKeys no cubre teclas multimedia, solo un teclado
estándar de 101 teclas. Se reemplazó por `keybd_event` (P/Invoke a
`user32.dll` desde PowerShell) con los códigos de tecla virtual reales de
Windows (`VK_VOLUME_UP` = 0xAF, etc., verificados contra la documentación
oficial de winuser.h). El mismo mecanismo corregido ahora también maneja
play/pausa y cambio de pista.

**10 comandos nuevos** en `core/hands.go`:

| Comando de voz | Acción |
|---|---|
| `pausar` / `reproducir` | Play/pausa de música (tecla de alternancia) |
| `siguiente canción` | Siguiente pista |
| `canción anterior` | Pista anterior |
| `copiar [texto]` | Copia texto al portapapeles |
| `cuéntame un chiste` | Chiste al azar (lista local, no necesita IA) |
| `batería` | Nivel de batería (WMI) |
| `cuál es mi ip` | IP local (sin PowerShell, con `net` de Go) |
| `abrir papelera` | Abre la papelera de reciclaje |
| `abrir documentos` | Abre la carpeta Documentos |
| `buscar en wikipedia [algo]` | Busca en Wikipedia en español |

No se agregaron dependencias nuevas a `go.mod`: todo usa la librería
estándar de Go (`net`, `math/rand`) o PowerShell, igual que el resto del
proyecto.

**Limitación conocida, sin resolver a propósito:** el matcheo de `"hora"` es
un `strings.Contains` simple, así que technically coincidiría con cualquier
frase que contenga "ahora" como subcadena. Viene de la versión original, no
es nuevo de esta ronda, y no se tocó para no meterse con lógica que no se
pidió tocar. Si se quiere, se puede ajustar a un chequeo de palabra completa.

## Estructura del proyecto

```
JarvisOS/
├── go.mod
├── main.go                 # arranca todo, loop principal de escucha
├── PROGRESO_JARVISOS.md     # este archivo
├── core/
│   ├── ears.go              # micrófono real (PortAudio) + STT (Vosk)
│   ├── hands.go              # ejecuta acciones en Windows (apps, volumen, etc.)
│   └── brain.go              # orquestador: comando local -> si no, IA de respaldo
├── config/
│   └── config.go             # configuración centralizada
├── ia/
│   └── conector.go           # conector de respaldo a OpenAI (opcional)
└── agents/
    └── coder_agent.go        # cimiento reservado para un futuro agente de código
```

## Qué se hizo en esta ronda (v0.3 → v0.4)

### Tarea 1 — Micrófono real
`core/ears.go` se reescribió por completo. La versión anterior usaba
`System.Speech.Recognition.SpeechRecognitionEngine` de Windows vía
PowerShell, pero sin una gramática (`Grammar`/`DictationGrammar`) cargada
ese motor no reconoce de forma confiable — de ahí que "faltara micrófono
real" aunque el código ya lo intentaba.

Ahora se usa `gordonklaus/portaudio` (captura real del micrófono) +
`alphacep/vosk-api/go` (motor Vosk, reconocimiento de voz **offline** real).
`Ears.Escuchar()` ahora devuelve `(string, error)` en vez de solo `string`
— es un cambio de firma respecto a la versión anterior, ya reflejado en
`main.go`.

**Necesitas descargar un modelo de voz en español** (no se puede generar
código para eso, es un archivo de datos):
1. Ir a https://alphacephei.com/vosk/models
2. Descargar un modelo español (busca "es", por ejemplo `vosk-model-small-es-*`)
3. Descomprimirlo en `JarvisOS/modelo-voz-es/` (o cambiar `ModeloVoz` en `config/config.go`)

### Tarea 2 — 10 comandos nuevos
Todos en `core/hands.go`, enrutados desde `RunCommand`:

| Comando de voz | Acción |
|---|---|
| `cerrar [app]` | Cierra una aplicación (`taskkill`) |
| `hora` | Dice la hora actual |
| `fecha` | Dice la fecha actual |
| `buscar [algo]` | Busca en Google |
| `buscar en youtube [algo]` | Busca en YouTube |
| `bloquear pantalla` | Bloquea la sesión de Windows |
| `minimizar todo` | Minimiza todas las ventanas |
| `captura de pantalla` | Guarda un screenshot en `Pictures` |
| `abrir descargas` / `abrir escritorio` | Abre esas carpetas en el Explorador |
| `abrir configuración` | Abre la Configuración de Windows |

De regalo también se agregó `abrir chrome` y `abrir spotify` al catálogo de
apps (estaban en el pedido original del proyecto pero no en el código subido).

**Decisión deliberada:** no se agregó ningún comando de apagar/reiniciar la
PC. Un falso positivo del reconocedor de voz activando eso sería demasiado
riesgoso (pérdida de trabajo sin guardar). `bloquear pantalla` cubre la
necesidad de "seguridad rápida" sin ese riesgo.

### Tarea 3 — Listo para conectar IA (ChatGPT)
Paquete nuevo `ia/conector.go`: llama a la API de OpenAI (Chat Completions)
usando solo `net/http` + `encoding/json` de la librería estándar (sin SDK).
Se eligió OpenAI por ser la más simple de integrar sin dependencias extra.

- Si la variable de entorno `OPENAI_API_KEY` **no** está configurada,
  `Disponible()` devuelve `false` y JarvisOS sigue funcionando 100% con
  comandos locales, sin errores ni bloqueos.
- Si **sí** está configurada, `Brain` la usa como respaldo: primero intenta
  un comando local, y solo si no reconoce nada, le pregunta a la IA.
- Para activarlo: `set OPENAI_API_KEY=sk-...` (cmd) o
  `$env:OPENAI_API_KEY="sk-..."` (PowerShell) antes de `go run .`
- Para cambiar a Gemini más adelante: solo hay que reescribir
  `ia/conector.go` (endpoint + formato de petición/respuesta). Nada más en
  el proyecto depende de OpenAI directamente — `Brain` solo conoce la
  interfaz `core.ConectorIA`.

### Extra: se conectó `Brain`, que estaba sin usar
`main.go` llamaba a `hands.RunCommand()` directamente; `Brain.Process()`
existía pero nunca se invocaba. Ahora todo pasa por `Brain`, que:
1. Prueba comandos locales (`Hands.RunCommand`)
2. Si nada coincide, prueba la IA de respaldo (si está configurada)
3. Si tampoco, responde con el mensaje genérico de "no entendí"

También se corrigió `agents/coder_agent.go`, que tenía una llave de cierre
faltante (no compilaba). Se dejó como cimiento documentado para una futura
capacidad de generar/ejecutar código con IA — deliberadamente **no** se
implementó ahora: ejecutar código sugerido por una IA sin supervisión es un
riesgo de seguridad real, no un detalle menor. Cuando se construya, debe
respetar `config.RequireApproval`.

## Antes de correrlo

Ver **`INSTALACION.md`** — guía paso a paso completa (MSYS2, PortAudio,
Vosk, variables de entorno, y una tabla de troubleshooting con el error más
probable que te va a aparecer, ya resuelto de antemano).

## Limitaciones conocidas / honestidad técnica

- Este entorno de trabajo es Linux, sin acceso a red y sin Go instalado, así
  que **no pude compilar ni ejecutar este código de verdad**. Está escrito
  con cuidado y las APIs de `portaudio`, `vosk-api/go` y la sintaxis de Go
  están verificadas contra documentación oficial, pero recomiendo que el
  primer `go build` sea tuyo — si tira un error, pégamelo y lo resuelvo.
- El nombre exacto del modelo Vosk en español (versión) cambia con el
  tiempo — por eso te mando al listado oficial en vez de inventar un
  nombre de archivo que podría estar desactualizado.
- `CoderAgent` sigue siendo solo un cimiento vacío, a propósito.

## Próximos pasos sugeridos

Las 4 fases pedidas están completas. Lo más urgente en términos absolutos
sigue siendo la prueba conjunta pendiente (memoria, recordatorios, trato
personalizado — nada de esto se probó en la máquina real todavía). Más
allá de eso, ideas para una posible ronda futura, ninguna comprometida:

- **Recordatorios con fecha**, no solo hora de hoy/mañana ("el jueves a
  las 5", "el 25 de julio")
- **Recordatorios recurrentes** ("todos los días a las 8, recordame la pastilla")
- **Buscar dentro de las notas guardadas** ("tengo alguna nota sobre el auto")
- Diagnosticar el error de `go build` (GOROOT) que quedó sin resolver — no
  bloquea nada porque `go run .` anda, pero sigue pendiente si en algún
  momento se quiere un `.exe` standalone
- Entrenar/ajustar el reconocimiento de voz si el ruido de fondo sigue
  siendo un problema (modelo Vosk más grande, si hace falta)
