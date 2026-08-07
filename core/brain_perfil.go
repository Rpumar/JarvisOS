package core

import (
	"fmt"
	"strings"
)

// manejarPerfil atiende los comandos de voz sobre quién opera Jarvis:
// seleccionar el perfil activo (dueño, admin, empleado o un usuario
// registrado), listar usuarios, registrar o borrar personas. Devuelve false
// si el texto no era un comando de perfil.
func (b *Brain) manejarPerfil(original string) (string, bool) {
	if b.perfil == nil {
		return "", false
	}
	original = strings.TrimSpace(original)
	entrada := strings.ToLower(original)
	if entrada == "" {
		return "", false
	}

	// Consulta: qué perfil está activo / quiénes son los usuarios.
	if contieneAlguna(entrada, []string{
		"que perfil esta activo", "qué perfil está activo", "quien esta operando",
		"quién está operando", "quien opero ahora", "quién operó ahora", "que usuario soy",
		"qué usuario soy", "con que perfil trabajo", "con qué perfil trabajo",
		"de quien soy", "de quién soy",
	}) {
		return fmt.Sprintf("El perfil activo es %s, señor.", b.perfil.Activo()), true
	}
	if contieneAlguna(entrada, []string{
		"que perfiles hay", "qué perfiles hay", "que usuarios hay", "qué usuarios hay",
		"quienes estan registrados", "quiénes están registrados", "cuales son los usuarios",
		"cuáles son los usuarios", "cuales son los perfiles", "cuáles son los perfiles",
		"mostrame los usuarios", "mostrame los perfiles",
	}) {
		return b.resumenPerfiles(), true
	}

	// Registro: "agregá (al) usuario X como admin", "registra un usuario X".
	if matchPorFrases(entrada, []string{"agregar un", "agregá un", "agrega un", "agregar al", "agregá al", "agrega al", "registra un", "registrá un", "alta de usuario", "dar de alta un"}) {
		return b.registrarPerfil(entrada, original), true
	}

	// Borrado: "borrá (al) usuario X".
	if matchPorFrases(entrada, []string{"borrar un", "borrá un", "borra un", "borrar al", "borrá al", "borra al", "eliminar un", "eliminá un", "eliminar al", "eliminá al", "baja de usuario", "dar de baja un"}) {
		return b.borrarPerfil(entrada, original), true
	}

	// Selección: "operá como admin", "soy el empleado", "perfil X".
	if prefijo, nombre, ok := extraerPerfilSeleccion(entrada); ok {
		if b.perfil.Seleccionar(nombre) {
			return fmt.Sprintf("Perfil activado: %s, señor.", b.perfil.Activo()), true
		}
		if prefijo != "" {
			return fmt.Sprintf("No encontré el perfil '%s'. Diga 'qué perfiles hay' para ver los registrados, señor.", nombre), true
		}
		return "", false
	}

	return "", false
}

// resumenPerfiles describe los usuarios registrados y el activo.
func (b *Brain) resumenPerfiles() string {
	usuarios := b.perfil.Usuarios()
	if len(usuarios) == 0 {
		return fmt.Sprintf("No hay usuarios registrados todavía. Puede registrar uno con 'agregá al usuario X como admin'. El perfil activo es %s, señor.", b.perfil.Activo())
	}
	nombres := make([]string, 0, len(usuarios))
	for _, u := range usuarios {
		nombre := u.Nombre
		if u.Area != "" {
			nombre += " (" + u.Area + ")"
		}
		nombres = append(nombres, fmt.Sprintf("%s: %s", nombre, u.Rol))
	}
	return fmt.Sprintf("Usuarios registrados: %s. Perfil activo: %s, señor.", strings.Join(nombres, ", "), b.perfil.Activo())
}

// extraerPerfilSeleccion detecta "operá como X", "soy X", "perfil X" y
// devuelve el nombre a seleccionar. prefijo queda vacío si el texto no era
// claramente una selección de perfil.
func extraerPerfilSeleccion(entrada string) (prefijo, nombre string, ok bool) {
	for _, p := range []string{
		"opera como", "operá como", "actua como perfil", "actuá como perfil",
		"cambia el perfil a", "cambiá el perfil a", "cambiar el perfil a",
		"cambia de perfil a", "cambiá de perfil a", "cambiar de perfil a",
		"perfil", "usuario", "soy", "soy el", "soy la",
	} {
		idx := strings.Index(entrada, p)
		if idx < 0 {
			continue
		}
		resto := strings.Trim(strings.TrimSpace(entrada[idx+len(p):]), " .,")
		resto = strings.TrimPrefix(resto, "a ")
		if resto == "" {
			continue
		}
		// Evitar capturar "soy" de frases no relacionadas.
		if p == "soy" && !matchPorFrases(entrada, []string{"soy el", "soy la", "soy dueno", "soy dueño", "soy admin", "soy empleado", "soy el dueno", "soy el dueño", "soy el admin", "soy el empleado", "soy la empleada", "soy operador", "soy admin", "soy el administrador"}) {
			return "", "", false
		}
		return p, resto, true
	}
	return "", "", false
}

