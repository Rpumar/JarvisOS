package core

import "math/rand"

// Este archivo es la Fase 1 del plan de mejoras: personalidad y carisma.
// En vez de una frase fija repetida en cada interacción, se elige al azar
// entre variantes con un tono más seguro, cálido y con algo de ingenio seco
// -inspirado en el arquetipo de mayordomo-IA competente, no en líneas de
// ninguna película puntual-. Se concentra en los puntos de mayor frecuencia
// (saludo, confusión, despedida) para que el cambio se note de verdad sin
// tocar la arquitectura del proyecto.

var saludos = []string{
	"A la orden, señor.",
	"Diga, señor.",
	"Presente, señor. ¿Qué necesita?",
	"Escuchando, señor.",
	"A su servicio, señor.",
	"Aquí estoy, señor. Usted dirá.",
}

var despedidas = []string{
	"Hasta luego, señor. Fue un placer.",
	"Apagando sistemas, señor. Que tenga un buen día.",
	"Me retiro, señor. Cualquier cosa, ya sabe dónde encontrarme.",
	"Sistemas fuera de línea. Hasta la próxima, señor.",
	"Cerrando todo, señor. Nos vemos pronto.",
}

var respuestasConfusion = []string{
	"No reconozco ese comando, señor. Diga 'ayuda' si quiere ver qué sé hacer.",
	"Eso se me escapó por completo. Pruebe con 'ayuda'.",
	"Ahí me perdió, señor. 'Ayuda' le muestra mis límites, que no son pocos.",
	"No le entendí bien. ¿Puede repetirlo, o decir 'ayuda'?",
	"Todavía no sé hacer eso. 'Ayuda' tiene la lista completa.",
}

var confirmacionesGenericas = []string{
	"Abriendo, señor.",
	"Ya casi está.",
	"Enseguida, señor.",
	"Dando inicio.",
	"Como usted ordene.",
}

func fraseAlAzar(opciones []string) string {
	if len(opciones) == 0 {
		return ""
	}
	return opciones[rand.Intn(len(opciones))]
}

// Saludo devuelve una variante al azar del saludo de activación por voz.
func Saludo() string { return fraseAlAzar(saludos) }

// Despedida devuelve una variante al azar del cierre al apagarse.
func Despedida() string { return fraseAlAzar(despedidas) }

// RespuestaConfusion devuelve una variante al azar para cuando ningún
// comando coincide (reemplaza la línea fija que había antes en Brain.Process).
func RespuestaConfusion() string { return fraseAlAzar(respuestasConfusion) }

// ConfirmacionGenerica devuelve una variante al azar para confirmar una
// acción simple (hoy se usa en abrirApp, el comando más repetido en la
// práctica; queda disponible para reusar en otros lados si hace falta).
func ConfirmacionGenerica() string { return fraseAlAzar(confirmacionesGenericas) }
