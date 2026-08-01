package security

// Roles RBAC (control de acceso por rol) del panel del dueño: quién puede
// aprobar acciones sensibles, ver la auditoría o cambiar la contraseña.
// El dueño autenticado es Admin; quien usa el panel sin iniciar sesión es
// Operador (solo conversa y consulta, no autoriza ni audita).

// Rol es el nivel de acceso de un usuario del panel.
type Rol string

const (
	// RolOperador es el visitante sin sesión: conversa con Jarvis y ve el
	// estado, pero no ejerce permisos protegidos.
	RolOperador Rol = "operador"
	// RolAdmin es el dueño autenticado: acceso completo.
	RolAdmin Rol = "admin"
)

// Permiso es una acción protegida del panel web.
type Permiso string

const (
	PermisoAprobar    Permiso = "aprobar"
	PermisoDenegar    Permiso = "denegar"
	PermisoAuditoria  Permiso = "auditoria"
	PermisoContrasena Permiso = "contrasena"
)

// PermisosDe devuelve los permisos que ejerce un rol.
func PermisosDe(rol Rol) []Permiso {
	if rol == RolAdmin {
		return []Permiso{PermisoAprobar, PermisoDenegar, PermisoAuditoria, PermisoContrasena}
	}
	return nil
}

// TienePermiso indica si el rol puede ejercer el permiso dado.
func TienePermiso(rol Rol, permiso Permiso) bool {
	for _, p := range PermisosDe(rol) {
		if p == permiso {
			return true
		}
	}
	return false
}
