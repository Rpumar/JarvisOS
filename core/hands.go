package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ComandoNoReconocido es el valor centinela que RunCommand devuelve cuando
// ningún comando local coincide. Brain lo usa para decidir si debe intentar
// una consulta de respaldo a la IA en vez de rendirse de inmediato.
const ComandoNoReconocido = "__NO_RECONOCIDO__"

// Códigos de tecla virtual de Windows para multimedia (winuser.h). Son
// valores estables desde Windows 2000. No existen como código nombrado en
// WScript.Shell.SendKeys (que solo cubre un teclado estándar de 101 teclas),
// por eso se simulan con keybd_event en vez de SendKeys.
const (
	vkVolumeMute     byte = 0xAD
	vkVolumeDown     byte = 0xAE
	vkVolumeUp       byte = 0xAF
	vkMediaNextTrack byte = 0xB0
	vkMediaPrevTrack byte = 0xB1
	vkMediaPlayPause byte = 0xB3
)

// Hands ejecuta acciones reales sobre Windows: abrir/cerrar apps, volumen,
// multimedia, navegador, sistema de archivos, capturas y voz (TTS).
type Hands struct {
	muCache  sync.Mutex
	cache    map[string]cacheEntry
	Apps     map[string]string
	ClimaKey string
	NewsKey  string
	Prefs    RegistroPreferencias
	rutinas  *RutinaManager
	clasif   *Clasificador
}

type cacheEntry struct {
	valor      string
	expiracion time.Time
}

type HandsOpciones struct {
	Apps      map[string]string
	ClimaKey string
	NewsKey  string
	Prefs    RegistroPreferencias
	Rutinas  *RutinaManager
}

func NewHands(opciones ...HandsOpciones) *Hands {
	h := &Hands{cache: make(map[string]cacheEntry), clasif: NuevoClasificador()}
	if len(opciones) > 0 {
		h.Apps = opciones[0].Apps
		h.ClimaKey = opciones[0].ClimaKey
		h.NewsKey = opciones[0].NewsKey
		h.Prefs = opciones[0].Prefs
		h.rutinas = opciones[0].Rutinas
	}
	return h
}

func (h *Hands) obtenerOCache(clave string, ttl time.Duration, fn func() string) string {
	h.muCache.Lock()
	if e, ok := h.cache[clave]; ok && time.Now().Before(e.expiracion) {
		h.muCache.Unlock()
		return e.valor
	}
	h.muCache.Unlock()
	valor := fn()
	h.muCache.Lock()
	h.cache[clave] = cacheEntry{valor: valor, expiracion: time.Now().Add(ttl)}
	h.muCache.Unlock()
	return valor
}

