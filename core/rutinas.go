package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Rutina struct {
	Nombre string   `json:"nombre"`
	Pasos  []string `json:"pasos"`
}

type RutinaManager struct {
	mu     sync.Mutex
	ruta   string
	rutinas map[string][]string
}

func NuevoRutinaManager(ruta string) *RutinaManager {
	m := &RutinaManager{ruta: ruta, rutinas: make(map[string][]string)}
	m.cargar()
	return m
}

func (m *RutinaManager) cargar() {
	datos, err := os.ReadFile(m.ruta)
	if err != nil {
		return
	}
	var listas []Rutina
	if err := json.Unmarshal(datos, &listas); err != nil {
		return
	}
	for _, r := range listas {
		m.rutinas[r.Nombre] = r.Pasos
	}
}

func (m *RutinaManager) guardar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(m.ruta), 0o700); err != nil {
		return
	}
	listas := make([]Rutina, 0, len(m.rutinas))
	for nombre, pasos := range m.rutinas {
		listas = append(listas, Rutina{Nombre: nombre, Pasos: pasos})
	}
	sort.Slice(listas, func(i, j int) bool { return listas[i].Nombre < listas[j].Nombre })
	datos, _ := json.MarshalIndent(listas, "", "  ")
	if err := os.WriteFile(m.ruta, datos, 0o600); err != nil {
		return
	}
}

func (m *RutinaManager) Crear(nombre string, pasos []string) {
	m.mu.Lock()
	m.rutinas[nombre] = pasos
	m.mu.Unlock()
	m.guardar()
}

func (m *RutinaManager) Borrar(nombre string) bool {
	m.mu.Lock()
	_, existe := m.rutinas[nombre]
	if existe {
		delete(m.rutinas, nombre)
	}
	m.mu.Unlock()
	if existe {
		m.guardar()
	}
	return existe
}

func (m *RutinaManager) Obtener(nombre string) ([]string, bool) {
	m.mu.Lock()
	pasos, ok := m.rutinas[nombre]
	m.mu.Unlock()
	return pasos, ok
}

func (m *RutinaManager) Listar() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	nombres := make([]string, 0, len(m.rutinas))
	for n := range m.rutinas {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	return nombres
}

func (h *Hands) manejarRutina(cmd string) string {
	entrada := strings.ToLower(strings.TrimSpace(cmd))

	switch {
	case strings.Contains(entrada, "crear rutina"), strings.Contains(entrada, "crea rutina"),
		strings.Contains(entrada, "creá rutina"), strings.Contains(entrada, "creá la rutina"),
		strings.Contains(entrada, "guarda rutina"), strings.Contains(entrada, "guardá rutina"),
		strings.Contains(entrada, "guardar rutina"):
		return h.crearRutina(entrada)

	case strings.Contains(entrada, "borrar rutina"), strings.Contains(entrada, "borra rutina"),
		strings.Contains(entrada, "borrá rutina"), strings.Contains(entrada, "eliminar rutina"),
		strings.Contains(entrada, "eliminá rutina"):
		nombre := strings.TrimSpace(strings.TrimPrefix(entrada, "borrar rutina"))
		nombre = strings.TrimSpace(strings.TrimPrefix(nombre, "borra rutina"))
		nombre = strings.TrimSpace(strings.TrimPrefix(nombre, "borrá rutina"))
		nombre = strings.TrimSpace(strings.TrimPrefix(nombre, "eliminar rutina"))
		nombre = strings.TrimSpace(strings.TrimPrefix(nombre, "eliminá rutina"))
		nombre = strings.ReplaceAll(nombre, "la rutina ", "")
		nombre = strings.TrimSpace(nombre)
		if nombre == "" {
			return "¿Qué rutina desea borrar, señor?"
		}
		if !h.rutinas.Borrar(nombre) {
			return fmt.Sprintf("No existe la rutina '%s', señor.", nombre)
		}
		return fmt.Sprintf("Rutina '%s' eliminada, señor.", nombre)

	case strings.Contains(entrada, "listar rutinas"), strings.Contains(entrada, "qué rutinas"),
		strings.Contains(entrada, "cuales rutinas"), strings.Contains(entrada, "mis rutinas"),
		strings.Contains(entrada, "rutinas tengo"):
		nombres := h.rutinas.Listar()
		if len(nombres) == 0 {
			return "No tiene rutinas guardadas, señor. Diga 'crear rutina trabajo que abrir chrome y abrir vs code' para crear una."
		}
		return fmt.Sprintf("Sus rutinas: %s, señor.", strings.Join(nombres, ", "))

	default:
		nombre := extraerNombreRutina(entrada)
		if nombre == "" {
			nombres := h.rutinas.Listar()
			if len(nombres) == 0 {
				return "No tiene rutinas guardadas, señor. Diga 'crear rutina trabajo que abrir chrome y abrir vs code'."
			}
			return fmt.Sprintf("Sus rutinas: %s, señor. Diga 'ejecutar rutina' seguido del nombre.", strings.Join(nombres, ", "))
		}
		pasos, ok := h.rutinas.Obtener(nombre)
		if !ok {
			return fmt.Sprintf("No tengo una rutina llamada '%s', señor. Sus rutinas: %s.", nombre, strings.Join(h.rutinas.Listar(), ", "))
		}
		resultados := make([]string, 0, len(pasos))
		for _, paso := range pasos {
			resultados = append(resultados, h.RunCommand(paso))
		}
		return fmt.Sprintf("Rutina '%s' ejecutada con %d pasos, señor.", nombre, len(resultados))
	}
}

