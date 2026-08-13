package core

import "testing"

func TestTokensSignificativos(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado []string
	}{
		{"subí el volumen", []string{"subi", "volumen"}},
		{"la hora del partido", []string{"hora", "partido"}},
		{"qué es un agujero negro", []string{"agujero", "negro"}},
		{"", nil},
	}
	for _, cso := range casos {
		got := tokensSignificativos(cso.entrada)
		if len(got) != len(cso.esperado) {
			t.Errorf("tokensSignificativos(%q) = %v, esperaba %v", cso.entrada, got, cso.esperado)
			continue
		}
		for i := range got {
			if got[i] != cso.esperado[i] {
				t.Errorf("tokensSignificativos(%q) = %v, esperaba %v", cso.entrada, got, cso.esperado)
				break
			}
		}
	}
}

func TestPalabrasVaciasNoSonPalabrasClave(t *testing.T) {
	claves := []string{"hora", "fecha", "clima", "ip", "ping", "pdf", "wifi", "ram", "tiempo", "temperatura", "grados", "nublado", "bateria", "volumen", "brillo", "musica", "cancion", "dia", "estamos", "subir", "bajar"}
	for _, c := range claves {
		if esPalabraVacia(c) {
			t.Errorf("esPalabraVacia(%q) = true, no debe vaciar palabras-clave", c)
		}
	}
}

func TestLema(t *testing.T) {
	casos := map[string]string{
		"subele":    "subir",
		"subi":      "subir",
		"bajele":    "bajar",
		"decime":    "decir",
		"hace":      "hacer",
		"pone":      "poner",
		"reproduci": "reproducir",
		"xyz":       "xyz",
		"sube":      "sube",
	}
	for entrada, esperado := range casos {
		if got := lema(entrada); got != esperado {
			t.Errorf("lema(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

func TestCoincideToken(t *testing.T) {
	casos := []struct {
		token, palabra string
		esperado       bool
	}{
		{"hora", "hora", true},
		{"subele", "subir", true},
		{"subele", "sube", false},
		{"clima", "climas", true},
		{"camara", "camaras", true},
		{"ip", "ips", false},
		{"subir", "bajar", false},
	}
	for _, cso := range casos {
		if got := coincideToken(cso.token, cso.palabra); got != cso.esperado {
			t.Errorf("coincideToken(%q, %q) = %v, esperaba %v", cso.token, cso.palabra, got, cso.esperado)
		}
	}
}

func TestDistanciaLevenshtein(t *testing.T) {
	casos := []struct {
		a, b     string
		esperado int
	}{
		{"kitten", "sitting", 3},
		{"hora", "hora", 0},
		{"clima", "climas", 1},
		{"abc", "xyz", 3},
		{"abcdef", "xyz", 3},
	}
	for _, cso := range casos {
		if got := distanciaLevenshtein(cso.a, cso.b); got != cso.esperado {
			t.Errorf("distanciaLevenshtein(%q, %q) = %d, esperaba %d", cso.a, cso.b, got, cso.esperado)
		}
	}
}
