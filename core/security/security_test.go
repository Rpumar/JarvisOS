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
		{"enviar un email a juan@gmail.com", RequiereAprobacion, "enviar un correo electrónico"},
		{"enviar correo al cliente", RequiereAprobacion, "enviar un correo electrónico"},
		{"mandar un mail de resumen", RequiereAprobacion, "enviar un correo electrónico"},
		{"leer mis correos", Segura, ""},
		{"publicar en x que estoy trabajando", RequiereAprobacion, "publicar en X (Twitter)"},
		{"publicar un tuit", RequiereAprobacion, "publicar en X (Twitter)"},
		{"twittear la novedad", RequiereAprobacion, "publicar en X (Twitter)"},
		{"publicar en linkedin un aviso", RequiereAprobacion, "publicar en LinkedIn"},
		{"publicar un post en linkedin", RequiereAprobacion, "publicar en LinkedIn"},
		{"rellenar el formulario factura", RequiereAprobacion, "rellenar un formulario web"},
		{"rellená el formulario presupuesto", RequiereAprobacion, "rellenar un formulario web"},
		{"completar el formulario de alta", RequiereAprobacion, "rellenar un formulario web"},
		{"qué formularios tengo", Segura, ""},
		{"creá un formulario factura para facturacion.com", Segura, ""},
		{"agregá el campo email con pedidos@x.com a factura", Segura, ""},
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
