# JarvisOS — Guía de instalación en Windows

Esta guía asume Windows 10 u 11, 64 bits. Todo lo que sigue está verificado
contra documentación oficial y reportes de errores reales (no es una lista
genérica) — pero yo no tengo forma de correrlo en este entorno (ver
`PROGRESO_JARVISOS.md`, sección Limitaciones), así que puede que algún paso
necesite un ajuste menor en tu máquina. Si te trabás, pegame el error exacto
y lo resolvemos.

Tiempo estimado: 30–60 minutos, la mayor parte descargando cosas.

---

## Paso 0: Go

Si no lo tenés: https://go.dev/dl/ → instalador `.msi` para Windows → Next,
Next, Finish. Verificá:
```
go version
```
Debería mostrar `go1.22` o más nuevo.

---

## Paso 1: MSYS2 + GCC (necesario para CGO)

`portaudio` y `vosk-api` usan CGO (código C llamado desde Go), así que Go
necesita un compilador de C. Windows no trae uno.

1. Descargá el instalador de https://www.msys2.org/ y ejecutalo (deja todo
   por defecto, instala en `C:\msys64`).
2. Al terminar, se abre una terminal MSYS2. Ahí corré:
   ```
   pacman -Syu
   ```
   Te va a pedir cerrar la terminal a la mitad — cerrala y volvé a abrir
   "MSYS2" desde el menú inicio, y corré `pacman -Syu` una segunda vez
   hasta que no haya más actualizaciones.
3. Instalá el compilador:
   ```
   pacman -S mingw-w64-x86_64-gcc
   ```
4. Agregá `C:\msys64\mingw64\bin` al PATH de Windows:
   - Buscá "Variables de entorno" en el menú inicio → "Editar las variables
     de entorno del sistema" → botón "Variables de entorno" → en "Path"
     (de Usuario o Sistema) → "Editar" → "Nuevo" → pegá
     `C:\msys64\mingw64\bin`.
5. Abrí una terminal **nueva** (PowerShell normal, no la de MSYS2) y
   verificá:
   ```
   gcc --version
   ```

---

## Paso 2: PortAudio

Misma terminal MSYS2 del paso 1:
```
pacman -S mingw-w64-x86_64-portaudio
```
Esto instala `portaudio.h` y `libportaudio.dll` dentro de
`C:\msys64\mingw64\`, ya en el lugar donde GCC los va a encontrar
automáticamente porque agregaste `mingw64\bin` al PATH. No necesitás
configurar nada más para PortAudio en particular.

**⚠️ Error conocido y ya resuelto de antemano:** con esta combinación
(gordonklaus/portaudio + MSYS2 en Windows) es común pegar este error al
compilar:
```
invalid flag in pkg-config --cflags: -mthreads
```
No es un bug tuyo: Go rechaza por seguridad ciertas banderas que
pkg-config devuelve. Se arregla con una variable de entorno (ver Paso 4).

---

## Paso 3: Vosk (reconocimiento de voz)

No tiene instalador — se descarga a mano:

1. Andá a https://github.com/alphacep/vosk-api/releases y descargá el zip
   de Windows de la versión más reciente (buscá algo como
   `vosk-win64-X.X.X.zip`; el número de versión cambia, por eso no te doy
   uno fijo).
2. Descomprimilo. Vas a tener `libvosk.dll` y `vosk_api.h` (entre otros
   archivos).
3. Copiá esos dos archivos a la raíz de tu proyecto `JarvisOS\` (junto a
   `go.mod`). Es el camino más simple: así CGO y el `.exe` final los
   encuentran sin configurar rutas absolutas.
4. Descargá el modelo de voz en **español**:
   - Andá a https://alphacephei.com/vosk/models
   - Buscá un modelo que empiece con `vosk-model-small-es-` (versión
     pequeña, más rápida) o `vosk-model-es-` (versión grande, más precisa)
   - Descomprimilo dentro de `JarvisOS\`, y **renombrá la carpeta
     resultante a `modelo-voz-es`** (o cambiá el valor de `ModeloVoz` en
     `config/config.go` para que apunte al nombre que le pusiste).

---

## Paso 4: variables de entorno para compilar

Abrí PowerShell **como Administrador** en la carpeta `JarvisOS\` y corré
(esto las deja guardadas permanentemente, no solo para esta sesión):
```powershell
setx CGO_ENABLED 1
setx CGO_CFLAGS_ALLOW "-mthreads"
setx CGO_CFLAGS "-I$PWD"
setx CGO_LDFLAGS "-L$PWD -lvosk"
```
**Cerrá la terminal y abrí una nueva** para que `setx` tome efecto (es
la forma en que funciona `setx`, no es un paso opcional).

---

## Paso 5: compilar

En la nueva terminal, parado en `JarvisOS\`:
```
go mod tidy
go build .
```
`go mod tidy` necesita internet: resuelve las versiones reales de
`portaudio` y `vosk-api/go` (no las fijé a mano en `go.mod` a propósito,
para no arriesgar una versión inventada).

Si compiló, vas a tener un `JarvisOS.exe`. Antes de correrlo, copiá
`libvosk.dll` (Paso 3) a la misma carpeta si `go build` lo puso en otro
lado.

---

## Paso 6: correrlo

```
.\JarvisOS.exe
```
Debería decir "Sistemas en línea" e imprimir "Di 'Jarvis' para
activarme...". Probá con el micrófono.

**Opcional — respaldo de IA y CoderAgent:**
```powershell
setx OPENAI_API_KEY "sk-tu-clave-real"
```
(nueva terminal después de esto también). Sin esto, JarvisOS funciona
igual, solo que sin el respaldo conversacional ni CoderAgent.

---

## Troubleshooting — errores esperables y su causa

| Error | Causa probable | Solución |
|---|---|---|
| `invalid flag in pkg-config --cflags: -mthreads` | Go bloquea esa bandera por seguridad | `setx CGO_CFLAGS_ALLOW "-mthreads"` (Paso 4) y abrir terminal nueva |
| `gcc: executable file not found` | El PATH no tiene `mingw64\bin`, o la terminal es vieja | Repetir Paso 1.4 y abrir una terminal nueva |
| `vosk_api.h: No such file or directory` | `CGO_CFLAGS` no apunta a donde está `vosk_api.h` | Confirmar que el archivo está en `JarvisOS\` y que `CGO_CFLAGS` se configuró en una terminal *después* de tenerlo ahí |
| El `.exe` abre y se cierra solo / `libvosk.dll no encontrado` | El `.dll` no está junto al `.exe` | Copiar `libvosk.dll` a la misma carpeta que `JarvisOS.exe` |
| El micrófono no reconoce nada | Sin gramática/modelo, o modelo mal ubicado | Confirmar que la carpeta `modelo-voz-es` existe y tiene archivos adentro (no un .zip sin descomprimir) |

Si aparece algo que no está en esta tabla: pegame el mensaje de error
completo (no un resumen) y lo resolvemos puntualmente.
