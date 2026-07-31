package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GeneradorFullstackIA es la interfaz que un asistente de código (por ejemplo,
// Ollama) implementa para mejorar los proyectos generados por Hands. La
// petición devuelve un archivo completo listo para escribir.
type GeneradorFullstackIA interface {
	Disponible() bool
	// ConsultarDesarrollo pide código para una petición de feature. Devuelve
	// la respuesta cruda en el formato "ARCHIVO: ... CONTENIDO: ... EXPLICACION: ...".
	ConsultarDesarrollo(peticion string) (string, string, error)
}

// ProyectoEjecutando registra una app levantada por JarvisOS para poder
// consultar su estado y detenerla después.
type ProyectoEjecutando struct {
	Nombre string
	Ruta   string
	Puerto int
	PID    int
}

var (
	prefijosCrearProyecto = []string{
		"crear proyecto web ", "crear una app web ", "crear una web ",
		"crear proyecto ", "crear una aplicacion web ", "crear aplicacion web ",
		"crea un proyecto web ", "crea una web ", "crea un proyecto ",
		"crea una app web ",
		"crea un proyecto web ", "crea un proyecto ",
		"crea una web ",
		"hace un proyecto web ", "hace un proyecto ", "hace una web ",
		"hace una app web ",
		"nueva app web ", "nuevo proyecto web ", "nuevo proyecto ",
		"scaffold ", "armar proyecto web ", "arma un proyecto web ", "arma un proyecto ",
		"hacer proyecto web ", "hacer un proyecto web ", "hacer un proyecto ",
	}
	prefijosCompilarProyecto = []string{
		"compilar proyecto ", "compilar el proyecto ", "compila el proyecto ", "compila proyecto ",
		"compilá el proyecto ", "build del proyecto ", "build del ", "compila la app ", "compile ",
	}
	prefijosEjecutarProyecto = []string{
		"ejecutar proyecto ", "ejecutar el proyecto ", "ejecuta el proyecto ", "ejecuta proyecto ",
		"ejecutá el proyecto ",
		"correr el proyecto ", "corre el proyecto ", "correr proyecto ", "corré el proyecto ",
		"levantar el proyecto ", "levanta el proyecto ", "levantar proyecto ", "levantá el proyecto ",
		"iniciar el proyecto ", "inicia el proyecto ", "iniciar proyecto ", "iniciá el proyecto ",
		"abre el proyecto ", "abrir el proyecto ", "abrir proyecto ",
	}
	prefijosDetenerProyecto = []string{
		"detener proyecto ", "detener el proyecto ", "detener la app ",
		"detener la aplicacion ",
		"para el proyecto ", "parar el proyecto ", "parar proyecto ", "pará el proyecto ",
		"cerrar el proyecto ", "cierra el proyecto ", "cerrar proyecto ", "cerrá el proyecto ",
		"apagar el proyecto ", "apaga el proyecto ", "apagar proyecto ", "apagá el proyecto ",
		"matar el proyecto ", "mata el proyecto ", "matar proyecto ", "matá el proyecto ",
		"cerrar la app ", "cerrar la aplicacion ",
	}
	prefijosEstadoProyecto = []string{
		"estado del proyecto ", "esta corriendo el proyecto ", "esta activo el proyecto ",
		"sigue andando el proyecto ", "sigue corriendo el proyecto ", "esta levantado el proyecto ",
	}
)

func sanitizarNombreProyecto(nombre string) string {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	var b strings.Builder
	for _, r := range nombre {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(strings.Trim(b.String(), "-"), " ")
}

func (h *Hands) proyectoRuta(nombre string) string {
	return filepath.Join(h.WorkspaceRoot, sanitizarNombreProyecto(nombre))
}

// crearProyectoWeb genera una aplicación fullstack real (backend Go con la
// librería estándar + frontend HTML/CSS/JS puro), la compila y la deja lista
// para ejecutar. No requiere IA: es el rol "ingeniero" de Jarvis.
func (h *Hands) crearProyectoWeb(comando string) string {
	if strings.TrimSpace(h.WorkspaceRoot) == "" {
		return "No tengo configurada una carpeta de proyectos, señor."
	}
	nombre := normalizarNombreProyecto(extraerObjeto(comando, prefijosCrearProyecto))
	if nombre == "" {
		nombre = "miapp"
	}
	nombre = sanitizarNombreProyecto(nombre)
	ruta := h.proyectoRuta(nombre)
	if existeRuta(ruta) {
		return fmt.Sprintf("Ya existe un proyecto '%s' en %s, señor. Elija otro nombre.", nombre, ruta)
	}
	files := plantillasProyecto(nombre)
	for rel, contenido := range files {
		destino := filepath.Join(ruta, rel)
		if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
			return "No pude crear la carpeta del proyecto: " + err.Error()
		}
		if err := os.WriteFile(destino, []byte(contenido), 0o644); err != nil {
			return fmt.Sprintf("No pude escribir '%s': %v", rel, err)
		}
	}
	salida, err := compilarProyectoEn(ruta, nombre)
	if err != nil {
		return fmt.Sprintf("Creé el proyecto '%s' en %s, pero la compilación falló: %s", nombre, ruta, strings.TrimSpace(salida))
	}
	return fmt.Sprintf("Proyecto '%s' creado y compilando en %s, señor. Diga 'ejecutar proyecto %s' para levantarlo en el navegador.", nombre, ruta, nombre)
}

func (h *Hands) compilarProyecto(comando string) string {
	nombre := sanitizarNombreProyecto(normalizarNombreProyecto(extraerObjeto(comando, prefijosCompilarProyecto)))
	if nombre == "" {
		return "¿Qué proyecto quiere que compile, señor? Diga 'compilar proyecto' seguido del nombre."
	}
	ruta := h.proyectoRuta(nombre)
	if !existeRuta(ruta) {
		return fmt.Sprintf("No encontré el proyecto '%s', señor.", nombre)
	}
	salida, err := compilarProyectoEn(ruta, nombre)
	if err != nil {
		return fmt.Sprintf("La compilación de '%s' falló: %s", nombre, strings.TrimSpace(salida))
	}
	return fmt.Sprintf("Proyecto '%s' compilado sin errores, señor.", nombre)
}

func (h *Hands) ejecutarProyecto(comando string) string {
	nombre := sanitizarNombreProyecto(normalizarNombreProyecto(extraerObjeto(comando, prefijosEjecutarProyecto)))
	if nombre == "" {
		return "¿Qué proyecto quiere que ejecute, señor? Diga 'ejecutar proyecto' seguido del nombre."
	}
	ruta := h.proyectoRuta(nombre)
	if !existeRuta(ruta) {
		return fmt.Sprintf("No encontré el proyecto '%s', señor.", nombre)
	}
	h.proyectosMu.Lock()
	if p, ok := h.proyectos[nombre]; ok {
		h.proyectosMu.Unlock()
		return fmt.Sprintf("El proyecto '%s' ya está corriendo en http://127.0.0.1:%d, señor.", nombre, p.Puerto)
	}
	h.proyectosMu.Unlock()

	salida, err := compilarProyectoEn(ruta, nombre)
	if err != nil {
		return fmt.Sprintf("No pude compilar '%s' para ejecutarlo: %s", nombre, strings.TrimSpace(salida))
	}
	puerto := h.puertoLibre()
	var logWriter *os.File
	logWriter, err = os.OpenFile(filepath.Join(ruta, "jarvis.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logWriter = nil
	}
	cmd := exec.Command(filepath.Join(ruta, nombre+".exe"))
	cmd.Dir = ruta
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(puerto))
	if logWriter != nil {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}
	if err := cmd.Start(); err != nil {
		return "No pude iniciar el proyecto: " + err.Error()
	}
	h.proyectosMu.Lock()
	h.proyectos[nombre] = &ProyectoEjecutando{Nombre: nombre, Ruta: ruta, Puerto: puerto, PID: cmd.Process.Pid}
	h.proyectosMu.Unlock()

	destino := fmt.Sprintf("http://127.0.0.1:%d", puerto)
	if h.esperarServidor(puerto, 20*time.Second) {
		_ = exec.Command("cmd", "/C", "start", "", destino).Run()
		return fmt.Sprintf("Proyecto '%s' corriendo en %s, señor. Ya está abierto en el navegador.", nombre, destino)
	}
	return fmt.Sprintf("Inicié el proyecto '%s', pero todavía no responde en %s. Revise jarvis.log, señor.", nombre, destino)
}

func (h *Hands) detenerProyecto(comando string) string {
	nombre := sanitizarNombreProyecto(normalizarNombreProyecto(extraerObjeto(comando, prefijosDetenerProyecto)))
	if nombre == "" {
		return "¿Qué proyecto quiere que detenga, señor?"
	}
	h.proyectosMu.Lock()
	p, ok := h.proyectos[nombre]
	if ok {
		delete(h.proyectos, nombre)
	}
	h.proyectosMu.Unlock()
	if !ok {
		return fmt.Sprintf("El proyecto '%s' no está corriendo, señor.", nombre)
	}
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(p.PID), "/T", "/F").Run()
	return fmt.Sprintf("Proyecto '%s' detenido, señor.", nombre)
}