// RunCommand interpreta cmd y ejecuta la acción correspondiente. Devuelve un
// texto listo para imprimir Y para hablar (por eso nunca incluye prefijos
// como "[MANOS]": eso se agrega solo al imprimir, no al hablar).
func (h *Hands) RunCommand(cmd string) string {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	if cmd == "" {
		return ComandoNoReconocido
	}

	if h.clasif != nil {
		if nombre, ok := h.clasif.Clasificar(cmd); ok {
			return h.clasif.Ejecutar(nombre, cmd, h)
		}
	}

	if nombreApp := h.buscarAppDirecto(cmd); nombreApp != "" {
		return h.abrirApp(nombreApp)
	}

	// NAVEGADOR
	if strings.Contains(cmd, "buscar en youtube ") {
		consulta := strings.TrimSpace(strings.Replace(cmd, "buscar en youtube ", "", 1))
		return h.buscarEnYoutube(consulta)
	}
	if strings.Contains(cmd, "buscar en wikipedia ") {
		consulta := strings.TrimSpace(strings.Replace(cmd, "buscar en wikipedia ", "", 1))
		return h.buscarEnWikipedia(consulta)
	}
	if strings.Contains(cmd, "buscar ") || strings.Contains(cmd, "busca ") || strings.Contains(cmd, "buscá ") || strings.Contains(cmd, "googleá ") || strings.Contains(cmd, "googlea ") {
		consulta := extraerObjeto(cmd, []string{"buscar ", "busca ", "buscá ", "googleá ", "googlea "})
		return h.buscarEnGoogle(consulta)
	}
	if strings.Contains(cmd, "ir a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "ir a ", "", 1))
		return h.irASitio(sitio)
	}
	if strings.Contains(cmd, "andá a ") || strings.Contains(cmd, "anda a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "andá a ", "", 1))
		return h.irASitio(sitio)
	}
	if strings.Contains(cmd, "navegar a ") || strings.Contains(cmd, "navegá a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "navegar a ", "", 1))
		return h.irASitio(sitio)
	}
	if strings.Contains(cmd, "dirigite a ") || strings.Contains(cmd, "dirigirse a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "dirigite a ", "", 1))
		return h.irASitio(sitio)
	}
	if strings.Contains(cmd, "entrá a ") || strings.Contains(cmd, "entra a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "entrá a ", "", 1))
		return h.irASitio(sitio)
	}
	if strings.Contains(cmd, "vamos a ") {
		sitio := strings.TrimSpace(strings.Replace(cmd, "vamos a ", "", 1))
		return h.irASitio(sitio)
	}

	// SISTEMA
	switch {
	case strings.Contains(cmd, "bloquear pantalla"), strings.Contains(cmd, "bloquear pc"):
		return h.bloquearPantalla()
	case strings.Contains(cmd, "minimizar todo"), strings.Contains(cmd, "minimizar ventanas"):
		return h.minimizarTodo()
	case strings.Contains(cmd, "captura de pantalla"), strings.Contains(cmd, "capturar pantalla"):
		return h.capturarPantalla()
	}

	// ARCHIVOS Y SISTEMA OPERATIVO
	switch {
	case strings.Contains(cmd, "abrir descargas"):
		return h.abrirCarpeta("Downloads")
	case strings.Contains(cmd, "abrir escritorio"):
		return h.abrirCarpeta("Desktop")
	case strings.Contains(cmd, "abrir documentos"):
		return h.abrirCarpeta("Documents")
	case strings.Contains(cmd, "abrir papelera"):
		return h.abrirPapelera()
	case strings.Contains(cmd, "abrir configuración"), strings.Contains(cmd, "abrir configuracion"):
		return h.abrirConfiguracion()
	case strings.Contains(cmd, "abrir imágenes") || strings.Contains(cmd, "abrir imagenes") || strings.Contains(cmd, "abrir fotos carpeta"):
		return h.abrirCarpeta("Pictures")
	case strings.Contains(cmd, "abrir música carpeta") || strings.Contains(cmd, "abrir musica carpeta"):
		return h.abrirCarpeta("Music")
	}

	// BOCA - REPETIR TEXTO
	if strings.Contains(cmd, "decir ") {
		mensaje := strings.TrimSpace(strings.Replace(cmd, "decir ", "", 1))
		if mensaje == "" {
			return "¿Qué desea que repita, señor?"
		}
		return mensaje
	}

	// === SISTEMA / INFORMACIÓN ===
	switch {
	case strings.Contains(cmd, "procesador") || strings.Contains(cmd, "cpu"):
		return h.infoCPU()
	case strings.Contains(cmd, "memoria ram") || strings.Contains(cmd, "ram"):
		return h.infoRAM()
	case strings.Contains(cmd, "disco") || strings.Contains(cmd, "almacenamiento"):
		return h.infoDisco()
	case strings.Contains(cmd, "sistema operativo") || strings.Contains(cmd, "versión de windows") || strings.Contains(cmd, "version de windows"):
		return h.infoSO()
	case strings.Contains(cmd, "tiempo activo") || strings.Contains(cmd, "uptime") || strings.Contains(cmd, "tiempo encendido"):
		return h.infoUptime()
	case strings.Contains(cmd, "usuario") || strings.Contains(cmd, "nombre de usuario") || strings.Contains(cmd, "quién soy"):
		return h.infoUsuario()
	case strings.Contains(cmd, "nombre del equipo") || strings.Contains(cmd, "nombre de pc") || strings.Contains(cmd, "nombre pc"):
		return h.infoPC()
	case strings.Contains(cmd, "arquitectura") || strings.Contains(cmd, "bits"):
		return h.infoArquitectura()
	case strings.Contains(cmd, "programas instalados") || strings.Contains(cmd, "aplicaciones instaladas"):
		return h.infoProgramas()
	case strings.Contains(cmd, "procesos") || strings.Contains(cmd, "ejecutando"):
		return h.infoProcesos()
	case strings.Contains(cmd, "núcleos") || strings.Contains(cmd, "nucleos") || strings.Contains(cmd, "cores"):
		return h.infoNucleos()
	case strings.Contains(cmd, "resolución") || strings.Contains(cmd, "resolucion") || strings.Contains(cmd, "pantalla"):
		return h.infoPantalla()
	case strings.Contains(cmd, "idioma") || strings.Contains(cmd, "lenguaje"):
		return h.infoIdioma()
	case strings.Contains(cmd, "zona horaria"):
		return h.infoZonaHoraria()
	case strings.Contains(cmd, "temperatura") || strings.Contains(cmd, "calor"):
		return h.infoTemperatura()
	}

	// === RED ===
	switch {
	case strings.Contains(cmd, "ip pública") || strings.Contains(cmd, "ip publica") || strings.Contains(cmd, "ip externa"):
		return h.ipPublica()
	case strings.Contains(cmd, "ping ") || strings.Contains(cmd, "latencia") || strings.Contains(cmd, "latency"):
		consulta := strings.TrimSpace(strings.Replace(cmd, "ping ", "", 1))
		if consulta == "" || consulta == "google" {
			consulta = "google.com"
		}
		return h.hacerPing(consulta)
	case strings.Contains(cmd, "dns"):
		return h.infoDNS()
	case strings.Contains(cmd, "mac") || strings.Contains(cmd, "dirección física"):
		return h.infoMAC()
	case strings.Contains(cmd, "velocidad red") || strings.Contains(cmd, "velocidad internet"):
		return h.infoVelocidadRed()
	case strings.Contains(cmd, "wifi") || strings.Contains(cmd, "red inalámbrica") || strings.Contains(cmd, "red inalambrica"):
		return h.infoWifi()
	case strings.Contains(cmd, "conexiones activas") || strings.Contains(cmd, "conexiones"):
		return h.infoConexiones()
	case strings.Contains(cmd, "interfaces red") || strings.Contains(cmd, "interfaces de red"):
		return h.infoInterfaces()
	}

	// === ENERGÍA / SISTEMA ===
	switch {
	case strings.Contains(cmd, "suspender") || strings.Contains(cmd, "dormir") || strings.Contains(cmd, "suspensión"):
		return h.suspender()
	case strings.Contains(cmd, "hibernar") || strings.Contains(cmd, "hibernación"):
		return h.hibernar()
	case strings.Contains(cmd, "reiniciar") || strings.Contains(cmd, "reinicio") || strings.Contains(cmd, "restart"):
		return h.reiniciar()
	case strings.Contains(cmd, "cerrar sesión") || strings.Contains(cmd, "cerrar sesion") || strings.Contains(cmd, "logoff"):
		return h.cerrarSesion()
	case strings.Contains(cmd, "apagar monitor") || strings.Contains(cmd, "apagar pantalla"):
		return h.apagarMonitor()
	case strings.Contains(cmd, "subir brillo") || strings.Contains(cmd, "más brillo") || strings.Contains(cmd, "mas brillo"):
		return h.brillo("up")
	case strings.Contains(cmd, "bajar brillo") || strings.Contains(cmd, "menos brillo"):
		return h.brillo("down")
	case strings.Contains(cmd, "brillo al "):
		return h.brilloPorcentaje(cmd)
	case strings.Contains(cmd, "modo avión") || strings.Contains(cmd, "modo avion") || strings.Contains(cmd, "airplane"):
		return h.modoAvion()
	}

	// === VENTANAS ===
	switch {
	case strings.Contains(cmd, "maximizar ventana") || strings.Contains(cmd, "maximizar"):
		return h.maximizarVentana()
	case strings.Contains(cmd, "minimizar ventana") || strings.Contains(cmd, "minimizar"):
		return h.minimizarVentana()
	case strings.Contains(cmd, "restaurar ventana") || strings.Contains(cmd, "restaurar"):
		return h.restaurarVentana()
	case strings.Contains(cmd, "cambiar ventana") || strings.Contains(cmd, "siguiente ventana"):
		return h.cambiarVentana()
	case strings.Contains(cmd, "organizar ventanas") || strings.Contains(cmd, "mosaico"):
		return h.organizarVentanas()
	case strings.Contains(cmd, "cerrar ventana") || strings.Contains(cmd, "cerrar esta"):
		return h.cerrarVentanaActiva()
	}

	// === ATAJOS WEB ===
	switch {
	case strings.Contains(cmd, "abrir youtube") || strings.Contains(cmd, "abre youtube") || strings.Contains(cmd, "abrime youtube"):
		return h.irASitio("youtube.com")
	case strings.Contains(cmd, "abrir gmail") || strings.Contains(cmd, "abrir correo"):
		return h.irASitio("gmail.com")
	case strings.Contains(cmd, "abrir github"):
		return h.irASitio("github.com")
	case strings.Contains(cmd, "abrir stackoverflow") || strings.Contains(cmd, "abrir stack overflow"):
		return h.irASitio("stackoverflow.com")
	case strings.Contains(cmd, "abrir amazon"):
		return h.irASitio("amazon.com")
	case strings.Contains(cmd, "abrir twitter") || strings.Contains(cmd, "abrir x.com"):
		return h.irASitio("x.com")
	case strings.Contains(cmd, "abrir reddit"):
		return h.irASitio("reddit.com")
	case strings.Contains(cmd, "abrir facebook"):
		return h.irASitio("facebook.com")
	case strings.Contains(cmd, "abrir instagram"):
		return h.irASitio("instagram.com")
	case strings.Contains(cmd, "abrir netflix"):
		return h.irASitio("netflix.com")
	}

	// === ARCHIVOS ===
	switch {
	case strings.Contains(cmd, "listar archivos") || strings.Contains(cmd, "listar directorio") || strings.Contains(cmd, "lista archivos") || strings.Contains(cmd, "ver archivos"):
		return h.listarDirectorio()
	case strings.Contains(cmd, "crear carpeta") || strings.Contains(cmd, "crear directorio"):
		return h.crearCarpeta(cmd)
	case strings.Contains(cmd, "crear archivo") || strings.Contains(cmd, "nuevo archivo"):
		return h.crearArchivo(cmd)
	case strings.Contains(cmd, "buscar archivo ") || strings.Contains(cmd, "buscar archivos "):
		return h.buscarArchivos(cmd)
	case strings.Contains(cmd, "vaciar papelera") || strings.Contains(cmd, "limpiar papelera") || strings.Contains(cmd, "vacíe la papelera"):
		return h.vaciarPapelera()
	case strings.Contains(cmd, "ruta actual") || strings.Contains(cmd, "dónde estoy") || strings.Contains(cmd, "pwd"):
		return h.rutaActual()
	case strings.Contains(cmd, "archivos recientes") || strings.Contains(cmd, "recientes"):
		return h.abrirCarpeta("Recent")
	}

	// === DIVERSIÓN ===
	switch {
	case strings.Contains(cmd, "moneda") || strings.Contains(cmd, "cara o cruz") || strings.Contains(cmd, "cara y cruz"):
		return h.lanzarMoneda()
	case strings.Contains(cmd, "dado") || strings.Contains(cmd, "tirar dado") || strings.Contains(cmd, "lanzar dado"):
		return h.tirarDado()
	case strings.Contains(cmd, "número aleatorio") || strings.Contains(cmd, "numero aleatorio") || strings.Contains(cmd, "random"):
		return h.numeroAleatorio()
	case strings.Contains(cmd, "cumplido") || strings.Contains(cmd, "piropo") || strings.Contains(cmd, "algo bonito"):
		return h.decirCumplido()
	case strings.Contains(cmd, "sí o no") || strings.Contains(cmd, "si o no") || strings.Contains(cmd, "decisión") || strings.Contains(cmd, "decision"):
		return h.decidir()
	case strings.Contains(cmd, "color aleatorio") || strings.Contains(cmd, "color random"):
		return h.colorAleatorio()
	case strings.Contains(cmd, "motivación") || strings.Contains(cmd, "motivacion") || strings.Contains(cmd, "frase motivadora"):
		return h.fraseMotivadora()
	case strings.Contains(cmd, "trabalenguas") || strings.Contains(cmd, "traba lenguas"):
		return h.trabalenguas()
	case strings.Contains(cmd, "signo zodiacal") || strings.Contains(cmd, "horóscopo") || strings.Contains(cmd, "horoscopo"):
		return h.signoZodiacal()
	case strings.Contains(cmd, "días para") || strings.Contains(cmd, "dias para") || strings.Contains(cmd, "cuánto falta para") || strings.Contains(cmd, "cuanto falta para"):
		return h.diasParaNavidad()
	}

	// === VOZ / SONIDO ===
	switch {
	case strings.Contains(cmd, "volumen actual") || strings.Contains(cmd, "qué volumen") || strings.Contains(cmd, "que volumen"):
		return h.volumenActual()
	case strings.Contains(cmd, "activar sonido") || strings.Contains(cmd, "quitar silencio") || strings.Contains(cmd, "quitar mute"):
		return h.volumen("mute")
	case strings.Contains(cmd, "fecha completa") || strings.Contains(cmd, "fecha de hoy completa"):
		return h.fechaCompleta()
	case strings.Contains(cmd, "eco ") || strings.Contains(cmd, "repite "):
		msg := strings.TrimSpace(strings.Replace(cmd, "eco ", "", 1))
		msg = strings.TrimSpace(strings.Replace(msg, "repite ", "", 1))
		if msg == "" {
			return "¿Qué desea que repita, señor?"
		}
		return msg
	case strings.Contains(cmd, "saludar") || strings.Contains(cmd, "saludame") || strings.Contains(cmd, "salúdame"):
		return h.saludarPersonalizado()
	}

	// === MISC ===
	switch {
	case strings.Contains(cmd, "tomar nota rápida") || strings.Contains(cmd, "tomar nota") && strings.Contains(cmd, "rápida"):
		return h.tomarNotaRapida()
	case strings.Contains(cmd, "respira") || strings.Contains(cmd, "respiración") || strings.Contains(cmd, "respiracion"):
		return h.ejercicioRespiración()
	case strings.Contains(cmd, "beber agua") || strings.Contains(cmd, "hidratación") || strings.Contains(cmd, "hidratate") || strings.Contains(cmd, "hidratación"):
		return h.recordatorioAgua()
	case strings.Contains(cmd, "estiramiento") || strings.Contains(cmd, "estirar") || strings.Contains(cmd, "estirarse"):
		return h.recordatorioEstiramiento()
	}

	// CERRAR APLICACIONES
	if strings.Contains(cmd, "cerrar ") {
		app := strings.TrimSpace(strings.Replace(cmd, "cerrar ", "", 1))
		return h.cerrarApp(app)
	}

	return ComandoNoReconocido
}

