//go:build cgo

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	stream      *portaudio.Stream
	modelo      *vosk.VoskModel
	reconocedor *vosk.VoskRecognizer
	buffer      []int16
	lineaStd    chan string
}

type resultadoVosk struct {
	Text string `json:"text"`
}

func NewEars(rutaModelo string) (*Ears, error) {
	if rutaModelo == "" {
		rutaModelo = "./modelo-voz-es"
	}

	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("error al inicializar PortAudio: %w", err)
	}

	modelo, err := vosk.NewModel(rutaModelo)
	if err != nil {
		portaudio.Terminate()
		return nil, fmt.Errorf("no se pudo cargar el modelo de voz en '%s' (descárgalo de https://alphacephei.com/vosk/models): %w", rutaModelo, err)
	}

	reconocedor, err := vosk.NewRecognizer(modelo, float64(tasaMuestreo))
	if err != nil {
		modelo.Free()
		portaudio.Terminate()
		return nil, fmt.Errorf("no se pudo crear el reconocedor de voz: %w", err)
	}

	buffer := make([]int16, muestrasPorBloque)

	stream, err := portaudio.OpenDefaultStream(1, 0, float64(tasaMuestreo), len(buffer), buffer)
	if err != nil {
		reconocedor.Free()
		modelo.Free()
		portaudio.Terminate()
		return nil, fmt.Errorf("no se pudo abrir el micrófono predeterminado: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		reconocedor.Free()
		modelo.Free()
		portaudio.Terminate()
		return nil, fmt.Errorf("no se pudo iniciar la captura de audio: %w", err)
	}

	e := &Ears{
		stream:      stream,
		modelo:      modelo,
		reconocedor: reconocedor,
		buffer:      buffer,
		lineaStd:    make(chan string, 1),
	}
	go e.leerStdin()
	return e, nil
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

// Cerrar detiene el stream y libera PortAudio y Vosk. Llamar una sola vez
// antes de salir del programa (idealmente con defer).
func (e *Ears) Cerrar() {
	_ = e.stream.Stop()
	_ = e.stream.Close()
	e.reconocedor.Free()
	e.modelo.Free()
	portaudio.Terminate()
}

// int16ABytes convierte las muestras PCM de PortAudio (int16) a []byte en
// little-endian, el formato que espera AcceptWaveform de Vosk.
func int16ABytes(muestras []int16) []byte {
	datos := make([]byte, len(muestras)*2)
	for i, m := range muestras {
		datos[i*2] = byte(m)
		datos[i*2+1] = byte(m >> 8)
	}
	return datos
}
