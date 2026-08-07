package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Perfiles del usuario (RBAC de F2): quién está operando Jarvis y con qué
// nivel de autoridad. Diferente de RolesManager (especialidad de la IA);
// acá es el puesto de la persona que da la orden.
const (
	PerfilDueno    = "dueno"
	PerfilAdmin    = "admin"
	PerfilEmpleado = "empleado"
)

// PerfilUsuario es una persona registrada en la instalación.
type PerfilUsuario struct {
	Nombre string `json:"nombre"`
	Area   string `json:"area,omitempty"`
	Rol    string `json:"rol"`
}

// GestorPerfil administra la lista de usuarios y cuál está activo.
type GestorPerfil struct {
	mu       sync.RWMutex
	ruta     string
	usuarios []PerfilUsuario
	activo   string
	// LimitePuestos es el tope de usuarios registrados que permite la
	// licencia (0 = sin límite, modo piloto/desarrollo).
	LimitePuestos int
}

// NuevoGestorPerfil carga los usuarios y el perfil activo persistidos.
func NuevoGestorPerfil(ruta string) *GestorPerfil {
	g := &GestorPerfil{ruta: ruta, activo: PerfilDueno}
	g.cargar()
	return g
}

func (g *GestorPerfil) cargar() {
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var estructura struct {
		Usuarios []PerfilUsuario `json:"usuarios"`
		Activo   string          `json:"activo"`
	}
	if err := json.Unmarshal(datos, &estructura); err != nil {
		return
	}
	if estructura.Usuarios != nil {
		g.usuarios = estructura.Usuarios
	}
	if estructura.Activo != "" {
		g.activo = estructura.Activo
	}
}

// Usuarios devuelve la lista de usuarios registrados.
func (g *GestorPerfil) Usuarios() []PerfilUsuario {
	g.mu.RLock()
	defer g.mu.RUnlock()
	res := make([]PerfilUsuario, len(g.usuarios))
	copy(res, g.usuarios)
	return res
}

// Activo devuelve el nombre del perfil activo (o un nivel directo).
func (g *GestorPerfil) Activo() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.activo
}

// Obtener devuelve el rol de un usuario registrado y si existe.
func (g *GestorPerfil) Obtener(nombre string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, u := range g.usuarios {
		if u.Nombre == nombre {
			return u.Rol, true
		}
	}
	return "", false
}

// ActivoRol devuelve el nivel de autoridad del perfil activo (dueno, admin
// o empleado), resolviendo también los nombres directos sin registrar.
func (g *GestorPerfil) ActivoRol() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.rolDeLocked(g.activo)
}

func (g *GestorPerfil) rolDeLocked(nombre string) string {
	switch nombre {
	case PerfilDueno, PerfilAdmin, PerfilEmpleado:
		return nombre
	}
	for _, u := range g.usuarios {
		if u.Nombre == nombre {
			return u.Rol
		}
	}
	return PerfilEmpleado
}

// Seleccionar cambia el perfil activo por nombre de usuario o nivel.
func (g *GestorPerfil) Seleccionar(texto string) bool {
	texto = strings.ToLower(strings.TrimSpace(texto))
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, u := range g.usuarios {
		if u.Nombre == texto {
			g.activo = u.Nombre
			g.guardar()
			return true
		}
	}

	// Coincidencia parcial contra el nombre.
	for _, u := range g.usuarios {
		if strings.Contains(strings.ToLower(u.Nombre), texto) {
			g.activo = u.Nombre
			g.guardar()
			return true
		}
	}

	switch texto {
	case "dueno", "dueño", "el dueno", "el dueño":
		g.activo = PerfilDueno
	case "admin", "administrador", "el admin", "administradores", "administración":
		g.activo = PerfilAdmin
	case "empleado", "el empleado", "la empleada", "empleada":
		g.activo = PerfilEmpleado
	default:
		return false
	}
	g.guardar()
	return true
}

// AgregarUsuario registra una persona con su nivel. Devuelve true si se
// creó, false si ya existía (en ese caso actualiza sus datos) o si la
// licencia ya no permite más puestos.
func (g *GestorPerfil) AgregarUsuario(nombre, area, rol string) bool {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return false
	}
	rol = normalizarRolPerfil(rol)
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.usuarios {
		if g.usuarios[i].Nombre == nombre {
			g.usuarios[i].Area = area
			g.usuarios[i].Rol = rol
			g.guardar()
			return false
		}
	}
	if g.LimitePuestos > 0 && len(g.usuarios) >= g.LimitePuestos {
		return false
	}
	g.usuarios = append(g.usuarios, PerfilUsuario{Nombre: nombre, Area: area, Rol: rol})
	g.guardar()
	return true
}

// LimiteAlcanzado indica si el tope de puestos de la licencia impide registrar
// un usuario nuevo (con LimitePuestos <= 0 nunca se alcanza).
func (g *GestorPerfil) LimiteAlcanzado() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.LimitePuestos > 0 && len(g.usuarios) >= g.LimitePuestos
}

// PuestosLibres devuelve cuántos puestos quedan disponibles (0 si ya no hay,
// -1 si no hay límite).
func (g *GestorPerfil) PuestosLibres() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.LimitePuestos <= 0 {
		return -1
	}
	libres := g.LimitePuestos - len(g.usuarios)
	if libres < 0 {
		return 0
	}
	return libres
}

// Eliminar quita un usuario; si era el activo, vuelve al dueño.
func (g *GestorPerfil) Eliminar(nombre string) bool {
	nombre = strings.TrimSpace(nombre)
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.usuarios {
		if g.usuarios[i].Nombre == nombre {
			g.usuarios = append(g.usuarios[:i], g.usuarios[i+1:]...)
			if g.activo == nombre {
				g.activo = PerfilDueno
			}
			g.guardar()
			return true
		}
	}
	return false
}

// normalizarRolPerfil traduce variantes de voz al valor interno.
func normalizarRolPerfil(rol string) string {
	rol = strings.ToLower(strings.TrimSpace(rol))
	switch {
	case strings.Contains(rol, "due"), strings.Contains(rol, "dueño"):
		return PerfilDueno
	case strings.Contains(rol, "admin"), strings.Contains(rol, "admin"):
		return PerfilAdmin
	default:
		return PerfilEmpleado
	}
}

func (g *GestorPerfil) guardar() {
	estructura := struct {
		Usuarios []PerfilUsuario `json:"usuarios"`
		Activo   string          `json:"activo"`
	}{Usuarios: g.usuarios, Activo: g.activo}
	datos, err := json.MarshalIndent(estructura, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(g.ruta), 0o700)
	_ = os.WriteFile(g.ruta, datos, 0o600)
}
