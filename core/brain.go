package core

import (
	"fmt"
	"strings"
)

type Brain struct {
	hands EjecutorComandos
	ia    ConectorIA
	coder AgenteDeCodigo
	mem   MemoriaPersistente
	ing   IngAgente
	prefs RegistroPreferencias

	ultimaApp      string
	ultimaBusqueda string
	ultimaRespuesta string
	historialIA    []TurnoConversacion
	maxHistorialIA int

	confirmacionPendiente *accionConfirmable
}

type accionConfirmable struct {
	ejecutar    func() string
	descripcion string
}

func NewBrain(h EjecutorComandos, opciones BrainOpciones) *Brain {
	b := &Brain{hands: h, ia: opciones.IA, coder: opciones.Coder, mem: opciones.Memoria, ing: opciones.IngAgente, prefs: opciones.Prefs, maxHistorialIA: 5}
	if opciones.MaxHistorialIA > 0 {
		b.maxHistorialIA = opciones.MaxHistorialIA
	}
	return b
}

func (b *Brain) Process(input string) string {
	respuesta := b.personalizarRespuesta(b.procesarInterno(input))
	if b.prefs != nil {
		b.prefs.RegistrarComando(input)
	}
	if respuesta != "" {
		b.ultimaRespuesta = respuesta
	}
	return respuesta
}

func (b *Brain) Saludar() string {
	return b.personalizarRespuesta(Saludo())
}

func (b *Brain) Despedirse() string {
	return b.personalizarRespuesta(Despedida())
}

func (b *Brain) sincronizarPrefs(clave, valor string) {
	if b.prefs == nil {
		return
	}
	switch clave {
	case "nombre":
		b.prefs.SetNombre(valor)
	}
}

func (b *Brain) personalizarRespuesta(respuesta string) string {
	if respuesta == "" || b.mem == nil {
		return respuesta
	}
	nombre, existe := b.mem.ObtenerHecho("nombre")
	if !existe || nombre == "" {
		return respuesta
	}
	return strings.ReplaceAll(respuesta, "señor", nombre)
}

func (b *Brain) procesarInterno(input string) string {
	original := strings.TrimSpace(input)
	entrada := strings.ToLower(original)
	if entrada == "" {
		return ""
	}

	if b.ing != nil && b.ing.TieneTareaPendiente() {
		if strings.Contains(entrada, "continuar") || strings.Contains(entrada, "seguir") || strings.Contains(entrada, "retomar") {
			return b.ing.ContinuarPlan()
		}
		if strings.Contains(entrada, "cancelar") && strings.Contains(entrada, "plan") {
			b.ing.Reset()
			return "Plan cancelado, señor."
		}
	}

	if b.coder != nil && b.coder.TienePropuestaPendiente() {
		switch {
		case strings.Contains(entrada, "confirmar"), strings.Contains(entrada, "confirmo"),
			strings.Contains(entrada, "ejecutar"), strings.Contains(entrada, "dale"):
			return b.coder.Confirmar()
		case strings.Contains(entrada, "cancelar"), strings.Contains(entrada, "cancelo"):
			return b.coder.Cancelar()
		default:
			return "Tiene una propuesta de script esperando, señor. Diga 'confirmar' para ejecutarla o 'cancelar' para descartarla."
		}
	}

	if b.confirmacionPendiente != nil {
		switch {
		case esPalabraExacta(entrada, "sí") || esPalabraExacta(entrada, "si") || entrada == "confirmar" || entrada == "confirmo" || entrada == "dale":
			accion := b.confirmacionPendiente
			b.confirmacionPendiente = nil
			return accion.ejecutar()
		case esPalabraExacta(entrada, "no") || strings.Contains(entrada, "cancelar") || strings.Contains(entrada, "cancelo") || strings.Contains(entrada, "para") || strings.Contains(entrada, "detener"):
			b.confirmacionPendiente = nil
			return "Cancelado, señor. No se realizó ninguna acción."
		default:
			return fmt.Sprintf("Tengo una acción pendiente: %s. Diga 'sí' para confirmar o 'no' para cancelar, señor.", b.confirmacionPendiente.descripcion)
		}
	}

	if strings.Contains(entrada, "ayuda") {
		fmt.Println(textoAyuda)
		return "Puede pedirme que abra aplicaciones, busque en internet, le diga la hora, escriba scripts, recuerde cosas, y más. Mire la consola para ver todos los comandos, señor."
	}

	if contieneAlguna(entrada, frasesQueDijiste) {
		if b.ultimaRespuesta == "" {
			return "Todavía no dije nada en esta sesión, señor."
		}
		return b.ultimaRespuesta
	}

	entrada = b.resolverPronombres(entrada)

	if respuesta, atendido := b.procesarMemoria(original); atendido {
		return respuesta
	}

	if b.coder != nil && esPeticionDeCodigo(entrada) {
		respuesta, err := b.coder.Proponer(input)
		if err != nil {
			return fmt.Sprintf("No pude generar el script: %v", err)
		}
		return respuesta
	}

	b.actualizarMemoriaDeSesion(entrada)

	if descripcion, peligroso := esAccionPeligrosa(entrada); peligroso && !esConsultaSegura(entrada) {
		ejecutar := func() string {
			return b.hands.RunCommand(entrada)
		}
		b.confirmacionPendiente = &accionConfirmable{
			ejecutar:    ejecutar,
			descripcion: descripcion,
		}
		return fmt.Sprintf("¿Está seguro de que desea %s, señor? Diga 'sí' para confirmar o 'no' para cancelar.", descripcion)
	}

	respuesta := b.hands.RunCommand(entrada)
	if respuesta != ComandoNoReconocido {
		return respuesta
	}

	if b.ia != nil && b.ia.Disponible() {
		respuestaIA, err := b.ia.Consultar(input, b.historialIA)
		if err == nil && respuestaIA != "" {
			b.historialIA = append(b.historialIA, TurnoConversacion{Usuario: input, Asistente: respuestaIA})
			if len(b.historialIA) > b.maxHistorialIA {
				b.historialIA = b.historialIA[len(b.historialIA)-b.maxHistorialIA:]
			}
			return respuestaIA
		}
	}

	if b.ing != nil && b.ing.Disponible() {
		if esPeticionIngenieria(entrada) {
			return b.ing.Procesar(input)
		}
	}

	return RespuestaConfusion()
}

