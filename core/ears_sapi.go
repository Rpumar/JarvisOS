//go:build !cgo && windows

package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Ears struct{}

func NewEars(rutaModelo string) (*Ears, error) {
	fmt.Println("[SAPI] Usando reconocimiento de voz de Windows (System.Speech).")
	fmt.Println("[SAPI] Para activarme, diga 'jarvis' por el micrófono.")
	return &Ears{}, nil
}

func (e *Ears) Escuchar() (string, error) {
	ps := `Add-Type -AssemblyName System.Speech; ` +
		`$ci=[System.Globalization.CultureInfo]::GetCultureInfo('es-ES'); ` +
		`$r=New-Object System.Speech.Recognition.SpeechRecognitionEngine($ci); ` +
		`$g=New-Object System.Speech.Recognition.DictationGrammar; ` +
		`$r.LoadGrammar($g); ` +
		`$r.SetInputToDefaultAudioDevice(); ` +
		`try { $res=$r.Recognize([System.TimeSpan]::FromSeconds(3)); if($res -and $res.Text -ne ''){ $res.Text }else{ '' } } catch { '' }; ` +
		`$r.UnloadAllGrammars(); $r.Dispose()`

	texto := ejecutarPowerShellYLeer(ps)
	if texto != "" {
		return texto, nil
	}

	fmt.Fprintf(os.Stderr, "[SAPI] No se capturó voz. ¿Micrófono conectado y encendido?\n")
	fmt.Fprintf(os.Stderr, "[SAPI] Escriba el comando manualmente (> ) o espere el próximo ciclo...\n")

	fmt.Print("> ")
	var entrada string
	fmt.Scanln(&entrada)
	entrada = strings.TrimSpace(entrada)
	if entrada != "" {
		return entrada, nil
	}

	time.Sleep(500 * time.Millisecond)
	return "", fmt.Errorf("no se detectó voz")
}

func (e *Ears) Cerrar() {}

func ejecutarPowerShellYLeer(comando string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", comando)
	salida, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(salida))
}