func (h *Hands) abrirApp(ruta string) string {
	if !esRutaSegura(ruta) {
		return fmt.Sprintf("No puedo abrir '%s', señor. Solo letras, números y guiones.", ruta)
	}
	if err := exec.Command("cmd", "/C", "start", "", ruta).Run(); err != nil {
		return fmt.Sprintf("No pude abrir eso, señor: %v", err)
	}
	if h.Prefs != nil {
		h.Prefs.RegistrarApp(filepath.Base(ruta))
	}
	return ConfirmacionGenerica()
}

func esRutaSegura(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
			c != ' ' && c != '-' && c != '.' && c != ':' && c != '_' && c != '/' &&
			c != 'á' && c != 'é' && c != 'í' && c != 'ó' && c != 'ú' &&
			c != 'Á' && c != 'É' && c != 'Í' && c != 'Ó' && c != 'Ú' && c != 'ñ' && c != 'Ñ' {
			return false
		}
	}
	return true
}

// procesosProtegidos son procesos críticos de Windows que cerrarApp se
// niega a matar, sin excepción. Antes de esta revisión, cerrarApp le pasaba
// CUALQUIER nombre directo a "taskkill /F" — un "cerrar explorer" mal
// reconocido por voz (o pedido literalmente sin pensarlo) hubiera hecho
// exactamente lo mismo que pedir cerrar Chrome. Esto cierra ese hueco.
var procesosProtegidos = []string{
	"winlogon.exe", "csrss.exe", "services.exe", "lsass.exe", "smss.exe",
	"wininit.exe", "svchost.exe", "explorer.exe",
}

func esProcesoProtegido(proceso string) bool {
	procesoMin := strings.ToLower(proceso)
	for _, p := range procesosProtegidos {
		if procesoMin == p {
			return true
		}
	}
	return false
}

var (
	patronURLSegura = regexp.MustCompile(`^[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(/[a-zA-Z0-9\-\._~:/?#\[\]@!$&'()*+,;=]*)?$`)
)

func argumentoUsuarioSeguro(arg string) bool {
	if len(arg) > 256 {
		return false
	}
	for _, r := range arg {
		if r < 32 || r > 126 {
			return false
		}
		if strings.ContainsRune("&|;`$<>(){}[]#!'\""+"\\\n\r\t", r) {
			return false
		}
	}
	return true
}

func urlSegura(sitio string) bool {
	if len(sitio) > 512 {
		return false
	}
	sitio = strings.TrimPrefix(sitio, "https://")
	sitio = strings.TrimPrefix(sitio, "http://")
	sitio = strings.TrimPrefix(sitio, "www.")
	return patronURLSegura.MatchString(sitio)
}

func validarArgumentoPowerShell(argumento string) string {
	return strings.NewReplacer(
		"`", "``",
		"'", "''",
		"$", "`$",
		"\n", " ",
		"\r", " ",
	).Replace(argumento)
}

type limitadorSensible struct {
	mu        sync.Mutex
	ultimaAcc time.Time
}

func (l *limitadorSensible) Permitir(intervalo time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if time.Since(l.ultimaAcc) < intervalo {
		return false
	}
	l.ultimaAcc = time.Now()
	return true
}

var limitadorGlobal = &limitadorSensible{}

func (h *Hands) buscarAppDirecto(cmd string) string {
	verbos := []string{"abrir ", "abri ", "abrime ", "abrim ", "abrí ", "abre ", "abrí ",
		"abrime ", "abrime ", "abr", "quiero ", "podes ", "podés ", "puedo ", "puede ",
		"necesito ", "podes abrir ", "podés abrir ", "puedo abrir ", "puede abrir "}

	sinVerbo := cmd
	for _, v := range verbos {
		sinVerbo = strings.TrimPrefix(sinVerbo, v)
	}

	if app, _ := extraerApp(cmd, h.Apps); app != "" {
		return app
	}

	palabras := strings.Fields(sinVerbo)
	for _, p := range palabras {
		if app, nombre := extraerApp(p, h.Apps); app != "" {
			_ = nombre
			return app
		}
	}

	for _, p := range strings.Fields(cmd) {
		if app, _ := extraerApp(p, h.Apps); app != "" {
			return app
		}
	}

	return ""
}

func (h *Hands) buscarAppEnComando(cmd string) string {
	nombres := make([]string, 0, len(h.Apps))
	for n := range h.Apps {
		nombres = append(nombres, n)
	}
	sort.Slice(nombres, func(i, j int) bool {
		return len(nombres[i]) > len(nombres[j])
	})
	for _, nombre := range nombres {
		if strings.Contains(cmd, nombre) {
			return h.Apps[nombre]
		}
	}
	return ""
}

func (h *Hands) apagarPC() string {
	if !limitadorGlobal.Permitir(30 * time.Second) {
		return "Ya se ha solicitado un apagado recientemente, señor. Espere unos segundos."
	}
	if err := exec.Command("shutdown", "/s", "/t", "5").Run(); err != nil {
		return "No pude iniciar el apagado, señor."
	}
	return "Apagando el equipo en 5 segundos, señor."
}

