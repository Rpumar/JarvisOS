//go:build cgo

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	vosk "github.com/alphacep/vosk-api/go"
	"github.com/gordonklaus/portaudio"
)

const (
	tasaMuestreo      = 16000
	muestrasPorBloque = 4096
	pollStdinInterval = 200 * time.Millisecond
)

type Ears struct {
	mu          sync.Mutex
	inicializado bool

	stream      *portaudio.Stream
	modelo      *vosk.VoskModel
	reconocedor *vosk.VoskRecognizer
	buffer      []int16

	lineaStd chan string
	rutaModelo string
}

type resultadoVosk struct {
	Text string `json:"text"`
}

func NewEars(rutaModelo string) (*Ears, error) {
	if rutaModelo == "" {
		rutaModelo = "./modelo-voz-es"
	}
	e := &Ears{
		lineaStd:   make(chan string, 1),
		rutaModelo: rutaModelo,
	}
	go e.leerStdin()
	return e, nil
}

func (e *Ears) inicializarVoz() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inicializado {
		return nil
	}

	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("error al inicializar PortAudio: %w", err)
	}

	modelo, err := vosk.NewModel(e.rutaModelo)
	if err != nil {
		portaudio.Terminate()
		return fmt.Errorf("no se pudo cargar el modelo de voz en '%s': %w", e.rutaModelo, err)
	}

	reconocedor, err := vosk.NewRecognizer(modelo, float64(tasaMuestreo))
	if err != nil {
		modelo.Free()
		portaudio.Terminate()
		return fmt.Errorf("no se pudo crear el reconocedor de voz: %w", err)
	}

	buffer := make([]int16, muestrasPorBloque)

	stream, err := portaudio.OpenDefaultStream(1, 0, float64(tasaMuestreo), len(buffer), buffer)
	if err != nil {
		reconocedor.Free()
		modelo.Free()
		portaudio.Terminate()
		return fmt.Errorf("no se pudo abrir el micrófono predeterminado: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		reconocedor.Free()
		modelo.Free()
		portaudio.Terminate()
		return fmt.Errorf("no se pudo iniciar la captura de audio: %w", err)
	}

	e.stream = stream
	e.modelo = modelo
	e.reconocedor = reconocedor
	e.buffer = buffer
	e.inicializado = true
	return nil
}

func (e *Ears) detenerVoz() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stream != nil {
		e.stream.Stop()
	}
}

func (e *Ears) reanudarVoz() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stream != nil {
		return e.stream.Start()
	}
	return nil
}

func (e *Ears) leerStdin() {
	lector := bufio.NewReader(os.Stdin)
	for {
		texto, err := lector.ReadString('\n')
		if err != nil {
			close(e.lineaStd)
			return
		}
		texto = strings.TrimSpace(texto)
		if texto != "" {
			e.lineaStd <- texto
		}
	}
}

func (e *Ears) EscucharTexto() (string, error) {
	texto, ok := <-e.lineaStd
	if !ok {
		return "", fmt.Errorf("stdin cerrado")
	}
	return texto, nil
}

func (e *Ears) Escuchar() (string, error) {
	if err := e.inicializarVoz(); err != nil {
		return "", err
	}

	e.reanudarVoz()
	defer e.detenerVoz()

	ultimoPoll := time.Now()
	for {
		if err := e.stream.Read(); err != nil {
			return "", fmt.Errorf("error al leer audio del micrófono: %w", err)
		}

		datos := int16ABytes(e.buffer)

		if e.reconocedor.AcceptWaveform(datos) != 0 {
			var resultado resultadoVosk
			if err := json.Unmarshal([]byte(e.reconocedor.Result()), &resultado); err != nil {
				continue
			}
			if resultado.Text != "" {
				fmt.Println()
				return resultado.Text, nil
			}
		}

		if time.Since(ultimoPoll) >= pollStdinInterval {
			ultimoPoll = time.Now()
			select {
			case texto, ok := <-e.lineaStd:
				if ok {
					return texto, nil
				}
			default:
			}
		}
	}
}

func (e *Ears) Cerrar() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.inicializado {
		return
	}
	if e.stream != nil {
		e.stream.Stop()
		e.stream.Close()
	}
	if e.reconocedor != nil {
		e.reconocedor.Free()
	}
	if e.modelo != nil {
		e.modelo.Free()
	}
	portaudio.Terminate()
}

func int16ABytes(muestras []int16) []byte {
	datos := make([]byte, len(muestras)*2)
	for i, m := range muestras {
		datos[i*2] = byte(m)
		datos[i*2+1] = byte(m >> 8)
	}
	return datos
}
