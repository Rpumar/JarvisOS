package core

import (
	"strings"
	"sync"
)

// palabrasVaciasSet es el set de palabras de relleno que no aportan
// significado al clasificador. Se construye una sola vez.
var palabrasVaciasSet = sync.OnceValue(func() map[string]bool {
	return map[string]bool{
		"el": true, "la": true, "los": true, "las": true,
		"un": true, "una": true, "unos": true, "unas": true,
		"me": true, "mi": true, "mis": true, "vos": true, "te": true, "che": true,
		"tu": true, "tus": true, "su": true, "sus": true,
		"que": true, "cual": true, "cuales": true, "como": true,
		"es": true, "son": true, "esta": true, "estan": true,
		"se": true, "le": true, "les": true, "lo": true, "nos": true,
		"esto": true, "eso": true, "aquello": true, "este": true, "ese": true,
		"ya": true, "no": true, "si": true, "ahora": true, "mismo": true,
		"podes": true, "puedes": true, "quiero": true, "necesito": true,
		"tengo": true, "tiene": true, "hay": true,
		"ser": true, "estar": true, "fue": true, "era": true, "soy": true,
		"somos": true, "sido": true,
		"por": true, "favor": true, "a": true, "al": true, "de": true, "del": true,
		"en": true, "con": true, "para": true, "y": true, "o": true, "ni": true,
		"hoy": true, "mañana": true, "manana": true, "ayer": true,
		"despues": true, "mas": true, "menos": true, "muy": true, "bien": true,
		"todo": true,
	}
})

// palabrasVacias devuelve el set de palabras vacías. No incluye palabras-clave
// de los intentos (hora, fecha, clima, subir, bajar, etc.) para no vaciar de
// significado frases cortas como "subi el volumen".
func palabrasVacias() map[string]bool {
	return palabrasVaciasSet()
}

func esPalabraVacia(token string) bool {
	return palabrasVacias()[token]
}

// tokensSignificativos simplifica la entrada y devuelve los tokens que no son
// palabras vacías.
func tokensSignificativos(entrada string) []string {
	var tokens []string
	for _, t := range strings.Fields(simplificar(entrada)) {
		if !esPalabraVacia(t) {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// lema normaliza formas conjugadas a su infinitivo. "sube" y "baja" quedan
// fuera a propósito: colisionan con el dominio de brillo (hands.go), donde
// "subir/bajar brillo" es otra acción.
func lema(token string) string {
	switch token {
	case "subi", "subile", "subele", "subilo", "subime":
		return "subir"
	case "bajale", "bajele", "bajalo", "bajame":
		return "bajar"
	case "reproduci":
		return "reproducir"
	case "continua":
		return "continuar"
	case "reanuda":
		return "reanudar"
	case "decime", "decis", "decile", "contame", "contale":
		return "decir"
	case "hace", "haceme", "hacelo":
		return "hacer"
	case "pone", "ponele", "poneme", "ponelo":
		return "poner"
	}
	return token
}

// coincideToken compara un token de la entrada contra una palabra de un
// intento: igualdad exacta, lemas iguales, o similitud a 1 edición para
// palabras de 5+ caracteres (tolera plurales y errores de tipeo).
func coincideToken(token, palabra string) bool {
	token = simplificar(token)
	palabra = simplificar(palabra)
	if token == palabra || lema(token) == lema(palabra) {
		return true
	}
	return len(token) >= 5 && len(palabra) >= 5 && distanciaLevenshtein(token, palabra) <= 1
}

// distanciaLevenshtein calcula la distancia de edición entre dos strings.
// Si las longitudes difieren en más de 1, devuelve esa diferencia sin
// recorrer la DP (basta para decidir "a 1 edición").
func distanciaLevenshtein(a, b string) int {
	if d := len(a) - len(b); d > 1 || d < -1 {
		if d < 0 {
			return -d
		}
		return d
	}
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			costo := 0
			if ar[i-1] != br[j-1] {
				costo = 1
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+costo)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