// registrarPerfil captura "agregá al usuario X como admin del área Y".
func (b *Brain) registrarPerfil(entrada, original string) string {
	nombre := extraerPerfilNombre(entrada, original)
	if nombre == "" {
		return "¿A quién registro? Diga por ejemplo 'agregá al usuario Ana como admin', señor."
	}
	rol := extraerPerfilRol(entrada)
	area := extraerPerfilArea(entrada, original)
	if b.perfil.LimiteAlcanzado() {
		if _, yaExiste := b.perfil.Obtener(nombre); yaExiste {
			b.perfil.AgregarUsuario(nombre, area, rol)
			return fmt.Sprintf("El usuario %s ya existía y quedó actualizado como %s%s, señor.", nombre, rol, sufijoArea(area))
		}
		libres := b.perfil.PuestosLibres()
		return fmt.Sprintf("No puedo registrar más puestos, señor: su licencia está completa (quedan %d libre(s)). Consulte 'qué licencia tengo' o activen un plan mayor.", libres)
	}
	if b.perfil.AgregarUsuario(nombre, area, rol) {
		return fmt.Sprintf("Usuario %s registrado como %s%s, señor.", nombre, rol, sufijoArea(area))
	}
	return fmt.Sprintf("El usuario %s ya existía y quedó actualizado como %s%s, señor.", nombre, rol, sufijoArea(area))
}

// borrarPerfil captura "borrá al usuario X".
func (b *Brain) borrarPerfil(entrada, original string) string {
	nombre := extraerPerfilNombre(entrada, original)
	if nombre == "" {
		return "¿A quién borro? Diga 'borrá al usuario Ana', señor."
	}
	if b.perfil.Eliminar(nombre) {
		return fmt.Sprintf("Usuario %s eliminado, señor.", nombre)
	}
	return fmt.Sprintf("No encontré al usuario %s, señor.", nombre)
}

// extraerPerfilNombre captura el nombre luego de "usuario X" / "al X".
func extraerPerfilNombre(entrada, original string) string {
	idx := strings.Index(entrada, "usuario")
	if idx >= 0 {
		resto := strings.TrimSpace(entrada[idx+len("usuario"):])
		resto = strings.TrimPrefix(resto, "a ")
		resto = strings.Trim(resto, " .,")
		if resto != "" {
			// Recortar en " como ", " del área ", " con rol ".
			partes := splitPrimero(resto, " como ", " del area ", " del área ", " con rol ", " de area ", " de área ")
			nombre := strings.TrimSpace(partes)
			if nombre != "" {
				return capitalizarNombre(nombre)
			}
		}
	}
	return ""
}

// extraerPerfilRol detecta el nivel mencionado.
func extraerPerfilRol(entrada string) string {
	switch {
	case strings.Contains(entrada, "dueno"), strings.Contains(entrada, "dueño"):
		return PerfilDueno
	case strings.Contains(entrada, "admin"), strings.Contains(entrada, "admin"):
		return PerfilAdmin
	case strings.Contains(entrada, "empleado"), strings.Contains(entrada, "empleada"):
		return PerfilEmpleado
	default:
		return PerfilEmpleado
	}
}

// extraerPerfilArea captura " del área Y".
func extraerPerfilArea(entrada, original string) string {
	for _, p := range []string{" del area ", " del área ", " de area ", " de área ", " del sector "} {
		idx := strings.Index(entrada, p)
		if idx >= 0 {
			area := strings.TrimSpace(entrada[idx+len(p):])
			area = strings.Trim(area, " .,")
			if area != "" {
				return capitalizarNombre(area)
			}
		}
	}
	return ""
}

func sufijoArea(area string) string {
	if area == "" {
		return ""
	}
	return " del área " + area
}

// splitPrimero corta el string en el primer separador encontrado.
func splitPrimero(s string, seps ...string) string {
	for _, sep := range seps {
		if idx := strings.Index(s, sep); idx >= 0 {
			return s[:idx]
		}
	}
	return s
}

// capitalizarNombre pone la primera letra en mayúscula.
func capitalizarNombre(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// matchPorFrases indica si la entrada contiene alguna de las frases.
func matchPorFrases(entrada string, frases []string) bool {
	for _, f := range frases {
		if strings.Contains(entrada, f) {
			return true
		}
	}
	return false
}
