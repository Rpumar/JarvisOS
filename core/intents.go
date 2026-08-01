package core

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type IntentoHandler func(cmd string, h *Hands) string

type Intento struct {
	Nombre   string
	Frases   []string
	Palabras []string
	Handler  IntentoHandler
}

type Clasificador struct {
	intentos []Intento
}

func NuevoClasificador() *Clasificador {
	c := &Clasificador{}
	c.intentos = []Intento{
		{
			Nombre: "orden",
			Frases: []string{"agendá una orden", "agenda una orden", "nueva orden", "registrá la orden", "registra la orden", "qué órdenes tengo", "que ordenes tengo", "órdenes pendientes", "ordenes pendientes", "todas las órdenes", "todas las ordenes", "retomá las órdenes", "retoma las ordenes", "ejecutá la orden", "ejecuta la orden", "tomá la orden", "toma la orden", "reportá la orden", "reporta la orden", "reporte de la orden", "terminá la orden", "termina la orden", "marcar la orden", "bloquear la orden", "bloqueá la orden", "bloquea la orden", "cancelar la orden", "cancela la orden", "cancelá la orden", "aprobar la orden", "aprobar orden", "denegar la orden", "rechazar la orden", "mis órdenes", "mis ordenes"},
			Palabras: []string{"orden", "órdenes", "ordenes"},
			Handler: func(cmd string, h *Hands) string { return h.manejarOrden(cmd) },
		},
		{
			Nombre: "volumen_subir",
			Frases: []string{"subir volumen", "súbele", "subile", "mas alto", "más alto", "aumenta volumen", "volumen arriba", "ponelo mas fuerte"},
			Palabras: []string{"subir", "súbele", "subile", "aumenta"},
			Handler: func(cmd string, h *Hands) string { return h.volumen("up") },
		},
		{
			Nombre: "volumen_bajar",
			Frases: []string{"bajar volumen", "bájale", "bajale", "mas bajo", "más bajo", "reduce volumen", "volumen abajo"},
			Palabras: []string{"bajar", "bájale", "bajale", "reduce"},
			Handler: func(cmd string, h *Hands) string { return h.volumen("down") },
		},
		{
			Nombre: "volumen_mute",
			Frases: []string{"silencio", "mutear", "silenciar", "sin sonido", "no se escucha"},
			Palabras: []string{"silencio", "mutear", "mute", "silenciar"},
			Handler: func(cmd string, h *Hands) string { return h.volumen("mute") },
		},
		{
			Nombre: "play_pause",
			Frases: []string{"pausar", "reproducir", "play", "pausa", "seguí", "seguir"},
			Palabras: []string{"pausar", "reproducir", "pausa", "play"},
			Handler: func(cmd string, h *Hands) string { return h.controlMedia("play_pause") },
		},
		{
			Nombre: "siguiente_cancion",
			Frases: []string{"siguiente canción", "siguiente cancion", "próximo tema", "proximo tema", "pasá", "pasa", "siguiente tema", "adelante"},
			Palabras: []string{"siguiente", "próximo", "proximo", "adelante"},
			Handler: func(cmd string, h *Hands) string { return h.controlMedia("next") },
		},
		{
			Nombre: "cancion_anterior",
			Frases: []string{"canción anterior", "cancion anterior", "tema anterior", "volvé", "volve", "atrás", "anterior"},
			Palabras: []string{"anterior", "volvé", "volve", "atrás"},
			Handler: func(cmd string, h *Hands) string { return h.controlMedia("prev") },
		},
		{
			Nombre: "hora",
			Frases: []string{"qué hora es", "que hora es", "decime la hora", "decí la hora", "deci la hora", "hora actual", "hora", "que hora tenés", "qué hora tenes"},
			Palabras: []string{"hora"},
			Handler: func(cmd string, h *Hands) string { return h.decirHora() },
		},
		{
			Nombre: "fecha",
			Frases: []string{"qué fecha es", "que fecha es", "decime la fecha", "decí la fecha", "deci la fecha", "fecha actual", "fecha de hoy", "a qué día estamos", "que dia es hoy", "qué día es hoy"},
			Palabras: []string{"fecha", "día", "dia", "estamos"},
			Handler: func(cmd string, h *Hands) string { return h.decirFecha() },
		},
		{
			Nombre: "bateria",
			Frases: []string{"nivel de batería", "nivel de bateria", "cuánta batería queda", "cuanta bateria queda", "batería", "bateria", "qué batería tengo", "que bateria tengo", "nivel de carga", "carga de la batería", "carga de bateria"},
			Palabras: []string{"batería", "bateria", "carga"},
			Handler: func(cmd string, h *Hands) string { return h.nivelBateria() },
		},
		{
			Nombre: "ip_publica",
			Frases: []string{"ip publica", "mi ip publica", "ip externa", "qué ip publica tengo", "que ip publica tengo"},
			Palabras: []string{"publica"},
			Handler: func(cmd string, h *Hands) string { return h.ipPublica() },
		},
		{
			Nombre: "voz_desactivar",
			Frases: []string{"no hables", "no hables más", "no me hables", "no respondas por voz", "silenciá tu voz", "silencia tu voz", "no me respondas por audio", "callate", "cállate", "calmate", "cállate la boca"},
			Palabras: []string{"callate", "cállate", "silenciá", "silencia", "no hables"},
			Handler: func(cmd string, h *Hands) string { return h.desactivarVoz() },
		},
		{
			Nombre: "voz_activar",
			Frases: []string{"activá tu voz", "activa tu voz", "hablá de nuevo", "habla de nuevo", "respondeme por voz", "respondé por voz", "volvé a hablar", "volve a hablar", "respondeme por audio", "no me escribas", "voz activada"},
			Palabras: []string{"activá tu voz", "activa tu voz", "hablá de nuevo", "habla de nuevo"},
			Handler: func(cmd string, h *Hands) string { return h.activarVoz() },
		},
		{
			Nombre: "voz_listar",
			Frases: []string{"qué voces tenés", "que voces tenes", "voces disponibles", "voces de voz", "lista de voces", "qué voces instaladas", "que voces instaladas", "voces instaladas"},
			Palabras: []string{"voces"},
			Handler: func(cmd string, h *Hands) string { return h.listarVoces() },
		},
		{
			Nombre: "ping_sitio",
			Frases: []string{"hacer ping", "ping a", "ping a google", "probá conexión", "proba conexion", "test de conexión", "test de conexion"},
			Palabras: []string{"ping"},
			Handler: func(cmd string, h *Hands) string {
				host := extraerObjeto(cmd, []string{"ping a ", "ping ", "hacer ping a "})
				if host == "" {
					host = "google.com"
				}
				return h.hacerPing(host)
			},
		},
		{
			Nombre: "velocidad_internet",
			Frases: []string{"velocidad de internet", "qué tan rápida es mi internet", "que tan rapida es mi internet", "velocidad de descarga", "test de velocidad", "testeá mi internet", "testea mi internet"},
			Palabras: []string{"velocidad"},
			Handler: func(cmd string, h *Hands) string { return h.velocidadInternet() },
		},
		{
			Nombre: "escanear_red",
			Frases: []string{"escanear red", "escanear la red", "dispositivos de la red", "dispositivos conectados", "qué hay en mi red", "que hay en mi red", "ver dispositivos conectados"},
			Palabras: []string{"escanear"},
			Handler: func(cmd string, h *Hands) string { return h.escanearRed() },
		},
		{
			Nombre: "limpiar_dns",
			Frases: []string{"limpiar dns", "liberar dns", "flush dns", "limpiar la caché dns", "limpiar la cache dns"},
			Palabras: []string{"flush", "liberar"},
			Handler: func(cmd string, h *Hands) string { return h.limpiarDNS() },
		},
		{
			Nombre: "info_red",
			Frases: []string{"adaptadores de red", "tarjeta de red", "redes activas", "mi conexión de red", "mi conexion de red", "info de red"},
			Palabras: []string{"adaptadores", "tarjeta"},
			Handler: func(cmd string, h *Hands) string { return h.infoRedDetallada() },
		},
		{
			Nombre: "uso_ram",
			Frases: []string{"cuánta ram me queda", "cuanta ram me queda", "ram libre", "memoria ram", "uso de ram", "cuánta memoria hay", "cuanta memoria hay"},
			Palabras: []string{"ram", "memoria"},
			Handler: func(cmd string, h *Hands) string { return h.usoRAM() },
		},
		{
			Nombre: "temp_cpu",
			Frases: []string{"temperatura del cpu", "temperatura del procesador", "temperatura de la pc", "está caliente la pc", "esta caliente la pc"},
			Palabras: []string{"temperatura", "caliente"},
			Handler: func(cmd string, h *Hands) string { return h.infoTemperatura() },
		},
		{
			Nombre: "uso_disco",
			Frases: []string{"espacio en disco", "disco duro", "cuánto espacio me queda", "cuanto espacio me queda", "almacenamiento", "espacio libre"},
			Palabras: []string{"disco", "espacio"},
			Handler: func(cmd string, h *Hands) string { return h.infoDisco() },
		},
		{
			Nombre: "plan_energia",
			Frases: []string{"plan de energía", "plan de energia", "modo ahorro", "rendimiento máximo", "rendimiento maximo", "plan de batería", "plan de bateria"},
			Palabras: []string{"energía", "energia", "ahorro"},
			Handler: func(cmd string, h *Hands) string { return h.planEnergia() },
		},
		{
			Nombre: "mi_ip",
			Frases: []string{"mi ip", "mi dirección ip", "mi direccion ip", "qué ip tengo", "que ip tengo", "ip local", "ip privada"},
			Palabras: []string{"ip"},
			Handler: func(cmd string, h *Hands) string { return h.obtenerIP() },
		},
		{
			Nombre: "copiar",
			Frases: []string{"copiá", "copia", "copiar", "copiar al portapapeles"},
			Palabras: []string{"copiar", "copiá", "copia"},
			Handler: func(cmd string, h *Hands) string {
				texto := extraerObjeto(cmd, []string{"copiar ", "copiá ", "copia "})
				if texto == "" {
					return h.copiarAlPortapapeles(cmd)
				}
				return h.copiarAlPortapapeles(texto)
			},
		},
		{
			Nombre: "clima",
			Frases: []string{"qué clima hace", "que clima hace", "cómo está el clima", "como esta el clima", "clima", "temperatura", "qué temperatura hace", "que temperatura hace", "qué calor", "que calor", "qué frío", "que frio", "está lloviendo", "va a llover"},
			Palabras: []string{"clima", "temperatura", "calor", "frío", "frio", "lloviendo", "llover", "pronóstico"},
			Handler: func(cmd string, h *Hands) string { return h.consultarClima() },
		},
		{
			Nombre: "noticias",
			Frases: []string{"últimas noticias", "ultimas noticias", "qué pasó hoy", "que paso hoy", "noticias", "titulares", "novedades", "qué hay de nuevo", "que hay de nuevo"},
			Palabras: []string{"noticias", "titulares", "novedades", "pasó", "paso"},
			Handler: func(cmd string, h *Hands) string { return h.consultarNoticias() },
		},
		{
			Nombre: "email",
			Frases: []string{"enviar un email", "enviar un correo", "enviar un mail", "enviá un email", "enviá un correo", "enviá un mail", "envia un email", "envia un correo", "envia un mail", "mandar un email", "mandar un correo", "mandar un mail", "mandá un email", "mandá un correo", "mandá un mail", "manda un email", "manda un correo", "manda un mail", "leer mis correos", "leer mi email", "leer mi correo", "revisar mis correos", "revisar mi email", "revisar mi correo", "ver mis correos", "cuántos correos tengo", "cuantos correos tengo"},
			Palabras: []string{"email", "correo", "mail", "correos", "mails"},
			Handler: func(cmd string, h *Hands) string { return h.manejarEmail(cmd) },
		},
		{
			Nombre: "bloquear_pantalla",
			Frases: []string{"bloquear pantalla", "bloquear pc", "bloquear equipo", "bloquear la pantalla", "bloqueá", "bloquea", "trancar pantalla"},
			Palabras: []string{"bloquear", "bloqueá", "bloquea", "trancar"},
			Handler: func(cmd string, h *Hands) string { return h.bloquearPantalla() },
		},
		{
			Nombre: "minimizar_todo",
			Frases: []string{"minimizar todo", "minimizar todas", "mostrar escritorio", "ver escritorio", "esconder todo"},
			Palabras: []string{"minimizar", "escritorio"},
			Handler: func(cmd string, h *Hands) string { return h.minimizarTodo() },
		},
		{
			Nombre: "captura_pantalla",
			Frases: []string{"captura de pantalla", "capturar pantalla", "tomar captura", "sacar captura", "screenshot", "foto de la pantalla"},
			Palabras: []string{"captura", "capturar", "screenshot", "pantallazo"},
			Handler: func(cmd string, h *Hands) string { return h.capturarPantalla() },
		},
		{
			Nombre: "apagar_pc",
			Frases: []string{"apagar la pc", "apagar pc", "apagar el equipo", "apagar computadora", "quiero apagar", "apaga la computadora", "apagá la compu", "apaga la compu"},
			Palabras: []string{"apagar"},
			Handler: func(cmd string, h *Hands) string { return h.apagarPC() },
		},
		{
			Nombre: "listar_wifi",
			Frases: []string{"redes wifi", "redes disponibles", "wifi disponible", "qué redes hay", "ver wifi", "listar wifi", "escaneá redes"},
			Palabras: []string{"wifi", "redes"},
			Handler: func(cmd string, h *Hands) string { return h.wifiListar() },
		},
		{
			Nombre: "desconectar_wifi",
			Frases: []string{"desconectar wifi", "cortar wifi", "apagar wifi", "desconectar red"},
			Palabras: []string{"desconectar", "cortar"},
			Handler: func(cmd string, h *Hands) string { return h.wifiDesconectar() },
		},
		{
			Nombre: "bluetooth_on",
			Frases: []string{"activar bluetooth", "prender bluetooth", "encender bluetooth", "bluetooth on"},
			Palabras: []string{"bluetooth"},
			Handler: func(cmd string, h *Hands) string { return h.bluetoothActivar() },
		},
		{
			Nombre: "bluetooth_off",
			Frases: []string{"desactivar bluetooth", "apagar bluetooth", "bluetooth off"},
			Palabras: []string{"bluetooth"},
			Handler: func(cmd string, h *Hands) string { return h.bluetoothDesactivar() },
		},
		{
			Nombre: "listar_procesos",
			Frases: []string{"listar procesos", "procesos activos", "qué procesos hay", "programas abiertos", "aplicaciones abiertas"},
			Palabras: []string{"procesos", "ejecutando"},
			Handler: func(cmd string, h *Hands) string { return h.listarProcesos() },
		},
		{
			Nombre: "listar_usb",
			Frases: []string{"dispositivos usb", "usb conectados", "qué usb hay", "listar usb", "puertos usb"},
			Palabras: []string{"usb"},
			Handler: func(cmd string, h *Hands) string { return h.listarUSB() },
		},
		{
			Nombre: "pantalla_duplicar",
			Frases: []string{"duplicar pantalla", "duplicar monitor", "pantalla duplicada"},
			Palabras: []string{"duplicar"},
			Handler: func(cmd string, h *Hands) string { return h.cambiarModoPantalla("duplicar") },
		},
		{
			Nombre: "pantalla_extender",
			Frases: []string{"extender pantalla", "extender monitor", "pantalla extendida", "segunda pantalla"},
			Palabras: []string{"extender"},
			Handler: func(cmd string, h *Hands) string { return h.cambiarModoPantalla("extender") },
		},
		{
			Nombre: "bateria_detallada",
			Frases: []string{"batería detallada", "bateria detallada", "estado batería", "estado bateria", "qué batería tengo", "que bateria tengo", "nivel de batería", "nivel de bateria", "carga de batería", "carga de bateria"},
			Palabras: []string{"batería", "bateria"},
			Handler: func(cmd string, h *Hands) string { return h.infoBateriaDetallada() },
		},
		{
			Nombre: "limpiar_temporales",
			Frases: []string{"limpiar temporales", "limpiar archivos temporales", "borrar temporales", "limpiar archivos basura", "borrar archivos temporales", "liberar espacio temporal"},
			Palabras: []string{"temporales"},
			Handler: func(cmd string, h *Hands) string { return h.limpiarTemporales() },
		},
		{
			Nombre: "organizar_descargas",
			Frases: []string{"organizar descargas", "organiza descargas", "ordenar descargas", "ordena descargas", "organizar mis descargas", "organiza mis descargas", "ordenar mis descargas", "limpiar descargas"},
			Palabras: []string{"organizar", "organiza", "ordenar", "ordena"},
			Handler: func(cmd string, h *Hands) string { return h.organizarDescargas() },
		},
		{
			Nombre: "grabar_pantalla",
			Frases: []string{"grabar pantalla", "grabar la pantalla", "graba la pantalla", "grabar video", "grabar video de la pantalla", "grabación de pantalla", "grabacion de pantalla"},
			Palabras: []string{"grabar", "graba"},
			Handler: func(cmd string, h *Hands) string { return h.grabarPantalla() },
		},
		{
			Nombre: "modo_oscuro",
			Frases: []string{"modo oscuro", "tema oscuro", "modo nocturno", "poné el modo oscuro", "pone el modo oscuro", "tema oscuro en todo"},
			Palabras: []string{"oscuro", "nocturno"},
			Handler: func(cmd string, h *Hands) string { return h.modoOscuro() },
		},
		{
			Nombre: "firewall",
			Frases: []string{"estado del firewall", "firewall activo", "está activo el firewall", "esta activo el firewall", "seguridad del firewall"},
			Palabras: []string{"firewall"},
			Handler: func(cmd string, h *Hands) string { return h.firewallEstado() },
		},
		{
			Nombre: "puertos_uso",
			Frases: []string{"puertos en uso", "qué puertos están abiertos", "que puertos estan abiertos", "puertos abiertos", "puertos escuchando", "puertos activos"},
			Palabras: []string{"puertos"},
			Handler: func(cmd string, h *Hands) string { return h.puertosEnUso() },
		},
		{
			Nombre: "procesos_red",
			Frases: []string{"procesos que usan red", "aplicaciones usando internet", "quién usa la red", "quien usa la red", "aplicaciones que usan internet", "qué apps usan internet"},
			Palabras: []string{"internet"},
			Handler: func(cmd string, h *Hands) string { return h.procesosConRed() },
		},
		{
			Nombre: "sesiones_activas",
			Frases: []string{"sesiones activas", "sesiones abiertas", "usuarios conectados", "quiénes están conectados", "quienes estan conectados", "sesiones de usuario"},
			Palabras: []string{"sesiones"},
			Handler: func(cmd string, h *Hands) string { return h.sesionesActivas() },
		},
		{
			Nombre: "rutina",
			Frases: []string{"crear rutina", "creá una rutina", "ejecutar rutina", "corré la rutina", "ejecutá la rutina", "listar rutinas", "mis rutinas", "qué rutinas tengo", "que rutinas tengo", "borrar rutina", "eliminar rutina", "rutina de"},
			Palabras: []string{"rutina", "rutinas"},
			Handler: func(cmd string, h *Hands) string { return h.manejarRutina(cmd) },
		},
		{
			Nombre: "tarea",
			Frases: []string{"agendá una tarea", "agenda una tarea", "agendame una tarea", "agendá la tarea", "qué tareas tengo", "que tareas tengo", "tareas pendientes", "tareas me faltan", "todas las tareas", "marcar tarea", "marca la tarea", "tarea como hecha", "completé la tarea", "borrar tarea", "borrá la tarea", "eliminar tarea", "registrá una tarea", "tarea nueva", "cuántas tareas tengo", "cuantas tareas tengo"},
			Palabras: []string{"tarea", "tareas", "agendá", "agenda", "agendame", "agendáme"},
			Handler: func(cmd string, h *Hands) string { return h.manejarTarea(cmd) },
		},
		{
			Nombre: "procedimiento",
			Frases: []string{"aprendé a hacer", "aprende a hacer", "aprendé que para", "aprende que para", "aprendé que", "aprende que", "recordá que para", "recuerda que para", "enseñame a", "enseñáme a", "cómo hago", "como hago", "cómo se hace", "como se hace", "ejecutá el procedimiento", "ejecuta el procedimiento", "qué procedimientos sabés", "que procedimientos sabes", "qué sabés hacer", "que sabes hacer", "borrar procedimiento", "olvidate el procedimiento", "olvidá el procedimiento", "los pasos son", "los pasos"},
			Palabras: []string{"procedimiento", "procedimientos", "aprendé", "aprende", "enseñame", "enseñáme", "cómo hago", "como hago", "los pasos"},
			Handler: func(cmd string, h *Hands) string { return h.manejarProcedimiento(cmd) },
		},
		{
			Nombre: "notificacion",
			Frases: []string{"avisame", "avisame en la pantalla", "avisame que", "avísame", "mostrar notificacion", "mostrame una notificacion", "mostrar notificación", "mostrame una notificación", "notificame", "notifica"},
			Palabras: []string{"avisame", "avísame", "notificame", "notifica"},
			Handler: func(cmd string, h *Hands) string {
				texto := extraerObjeto(cmd, []string{"avisame en la pantalla ", "avísame en la pantalla ", "avisame que ", "avísame que ", "avisame ", "avísame ", "notificame que ", "notificame ", "notificá "})
				return h.enviarNotificacion(texto)
			},
		},
		{
			Nombre: "comprimir",
			Frases: []string{"comprimir carpeta", "comprimir la carpeta", "comprimí la carpeta", "comprimi la carpeta", "comprimir descargas"},
			Palabras: []string{"comprimir", "comprimí", "comprimi", "comprime"},
			Handler: func(cmd string, h *Hands) string { return h.comprimirCarpeta(cmd) },
		},
		{
			Nombre: "descomprimir",
			Frases: []string{"descomprimir archivo", "descomprimir", "descomprimí", "descomprimi", "descomprime"},
			Palabras: []string{"descomprimir", "descomprimí", "descomprimi", "descomprime"},
			Handler: func(cmd string, h *Hands) string { return h.descomprimirArchivo(cmd) },
		},
		{
			Nombre: "expulsar_disco",
			Frases: []string{"expulsar usb", "expulsar el usb", "expulsá el usb", "expulsa el usb", "sacar el pendrive", "expulsar disco"},
			Palabras: []string{"expulsar", "expulsá", "expulsa"},
			Handler: func(cmd string, h *Hands) string { return h.expulsarDisco() },
		},
		{
			Nombre: "mantener_despierto",
			Frases: []string{"mantené la pc despierta", "mantene la pc despierta", "mantener la pc despierta", "modo no dormir", "que la pc no duerma", "desactivar suspensión"},
			Palabras: []string{"despierta"},
			Handler: func(cmd string, h *Hands) string { return h.mantenerDespierto() },
		},
		{
			Nombre: "activar_suspension",
			Frases: []string{"activar suspensión", "activar suspension", "restaurar la suspensión", "restaurar la suspension", "que la pc duerma"},
			Palabras: []string{"suspensión", "suspension"},
			Handler: func(cmd string, h *Hands) string { return h.activarSuspension() },
		},
		{
			Nombre: "probar_sonido",
			Frases: []string{"probar el sonido", "probá el sonido", "proba el sonido", "test de audio", "probar audio", "probá los parlantes", "proba los parlantes", "probar los altavoces"},
			Palabras: []string{"altavoces", "parlantes"},
			Handler: func(cmd string, h *Hands) string { return h.probarSonido() },
		},
		{
			Nombre: "listar_audio",
			Frases: []string{"dispositivos de audio", "dispositivos de sonido", "salidas de audio", "qué audio tengo"},
			Palabras: []string{"audio"},
			Handler: func(cmd string, h *Hands) string { return h.listarAudio() },
		},
		{
			Nombre: "listar_camaras",
			Frases: []string{"qué cámaras tengo", "que camaras tengo", "cámaras conectadas", "camaras conectadas", "listar cámaras", "listar camaras", "webcam"},
			Palabras: []string{"cámaras", "camaras", "webcam"},
			Handler: func(cmd string, h *Hands) string { return h.listarCamaras() },
		},
		{
			Nombre: "informe_sistema",
			Frases: []string{"informe del sistema", "reporte del sistema", "resumen del sistema", "diagnóstico del sistema", "diagnostico del sistema", "estado general de la pc"},
			Palabras: []string{"informe", "reporte", "diagnóstico", "diagnostico"},
			Handler: func(cmd string, h *Hands) string { return h.informeSistema() },
		},
		{
			Nombre: "crear_carpeta",
			Frases: []string{"crear carpeta", "creá una carpeta", "crea una carpeta", "nueva carpeta", "hacé una carpeta", "hace una carpeta", "crear una carpeta"},
			Palabras: []string{"crear carpeta", "creá una carpeta", "nueva carpeta", "hacé una carpeta"},
			Handler: func(cmd string, h *Hands) string { return h.crearCarpeta(cmd) },
		},
		{
			Nombre: "crear_archivo",
			Frases: []string{"crear archivo", "creá un archivo", "crea un archivo", "nuevo archivo", "hacé un archivo", "hace un archivo", "crear un archivo de texto", "crear txt"},
			Palabras: []string{"crear archivo", "creá un archivo", "nuevo archivo"},
			Handler: func(cmd string, h *Hands) string { return h.crearArchivo(cmd) },
		},
		{
			Nombre: "buscar_archivo",
			Frases: []string{"buscar archivo", "buscame el archivo", "busca el archivo", "buscá el archivo", "encontrá el archivo", "encontra el archivo", "dónde está mi archivo", "donde esta mi archivo", "buscame un archivo"},
			Palabras: []string{"buscar archivo", "buscame el archivo", "encontrá el archivo", "encontra el archivo"},
			Handler: func(cmd string, h *Hands) string {
				nombre := extraerObjeto(cmd, []string{"buscar archivo ", "buscame el archivo ", "busca el archivo ", "buscá el archivo ", "encontrá el archivo ", "encontra el archivo ", "dónde está mi archivo ", "donde esta mi archivo ", "buscame un archivo "})
				return h.buscarArchivo(nombre)
			},
		},
		{
			Nombre: "abrir_ubicacion",
			Frases: []string{"abrir ubicación de", "abrir ubicacion de", "mostrame dónde está", "mostrame donde esta", "mostrá la carpeta de", "mostra la carpeta de", "abrir carpeta que contiene", "ubicación de mi archivo", "ubicacion de mi archivo"},
			Palabras: []string{"ubicación de", "ubicacion de", "dónde está mi", "donde esta mi"},
			Handler: func(cmd string, h *Hands) string {
				nombre := extraerObjeto(cmd, []string{"abrir ubicación de ", "abrir ubicacion de ", "mostrame dónde está ", "mostrame donde esta ", "mostrá la carpeta de ", "mostra la carpeta de ", "abrir carpeta que contiene ", "ubicación de mi archivo ", "ubicacion de mi archivo "})
				return h.abrirUbicacion(nombre)
			},
		},
		{
			Nombre: "borrar_archivo",
			Frases: []string{"borrar archivo", "borrá el archivo", "borra el archivo", "borrar el archivo", "eliminar archivo", "eliminá el archivo", "elimina el archivo", "eliminar el archivo", "mover a la papelera"},
			Palabras: []string{"borrar archivo", "borrá el archivo", "eliminar archivo", "eliminá el archivo", "papelera de reciclaje"},
			Handler: func(cmd string, h *Hands) string {
				nombre := extraerObjeto(cmd, []string{"borrar archivo ", "borrá el archivo ", "borra el archivo ", "borrar el archivo ", "eliminar archivo ", "eliminá el archivo ", "elimina el archivo ", "eliminar el archivo ", "mover a la papelera "})
				return h.borrarArchivo(nombre)
			},
		},
		{
			Nombre: "tomar_nota",
			Frases: []string{"tomá nota", "toma nota", "anotá", "anota", "escribí una nota", "escribe una nota", "tomá nota de", "anotá esto", "guarda esta nota", "guardá esta nota"},
			Palabras: []string{"anotá", "anota", "tomá nota", "guarda esta nota"},
			Handler: func(cmd string, h *Hands) string {
				texto := extraerObjeto(cmd, []string{"tomá nota de ", "tomá nota ", "toma nota de ", "toma nota ", "anotá esto ", "anotá ", "anota esto ", "anota ", "escribí una nota ", "escribe una nota ", "guarda esta nota ", "guardá esta nota "})
				return h.tomarNota(texto)
			},
		},
		{
			Nombre: "leer_notas",
			Frases: []string{"leé mis notas", "lee mis notas", "qué notas tengo", "que notas tengo", "mostrame las notas", "mostrame mis notas", "mis notas", "leé las notas", "lee las notas"},
			Palabras: []string{"mis notas", "las notas"},
			Handler: func(cmd string, h *Hands) string { return h.leerNotas() },
		},
		{
			Nombre: "ver_portapapeles",
			Frases: []string{"qué hay en el portapapeles", "que hay en el portapapeles", "contenido del portapapeles", "qué copié", "que copie"},
			Palabras: []string{"portapapeles"},
			Handler: func(cmd string, h *Hands) string { return h.verPortapapeles() },
		},
		{
			Nombre: "diagnostico",
			Frases: []string{"diagnóstico completo", "diagnostico completo", "revisá mi sistema", "revisa mi sistema", "revisame el sistema", "análisis profundo", "analisis profundo", "análisis del sistema", "analisis del sistema", "estado completo de la pc", "chequeo general", "revisá la pc", "revisa la pc", "escaneo completo"},
			Palabras: []string{"revisá mi", "revisa mi", "revisame", "análisis profundo", "analisis profundo", "chequeo", "escaneo completo"},
			Handler: func(cmd string, h *Hands) string { return h.diagnosticoCompleto() },
		},
		{
			Nombre: "salud_sistema",
			Frases: []string{"salud de mi pc", "salud del sistema", "puntaje de salud", "qué tan saludable está mi pc", "que tan saludable esta mi pc", "qué tan sano está mi pc", "que tan sano esta mi pc", "score de mi pc", "nota de mi pc", "estado de salud"},
			Palabras: []string{"salud", "puntaje de salud", "sano está", "sano esta"},
			Handler: func(cmd string, h *Hands) string { return h.saludSistema() },
		},
		{
			Nombre: "problemas_sistema",
			Frases: []string{"qué problemas tiene mi pc", "que problemas tiene mi pc", "problemas del sistema", "qué le pasa a mi pc", "que le pasa a mi pc", "encontrá problemas", "encontra problemas", "detectá problemas", "detecta problemas", "listar problemas", "quejas del sistema"},
			Palabras: []string{"problemas", "quejas del sistema"},
			Handler: func(cmd string, h *Hands) string { return h.problemasSistema() },
		},
		{
			Nombre: "mantenimiento",
			Frases: []string{"limpiá mi pc", "limpia mi pc", "mantenimiento rápido", "mantenimiento rapido", "optimizá mi pc", "optimiza mi pc", "limpieza general", "hacé mantenimiento", "hace mantenimiento", "mantenimiento general", "poné a punto mi pc"},
			Palabras: []string{"mantenimiento", "optimizá", "optimiza", "limpieza general"},
			Handler: func(cmd string, h *Hands) string { return h.mantenimientoRapido() },
		},
		{
			Nombre: "integridad",
			Frases: []string{"verificar integridad", "verificá los archivos del sistema", "verifica los archivos del sistema", "verificá la integridad", "verifica la integridad", "scaneá la integridad", "scanea la integridad", "sfc", "dism checkhealth", "chequeo de archivos del sistema"},
			Palabras: []string{"integridad", "sfc"},
			Handler: func(cmd string, h *Hands) string { return h.verificarIntegridad() },
		},
		{
			Nombre: "servicios_caidos",
			Frases: []string{"qué servicios fallaron", "que servicios fallaron", "servicios caídos", "servicios caidos", "servicios detenidos", "servicios con problemas", "servicios automáticos detenidos", "servicios automaticos detenidos"},
			Palabras: []string{"servicios caídos", "servicios caidos", "servicios detenidos", "servicios fallaron"},
			Handler: func(cmd string, h *Hands) string { return h.serviciosCaidos() },
		},
		{
			Nombre: "top_procesos",
			Frases: []string{"qué procesos consumen más", "que procesos consumen mas", "procesos que más usan cpu", "procesos que mas usan cpu", "procesos que más memoria usan", "procesos que mas memoria usan", "top procesos", "los procesos más pesados", "los procesos mas pesados"},
			Palabras: []string{"consumen", "más pesados", "mas pesados"},
			Handler: func(cmd string, h *Hands) string { return h.topProcesos() },
		},
		{
			Nombre: "eventos_error",
			Frases: []string{"eventos de error recientes", "errores del sistema recientes", "qué errores hubo", "que errores hubo", "críticas recientes", "criticas recientes", "eventos críticos", "eventos criticos", "errores recientes"},
			Palabras: []string{"eventos", "errores recientes", "críticas", "criticas"},
			Handler: func(cmd string, h *Hands) string { return h.eventosRecientes() },
		},
		{
			Nombre: "programas_inicio",
			Frases: []string{"programas de inicio", "qué se inicia con windows", "que se inicia con windows", "programas en el arranque", "aplicaciones de inicio", "programas al iniciar"},
			Palabras: []string{"programas de inicio", "aplicaciones de inicio", "se inicia con windows", "arranque"},
			Handler: func(cmd string, h *Hands) string { return h.programasInicio() },
		},
		{
			Nombre: "carpetas_grandes",
			Frases: []string{"qué ocupa espacio", "que ocupa espacio", "carpetas más pesadas", "carpetas mas pesadas", "archivos más grandes", "archivos mas grandes", "qué está llenando mi disco", "que esta llenando mi disco", "analizá mi almacenamiento", "analiza mi almacenamiento", "qué ocupa mi disco", "que ocupa mi disco"},
			Palabras: []string{"ocupa", "carpetas más pesadas", "carpetas mas pesadas"},
			Handler: func(cmd string, h *Hands) string { return h.carpetasGrandes() },
		},
		{
			Nombre: "vigilante_on",
			Frases: []string{"modo vigilante", "modo centinela", "vigilame el sistema", "vigilame la pc", "monitoreame la pc", "activá la vigilancia", "activa la vigilancia", "modo guardián", "modo guardian"},
			Palabras: []string{"vigilante", "centinela", "vigilancia", "monitoreame", "guardián", "guardian"},
			Handler: func(cmd string, h *Hands) string { return h.modoVigilante() },
		},
		{
			Nombre: "vigilante_off",
			Frases: []string{"pará la vigilancia", "para la vigilancia", "salí del modo vigilante", "salir del modo vigilante", "apagá la vigilancia", "apaga la vigilancia", "desactivá el modo vigilante", "desactiva el modo vigilante", "dejá de vigilar", "deja de vigilar", "pará de monitorear"},
			Palabras: []string{"pará la vigilancia", "para la vigilancia", "salir del modo vigilante", "apagá la vigilancia", "dejá de vigilar"},
			Handler: func(cmd string, h *Hands) string { return h.salirVigilancia() },
		},
		{
			Nombre: "vigilante_estado",
			Frases: []string{"estás vigilando", "estas vigilando", "estado de la vigilancia", "seguís monitoreando", "seguis monitoreando", "seguís vigilando", "seguis vigilando"},
			Palabras: []string{"vigilando", "monitoreando"},
			Handler: func(cmd string, h *Hands) string { return h.estadoVigilancia() },
		},
		{
			Nombre: "plan_accion",
			Frases: []string{"plan de acción", "plan de accion", "generá un plan", "genera un plan", "qué debería hacer", "que deberia hacer", "plan de mejora", "plan de optimización", "plan de optimizacion", "armá un plan", "arma un plan", "decime qué hago", "plan de mantenimiento", "qué me recomendás", "que me recomendas"},
			Palabras: []string{"plan de acción", "plan de accion", "plan de mejora", "qué debería hacer", "que deberia hacer", "armá un plan", "arma un plan"},
			Handler: func(cmd string, h *Hands) string { return h.planAccion() },
		},
		{
			Nombre: "ejecutar_plan",
			Frases: []string{"ejecutá el plan", "ejecuta el plan", "aplicá el plan", "aplica el plan", "ejecutá el plan de acción", "ejecuta el plan de accion", "hacé lo que corresponde", "hace lo que corresponde", "corré el plan", "corre el plan", "ejecutá las acciones", "ejecuta las acciones", "poné en marcha el plan"},
			Palabras: []string{"ejecutá el plan", "ejecuta el plan", "aplicá el plan", "corré el plan", "ejecutá las acciones"},
			Handler: func(cmd string, h *Hands) string { return h.ejecutarPlan() },
		},
		{
			Nombre: "crear_proyecto",
			Frases: []string{
				"crear proyecto web", "crear un proyecto web", "crear un proyecto", "crear proyecto",
				"crea un proyecto web", "crea un proyecto", "crea proyecto web", "crea una web",
				"crea una app web", "crea una aplicacion web",
				"hace un proyecto web", "hace un proyecto", "hace una web", "hace una app web",
				"nueva app web", "nuevo proyecto web", "nuevo proyecto", "scaffold",
				"armar proyecto web", "arma un proyecto", "hacer un proyecto", "hacer proyecto web",
			},
			Palabras: []string{"scaffold"},
			Handler:  func(cmd string, h *Hands) string { return h.crearProyectoWeb(cmd) },
		},
		{
			Nombre: "listar_proyectos",
			Frases: []string{
				"que proyectos tengo", "mis proyectos", "lista de proyectos", "listar proyectos",
				"que proyectos hay", "cuales proyectos tengo", "lista de mis proyectos",
				"mostrame mis proyectos", "proyectos creados",
			},
			Palabras: []string{"proyectos"},
			Handler:  func(cmd string, h *Hands) string { return h.listarProyectos() },
		},
		{
			Nombre: "compilar_proyecto",
			Frases: []string{
				"compilar proyecto", "compilar el proyecto", "compila el proyecto", "compila proyecto",
				"compilar la app", "compilar la aplicacion", "build del proyecto",
			},
			Palabras: []string{},
			Handler:  func(cmd string, h *Hands) string { return h.compilarProyecto(cmd) },
		},
		{
			Nombre: "ejecutar_proyecto",
			Frases: []string{
				"ejecutar proyecto", "ejecutar el proyecto", "ejecuta el proyecto", "ejecuta proyecto",
				"correr el proyecto", "corre el proyecto", "correr proyecto",
				"levantar el proyecto", "levanta el proyecto", "levantar proyecto",
				"iniciar el proyecto", "inicia el proyecto", "iniciar proyecto",
				"abrir el proyecto", "abre el proyecto", "abrir proyecto",
			},
			Palabras: []string{},
			Handler:  func(cmd string, h *Hands) string { return h.ejecutarProyecto(cmd) },
		},
		{
			Nombre: "detener_proyecto",
			Frases: []string{
				"detener proyecto", "detener el proyecto", "detener la app", "detener la aplicacion",
				"parar el proyecto", "para el proyecto", "parar proyecto",
				"cerrar el proyecto", "cierra el proyecto", "cerra el proyecto", "cerrar proyecto",
				"apagar el proyecto", "apaga el proyecto", "apagar proyecto",
				"matar el proyecto", "mata el proyecto", "matar proyecto",
			},
			Palabras: []string{},
			Handler:  func(cmd string, h *Hands) string { return h.detenerProyecto(cmd) },
		},
		{
			Nombre: "estado_proyecto",
			Frases: []string{
				"estado del proyecto", "esta corriendo el proyecto", "esta activo el proyecto",
				"sigue andando el proyecto", "sigue corriendo el proyecto", "esta levantado el proyecto",
				"estado de los proyectos", "que proyectos estan corriendo",
			},
			Palabras: []string{},
			Handler:  func(cmd string, h *Hands) string { return h.estadoProyecto(cmd) },
		},
		{
			Nombre: "mejorar_proyecto",
			Frases: []string{
				"mejorar proyecto", "mejorar el proyecto", "mejorar la app", "mejorar la aplicacion",
				"agregar al proyecto", "agregar algo al proyecto", "agregar una feature al proyecto",
				"agrega una feature al proyecto", "agregar feature al proyecto",
				"agregar una funcion al proyecto", "agregar una funcion a la app",
				"agregar algo a la app", "agregar un boton a la app", "agregar un boton al proyecto",
				"agregar una seccion a la app", "agregar una seccion al proyecto",
			},
			Palabras: []string{"feature"},
			Handler:  func(cmd string, h *Hands) string { return h.mejorarProyecto(cmd) },
		},
		{
			Nombre: "listar_skills",
			Frases: []string{
				"qué skills tenés", "que skills tenes", "lista de skills", "listar skills",
				"qué habilidades tenés", "que habilidades tenes", "mostrame las skills",
				"skills disponibles", "qué skills tengo instaladas",
			},
			Palabras: []string{"skills", "skill"},
			Handler:  func(cmd string, h *Hands) string { return h.listarSkills() },
		},
		{
			Nombre: "ayuda",
			Frases: []string{"ayuda", "qué podés hacer", "que podes hacer", "comandos", "qué sabes hacer", "que sabes hacer", "funciones", "instrucciones"},
			Palabras: []string{"ayuda", "comandos", "funciones", "instrucciones"},
			Handler: func(cmd string, h *Hands) string {
				fmt.Println(textoAyuda)
				return "Puede pedirme que abra aplicaciones, busque en internet, le diga la hora, escriba scripts, recuerde cosas, y más. Mire la consola para ver todos los comandos, señor."
			},
		},
	}
	return c
}

