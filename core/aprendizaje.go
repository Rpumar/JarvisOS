package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ComandoAprendido struct {
	Frases    []string `json:"frases"`
	Comando   string   `json:"comando"`
	CreadoEl  string   `json:"creado_el"`
	Usos      int      `json:"usos"`
}

type RegistroAprendizaje struct {
	mu        sync.RWMutex
	comandos  []ComandoAprendido
	ruta      string
}

func NuevoRegistroAprendizaje(ruta string) *RegistroAprendizaje {
	r := &RegistroAprendizaje{ruta: ruta}
	r.cargar()
	return r
}

func (r *RegistroAprendizaje) cargar() {
	contenido, err := os.ReadFile(r.ruta)
	if err != nil {
		r.comandos = []ComandoAprendido{}
		return
	}
	if err := json.Unmarshal(contenido, &r.comandos); err != nil {
		r.comandos = []ComandoAprendido{}
	}
}

func (r *RegistroAprendizaje) guardar() {
	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o700); err != nil {
		return
	}
	datos, _ := json.MarshalIndent(r.comandos, "", "  ")
	if err := os.WriteFile(r.ruta, datos, 0o600); err != nil {
		return
	}
}

func (r *RegistroAprendizaje) Aprender(frases []string, comando string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.comandos {
		if c.Comando == comando {
			r.comandos[i].Frases = agregarFrasesNoRepetidas(c.Frases, frases)
			r.comandos[i].Usos++
			r.guardar()
			return
		}
	}

	r.comandos = append(r.comandos, ComandoAprendido{
		Frases:   frases,
		Comando:  comando,
		CreadoEl: time.Now().Format(time.RFC3339),
		Usos:     1,
	})
	r.guardar()
}

func (r *RegistroAprendizaje) Buscar(entrada string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entrada = strings.ToLower(strings.TrimSpace(entrada))
	entrada = simplificar(entrada)

	var mejor string
	var mejorScore int

	for _, c := range r.comandos {
		for _, frase := range c.Frases {
			f := simplificar(strings.ToLower(frase))
			if strings.Contains(entrada, f) || strings.Contains(f, entrada) {
				score := len(f)
				if score > mejorScore {
					mejorScore = score
					mejor = c.Comando
				}
			}
		}
	}

	if mejor != "" {
		return mejor, true
	}
	return "", false
}

func (r *RegistroAprendizaje) Listar() []ComandoAprendido {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]ComandoAprendido, len(r.comandos))
	copy(cp, r.comandos)
	return cp
}

func (r *RegistroAprendizaje) Olvidar(comando string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.comandos {
		if c.Comando == comando {
			r.comandos = append(r.comandos[:i], r.comandos[i+1:]...)
			r.guardar()
			return true
		}
	}
	return false
}

func (r *RegistroAprendizaje) AprenderDeInteraccion(entradaUsuario string, comandoEjecutado string) {
	frases := generarFrases(entradaUsuario)
	r.Aprender(frases, comandoEjecutado)
}

func agregarFrasesNoRepetidas(existentes, nuevas []string) []string {
	set := make(map[string]bool, len(existentes))
	for _, f := range existentes {
		set[f] = true
	}
	for _, f := range nuevas {
		if !set[f] {
			existentes = append(existentes, f)
		}
	}
	return existentes
}

func generarFrases(entrada string) []string {
	entrada = strings.ToLower(strings.TrimSpace(entrada))
	palabras := strings.Fields(entrada)
	if len(palabras) <= 3 {
		return []string{entrada}
	}

	frases := []string{entrada}

	if len(palabras) >= 4 {
		frases = append(frases, strings.Join(palabras[:3], " "))
	}

	for i := 0; i < len(palabras); i++ {
		for j := i + 2; j <= len(palabras) && j <= i+4; j++ {
			f := strings.Join(palabras[i:j], " ")
			if f != entrada {
				frases = append(frases, f)
			}
		}
	}

	return frases
}

var _ = fmt.Sprintf