func (h *Hands) cerrarApp(app string) string {
	if app == "" {
		return "¿Qué aplicación desea que cierre, señor?"
	}
	proceso := app
	if !strings.HasSuffix(proceso, ".exe") {
		proceso += ".exe"
	}
	if !esRutaSegura(proceso) {
		return fmt.Sprintf("No puedo cerrar '%s', señor: nombre no válido.", app)
	}
	if esProcesoProtegido(proceso) {
		return fmt.Sprintf("No voy a cerrar %s, señor: es un proceso del sistema y podría dejar Windows inestable.", app)
	}
	if err := exec.Command("taskkill", "/IM", proceso, "/F").Run(); err != nil {
		return fmt.Sprintf("No pude cerrar %s. ¿Está abierta, señor?", app)
	}
	return fmt.Sprintf("%s cerrada, señor.", app)
}

// enviarTeclaVirtual simula la pulsación de una tecla virtual de Windows
// (volumen, multimedia) mediante keybd_event de user32.dll, llamado desde
// PowerShell vía Add-Type. CORRECCIÓN vs. la versión anterior: estas teclas
// no existen en el lenguaje de códigos de WScript.Shell.SendKeys (que solo
// cubre un teclado estándar de 101 teclas) — "{volume_up}" no es un código
// válido de SendKeys y no funcionaba. keybd_event con el código de tecla
// virtual real (verificado contra winuser.h) sí funciona.
func (h *Hands) enviarTeclaVirtual(codigoVK byte) error {
	ps := fmt.Sprintf(`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; `+
		`public class TeclaVirtual { [DllImport("user32.dll")] `+
		`public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, uint dwExtraInfo); }'; `+
		`[TeclaVirtual]::keybd_event(%d, 0, 0, 0); Start-Sleep -Milliseconds 50; `+
		`[TeclaVirtual]::keybd_event(%d, 0, 2, 0)`, codigoVK, codigoVK)

	return exec.Command("powershell", "-Command", ps).Run()
}

func (h *Hands) volumen(accion string) string {
	var codigo byte
	var mensaje string
	switch accion {
	case "up":
		codigo, mensaje = vkVolumeUp, "Subiendo volumen, señor."
	case "down":
		codigo, mensaje = vkVolumeDown, "Bajando volumen, señor."
	case "mute":
		codigo, mensaje = vkVolumeMute, "Silenciando, señor."
	}
	if err := h.enviarTeclaVirtual(codigo); err != nil {
		return fmt.Sprintf("No pude ajustar el volumen: %v", err)
	}
	return mensaje
}

// controlMedia maneja play/pausa y cambio de pista (nuevo). VK_MEDIA_PLAY_PAUSE
// es una tecla de alternancia: no hay forma de saber desde acá si el
// reproductor queda en play o en pausa, por eso el mensaje es neutro.
func (h *Hands) controlMedia(accion string) string {
	var codigo byte
	var mensaje string
	switch accion {
	case "play_pause":
		codigo, mensaje = vkMediaPlayPause, "Listo, señor."
	case "next":
		codigo, mensaje = vkMediaNextTrack, "Siguiente canción, señor."
	case "prev":
		codigo, mensaje = vkMediaPrevTrack, "Canción anterior, señor."
	}
	if err := h.enviarTeclaVirtual(codigo); err != nil {
		return fmt.Sprintf("No pude controlar la reproducción: %v", err)
	}
	return mensaje
}

func (h *Hands) decirHora() string {
	return formatearHora(time.Now())
}

// formatearHora es la lógica pura de decirHora, separada para poder
// testearla sin depender del reloj real (se le inyecta el momento).
func formatearHora(ahora time.Time) string {
	return "Son las " + ahora.Format("15:04") + ", señor."
}

func (h *Hands) decirFecha() string {
	return formatearFecha(time.Now())
}

// formatearFecha es la lógica pura de decirFecha, separada para poder
// testearla sin depender del reloj real (se le inyecta el momento).
func formatearFecha(ahora time.Time) string {
	meses := [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
	return fmt.Sprintf("Hoy es %d de %s de %d, señor.", ahora.Day(), meses[ahora.Month()-1], ahora.Year())
}

// nivelBateria consulta el nivel de batería vía WMI/CIM (nuevo).
func (h *Hands) nivelBateria() string {
	ps := "(Get-CimInstance -ClassName Win32_Battery | Select-Object -ExpandProperty EstimatedChargeRemaining)"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude leer el nivel de batería, señor."
	}
	nivel := strings.TrimSpace(string(salida))
	if nivel == "" {
		return "No detecté ninguna batería, señor. ¿Es una PC de escritorio?"
	}
	return fmt.Sprintf("La batería está al %s por ciento, señor.", nivel)
}

// obtenerIP devuelve la IP local (LAN) sin depender de PowerShell (nuevo).
func (h *Hands) obtenerIP() string {
	direcciones, err := net.InterfaceAddrs()
	if err != nil {
		return fmt.Sprintf("No pude obtener la IP: %v", err)
	}
	for _, direccion := range direcciones {
		ipNet, ok := direccion.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return fmt.Sprintf("Su dirección IP local es %s, señor.", ip4.String())
		}
	}
	return "No pude encontrar una dirección IP local, señor."
}

var chistes = []string{
	"¿Por qué los pájaros no usan Facebook? Porque ya tienen Twitter.",
	"¿Qué le dijo un semáforo a otro? No me mires que me estoy cambiando.",
	"¿Cómo se llama el campeón de buceo? Miguel Ángel.",
	"¿Por qué el libro de matemáticas está triste? Porque tiene muchos problemas.",
	"Un bit le dice a otro: ¿nos vamos de bytes esta noche?",
	"¿Cómo se despiden los químicos? Ácido un placer.",
}

// contarChiste devuelve un chiste al azar de una lista local (nuevo). Es
// local a propósito: funciona incluso sin OPENAI_API_KEY configurada.
func (h *Hands) contarChiste() string {
	return chistes[rand.Intn(len(chistes))]
}

// copiarAlPortapapeles copia texto al portapapeles de Windows (nuevo).
func (h *Hands) copiarAlPortapapeles(texto string) string {
	if texto == "" {
		return "¿Qué texto desea que copie, señor?"
	}
	textoEscapado := strings.ReplaceAll(texto, "'", "''")
	ps := fmt.Sprintf("Set-Clipboard -Value '%s'", textoEscapado)
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return fmt.Sprintf("No pude copiar el texto: %v", err)
	}
	return "Copiado al portapapeles, señor."
}

func (h *Hands) buscarEnGoogle(consulta string) string {
	if consulta == "" {
		return "¿Qué desea que busque, señor?"
	}
	destino := "https://www.google.com/search?q=" + url.QueryEscape(consulta)
	if err := exec.Command("cmd", "/C", "start", "", destino).Run(); err != nil {
		return fmt.Sprintf("No pude realizar la búsqueda: %v", err)
	}
	return fmt.Sprintf("Buscando %s en Google, señor.", consulta)
}

func (h *Hands) buscarEnYoutube(consulta string) string {
	if consulta == "" {
		return "¿Qué desea que busque en YouTube, señor?"
	}
	destino := "https://www.youtube.com/results?search_query=" + url.QueryEscape(consulta)
	if err := exec.Command("cmd", "/C", "start", "", destino).Run(); err != nil {
		return fmt.Sprintf("No pude abrir YouTube: %v", err)
	}
	return fmt.Sprintf("Buscando %s en YouTube, señor.", consulta)
}

// buscarEnWikipedia busca en la Wikipedia en español (nuevo).
func (h *Hands) buscarEnWikipedia(consulta string) string {
	if consulta == "" {
		return "¿Qué desea que busque en Wikipedia, señor?"
	}
	destino := "https://es.wikipedia.org/wiki/Special:Search?search=" + url.QueryEscape(consulta)
	if err := exec.Command("cmd", "/C", "start", "", destino).Run(); err != nil {
		return fmt.Sprintf("No pude abrir Wikipedia: %v", err)
	}
	return fmt.Sprintf("Buscando %s en Wikipedia, señor.", consulta)
}

