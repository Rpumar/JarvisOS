package core

import (
	"strings"

	"JarvisOS/core/audit"
)

// EstablecerContrasena configura la contraseña de acceso al panel web
// (6-32 caracteres). Cuando está definida, el panel pasa a exigir inicio de
// sesión: el dueño autenticado es Admin; el resto queda como Operador. Se
// normaliza a minúsculas para que "MiClave" y "miclave" sean lo mismo.
func (h *Hands) EstablecerContrasena(clave string) string {
	clave = normalizarContrasena(clave)
	if len(clave) < 6 || len(clave) > 32 {
		return "La contraseña debe tener entre 6 y 32 caracteres, señor."
	}
	hash := HashTexto(clave)
	if h.ContrasenaSetter != nil && !h.ContrasenaSetter(hash) {
		return "No pude guardar la contraseña, señor. Verifique los permisos del archivo de configuración."
	}
	h.ContrasenaHash = hash
	return "Contraseña de acceso configurada, señor. El panel web pedirá inicio de sesión desde ahora."
}

// contrasenaValida indica si la clave coincide con la configurada. Sin
// contraseña configurada, el acceso está abierto (equivale a Admin).
func (h *Hands) contrasenaValida(clave string) bool {
	if h.ContrasenaHash == "" {
		return true
	}
	return HashTexto(normalizarContrasena(clave)) == h.ContrasenaHash
}

func normalizarContrasena(clave string) string {
	return strings.ToLower(strings.TrimSpace(clave))
}

// AuditoriaPanel devuelve las entradas recientes del registro inmutable para
// el visor del panel del dueño.
func (h *Hands) AuditoriaPanel() []audit.Entrada {
	if h.Auditoria == nil {
		return nil
	}
	return h.Auditoria.Recientes(200)
}

// extraerContrasena toma el texto que sigue a "de acceso / acceso /
// contraseña / password" en un comando de voz.
func extraerContrasena(entrada string) string {
	// Busca el marcador que aparezca más tarde: "configurá la contraseña de
	// acceso miClave" → "miClave" (con "acceso" como último marcador), y
	// "configurá la contraseña 123456" → "123456".
	idx := -1
	marcadorLargo := 0
	for _, marcador := range []string{"de acceso", "acceso", "contraseña", "contrasena", "password"} {
		if i := strings.LastIndex(entrada, marcador); i > idx {
			idx = i
			marcadorLargo = len(marcador)
		}
	}
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(entrada[idx+marcadorLargo:])
}
