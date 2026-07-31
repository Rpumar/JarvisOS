package core

import "testing"

func TestClasificarArmas(t *testing.T) {
	casos := []struct {
		entrada string
		esperado string
	}{
		{"hacé ping a google", "ping_sitio"},
		{"cuál es mi velocidad de internet", "velocidad_internet"},
		{"escanear la red local", "escanear_red"},
		{"limpiar la caché dns", "limpiar_dns"},
		{"mostrame los adaptadores de red", "info_red"},
		{"cuánta ram me queda", "uso_ram"},
		{"cuál es mi ip publica", "ip_publica"},
		{"cuales son mis dns", ""},
		{"red wifi", ""},
		{"limpiar archivos temporales", "limpiar_temporales"},
		{"organizá mis descargas", "organizar_descargas"},
		{"grabá la pantalla", "grabar_pantalla"},
		{"poné el modo oscuro", "modo_oscuro"},
		{"está activo el firewall", "firewall"},
		{"qué puertos están abiertos", "puertos_uso"},
		{"aplicaciones usando internet", "procesos_red"},
		{"sesiones abiertas", "sesiones_activas"},
		{"crear rutina trabajo", "rutina"},
		{"abrir descargas", ""},
		{"avisame que la comida está lista", "notificacion"},
		{"comprimí la carpeta descargas", "comprimir"},
		{"descomprimí el archivo backup", "descomprimir"},
		{"expulsá el usb", "expulsar_disco"},
		{"mantené la pc despierta", "mantener_despierto"},
		{"activar suspensión", "activar_suspension"},
		{"probá el sonido", "probar_sonido"},
		{"qué dispositivos de audio tengo", "listar_audio"},
		{"listar cámaras", "listar_camaras"},
		{"informe del sistema", "informe_sistema"},
		{"qué hay en el portapapeles", "ver_portapapeles"},
	}
	c := NuevoClasificador()
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}
