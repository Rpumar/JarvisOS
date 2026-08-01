package security

import "testing"

func TestClasificar(t *testing.T) {
	casos := []struct {
		entrada  string
		nivel    NivelRiesgo
		desc     string
	}{
		{"abrir chrome", Segura, ""},
		{"qué hora es", Segura, ""},
		{"tomar nota de la reunión", Segura, ""},
		{"suspender la pc", RequiereAprobacion, "suspender el equipo"},
		{"vaciar papelera", RequiereAprobacion, "vaciar la papelera de reciclaje"},
		{"borrar la carpeta de respaldos", RequiereAprobacion, "borrar archivos o datos"},
		{"instalar el programa", RequiereAprobacion, "instalar un programa"},
		{"formatear el disco", RequiereAprobacion, "formatear un disco"},
		{"activar suspensión", Segura, ""},
		{"desactivar suspensión", Segura, ""},
		{"mantené la pc despierta", Segura, ""},
	}
	for _, c := range casos {
		got := Clasificar(c.entrada)
		if got.Nivel != c.nivel {
			t.Errorf("Clasificar(%q).Nivel = %v, esperaba %v (%q)", c.entrada, got.Nivel, c.nivel, got.Descripcion)
		}
		if c.nivel != Segura && got.Descripcion != c.desc {
			t.Errorf("Clasificar(%q).Descripcion = %q, esperaba %q", c.entrada, got.Descripcion, c.desc)
		}
	}
}
