package security

import "testing"

func TestPermisosDeRol(t *testing.T) {
	admin := PermisosDe(RolAdmin)
	esperados := []Permiso{PermisoAprobar, PermisoDenegar, PermisoAuditoria, PermisoContrasena}
	if len(admin) != len(esperados) {
		t.Fatalf("admin debería tener %d permisos, tiene %d", len(esperados), len(admin))
	}
	for _, p := range esperados {
		if !TienePermiso(RolAdmin, p) {
			t.Errorf("admin debería tener el permiso %s", p)
		}
	}

	if !TienePermiso(RolAdmin, PermisoAuditoria) {
		t.Error("admin debe poder ver la auditoría")
	}
}

func TestOperadorSinPermisos(t *testing.T) {
	for _, p := range []Permiso{PermisoAprobar, PermisoDenegar, PermisoAuditoria, PermisoContrasena} {
		if TienePermiso(RolOperador, p) {
			t.Errorf("el operador no debería tener el permiso %s", p)
		}
	}
	if len(PermisosDe(RolOperador)) != 0 {
		t.Error("el operador no debería tener ningún permiso")
	}
}
