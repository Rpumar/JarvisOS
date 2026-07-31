package core

import "testing"

func TestAnalizarSalud(t *testing.T) {
	casos := []struct {
		nombre     string
		diag       *Diagnostico
		esperadoMin int
		esperadoMax int
	}{
		{
			nombre: "sistema sano",
			diag: &Diagnostico{
				RAMPorcentaje: 40,
				Discos: []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 60, Porcentaje: 40}},
				DiscosFisicos: []DiscoFisico{{Nombre: "SSD", Salud: "OK"}},
				TempGB: 0.1,
			},
			esperadoMin: 90,
			esperadoMax: 100,
		},
		{
			nombre: "memoria saturada",
			diag: &Diagnostico{
				RAMPorcentaje: 95,
				Discos: []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 60, Porcentaje: 40}},
			},
			esperadoMin: 80,
			esperadoMax: 90,
		},
		{
			nombre: "disco casi lleno",
			diag: &Diagnostico{
				RAMPorcentaje: 40,
				Discos: []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 5, Porcentaje: 95}},
			},
			esperadoMin: 80,
			esperadoMax: 90,
		},
		{
			nombre: "disco fisico enfermo",
			diag: &Diagnostico{
				RAMPorcentaje: 40,
				Discos: []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 60, Porcentaje: 40}},
				DiscosFisicos: []DiscoFisico{{Nombre: "HDD", Salud: "Unhealthy"}},
			},
			esperadoMin: 75,
			esperadoMax: 85,
		},
		{
			nombre: "muchos problemas acumulados",
			diag: &Diagnostico{
				RAMPorcentaje: 95,
				CPUPorcentaje: 96,
				Discos: []DiscoLogico{{Letra: "C:", TotalGB: 100, LibreGB: 3, Porcentaje: 97}},
				DiscosFisicos: []DiscoFisico{{Nombre: "HDD", Salud: "Unhealthy"}},
				ServiciosCaidos: []ServicioInfo{{Nombre: "X", Estado: "Stopped"}},
				EventosError: []EventoInfo{{Hora: "a", Fuente: "b", Mensaje: "c"}},
				TempGB: 5,
				ReinicioPendiente: true,
			},
			esperadoMin: 0,
			esperadoMax: 60,
		},
	}

	for _, cso := range casos {
		puntaje, _ := analizarSalud(cso.diag)
		if puntaje < cso.esperadoMin || puntaje > cso.esperadoMax {
			t.Errorf("%s: puntaje=%d, esperado entre %d y %d", cso.nombre, puntaje, cso.esperadoMin, cso.esperadoMax)
		}
	}
}