func (c *Clasificador) Clasificar(entrada string) (string, bool) {
	entrada = strings.ToLower(strings.TrimSpace(entrada))
	if entrada == "" {
		return "", false
	}

	entrada = simplificar(entrada)

	for _, intento := range c.intentos {
		if matchPorFrase(entrada, intento.Frases) {
			return intento.Nombre, true
		}
	}

	for _, intento := range c.intentos {
		if matchPorPalabras(entrada, intento.Palabras) {
			return intento.Nombre, true
		}
	}

	return "", false
}

func (c *Clasificador) Ejecutar(nombre string, entrada string, h *Hands) string {
	for _, intento := range c.intentos {
		if intento.Nombre == nombre {
			return intento.Handler(entrada, h)
		}
	}
	return ComandoNoReconocido
}

func simplificar(s string) string {
	s = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"ü", "u", "ñ", "n",
		"¿", "", "?", "", "¡", "", "!", "",
		".", "", ",", "", ":", "", ";", "",
	).Replace(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func matchPorFrase(entrada string, frases []string) bool {
	for _, f := range frases {
		if strings.Contains(entrada, f) {
			return true
		}
	}
	return false
}

func matchPorPalabras(entrada string, palabras []string) bool {
	parts := strings.Fields(entrada)
	if len(parts) == 0 {
		return false
	}
	aciertos := 0
	for _, p := range parts {
		for _, pal := range palabras {
			if p == pal {
				aciertos++
				break
			}
		}
	}
	if len(parts) <= 2 {
		return aciertos >= len(parts)
	}
	return aciertos >= 1 && aciertos >= len(parts)/2
}

func extraerObjeto(entrada string, prefijos []string) string {
	for _, p := range prefijos {
		idx := strings.Index(entrada, p)
		if idx >= 0 {
			return strings.TrimSpace(entrada[idx+len(p):])
		}
	}
	return ""
}

func extraerApp(entrada string, apps map[string]string) (string, string) {
	nombres := make([]string, 0, len(apps))
	for n := range apps {
		nombres = append(nombres, n)
	}
	sort.Slice(nombres, func(i, j int) bool {
		return len(nombres[i]) > len(nombres[j])
	})
	for _, nombre := range nombres {
		if strings.Contains(entrada, nombre) {
			return apps[nombre], nombre
		}
	}
	return "", ""
}
