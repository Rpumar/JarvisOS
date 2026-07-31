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
			Nombre: "chiste",
			Frases: []string{"contá un chiste", "conta un chiste", "decí un chiste", "deci un chiste", "chiste", "hacerme reír", "hacerme reir", "un chiste"},
			Palabras: []string{"chiste", "reír", "reir"},
			Handler: func(cmd string, h *Hands) string { return h.contarChiste() },
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
			Nombre: "ver_portapapeles",
			Frases: []string{"qué hay en el portapapeles", "que hay en el portapapeles", "contenido del portapapeles", "qué copié", "que copie"},
			Palabras: []string{"portapapeles"},
			Handler: func(cmd string, h *Hands) string { return h.verPortapapeles() },
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