func (h *Hands) estadoProyecto(comando string) string {
	nombre := sanitizarNombreProyecto(normalizarNombreProyecto(extraerObjeto(comando, prefijosEstadoProyecto)))
	if nombre == "" {
		h.proyectosMu.Lock()
		names := make([]string, 0, len(h.proyectos))
		for n := range h.proyectos {
			names = append(names, n)
		}
		sort.Strings(names)
		var lineas []string
		for _, n := range names {
			p := h.proyectos[n]
			lineas = append(lineas, fmt.Sprintf("%s en http://127.0.0.1:%d", n, p.Puerto))
		}
		h.proyectosMu.Unlock()
		if len(lineas) == 0 {
			return "No hay proyectos corriendo, señor."
		}
		return "Proyectos activos: " + strings.Join(lineas, ", ") + "."
	}
	h.proyectosMu.Lock()
	p, ok := h.proyectos[nombre]
	h.proyectosMu.Unlock()
	if !ok {
		return fmt.Sprintf("El proyecto '%s' no está corriendo, señor.", nombre)
	}
	return fmt.Sprintf("El proyecto '%s' está activo en http://127.0.0.1:%d, señor.", nombre, p.Puerto)
}

func (h *Hands) listarProyectos() string {
	if strings.TrimSpace(h.WorkspaceRoot) == "" {
		return "No tengo configurada una carpeta de proyectos, señor."
	}
	entradas, err := os.ReadDir(h.WorkspaceRoot)
	if err != nil {
		return "No pude leer la carpeta de proyectos: " + err.Error()
	}
	var nombres []string
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(h.WorkspaceRoot, e.Name(), "main.go")); err == nil {
			nombres = append(nombres, e.Name())
		}
	}
	if len(nombres) == 0 {
		return fmt.Sprintf("No hay proyectos en %s, señor.", h.WorkspaceRoot)
	}
	sort.Strings(nombres)
	return fmt.Sprintf("Proyectos en %s: %s. Para crear uno nuevo diga 'crear proyecto web' con un nombre, señor.", h.WorkspaceRoot, strings.Join(nombres, ", "))
}