// irASitio abre un sitio web directo (no una búsqueda). Completa ".com" y
// "https://" si el usuario no los dijo, para que "ir a youtube" alcance
// (nuevo).
func (h *Hands) irASitio(sitio string) string {
	if sitio == "" {
		return "¿A qué sitio quiere que vaya, señor?"
	}
	destino := sitio
	if !strings.Contains(destino, ".") {
		destino += ".com"
	}
	if !strings.HasPrefix(destino, "http://") && !strings.HasPrefix(destino, "https://") {
		destino = "https://" + destino
	}
	parsed, err := url.Parse(destino)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Sprintf("'%s' no parece una URL válida, señor.", sitio)
	}
	if err := exec.Command("cmd", "/C", "start", "", parsed.String()).Run(); err != nil {
		return fmt.Sprintf("No pude abrir %s: %v", sitio, err)
	}
	return fmt.Sprintf("Abriendo %s, señor.", sitio)
}

func (h *Hands) bloquearPantalla() string {
	if err := exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run(); err != nil {
		return fmt.Sprintf("No pude bloquear la pantalla: %v", err)
	}
	return "Pantalla bloqueada, señor."
}

func (h *Hands) minimizarTodo() string {
	ps := "(New-Object -ComObject Shell.Application).MinimizeAll()"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return fmt.Sprintf("No pude minimizar las ventanas: %v", err)
	}
	return "Ventanas minimizadas, señor."
}

func (h *Hands) capturarPantalla() string {
	carpeta := filepath.Join(os.Getenv("USERPROFILE"), "Pictures")
	ruta := filepath.Join(carpeta, fmt.Sprintf("jarvis_captura_%d.png", time.Now().Unix()))

	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms,System.Drawing;`+
		`$b=[System.Windows.Forms.SystemInformation]::VirtualScreen;`+
		`$bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height;`+
		`$g=[System.Drawing.Graphics]::FromImage($bmp);`+
		`$g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size);`+
		`$bmp.Save('%s')`, ruta)

	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return fmt.Sprintf("No pude tomar la captura: %v", err)
	}
	return fmt.Sprintf("Captura guardada en %s, señor.", ruta)
}

func (h *Hands) abrirCarpeta(nombre string) string {
	ruta := filepath.Join(os.Getenv("USERPROFILE"), nombre)
	// explorer.exe casi siempre devuelve código de salida distinto de cero
	// aunque la carpeta se abra correctamente, así que no tratamos err como fallo real.
	_ = exec.Command("explorer.exe", ruta).Run()
	return fmt.Sprintf("Abriendo %s, señor.", nombre)
}

// abrirPapelera abre la papelera de reciclaje (nuevo).
func (h *Hands) abrirPapelera() string {
	_ = exec.Command("explorer.exe", "shell:RecycleBinFolder").Run()
	return "Abriendo la papelera de reciclaje, señor."
}

func (h *Hands) abrirConfiguracion() string {
	if err := exec.Command("cmd", "/C", "start", "ms-settings:").Run(); err != nil {
		return fmt.Sprintf("No pude abrir la configuración: %v", err)
	}
	return "Abriendo configuración de Windows, señor."
}

// Hablar reproduce texto en voz alta usando la síntesis de voz nativa de
// Windows (SAPI) a través de PowerShell. No requiere ninguna dependencia
// externa de Go.
func (h *Hands) Hablar(texto string) string {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return ""
	}
	fmt.Printf("[JARVIS HABLANDO]: %s\n", texto)

	textoEscapado := strings.ReplaceAll(texto, "'", "''")
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Speech; `+
		`$speak = New-Object System.Speech.Synthesis.SpeechSynthesizer; `+
		`$speak.Speak('%s')`, textoEscapado)

	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		fmt.Printf("[ADVERTENCIA] No se pudo reproducir voz: %v\n", err)
	}
	return "Listo, señor."
}

// === NUEVOS SISTEMA / INFORMACIÓN ===