var palabrasPeligrosas = []struct {
	frase  string
	accion string
}{
	{"reiniciar", "reiniciar el equipo"},
	{"reinicio", "reiniciar el equipo"},
	{"restart", "reiniciar el equipo"},
	{"suspender", "suspender el equipo"},
	{"dormir", "suspender el equipo"},
	{"suspensión", "suspender el equipo"},
	{"hibernar", "hibernar el equipo"},
	{"hibernación", "hibernar el equipo"},
	{"cerrar sesión", "cerrar su sesión"},
	{"cerrar sesion", "cerrar su sesión"},
	{"logoff", "cerrar su sesión"},
	{"vaciar papelera", "vaciar la papelera de reciclaje"},
	{"limpiar papelera", "vaciar la papelera de reciclaje"},
	{"formatear", "formatear un disco"},
	{"format ", "formatear un disco"},
	{"diskpart", "ejecutar diskpart"},
	{"desinstalar", "desinstalar un programa"},
	{"uninstall", "desinstalar un programa"},
	{"instalar", "instalar un programa"},
	{"install ", "instalar un programa"},
	{"descargar", "descargar algo de internet"},
	{"download", "descargar algo de internet"},
	{"comprar", "comprar algo"},
	{"compra ", "comprar algo"},
	{"vender", "vender algo"},
	{"venta ", "vender algo"},
	{"borrar ", "borrar archivos o datos"},
	{"eliminar ", "eliminar archivos o datos"},
	{"suprimir ", "suprimir archivos o datos"},
}

func esAccionPeligrosa(entrada string) (string, bool) {
	entrada = strings.TrimSpace(entrada)
	if strings.Contains(entrada, "activar suspensión") || strings.Contains(entrada, "activar suspension") ||
		strings.Contains(entrada, "desactivar suspensión") || strings.Contains(entrada, "desactivar suspension") {
		return "", false
	}
	for _, p := range palabrasPeligrosas {
		if strings.Contains(entrada, p.frase) {
			return p.accion, true
		}
	}
	return "", false
}