// maxIteracionesDesarrollo es cuántas veces Jarvis le devuelve a la IA el
// error de verificación para que corrija antes de rendirse (estilo Claude Code:
// editar, verificar y reparar hasta que quede verde).
const maxIteracionesDesarrollo = 3

// mejorarProyecto usa la IA (si está disponible) para escribir una mejora real
// en un proyecto existente y la itera hasta que compila, pasa vet y los tests.
// Nunca ejecuta el resultado.
func (h *Hands) mejorarProyecto(comando string) string {
	nombre, feature := extraerNombreYFeatureProyecto(comando)
	if nombre == "" {
		nombre = "miapp"
	}
	ruta := h.proyectoRuta(nombre)
	if !existeRuta(ruta) {
		return fmt.Sprintf("No encontré el proyecto '%s', señor.", nombre)
	}
	if h.DesarrolladorIA == nil || !h.DesarrolladorIA.Disponible() {
		return "No tengo una IA de código disponible para mejorar el proyecto, señor. Active Ollama con el modelo qwen2.5-coder."
	}
	if feature == "" {
		feature = "realiza una mejora general al panel, manteniendo el estilo existente"
	}

	instrucciones := ""
	if h.Skills != nil {
		instrucciones = h.Skills.TextoParaIA(feature)
	}

	peticion := fmt.Sprintf("CONTEXTO DEL PROYECTO (%s):\n%s\n\nPETICION: %s", nombre, contextoProyecto(ruta), feature)
	if instrucciones != "" {
		peticion += "\n\n" + instrucciones
	}

	archivo, contenido, explicacion, err := h.pedirCodigoDesarrollo(peticion)
	if err != nil {
		return "La IA no respondió: " + err.Error()
	}
	if archivo == "" {
		return "La IA devolvió algo inesperado. " + explicacion
	}

	for i := 1; i <= maxIteracionesDesarrollo; i++ {
		if msg := escribirMejora(ruta, archivo, contenido); msg != "" {
			return msg
		}
		salida, errV := verificarProyecto(ruta, nombre)
		if errV == nil {
			h.checkpointProyecto(ruta)
			if i == 1 {
				return fmt.Sprintf("Mejora aplicada en '%s' del proyecto '%s', señor. Compila, pasa vet y los tests. %s", archivo, nombre, explicacion)
			}
			return fmt.Sprintf("Mejora aplicada en '%s' del proyecto '%s' tras %d intentos de corrección, señor. %s", archivo, nombre, i, explicacion)
		}
		if i == maxIteracionesDesarrollo {
			return fmt.Sprintf("La mejora en '%s' quedó escrita pero el proyecto no compila tras %d intentos, señor. Error: %s", archivo, maxIteracionesDesarrollo, strings.TrimSpace(salida))
		}
		correccion := fmt.Sprintf(
			"La verificación del proyecto falló con este error:\n%s\n\nEl archivo actual '%s' es:\n%s\n\nCorregí el error y devolvé el archivo completo corregido en el mismo formato (ARCHIVO: / CONTENIDO: / EXPLICACION:).",
			salida, archivo, contenido)
		if instrucciones != "" {
			correccion += "\n\n" + instrucciones
		}
		archivo, contenido, explicacion, err = h.pedirCodigoDesarrollo(correccion)
		if err != nil {
			return "La IA falló al corregir: " + err.Error()
		}
		if archivo == "" {
			return "La IA no devolvió una corrección válida. " + explicacion
		}
	}
	return "No pude terminar la mejora, señor."
}

