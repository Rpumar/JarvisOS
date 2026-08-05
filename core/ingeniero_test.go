package core

import (
	"strings"
	"testing"
)

func TestAnalizarSaludPerfecto(t *testing.T) {
	d := &Diagnostico{
		OS:             "Windows 11",
		RAMPorcentaje:  50,
		CPUPorcentaje:  30,
		RAMTotalGB:     16,
		RAMUsadaGB:     8,
		UptimeDias:     2.5,
		Discos:         []DiscoLogico{{Letra: "C:", TotalGB: 500, LibreGB: 300, Porcentaje: 40}},
		DiscosFisicos:  []DiscoFisico{{Nombre: "SSD", Salud: "OK"}},
		ServiciosCaidos: []ServicioInfo{},
		EventosError:   []EventoInfo{},
		TempGB:         0.5,
		Bateria:        &BateriaInfo{Porcentaje: 80, Conectada: true},
	}
	puntaje, problemas := analizarSalud(d)
	if puntaje != 100 {
		t.Errorf("esperaba 100, obtuve %d (problemas=%v)", puntaje, problemas)
	}
	if len(problemas) != 0 {
		t.Errorf("esperaba 0 problemas, obtuve %d", len(problemas))
	}
}

func TestAnalizarSaludPenalizaciones(t *testing.T) {
	d := &Diagnostico{
		RAMPorcentaje:   95,
		CPUPorcentaje:   95,
		Discos:          []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 5, Porcentaje: 95}},
		DiscosFisicos:   []DiscoFisico{{Nombre: "HDD", Salud: "Pred Fail"}},
		ServiciosCaidos: []ServicioInfo{{Nombre: "Spooler", Estado: "Stopped"}, {Nombre: "Wuauserv", Estado: "Stopped"}},
		EventosError:    []EventoInfo{{Hora: "01/01", Fuente: "Kernel", Mensaje: "x"}, {Hora: "02/01", Fuente: "Kernel", Mensaje: "y"}},
		TempGB:          3.5,
		ReinicioPendiente: true,
		Bateria:         &BateriaInfo{Porcentaje: 10, Conectada: false},
	}
	puntaje, problemas := analizarSalud(d)
	if puntaje >= 100 {
		t.Errorf("esperaba penalización, obtuve %d", puntaje)
	}
	if len(problemas) < 6 {
		t.Errorf("esperaba varios problemas, obtuve %d", len(problemas))
	}
	if puntaje < 0 {
		t.Errorf("el puntaje no debería ser negativo, obtuve %d", puntaje)
	}
}

func TestAnalizarSaludClampNegativo(t *testing.T) {
	d := &Diagnostico{
		RAMPorcentaje:   99,
		CPUPorcentaje:   99,
		Discos:          []DiscoLogico{{Letra: "C:", TotalGB: 10, LibreGB: 0, Porcentaje: 100}},
		DiscosFisicos:   []DiscoFisico{{Nombre: "x", Salud: "bad"}},
		ServiciosCaidos: []ServicioInfo{{Nombre: "a", Estado: "Stopped"}},
		EventosError:    []EventoInfo{{}, {}, {}, {}, {}},
		TempGB:          99,
		ReinicioPendiente: true,
	}
	puntaje, _ := analizarSalud(d)
	if puntaje < 0 {
		t.Errorf("puntaje negativo sin clampear: %d", puntaje)
	}
}

func TestAnalizarSaludDiscoSinTamañoNoPenaliza(t *testing.T) {
	d := &Diagnostico{
		Discos: []DiscoLogico{{Letra: "X:", TotalGB: 0, LibreGB: 0}},
	}
	puntaje, problemas := analizarSalud(d)
	if puntaje != 100 {
		t.Errorf("disco sin tamaño no debería penalizar: %d", puntaje)
	}
	if len(problemas) != 0 {
		t.Errorf("esperaba 0 problemas, obtuve %d", len(problemas))
	}
}

func TestAnalizarSaludEventosCap(t *testing.T) {
	d := &Diagnostico{
		EventosError: []EventoInfo{
			{}, {}, {}, {}, {}, {},
		},
	}
	puntaje, problemas := analizarSalud(d)
	if puntaje != 85 {
		t.Errorf("cap de eventos: esperaba 85, obtuve %d", puntaje)
	}
	if len(problemas) != 1 {
		t.Errorf("esperaba 1 problema, obtuve %d", len(problemas))
	}
}

func TestDiagnosticoReporte(t *testing.T) {
	d := &Diagnostico{
		OS:             "Windows 11",
		Version:        "10.0.22631",
		CPUNombre:      "Intel i7",
		CPUNucleos:     8,
		CPUPorcentaje:  25,
		RAMPorcentaje:  60,
		RAMUsadaGB:     9.6,
		RAMTotalGB:     16,
		UptimeDias:     3.2,
		Discos:         []DiscoLogico{{Letra: "C:", TotalGB: 500, LibreGB: 250, Porcentaje: 50}},
		DiscosFisicos:  []DiscoFisico{{Nombre: "NVMe", Salud: "Healthy"}},
		TempGB:         1.2,
		IPs:            []string{"192.168.1.10"},
		TopCPU:         []ProcInfo{{Nombre: "chrome.exe", CPUSeg: 1234, PID: 100}},
		TopRAM:         []ProcInfo{{Nombre: "powershell", MB: 512, PID: 200}},
		ServiciosCaidos: []ServicioInfo{{Nombre: "BITS", Estado: "Stopped"}},
		EventosError:   []EventoInfo{{Hora: "01/08 10:00", Fuente: "Kernel-Power", Mensaje: "fallo"}},
		Inicio:         []string{"OneDrive"},
	}
	rep := d.reporte()
	for _, esperado := range []string{"Windows 11", "Intel i7", "chrome.exe", "BITS", "Kernel-Power", "OneDrive", "192.168.1.10"} {
		if !strings.Contains(rep, esperado) {
			t.Errorf("reporte() no contiene %q\n%s", esperado, rep)
		}
	}
}

func TestDiagnosticoReporteVacio(t *testing.T) {
	d := &Diagnostico{}
	rep := d.reporte()
	if !strings.Contains(rep, "INFORME DE INGENIERÍA") {
		t.Errorf("reporte vacío debería tener encabezado: %q", rep)
	}
	if strings.Contains(rep, "BITS") {
		t.Errorf("reporte vacío no debería listar servicios")
	}
}