func esConsultaSegura(entrada string) bool {
	entrada = strings.TrimSpace(entrada)
	return strings.HasPrefix(entrada, "buscar ") ||
		strings.HasPrefix(entrada, "buscar en ") ||
		strings.HasPrefix(entrada, "ir a ") ||
		strings.HasPrefix(entrada, "andá a ") ||
		strings.HasPrefix(entrada, "anda a ") ||
		strings.HasPrefix(entrada, "decir ") ||
		strings.HasPrefix(entrada, "copiar ") ||
		strings.HasPrefix(entrada, "eco ") ||
		strings.HasPrefix(entrada, "repite ") ||
		strings.HasPrefix(entrada, "busca ") ||
		strings.HasPrefix(entrada, "buscá ") ||
		strings.HasPrefix(entrada, "tomá nota") ||
		strings.HasPrefix(entrada, "toma nota") ||
		strings.HasPrefix(entrada, "tomá nota de") ||
		strings.HasPrefix(entrada, "anotá esto") ||
		strings.HasPrefix(entrada, "anotá ") ||
		strings.HasPrefix(entrada, "anota esto") ||
		strings.HasPrefix(entrada, "anota ")
}

func esPeticionIngenieria(entrada string) bool {
	tieneAccion := contieneAlguna(entrada, []string{
		"programa", "codigo", "código", "script", "implementa", "implementá",
		"crea", "creá", "hace", "hacé", "modifica", "modificá", "cambia",
		"cambiá", "agrega", "agregá", "refactoriza", "refactoreá",
		"arregla", "arreglá", "repara", "repará", "proyecto", "archivo",
		"funcion", "función", "clase", "estructura", "test", "prueba",
	})
	return tieneAccion
}

func esPeticionDeCodigo(entrada string) bool {
	tieneVerbo := strings.Contains(entrada, "escribe") || strings.Contains(entrada, "escribime") ||
		strings.Contains(entrada, "crea") || strings.Contains(entrada, "generame") ||
		strings.Contains(entrada, "genera") || strings.Contains(entrada, "hazme") ||
		strings.Contains(entrada, "haceme")
	tieneSustantivo := strings.Contains(entrada, "script") || strings.Contains(entrada, "código") ||
		strings.Contains(entrada, "codigo")
	return tieneVerbo && tieneSustantivo
}

func esPalabraExacta(entrada, palabra string) bool {
	entrada = strings.TrimSpace(entrada)
	if entrada == palabra {
		return true
	}
	if strings.HasPrefix(entrada, palabra+" ") || strings.HasSuffix(entrada, " "+palabra) {
		return true
	}
	if strings.Contains(entrada, " "+palabra+" ") {
		return true
	}
	return false
}