func (h *Hands) crearRutina(entrada string) string {
	idx := strings.Index(entrada, "rutina")
	if idx < 0 {
		return "No entendí, señor. Diga 'crear rutina nombre que abrir chrome y abrir spotify'."
	}
	resto := strings.TrimSpace(entrada[idx+len("rutina"):])
	resto = strings.TrimLeft(resto, " ")
	resto = strings.Replace(resto, "que ", "", 1)
	partes := strings.SplitN(resto, " ", 2)
	if len(partes) == 0 || strings.TrimSpace(partes[0]) == "" {
		return "Dígame un nombre para la rutina, señor. Ejemplo: 'crear rutina trabajo que abrir chrome y abrir spotify'."
	}
	nombre := strings.TrimSpace(partes[0])
	acciones := ""
	if len(partes) > 1 {
		acciones = strings.TrimSpace(partes[1])
	}
	if acciones == "" {
		return "Dígame qué debe hacer la rutina, señor. Ejemplo: 'crear rutina trabajo que abrir chrome y abrir spotify'."
	}
	pasos := dividirPasos(acciones)
	if len(pasos) == 0 {
		return "No entendí los pasos de la rutina, señor."
	}
	if h.rutinas != nil {
		h.rutinas.Crear(nombre, pasos)
	}
	return fmt.Sprintf("Rutina '%s' creada con %d pasos, señor.", nombre, len(pasos))
}

func extraerNombreRutina(entrada string) string {
	idx := strings.Index(entrada, "rutina")
	if idx < 0 {
		return ""
	}
	resto := strings.TrimSpace(entrada[idx+len("rutina"):])
	if resto == "" {
		return ""
	}
	primer := strings.Fields(resto)
	if len(primer) == 0 {
		return ""
	}
	nombre := primer[0]
	if nombre == "que" || nombre == "tengo" || nombre == "con" {
		return ""
	}
	return nombre
}

func dividirPasos(acciones string) []string {
	acciones = strings.ReplaceAll(acciones, ",", " y ")
	partes := strings.Split(acciones, " y ")
	pasos := make([]string, 0, len(partes))
	for _, p := range partes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = normalizarPaso(p)
		pasos = append(pasos, p)
	}
	return pasos
}

func normalizarPaso(paso string) string {
	paso = strings.ReplaceAll(paso, "abrí ", "abrir ")
	paso = strings.ReplaceAll(paso, "abri ", "abrir ")
	paso = strings.ReplaceAll(paso, "abrís ", "abrir ")
	paso = strings.ReplaceAll(paso, "abra ", "abrir ")
	paso = strings.ReplaceAll(paso, "abre ", "abrir ")
	return paso
}
