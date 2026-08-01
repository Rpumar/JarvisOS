// Package security clasifica el riesgo de cada acción que el agente
// intenta ejecutar, antes de que toque el sistema. Es la primera
// barrera de F2: distinguir lo seguro de lo que exige aprobación.
package security

import "strings"

// NivelRiesgo indica cuánta autorización requiere una acción.
type NivelRiesgo int

const (
	// Segura se ejecuta sin pedir nada.
	Segura NivelRiesgo = iota
	// RequiereAprobacion necesita confirmación del dueño (PIN o botón).
	RequiereAprobacion
	// Denegada nunca se ejecuta (bloqueada de antemano).
	Denegada
)

// Clasificacion resume el análisis de una acción.
type Clasificacion struct {
	Nivel       NivelRiesgo
	Descripcion string
}

type frasePeligrosa struct {
	frase  string
	accion string
}

// frasesPeligrosas son acciones que alteran el equipo o datos. La
// coincidencia es por subcadena para tolerar variaciones de redacción.
var frasesPeligrosas = []frasePeligrosa{
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
	// Acciones externas: envían datos fuera del equipo, exigen aprobación.
	{"enviar email", "enviar un correo electrónico"},
	{"enviar un email", "enviar un correo electrónico"},
	{"enviar correo", "enviar un correo electrónico"},
	{"enviar un correo", "enviar un correo electrónico"},
	{"enviar mail", "enviar un correo electrónico"},
	{"enviar un mail", "enviar un correo electrónico"},
	{"mandar un email", "enviar un correo electrónico"},
	{"mandar un correo", "enviar un correo electrónico"},
	{"mandar un mail", "enviar un correo electrónico"},
}

// Clasificar devuelve el riesgo de una acción y una descripción de qué
// supone. Excepciones explícitas (ej. "activar suspensión", que solo
// configura un plan de energía) quedan como Segura.
func Clasificar(entrada string) Clasificacion {
	entrada = strings.TrimSpace(entrada)
	if strings.Contains(entrada, "activar suspensión") || strings.Contains(entrada, "activar suspension") ||
		strings.Contains(entrada, "desactivar suspensión") || strings.Contains(entrada, "desactivar suspension") {
		return Clasificacion{Nivel: Segura}
	}
	for _, p := range frasesPeligrosas {
		if strings.Contains(entrada, p.frase) {
			return Clasificacion{Nivel: RequiereAprobacion, Descripcion: p.accion}
		}
	}
	return Clasificacion{Nivel: Segura}
}
