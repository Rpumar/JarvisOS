//go:build cgo

package core

import (
	"encoding/json"
	"fmt"

	vosk "github.com/alphacep/vosk-api/go"
	"github.com/gordonklaus/portaudio"
)

const (
	tasaMuestreo      = 16000 // Hz — la tasa que espera el modelo de Vosk
	muestrasPorBloque = 4096  // tamaño del bloque de audio leído en cada iteración
)

// Ears encapsula la captura de audio real del micrófono (PortAudio) y el
// reconocimiento de voz offline (Vosk).
//
// NOTA DE ARQUITECTURA (reemplaza la versión anterior basada en PowerShell):
// System.Speech.Recognition.SpeechRecognitionEngine de Windows, sin una
// gramática (Grammar/DictationGrammar) cargada, no reconoce de forma fiable —
// por eso "faltaba micrófono real". PortAudio por sí solo tampoco resuelve esto:
// solo entrega audio en crudo, no hace STT. La combinación PortAudio (captura)
// + Vosk (reconocimiento offline real) sí resuelve el requisito por completo.
type Ears struct {
	stream      *portaudio.Stream
	modelo      *vosk.VoskModel
	reconocedor *vosk.VoskRecognizer
	buffer      []int16
}

type resultadoVosk struct {
	Text string `json:"text"`
}

// NewEars inicializa PortAudio, carga el modelo de Vosk desde rutaModelo
// (si viene vacío, usa "./modelo-voz-es") y abre el micrófono predeterminado
// del sistema. Devuelve error si algo falla, en vez de fallar en silencio.
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

	return &Ears{stream: stream, modelo: modelo, reconocedor: reconocedor, buffer: buffer}, nil
}

// Escuchar bloquea hasta que Vosk detecta el final de una frase (silencio tras
// voz) y devuelve el texto reconocido. Los bloques sin voz se ignoran solos.
func (e *Ears) Escuchar() (string, error) {
	for {
		if err := e.stream.Read(); err != nil {
			return "", fmt.Errorf("error al leer audio del micrófono: %w", err)
		}

		datos := int16ABytes(e.buffer)

		if e.reconocedor.AcceptWaveform(datos) != 0 {
			var resultado resultadoVosk
			if err := json.Unmarshal([]byte(e.reconocedor.Result()), &resultado); err != nil {
				continue // resultado no parseable, se descarta y se sigue escuchando
			}
			if resultado.Text != "" {
				return resultado.Text, nil
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