func (h *Hands) pedirCodigoDesarrollo(peticion string) (archivo, contenido, explicacion string, err error) {
	cruda, explicacion, err := h.DesarrolladorIA.ConsultarDesarrollo(peticion)
	if err != nil {
		return "", "", explicacion, err
	}
	archivo, contenido = extraerArchivoDesarrollo(cruda)
	return archivo, contenido, explicacion, nil
}

// escribirMejora valida y escribe el archivo generado. Devuelve "" si escribió
// bien, o un mensaje de error si el archivo salía del proyecto o no se pudo.
func escribirMejora(ruta, archivo, contenido string) string {
	archivo = strings.ReplaceAll(archivo, "\\", "/")
	destino := filepath.Join(ruta, archivo)
	absRuta, _ := filepath.Abs(ruta)
	absDestino, _ := filepath.Abs(destino)
	if !strings.HasPrefix(absDestino, absRuta+string(filepath.Separator)) {
		return "La IA intentó escribir fuera del proyecto y lo bloqueé, señor."
	}
	ext := strings.ToLower(filepath.Ext(archivo))
	if ext != ".go" && ext != ".js" && ext != ".css" && ext != ".html" {
		return "La IA devolvió un archivo con extensión no permitida y lo bloqueé, señor."
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return "No pude escribir la mejora: " + err.Error()
	}
	if err := os.WriteFile(destino, []byte(contenido), 0o644); err != nil {
		return "No pude escribir la mejora: " + err.Error()
	}
	return ""
}

// contextoProyecto arma el árbol de archivos y el contenido de los archivos
// clave del proyecto para dárselo de contexto a la IA.
func contextoProyecto(ruta string) string {
	var b strings.Builder
	b.WriteString("Estructura del proyecto:\n")
	_ = filepath.WalkDir(ruta, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".git") {
			return nil
		}
		rel, err := filepath.Rel(ruta, path)
		if err == nil {
			b.WriteString(rel + "\n")
		}
		return nil
	})
	for _, f := range []string{"main.go", "README.md", "frontend/app.js", "frontend/index.html"} {
		datos, err := os.ReadFile(filepath.Join(ruta, f))
		if err != nil {
			continue
		}
		b.WriteString("\n===== " + f + " =====\n")
		b.WriteString(string(datos))
	}
	return b.String()
}

// verificarProyecto corre build + vet + test y devuelve la salida y un error si
// alguna de las tres falla.
func verificarProyecto(ruta, exe string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	build := exec.CommandContext(ctx, "go", "build", "-o", exe+".exe", ".")
	build.Dir = ruta
	salidaBuild, errBuild := build.CombinedOutput()
	if errBuild != nil {
		return "BUILD:\n" + string(salidaBuild), errBuild
	}

	vet := exec.CommandContext(ctx, "go", "vet", ".")
	vet.Dir = ruta
	salidaVet, errVet := vet.CombinedOutput()
	if errVet != nil {
		return "VET:\n" + string(salidaVet), errVet
	}

	test := exec.CommandContext(ctx, "go", "test", "./...")
	test.Dir = ruta
	salidaTest, errTest := test.CombinedOutput()
	if errTest != nil {
		return "TEST:\n" + string(salidaTest), errTest
	}
	return "build, vet y tests OK", nil
}