func (h *Hands) infoCPU() string {
	return h.obtenerOCache("cpu", 0, func() string {
		ps := "(Get-CimInstance Win32_Processor | Select-Object -First 1 | ForEach-Object { $_.Name })"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude obtener información del procesador, señor."
		}
		return fmt.Sprintf("Su procesador es %s, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoRAM() string {
	return h.obtenerOCache("ram", 0, func() string {
		ps := "Get-CimInstance Win32_ComputerSystem | ForEach-Object { [math]::Round($_.TotalPhysicalMemory / 1GB, 1) }"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude leer la memoria RAM, señor."
		}
		return fmt.Sprintf("Tiene %s GB de RAM instalados, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoDisco() string {
	return h.obtenerOCache("disco", 30*time.Second, func() string {
		ps := "Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | ForEach-Object { $d=$_; $f=[math]::Round($d.FreeSpace/1GB,1); $t=[math]::Round($d.Size/1GB,1); \"$($d.DeviceID) $f GB libres de $t GB\" }"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude leer el disco, señor."
		}
		return fmt.Sprintf("Almacenamiento: %s", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoSO() string {
	return h.obtenerOCache("so", 0, func() string {
		ps := "(Get-CimInstance Win32_OperatingSystem | ForEach-Object { $_.Caption })"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude obtener la versión del sistema, señor."
		}
		return fmt.Sprintf("Su sistema operativo es %s, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoUptime() string {
	return h.obtenerOCache("uptime", 30*time.Second, func() string {
		ps := "$boot=(Get-CimInstance Win32_OperatingSystem).LastBootUpTime; $up=(Get-Date)-$boot; \"{0} días, {1} horas y {2} minutos\" -f $up.Days, $up.Hours, $up.Minutes"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude calcular el tiempo activo, señor."
		}
		return fmt.Sprintf("Llevo encendido %s, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoUsuario() string {
	return fmt.Sprintf("El usuario actual es %s, señor.", os.Getenv("USERNAME"))
}

func (h *Hands) infoPC() string {
	return fmt.Sprintf("El nombre del equipo es %s, señor.", os.Getenv("COMPUTERNAME"))
}

func (h *Hands) infoArquitectura() string {
	return h.obtenerOCache("arq", 0, func() string {
		ps := "(Get-CimInstance Win32_ComputerSystem).SystemType"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude obtener la arquitectura, señor."
		}
		return fmt.Sprintf("La arquitectura del sistema es %s, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoProgramas() string {
	return h.obtenerOCache("programas", 60*time.Second, func() string {
		ps := "(Get-ItemProperty 'HKLM:\\Software\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*' | Measure-Object).Count"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude contar los programas instalados, señor."
		}
		return fmt.Sprintf("Tiene aproximadamente %s programas instalados, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoProcesos() string {
	return h.obtenerOCache("procesos", 10*time.Second, func() string {
		ps := "(Get-Process | Measure-Object).Count"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude contar los procesos, señor."
		}
		return fmt.Sprintf("Hay %s procesos ejecutándose, señor.", strings.TrimSpace(string(salida)))
	})
}

func (h *Hands) infoNucleos() string {
	return h.obtenerOCache("nucleos", 0, func() string {
		ps := "(Get-CimInstance Win32_Processor | ForEach-Object { $_.NumberOfCores })"
		salida, err := exec.Command("powershell", "-Command", ps).Output()
		if err != nil {
			return "No pude obtener los núcleos, señor."
		}
		nucleos := strings.TrimSpace(string(salida))
		lineas := strings.Split(nucleos, "\n")
		total := 0
		for _, l := range lineas {
			n := 0
			fmt.Sscanf(l, "%d", &n)
			total += n
		}
		return fmt.Sprintf("Tiene %d núcleos de procesamiento, señor.", total)
	})
}

func (h *Hands) infoPantalla() string {
	ps := "Add-Type -AssemblyName System.Windows.Forms; $s=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; \"$($s.Width)x$($s.Height)\""
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener la resolución, señor."
	}
	return fmt.Sprintf("La resolución de pantalla es %s, señor.", strings.TrimSpace(string(salida)))
}

func (h *Hands) infoIdioma() string {
	ps := "(Get-CimInstance Win32_OperatingSystem).MUILanguages[0]"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener el idioma, señor."
	}
	return fmt.Sprintf("El idioma del sistema es %s, señor.", strings.TrimSpace(string(salida)))
}

func (h *Hands) infoZonaHoraria() string {
	ps := "(Get-CimInstance Win32_TimeZone).Caption"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener la zona horaria, señor."
	}
	return fmt.Sprintf("La zona horaria es %s, señor.", strings.TrimSpace(string(salida)))
}

func (h *Hands) infoTemperatura() string {
	ps := "Get-CimInstance -Namespace 'root/wmi' -ClassName MSAcpi_ThermalZoneTemperature 2>$null | ForEach-Object { [math]::Round(($_.CurrentTemperature/10-273.15),1) }"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil || strings.TrimSpace(string(salida)) == "" {
		return "No pude leer la temperatura, señor. Es posible que su equipo no tenga sensor térmico accesible."
	}
	return fmt.Sprintf("La temperatura actual es de %s grados Celsius, señor.", strings.TrimSpace(string(salida)))
}

// === NUEVAS RED ===

func (h *Hands) ipPublica() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "No pude obtener la IP pública, señor."
	}
	defer resp.Body.Close()
	ip, _ := io.ReadAll(resp.Body)
	if len(ip) == 0 {
		return "No pude obtener la IP pública, señor."
	}
	return fmt.Sprintf("Su dirección IP pública es %s, señor.", strings.TrimSpace(string(ip)))
}

func (h *Hands) hacerPing(host string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ping", "-n", "1", host)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("No hubo respuesta de %s, señor. Puede que esté caído o no tenga conexión.", host)
	}
	return fmt.Sprintf("Recibí respuesta de %s, su conexión está funcionando, señor.", host)
}

func (h *Hands) infoDNS() string {
	ps := "Get-DnsClientServerAddress -AddressFamily IPv4 | Where-Object {$_.ServerAddresses} | ForEach-Object { $_.ServerAddresses -join ', ' }"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener los DNS, señor."
	}
	dns := strings.TrimSpace(string(salida))
	if dns == "" {
		return "No encontré servidores DNS configurados, señor."
	}
	return fmt.Sprintf("Sus servidores DNS son: %s, señor.", dns)
}

func (h *Hands) infoMAC() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "No pude obtener la dirección MAC, señor."
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && len(iface.HardwareAddr) > 0 {
			return fmt.Sprintf("La dirección MAC de su red activa es %s, señor.", iface.HardwareAddr.String())
		}
	}
	return "No encontré una dirección MAC activa, señor."
}

func (h *Hands) infoVelocidadRed() string {
	ps := "Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | Select-Object -First 1 | ForEach-Object { [math]::Round($_.LinkSpeed/1000,1) }"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener la velocidad de red, señor."
	}
	vel := strings.TrimSpace(string(salida))
	if vel == "" {
		return "No detecté una conexión de red activa, señor."
	}
	return fmt.Sprintf("Su velocidad de conexión es de %s Gbps, señor.", vel)
}

func (h *Hands) infoWifi() string {
	ps := "(netsh wlan show interfaces | Select-String 'SSID' | Select-String -NotMatch 'BSSID')"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener la red WiFi, señor."
	}
	texto := strings.TrimSpace(string(salida))
	if texto == "" {
		return "No está conectado a ninguna red WiFi, señor."
	}
	partes := strings.Split(texto, ":")
	if len(partes) >= 2 {
		return fmt.Sprintf("Está conectado a la red '%s', señor.", strings.TrimSpace(partes[1]))
	}
	return fmt.Sprintf("Red WiFi detectada: %s, señor.", texto)
}

func (h *Hands) infoConexiones() string {
	ps := "(Get-NetTCPConnection | Where-Object {$_.State -eq 'Established'} | Measure-Object).Count"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude contar las conexiones activas, señor."
	}
	return fmt.Sprintf("Tiene %s conexiones activas, señor.", strings.TrimSpace(string(salida)))
}

func (h *Hands) infoInterfaces() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "No pude obtener las interfaces de red, señor."
	}
	activas := 0
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 {
			activas++
		}
	}
	return fmt.Sprintf("Tiene %d interfaces de red activas de un total de %d, señor.", activas, len(interfaces))
}

// === NUEVAS ENERGÍA ===

func (h *Hands) suspender() string {
	if err := exec.Command("powershell", "-Command", "(Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false))").Run(); err != nil {
		return "No pude suspender el equipo, señor."
	}
	return "Suspendiendo el equipo, señor."
}

func (h *Hands) hibernar() string {
	if err := exec.Command("powershell", "-Command", "(Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Hibernate', $false, $false))").Run(); err != nil {
		return "No pude hibernar el equipo, señor."
	}
	return "Hibernando el equipo, señor."
}

func (h *Hands) reiniciar() string {
	if err := exec.Command("shutdown", "/r", "/t", "5").Run(); err != nil {
		return "No pude iniciar el reinicio, señor."
	}
	return "Reiniciando el equipo en 5 segundos, señor."
}

func (h *Hands) cerrarSesion() string {
	if err := exec.Command("shutdown", "/l").Run(); err != nil {
		return "No pude cerrar la sesión, señor."
	}
	return "Cerrando su sesión, señor."
}

func (h *Hands) apagarMonitor() string {
	ps := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class Monitor { [DllImport(\"user32.dll\")] public static extern int SendMessage(int hWnd, int hMsg, int wParam, int lParam); }'; [Monitor]::SendMessage(-1, 0x0112, 0xF170, 2)"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude apagar el monitor, señor."
	}
	return "Monitor apagado, señor."
}

func (h *Hands) brillo(accion string) string {
	nivel := 50
	if accion == "up" {
		actual := h.obtenerBrillo()
		nivel = actual + 20
		if nivel > 100 {
			nivel = 100
		}
	} else {
		actual := h.obtenerBrillo()
		nivel = actual - 20
		if nivel < 0 {
			nivel = 0
		}
	}
	return h.establecerBrillo(nivel)
}

func (h *Hands) brilloPorcentaje(cmd string) string {
	resto := strings.TrimSpace(strings.Replace(cmd, "brillo al ", "", 1))
	resto = strings.TrimSpace(strings.Replace(resto, "%", "", 1))
	nivel := 0
	if _, err := fmt.Sscanf(resto, "%d", &nivel); err != nil || nivel < 0 || nivel > 100 {
		return "Dígame un nivel de brillo entre 0 y 100, señor."
	}
	return h.establecerBrillo(nivel)
}

func (h *Hands) obtenerBrillo() int {
	ps := "(Get-CimInstance -Namespace root\\wmi -ClassName WmiMonitorBrightness | Select-Object -ExpandProperty CurrentBrightness)"
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return 50
	}
	nivel := 50
	fmt.Sscanf(strings.TrimSpace(string(salida)), "%d", &nivel)
	return nivel
}

func (h *Hands) establecerBrillo(nivel int) string {
	ps := fmt.Sprintf("(Get-WmiObject -Namespace root\\wmi -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)", nivel)
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return fmt.Sprintf("No pude establecer el brillo, señor.")
	}
	return fmt.Sprintf("Brillo al %d%%, señor.", nivel)
}

func (h *Hands) modoAvion() string {
	ps := `$ad=Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Up'}; if($ad){try{$ad | Disable-NetAdapter -Confirm:$false -ErrorAction Stop; "Modo avión activado, señor."}catch{"No pude activar modo avión. ¿Ejecuta como administrador, señor?"}}else{$ad=Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Disabled'}; if($ad){try{$ad | Enable-NetAdapter -Confirm:$false -ErrorAction Stop; "Modo avión desactivado, señor."}catch{"No pude desactivar modo avión. ¿Ejecuta como administrador, señor?"}}else{"No encontré adaptadores de red física, señor."}}`
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude cambiar el modo avión, señor."
	}
	return strings.TrimSpace(string(salida))
}