const textoAyuda = `
Comandos disponibles (100+):
  jarvis                          -> actívame
  decir [texto]                   -> repito lo que digas
  qué dijiste                     -> repito mi última respuesta
  saludar / salúdame              -> saludo según la hora
  eco [texto] / repite [texto]    -> repito exactamente

APPS:
  abrir chrome / firefox / edge / opera / vscode / bloc / word / excel / powerpoint
  abrir spotify / discord / whatsapp / telegram / outlook / steam / zoom / teams
  abrir calculadora / terminal / cmd / powershell / paint
  abrir calendario / cámara / fotos / música / videos / mapas / noticias / clima
  cerrar [app] / cerralo          -> "cerralo" cierra la última app abierta

VOLUMEN / MULTIMEDIA:
  subir volumen / bajar volumen / silencio / activar sonido
  qué volumen / volumen actual
  pausar / reproducir / siguiente canción / canción anterior

INFORMACIÓN DEL SISTEMA:
  hora / fecha / fecha completa  -> día, fecha y hora
  batería                        -> nivel de batería
  procesador / cpu               -> modelo del procesador
  memoria ram / ram              -> total de RAM instalada
  disco / almacenamiento         -> espacio en discos
  sistema operativo / os         -> versión de Windows
  tiempo activo / uptime         -> cuánto llevo encendido
  usuario                        -> nombre de usuario actual
  nombre pc / nombre del equipo  -> nombre del equipo
  arquitectura / bits            -> 32 o 64 bits
  programas instalados           -> cantidad de software
  procesos                       -> procesos ejecutándose
  núcleos / cores               -> cantidad de núcleos CPU
  resolución / pantalla          -> resolución de monitor
  idioma                         -> idioma del sistema
  zona horaria                   -> zona horaria configurada
  temperatura                    -> temperatura del CPU

RED:
  mi ip / ip local               -> IP de red local
  ip pública / ip externa       -> IP pública
  ping [host]                    -> prueba conexión
  dns                            -> servidores DNS
  mac                            -> dirección MAC
  velocidad red                  -> velocidad del adaptador
  wifi                           -> red WiFi actual
  conexiones activas             -> conexiones TCP activas
  interfaces red                 -> interfaces de red

ENERGÍA / SISTEMA:
  suspender / dormir             -> suspende el equipo
  hibernar                       -> hiberna el equipo
  reiniciar                      -> reinicia el equipo
  cerrar sesión                  -> cierra tu sesión
  bloquear pantalla              -> bloquea la PC
  apagar monitor                 -> apaga la pantalla
  subir brillo / bajar brillo    -> ajusta el brillo
  brillo al [X]%                 -> brillo al porcentaje exacto
  modo avión                     -> alterna modo avión

VENTANAS:
  maximizar / minimizar / restaurar ventana
  cerrar ventana                 -> cierra la ventana activa
  cambiar ventana                -> cambia a otra ventana
  mostrar escritorio             -> muestra el escritorio
  organizar ventanas             -> organiza en mosaico
  minimizar todo                 -> minimiza todas las ventanas

NAVEGADOR / WEB:
  buscar [algo]                  -> busca en Google
  buscar en youtube [algo]       -> busca en YouTube
  buscar en wikipedia [algo]     -> busca en Wikipedia
  ir a [sitio]                   -> abre un sitio web
  abrir youtube / gmail / github / stackoverflow
  abrir amazon / twitter / reddit / facebook / instagram / netflix

ARCHIVOS:
  listar archivos                -> muestra el directorio actual
  ruta actual / pwd              -> muestra la ruta actual
  crear carpeta [nombre]         -> crea carpeta en escritorio
  crear archivo [nombre]         -> crea archivo en escritorio
  buscar archivo [nombre]        -> busca archivos en el sistema
  vaciar papelera                -> vacía la papelera
  abrir descargas / escritorio / documentos / imágenes / música
  abrir papelera / configuración

MEMORIA:
  recordá que [algo]             -> guarda una nota libre
  recordame [algo] a las [hora]  -> recordatorio con hora
  recordame [algo] mañana        -> recordatorio para mañana
  recordame [algo] el lunes      -> recordatorio para el próximo día
  recordame [algo] cada día      -> recordatorio diario
  buscá en mis notas [texto]     -> busca en notas guardadas
  poné un timer de [X] min      -> temporizador
  qué recordatorios tengo        -> lista pendientes
  cancelá el recordatorio de [algo] / cancelá todos
  llamame [nombre]               -> así te llamo
  cuál es mi nombre / dónde vivo / cuándo es mi cumpleaños
  dónde trabajo / qué recordás

LISTAS:
  creá una lista de [nombre]     -> crear lista nueva
  agregá [item] a la lista [nom] -> agregar item
  marcá [item] como hecho [lista]-> marcar completado
  mostrame las listas            -> todas las listas
  mostrame la lista de [nombre]  -> contenido de una lista
  eliminá la lista [nombre]      -> borrar lista

CÓDIGO / IA:
  escribe un script que [tarea]  -> genera script con IA
  confirmar / cancelar           -> confirma o cancela script propuesto

DESARROLLADOR FULLSTACK:
  crear proyecto web [nombre]    -> genera app Go + frontend y la compila
  mis proyectos                 -> lista los proyectos creados
  compilar el proyecto [nombre] -> compila un proyecto
  ejecutar proyecto [nombre]    -> lo levanta y lo abre en el navegador
  detener proyecto [nombre]     -> detiene la app en ejecución
  estado del proyecto [nombre]  -> si está corriendo y en qué puerto
  mejorar el proyecto [nombre]  -> mejora con IA (requiere Ollama)

CLIMA / NOTICIAS (requiere API key en config.json):
  clima / qué temperatura hace   -> clima de tu ciudad (si configurado)
  noticias / últimas noticias    -> titulares de noticias (si configurado)

DIVERSIÓN:
  cuéntame un chiste / cumplido / motivación
  moneda / cara o cruz           -> lanza una moneda
  dado                           -> tira un dado
  número aleatorio / random      -> número del 1 al 100
  sí o no / decisión             -> decide por ti
  color aleatorio                -> elige un color
  trabalenguas                   -> un trabalenguas
  signo zodiacal                 -> signo del día
  días para navidad              -> días que faltan
  copiar [texto]                 -> copia al portapapeles
  captura de pantalla            -> toma captura de pantalla

BIENESTAR:
  respira / respiración          -> ejercicio de respiración
  beber agua / hidratación       -> recordatorio de agua
  estiramiento / estirar         -> recordatorio de estirarse

  apagar                         -> me apago
Si ningún comando coincide y hay una clave de IA configurada, te responderé con IA.`