// checkpointProyecto crea un commit de git en el proyecto tras un cambio
// exitoso (comportamiento de checkpoints de Claude Code). Best effort.
func (h *Hands) checkpointProyecto(ruta string) {
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = ruta
		_ = cmd.Run()
	}
	if !existeRuta(filepath.Join(ruta, ".git")) {
		git("init", "-q")
	}
	git("add", "-A")
	git("commit", "-q", "-m", "mejora: "+time.Now().Format("2006-01-02 15:04"))
}

// listarSkills responde por voz cuáles skills tiene cargadas Jarvis.
func (h *Hands) listarSkills() string {
	if h.Skills == nil {
		return "No tengo skills cargadas, señor."
	}
	nombres := h.Skills.Listar()
	if len(nombres) == 0 {
		return "No tengo skills cargadas, señor."
	}
	return "Tengo estas skills: " + strings.Join(nombres, ", ") + ". Se activan solas según lo que pida, señor."
}

func compilarProyectoEn(ruta, exe string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", exe+".exe", ".")
	cmd.Dir = ruta
	salida, err := cmd.CombinedOutput()
	return string(salida), err
}

func (h *Hands) esperarServidor(puerto int, maximo time.Duration) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/estado", puerto)
	fin := time.Now().Add(maximo)
	cliente := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(fin) {
		resp, err := cliente.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (h *Hands) puertoLibre() int {
	h.proyectosMu.Lock()
	usados := map[int]bool{}
	for _, p := range h.proyectos {
		usados[p.Puerto] = true
	}
	h.proyectosMu.Unlock()
	puerto := 9090
	for usados[puerto] || puertoOcupado(puerto) {
		puerto++
	}
	return puerto
}

func puertoOcupado(puerto int) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(puerto))
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

func normalizarNombreProyecto(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	nombre = strings.TrimPrefix(nombre, "llamado ")
	nombre = strings.TrimPrefix(nombre, "llamada ")
	nombre = strings.TrimPrefix(nombre, "que se llame ")
	nombre = strings.TrimPrefix(nombre, "llamado el ")
	return strings.TrimSpace(nombre)
}

// extraerNombreYFeatureProyecto encuentra "proyecto <nombre>" dentro de un
// comando de mejora y separa el nombre del resto (la feature pedida).
func extraerNombreYFeatureProyecto(comando string) (nombre, feature string) {
	comando = strings.TrimSpace(strings.ToLower(comando))
	re := regexp.MustCompile(`proyecto\s+([a-z0-9][a-z0-9\-_]*)`)
	loc := re.FindStringSubmatchIndex(comando)
	if loc != nil {
		nombre = comando[loc[2]:loc[3]]
		feature = strings.TrimSpace(comando[:loc[0]] + " " + comando[loc[1]:])
	} else {
		feature = comando
	}
	feature = recortarConectores(feature)
	return
}

var conectoresFeature = map[string]bool{
	"al": true, "del": true, "el": true, "la": true, "los": true, "las": true,
	"para": true, "en": true, "de": true, "una": true, "un": true, "a": true,
}

func recortarConectores(s string) string {
	partes := strings.Fields(s)
	for len(partes) > 0 && conectoresFeature[partes[len(partes)-1]] {
		partes = partes[:len(partes)-1]
	}
	return strings.Join(partes, " ")
}

func extraerArchivoDesarrollo(texto string) (archivo, contenido string) {
	lineas := strings.Split(texto, "\n")
	for _, l := range lineas {
		trim := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(trim, "archivo:") {
			archivo = strings.TrimSpace(l[len("ARCHIVO:"):])
			archivo = strings.Trim(strings.Trim(archivo, `"'`), "`")
			break
		}
	}
	inicio := -1
	for i, l := range lineas {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "contenido:") {
			inicio = i
			break
		}
	}
	if inicio >= 0 {
		var b strings.Builder
		for _, l := range lineas[inicio+1:] {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "explicacion:") {
				break
			}
			b.WriteString(l)
			b.WriteString("\n")
		}
		contenido = strings.TrimRight(b.String(), "\n")
	}
	if archivo == "" || contenido == "" {
		return "", strings.TrimSpace(texto)
	}
	return archivo, contenido
}