// === NUEVAS VENTANAS ===

func (h *Hands) maximizarVentana() string {
	ps := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class Win { [DllImport(\"user32.dll\")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow); [DllImport(\"user32.dll\")] public static extern IntPtr GetForegroundWindow(); }'; $h=0x0; do { Start-Sleep -Milliseconds 100; $h=[Win]::GetForegroundWindow() } while($h -eq 0); [Win]::ShowWindowAsync($h, 3)"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude maximizar la ventana, señor."
	}
	return "Ventana maximizada, señor."
}

func (h *Hands) minimizarVentana() string {
	ps := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class Win { [DllImport(\"user32.dll\")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow); [DllImport(\"user32.dll\")] public static extern IntPtr GetForegroundWindow(); }'; $h=[Win]::GetForegroundWindow(); [Win]::ShowWindowAsync($h, 6)"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude minimizar la ventana, señor."
	}
	return "Ventana minimizada, señor."
}

func (h *Hands) restaurarVentana() string {
	ps := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class Win { [DllImport(\"user32.dll\")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow); [DllImport(\"user32.dll\")] public static extern IntPtr GetForegroundWindow(); }'; $h=[Win]::GetForegroundWindow(); [Win]::ShowWindowAsync($h, 9)"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude restaurar la ventana, señor."
	}
	return "Ventana restaurada, señor."
}

func (h *Hands) cambiarVentana() string {
	if err := exec.Command("powershell", "-Command", "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class KB { [DllImport(\"user32.dll\")] public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, uint dwExtraInfo); }'; [KB]::keybd_event(0x09, 0, 0, 0); [KB]::keybd_event(0x09, 0, 2, 0)").Run(); err != nil {
		return "No pude cambiar de ventana, señor."
	}
	return "Cambiando de ventana, señor."
}

func (h *Hands) mostrarEscritorio() string {
	ps := "(New-Object -ComObject Shell.Application).ToggleDesktop()"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude mostrar el escritorio, señor."
	}
	return "Mostrando el escritorio, señor."
}

func (h *Hands) organizarVentanas() string {
	ps := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class Win { [DllImport(\"user32.dll\")] public static extern bool PostMessage(IntPtr hWnd, uint Msg, int wParam, int lParam); }'; $shell=(New-Object -ComObject Shell.Application); $shell.TileHorizontally()"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude organizar las ventanas, señor."
	}
	return "Ventanas organizadas, señor."
}

func (h *Hands) cerrarVentanaActiva() string {
	if err := exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait('%{F4}')").Run(); err != nil {
		return "No pude cerrar la ventana, señor."
	}
	return "Ventana cerrada, señor."
}

// === NUEVAS ARCHIVOS ===

func (h *Hands) listarDirectorio() string {
	ruta, _ := os.Getwd()
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return fmt.Sprintf("No pude leer el directorio actual: %v", err)
	}
	archivos := 0
	carpetas := 0
	for _, e := range entradas {
		if e.IsDir() {
			carpetas++
		} else {
			archivos++
		}
	}
	return fmt.Sprintf("En %s hay %d carpetas y %d archivos, señor.", ruta, carpetas, archivos)
}

func (h *Hands) crearCarpeta(cmd string) string {
	nombre := strings.TrimSpace(strings.Replace(cmd, "crear carpeta ", "", 1))
	nombre = strings.TrimSpace(strings.Replace(nombre, "crear directorio ", "", 1))
	if nombre == "" {
		nombre = fmt.Sprintf("nueva_carpeta_%d", time.Now().Unix())
	}
	ruta := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", nombre)
	if err := os.MkdirAll(ruta, 0o700); err != nil {
		return fmt.Sprintf("No pude crear la carpeta: %v", err)
	}
	return fmt.Sprintf("Carpeta '%s' creada en el escritorio, señor.", nombre)
}

func (h *Hands) crearArchivo(cmd string) string {
	nombre := strings.TrimSpace(strings.Replace(cmd, "crear archivo ", "", 1))
	nombre = strings.TrimSpace(strings.Replace(nombre, "nuevo archivo ", "", 1))
	if nombre == "" {
		nombre = fmt.Sprintf("nuevo_archivo_%d.txt", time.Now().Unix())
	}
	ruta := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", nombre)
	if err := os.WriteFile(ruta, []byte(""), 0o600); err != nil {
		return fmt.Sprintf("No pude crear el archivo: %v", err)
	}
	return fmt.Sprintf("Archivo '%s' creado en el escritorio, señor.", nombre)
}

func (h *Hands) vaciarPapelera() string {
	ps := "(New-Object -ComObject Shell.Application).NameSpace(0x0a).Items() | ForEach-Object { $_.InvokeVerb('delete') }"
	if err := exec.Command("powershell", "-Command", ps).Run(); err != nil {
		return "No pude vaciar la papelera, señor."
	}
	return "Papelera vaciada, señor."
}

func (h *Hands) rutaActual() string {
	ruta, err := os.Getwd()
	if err != nil {
		return "No pude obtener la ruta actual, señor."
	}
	return fmt.Sprintf("La ruta actual es %s, señor.", ruta)
}

// === NUEVAS DIVERSIÓN ===

func (h *Hands) lanzarMoneda() string {
	if rand.Intn(2) == 0 {
		return "Cara, señor."
	}
	return "Cruz, señor."
}

func (h *Hands) tirarDado() string {
	return fmt.Sprintf("Salió %d, señor.", rand.Intn(6)+1)
}

func (h *Hands) numeroAleatorio() string {
	return fmt.Sprintf("Su número aleatorio es %d, señor.", rand.Intn(100)+1)
}

var cumplidos = []string{
	"Es usted una persona increíble, señor.",
	"Admiro su inteligencia, señor.",
	"Hoy está especialmente brillante, señor.",
	"Tiene un gusto excelente, señor.",
	"Es un placer trabajar para usted, señor.",
	"Su determinación es inspiradora, señor.",
}

func (h *Hands) decirCumplido() string {
	return cumplidos[rand.Intn(len(cumplidos))]
}

func (h *Hands) decidir() string {
	if rand.Intn(2) == 0 {
		return "Sí, señor."
	}
	return "No, señor."
}

var colores = []string{"rojo", "azul", "verde", "amarillo", "naranja", "violeta", "rosa", "negro", "blanco", "gris", "marrón", "turquesa", "dorado", "plateado"}

func (h *Hands) colorAleatorio() string {
	return fmt.Sprintf("Su color aleatorio es %s, señor.", colores[rand.Intn(len(colores))])
}

var motivacion = []string{
	"El éxito no es la clave de la felicidad. La felicidad es la clave del éxito.",
	"El único modo de hacer un gran trabajo es amar lo que haces.",
	"No cuentes los días, haz que los días cuenten.",
	"El futuro pertenece a quienes creen en la belleza de sus sueños.",
	"Todo lo que siempre has querido está al otro lado del miedo.",
	"La mejor manera de predecir el futuro es crearlo.",
}

func (h *Hands) fraseMotivadora() string {
	return fmt.Sprintf("%s, señor.", motivacion[rand.Intn(len(motivacion))])
}

var trabalenguasLista = []string{
	"Tres tristes tigres tragan trigo en un trigal.",
	"El perro de San Roque no tiene rabo porque Ramón Ramírez se lo ha robado.",
	"Pepe pisa pipas, pipas pisa Pepe.",
	"El cielo está enladrillado ¿quién lo desenladrillará?",
	"Pablito clavó un clavito, ¿qué clavo clavó Pablito?",
}

func (h *Hands) trabalenguas() string {
	return fmt.Sprintf("Repita conmigo: %s", trabalenguasLista[rand.Intn(len(trabalenguasLista))])
}

func (h *Hands) signoZodiacal() string {
	now := time.Now()
	dia := now.YearDay()
	if dia <= 19 { return "Su signo es Capricornio, señor." }
	if dia <= 49 { return "Su signo es Acuario, señor." }
	if dia <= 79 { return "Su signo es Piscis, señor." }
	if dia <= 110 { return "Su signo es Aries, señor." }
	if dia <= 140 { return "Su signo es Tauro, señor." }
	if dia <= 171 { return "Su signo es Géminis, señor." }
	if dia <= 203 { return "Su signo es Cáncer, señor." }
	if dia <= 234 { return "Su signo es Leo, señor." }
	if dia <= 265 { return "Su signo es Virgo, señor." }
	if dia <= 295 { return "Su signo es Libra, señor." }
	if dia <= 325 { return "Su signo es Escorpio, señor." }
	if dia <= 355 { return "Su signo es Sagitario, señor." }
	return "Su signo es Capricornio, señor."
}

func (h *Hands) diasParaNavidad() string {
	now := time.Now()
	navidad := time.Date(now.Year(), time.December, 25, 0, 0, 0, 0, now.Location())
	if now.After(navidad) {
		navidad = time.Date(now.Year()+1, time.December, 25, 0, 0, 0, 0, now.Location())
	}
	dias := int(navidad.Sub(now).Hours() / 24)
	return fmt.Sprintf("Faltan %d días para Navidad, señor.", dias)
}

// === NUEVAS VOZ / SONIDO ===

func (h *Hands) volumenActual() string {
	ps := `Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class Vol { [DllImport("winmm.dll")] public static extern int waveOutGetVolume(IntPtr hwo, out uint dwVolume); }'; $v=[uint32]0; [Vol]::waveOutGetVolume([IntPtr]::Zero, [ref]$v); $left=[math]::Round(($v -band 0xFFFF)/65535.0*100); $right=[math]::Round(($v -shr 16)/65535.0*100); [math]::Max($left,$right)`
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude leer el volumen actual, señor."
	}
	nivel := strings.TrimSpace(string(salida))
	return fmt.Sprintf("El volumen actual es de %s por ciento, señor.", nivel)
}

func (h *Hands) fechaCompleta() string {
	now := time.Now()
	diasSemana := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	meses := []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
	return fmt.Sprintf("Hoy es %s %d de %s de %d, señor.", diasSemana[now.Weekday()], now.Day(), meses[now.Month()-1], now.Year())
}

func (h *Hands) saludarPersonalizado() string {
	now := time.Now()
	msg := "Hola, señor."
	hora := now.Hour()
	if hora < 12 {
		msg = "Buenos días, señor."
	} else if hora < 18 {
		msg = "Buenas tardes, señor."
	} else {
		msg = "Buenas noches, señor."
	}
	return msg
}

// === NUEVAS MISC ===

func (h *Hands) tomarNotaRapida() string {
	return "Dígame qué nota quiere guardar, señor. Por ejemplo: 'recordá que tengo reunión mañana'."
}

func (h *Hands) ejercicioRespiración() string {
	return "Inhalé profundamente por 4 segundos... sostenga 4 segundos... exhale lentamente por 6 segundos. Repita 3 veces, señor."
}

func (h *Hands) recordatorioAgua() string {
	return "Recuerde beber agua, señor. La hidratación es importante para su salud."
}

func (h *Hands) recordatorioEstiramiento() string {
	return "Es momento de estirarse, señor. Levántese y estire los brazos y la espalda por unos segundos."
}

func (h *Hands) buscarArchivos(cmd string) string {
	consulta := strings.TrimSpace(strings.Replace(cmd, "buscar archivo ", "", 1))
	consulta = strings.TrimSpace(strings.Replace(consulta, "buscar archivos ", "", 1))
	if consulta == "" {
		return "¿Qué archivo desea buscar, señor?"
	}
	// Sanitizar: solo caracteres seguros para evitar inyección de PowerShell
	sanitizado := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == ' ' || r == '-' || r == '_' || r == '.' || r == '(' || r == ')' {
			return r
		}
		return -1
	}, consulta)
	ps := fmt.Sprintf(`Get-ChildItem -Path "$env:USERPROFILE" -Recurse -Filter "*%s*" -ErrorAction SilentlyContinue | Select-Object -First 10 -ExpandProperty FullName`, sanitizado)
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return fmt.Sprintf("No pude buscar archivos, señor: %v", err)
	}
	lineas := strings.Split(strings.TrimSpace(string(salida)), "\n")
	lineasFiltradas := make([]string, 0, len(lineas))
	for _, l := range lineas {
		l = strings.TrimSpace(l)
		if l != "" {
			lineasFiltradas = append(lineasFiltradas, l)
		}
	}
	if len(lineasFiltradas) == 0 {
		return fmt.Sprintf("No encontré archivos con '%s', señor.", consulta)
	}
	fmt.Printf("Resultados de búsqueda para '%s':\n", consulta)
	for _, l := range lineasFiltradas {
		fmt.Println(" -", l)
	}
	return fmt.Sprintf("Encontré %d archivos con '%s'. Mire la consola para ver las rutas, señor.", len(lineasFiltradas), consulta)
}

func (h *Hands) consultarClima() string {
	if h.ClimaKey != "" {
		url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=Buenos+Ayres,ar&appid=%s&units=metric&lang=es", h.ClimaKey)
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var resultado map[string]any
			if json.Unmarshal(body, &resultado) == nil {
				if main, ok := resultado["main"].(map[string]any); ok {
					if temp, ok := main["temp"].(float64); ok {
						desc := ""
						if weather, ok := resultado["weather"].([]any); ok && len(weather) > 0 {
							if w, ok := weather[0].(map[string]any); ok {
								desc, _ = w["description"].(string)
							}
						}
						return fmt.Sprintf("Clima actual: %.0f°C, %s, señor.", temp, desc)
					}
				}
			}
		}
	}
	resp, err := http.Get("https://wttr.in?format=%C+%t&lang=es")
	if err != nil {
		return "No pude consultar el clima, señor. ¿Tiene conexión a internet?"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	texto := strings.TrimSpace(string(body))
	if texto == "" {
		return "No pude obtener información del clima, señor."
	}
	return fmt.Sprintf("El clima actual: %s, señor.", texto)
}

func (h *Hands) consultarNoticias() string {
	if h.NewsKey != "" && h.NewsKey != "demo" {
		url := fmt.Sprintf("https://newsapi.org/v2/top-headlines?country=ar&apiKey=%s", h.NewsKey)
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var resultado map[string]any
			if json.Unmarshal(body, &resultado) == nil {
				if articulos, ok := resultado["articles"].([]any); ok && len(articulos) > 0 {
					titulares := make([]string, 0, min(5, len(articulos)))
					for i, a := range articulos {
						if i >= 5 {
							break
						}
						if art, ok := a.(map[string]any); ok {
							if titulo, ok := art["title"].(string); ok {
								titulares = append(titulares, titulo)
							}
						}
					}
					if len(titulares) > 0 {
						fmt.Println("Últimas noticias:")
						for i, t := range titulares {
							fmt.Printf(" %d. %s\n", i+1, t)
						}
						return fmt.Sprintf("Últimas %d noticias, señor. Mire la consola.", len(titulares))
					}
				}
			}
		}
	}
	return h.noticiasFallback()
}

func (h *Hands) noticiasFallback() string {
	ps := `(Invoke-WebRequest -Uri "https://news.google.com/rss?hl=es-419&gl=AR&ceid=AR:es-419" -UseBasicParsing).Content`
	salida, err := exec.Command("powershell", "-Command", ps).Output()
	if err != nil {
		return "No pude obtener las noticias, señor."
	}
	lineas := strings.Split(string(salida), "\n")
	var titulares []string
	for _, l := range lineas {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "<title>") && !strings.Contains(l, "Google Noticias") {
			t := strings.TrimPrefix(l, "<title>")
			t = strings.TrimSuffix(t, "</title>")
			titulares = append(titulares, t)
		}
	}
	if len(titulares) == 0 {
		return "No pude obtener las noticias, señor."
	}
	if len(titulares) > 5 {
		titulares = titulares[:5]
	}
	fmt.Println("Últimas noticias:")
	for i, t := range titulares {
		fmt.Printf(" %d. %s\n", i+1, t)
	}
	return fmt.Sprintf("Últimas %d noticias, señor. Mire la consola.", len(titulares))
}
