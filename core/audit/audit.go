// Package audit mantiene el registro inmutable (append-only) de todo lo
// que el empleado ejecuta: quién lo pidió, con qué rol, qué comando y el
// resultado exacto. Es la base del cumplimiento corporativo de F2; más
// adelante alimentará SQLite y el panel del dueño.
package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entrada es un evento de auditoría único.
type Entrada struct {
	Momento   string `json:"momento"`
	Usuario   string `json:"usuario"`
	Rol       string `json:"rol"`
	Orden     int    `json:"orden,omitempty"`
	Comando   string `json:"comando"`
	Resultado string `json:"resultado"`
}

// MaxAuditoriaBytes es el tamaño máximo del archivo de auditoría antes de
// rotarlo automáticamente (10 MB).
const MaxAuditoriaBytes = 10 * 1024 * 1024

// Registro escribe cada entrada como una línea JSON (JSONL) en un archivo,
// de modo que el historial previo nunca se reescribe. Cuando el archivo
// supera MaxAuditoriaBytes se rota: se renombra con sufijo de timestamp y
// se sigue escribiendo en un archivo nuevo limpio.
type Registro struct {
	mu       sync.Mutex
	ruta     string
	entradas []Entrada
	maxBytes int64
}

// NuevoRegistro carga las líneas existentes del archivo (si hay) y deja el
// registro listo para seguir agregando.
func NuevoRegistro(ruta string) *Registro {
	r := &Registro{ruta: ruta, entradas: make([]Entrada, 0), maxBytes: MaxAuditoriaBytes}
	if ruta == "" {
		return r
	}
	f, err := os.Open(ruta)
	if err != nil {
		return r
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Entrada
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			r.entradas = append(r.entradas, e)
		}
	}
	return r
}

// Registrar agrega una entrada en memoria y la persiste como una nueva
// línea al final del archivo (nunca se modifica lo anterior). Si el archivo
// supera el tamaño máximo, primero se rota. Thread-safe: toda la operación
// de disco corre bajo el mismo mutex para evitar reescrituras concurrentes.
func (r *Registro) Registrar(e Entrada) {
	if e.Momento == "" {
		e.Momento = time.Now().Format("2006-01-02 15:04:05")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entradas = append(r.entradas, e)

	if r.ruta == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o700); err != nil {
		return
	}

	if r.superaMaximo() {
		r.rotar()
	}

	f, err := os.OpenFile(r.ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	linea, _ := json.Marshal(e)
	if _, err := f.Write(append(linea, '\n')); err != nil {
		return
	}
}

// superaMaximo indica si el archivo activo ya alcanzó el límite de bytes.
func (r *Registro) superaMaximo() bool {
	info, err := os.Stat(r.ruta)
	return err == nil && info.Size() >= r.maxBytes
}

// rotar renombra el archivo activo con sufijo de timestamp y deja que el
// siguiente registro cree un archivo nuevo limpio. Es inmutable: lo escrito
// jamás se reescribe, solo se archiva.
func (r *Registro) rotar() {
	archivado := r.ruta + "." + time.Now().Format("20060102_150405")
	_ = os.Rename(r.ruta, archivado)
}

// Listar devuelve todas las entradas, de la más antigua a la más nueva.
func (r *Registro) Listar() []Entrada {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]Entrada, len(r.entradas))
	copy(res, r.entradas)
	return res
}

// Recientes devuelve las n entradas más nuevas (o todas si hay menos).
func (r *Registro) Recientes(n int) []Entrada {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > len(r.entradas) {
		n = len(r.entradas)
	}
	res := make([]Entrada, n)
	copy(res, r.entradas[len(r.entradas)-n:])
	return res
}
