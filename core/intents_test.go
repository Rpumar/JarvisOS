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
