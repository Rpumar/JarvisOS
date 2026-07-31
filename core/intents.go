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
			Frases: []string{"nivel de batería", "nivel de bateria", "cuánta batería queda", "cuanta bateria queda", "batería", "bateria", "qué batería tengo", "que bateria tengo", "carga"},
			Palabras: []string{"batería", "bateria", "carga"},
			Handler: func(cmd string, h *Hands) string { return h.nivelBateria() },
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
