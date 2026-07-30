package core

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// --- Mocks (implementan las interfaces de las que depende Brain) ---

type manosFalsas struct {
	respuesta       string
	comandoRecibido string
}

func (m *manosFalsas) RunCommand(cmd string) string {
	m.comandoRecibido = cmd
	return m.respuesta
}

func (m *manosFalsas) Hablar(texto string) string { return "Listo, señor." }

type iaFalsa struct {
	disponible        bool
	respuesta         string
	err               error
	consultaRecibida  string
	historialRecibido []TurnoConversacion
}

func (i *iaFalsa) Disponible() bool { return i.disponible }

func (i *iaFalsa) Consultar(prompt string, historial []TurnoConversacion) (string, error) {
	i.consultaRecibida = prompt
	i.historialRecibido = historial
	return i.respuesta, i.err
}

type coderFalso struct {
	pendiente          bool
	respuestaProponer  string
	errProponer        error
	respuestaConfirmar string
	respuestaCancelar  string
	peticionRecibida   string
}

func (c *coderFalso) Proponer(peticion string) (string, error) {
	c.peticionRecibida = peticion
	return c.respuestaProponer, c.errProponer
}
func (c *coderFalso) Confirmar() string             { return c.respuestaConfirmar }
func (c *coderFalso) Cancelar() string              { return c.respuestaCancelar }
func (c *coderFalso) TienePropuestaPendiente() bool { return c.pendiente }

type memoriaFalsa struct {
	hechos          map[string]string
	notas           []string
	recordatorios   []recordatorioFalso
	errGuardarHecho error
	errAgregarNota  error
	claveRecibida   string
	valorRecibido   string
	notaRecibida    string
}

type recordatorioFalso struct {
	texto   string
	momento time.Time
}

func nuevaMemoriaFalsa() *memoriaFalsa {
	return &memoriaFalsa{hechos: map[string]string{}}
}

func (m *memoriaFalsa) GuardarHecho(clave, valor string) error {
	m.claveRecibida, m.valorRecibido = clave, valor
	if m.errGuardarHecho != nil {
		return m.errGuardarHecho
	}
	m.hechos[clave] = valor
	return nil
}

func (m *memoriaFalsa) ObtenerHecho(clave string) (string, bool) {
	v, ok := m.hechos[clave]
	return v, ok
}

func (m *memoriaFalsa) AgregarNota(texto string) error {
	m.notaRecibida = texto
	if m.errAgregarNota != nil {
		return m.errAgregarNota
	}
	m.notas = append(m.notas, texto)
	return nil
}

func (m *memoriaFalsa) ObtenerNotas() []string { return m.notas }

func (m *memoriaFalsa) AgregarRecordatorio(texto string, momento time.Time) error {
	m.recordatorios = append(m.recordatorios, recordatorioFalso{texto: texto, momento: momento})
	return nil
}

func (m *memoriaFalsa) ObtenerRecordatoriosPendientesTexto() []string {
	resultado := make([]string, len(m.recordatorios))
	for i, r := range m.recordatorios {
		resultado[i] = r.texto
	}
	return resultado
}

func (m *memoriaFalsa) CancelarRecordatorios(textoBusqueda string) (int, error) {
	restantes := m.recordatorios[:0]
	cancelados := 0
	for _, r := range m.recordatorios {
		if textoBusqueda == "" || strings.Contains(r.texto, textoBusqueda) {
			cancelados++
			continue
		}
		restantes = append(restantes, r)
	}
	m.recordatorios = restantes
	return cancelados, nil
}

func (m *memoriaFalsa) BuscarNotas(texto string) []string { return nil }
func (m *memoriaFalsa) AgregarRecordatorioConPeriodo(texto string, momento time.Time, periodo string) error { return nil }
func (m *memoriaFalsa) CrearLista(nombre string) error { return nil }
func (m *memoriaFalsa) AgregarItemLista(nombreLista, item string) error { return nil }
func (m *memoriaFalsa) MarcarItemLista(nombreLista, item string) (string, error) { return "", nil }
func (m *memoriaFalsa) ObtenerListas() []string { return nil }
func (m *memoriaFalsa) ObtenerLista(nombre string) (string, bool) { return "", false }
func (m *memoriaFalsa) EliminarLista(nombre string) error { return nil }

// --- Tests: comportamiento general (ya existían, adaptados a BrainOpciones) ---

func TestProcess_EntradaVacia(t *testing.T) {
	b := NewBrain(&manosFalsas{}, BrainOpciones{})
	if got := b.Process("   "); got != "" {
		t.Errorf("esperaba respuesta vacía para entrada vacía, obtuve %q", got)
	}
}

func TestProcess_ComandoLocalReconocido(t *testing.T) {
	manos := &manosFalsas{respuesta: "Abriendo, señor."}
	b := NewBrain(manos, BrainOpciones{})

	got := b.Process("abrir chrome")

	if got != "Abriendo, señor." {
		t.Errorf("respuesta = %q, esperaba %q", got, "Abriendo, señor.")
	}
	if manos.comandoRecibido != "abrir chrome" {
		t.Errorf("Hands.RunCommand recibió %q, esperaba %q", manos.comandoRecibido, "abrir chrome")
	}
}

func TestProcess_RespaldoIA_CuandoComandoNoReconocido(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, respuesta: "La capital de Francia es París."}
	b := NewBrain(manos, BrainOpciones{IA: ia})

	got := b.Process("cuál es la capital de Francia")

	if got != "La capital de Francia es París." {
		t.Errorf("respuesta = %q, esperaba el respaldo de IA", got)
	}
	if ia.consultaRecibida == "" {
		t.Error("se esperaba que se consultara la IA, pero no se llamó")
	}
}

func TestProcess_SinConectorIA_MensajeGenerico(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	b := NewBrain(manos, BrainOpciones{})

	got := b.Process("algo que no reconozco")

	if got == "" || got == ComandoNoReconocido {
		t.Errorf("respuesta inesperada sin conector de IA: %q", got)
	}
}

func TestProcess_IANoDisponible_NoSeConsulta(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: false, respuesta: "esto no debería devolverse"}
	b := NewBrain(manos, BrainOpciones{IA: ia})

	got := b.Process("algo raro")

	if got == "esto no debería devolverse" {
		t.Error("no debería usar la IA si Disponible() devuelve false")
	}
}

func TestProcess_IAConError_CaeAlMensajeGenerico(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, err: errors.New("timeout")}
	b := NewBrain(manos, BrainOpciones{IA: ia})

	got := b.Process("algo raro")

	if got == "" {
		t.Error("esperaba un mensaje de respaldo, no una respuesta vacía")
	}
}

func TestProcess_IA_AcumulaHistorialEntreTurnos(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, respuesta: "Fue un físico muy conocido."}
	b := NewBrain(manos, BrainOpciones{IA: ia})

	b.Process("quién fue Einstein")
	if len(ia.historialRecibido) != 0 {
		t.Errorf("el primer turno no debería tener historial previo, tiene %d", len(ia.historialRecibido))
	}

	b.Process("qué más hizo")
	if len(ia.historialRecibido) != 1 {
		t.Fatalf("el segundo turno debería recibir 1 turno de historial, recibió %d", len(ia.historialRecibido))
	}
	if ia.historialRecibido[0].Usuario != "quién fue Einstein" {
		t.Errorf("historial[0].Usuario = %q, esperaba la primera pregunta", ia.historialRecibido[0].Usuario)
	}
}

func TestProcess_PeticionDeCodigo_SeRuteaACoderAgent(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	coder := &coderFalso{respuestaProponer: "¿Confirmo, señor?"}
	b := NewBrain(manos, BrainOpciones{Coder: coder})

	got := b.Process("escribime un script que organice mis descargas")

	if got != "¿Confirmo, señor?" {
		t.Errorf("respuesta = %q, esperaba la propuesta del CoderAgent", got)
	}
	if coder.peticionRecibida == "" {
		t.Error("se esperaba que se llamara a Proponer, pero no se llamó")
	}
	if manos.comandoRecibido != "" {
		t.Error("no debería haber llegado a Hands.RunCommand: la petición de código se intercepta antes")
	}
}

func TestProcess_BuscarScriptDePython_NoSeConfundeConPeticionDeCodigo(t *testing.T) {
	manos := &manosFalsas{respuesta: "Buscando script de python en Google, señor."}
	coder := &coderFalso{respuestaProponer: "no debería llegar acá"}
	b := NewBrain(manos, BrainOpciones{Coder: coder})

	got := b.Process("buscar script de python")

	if got != "Buscando script de python en Google, señor." {
		t.Errorf("respuesta = %q; se esperaba que fuera a Hands, no a CoderAgent", got)
	}
	if coder.peticionRecibida != "" {
		t.Error("no debería haberse llamado a CoderAgent.Proponer")
	}
}

func TestProcess_PropuestaPendiente_Confirmar(t *testing.T) {
	coder := &coderFalso{pendiente: true, respuestaConfirmar: "Listo, señor."}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Coder: coder})

	if got := b.Process("confirmar"); got != "Listo, señor." {
		t.Errorf("respuesta = %q, esperaba la confirmación", got)
	}
}

func TestProcess_PropuestaPendiente_Cancelar(t *testing.T) {
	coder := &coderFalso{pendiente: true, respuestaCancelar: "Descartado, señor."}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Coder: coder})

	if got := b.Process("cancelar"); got != "Descartado, señor." {
		t.Errorf("respuesta = %q, esperaba la cancelación", got)
	}
}

func TestProcess_PropuestaPendiente_BloqueaComandosNormales(t *testing.T) {
	coder := &coderFalso{pendiente: true}
	manos := &manosFalsas{respuesta: "no debería usarse"}
	b := NewBrain(manos, BrainOpciones{Coder: coder})

	got := b.Process("abrir chrome")

	if got == "no debería usarse" {
		t.Error("con una propuesta pendiente, no debería procesarse un comando normal")
	}
	if manos.comandoRecibido != "" {
		t.Error("no debería haber llegado a Hands.RunCommand mientras hay una propuesta pendiente")
	}
}

func TestEsPeticionDeCodigo(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado bool
	}{
		{"escribe un script que borre archivos temporales", true},
		{"escribime un script para organizar mis descargas", true},
		{"crea un script que cuente archivos", true},
		{"generame un código para esto", true},
		{"hazme un script rápido", true},
		{"buscar script de python", false},
		{"abrir chrome", false},
		{"cuéntame un chiste", false},
		{"", false},
	}

	for _, c := range casos {
		if got := esPeticionDeCodigo(c.entrada); got != c.esperado {
			t.Errorf("esPeticionDeCodigo(%q) = %v, esperaba %v", c.entrada, got, c.esperado)
		}
	}
}

// --- Tests: Fase 2, memoria de sesión (resolución de pronombres) ---

func TestProcess_Cerralo_UsaLaUltimaAppAbierta(t *testing.T) {
	manos := &manosFalsas{respuesta: "Abriendo, señor."}
	b := NewBrain(manos, BrainOpciones{})

	b.Process("abrir chrome")
	manos.respuesta = "chrome cerrada, señor."
	got := b.Process("cerralo")

	if manos.comandoRecibido != "cerrar chrome" {
		t.Errorf("Hands.RunCommand recibió %q, esperaba %q", manos.comandoRecibido, "cerrar chrome")
	}
	if got != "chrome cerrada, señor." {
		t.Errorf("respuesta = %q, esperaba la confirmación de cierre", got)
	}
}

func TestProcess_Cerralo_SinAppPrevia_NoRompeNada(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	b := NewBrain(manos, BrainOpciones{})

	// Sin haber abierto nada antes, "cerralo" no tiene nada que resolver:
	// debería pasar tal cual y terminar en la respuesta de confusión.
	got := b.Process("cerralo")

	if got == "" {
		t.Error("no debería devolver una respuesta vacía")
	}
}

func TestProcess_LoMismoEnYoutube_UsaLaUltimaBusqueda(t *testing.T) {
	manos := &manosFalsas{respuesta: "Buscando gatos en Google, señor."}
	b := NewBrain(manos, BrainOpciones{})

	b.Process("buscar gatos")
	manos.respuesta = "Buscando gatos en YouTube, señor."
	got := b.Process("lo mismo en youtube")

	if manos.comandoRecibido != "buscar en youtube gatos" {
		t.Errorf("Hands.RunCommand recibió %q, esperaba %q", manos.comandoRecibido, "buscar en youtube gatos")
	}
	if got != "Buscando gatos en YouTube, señor." {
		t.Errorf("respuesta = %q, esperaba la confirmación de búsqueda en YouTube", got)
	}
}

// --- Tests: Fase 2, memoria persistente ---

func TestProcess_RecordarNombre_SeGuardaComoHechoEstructurado(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordá que me llamo Juan")

	if mem.claveRecibida != "nombre" || mem.valorRecibido != "Juan" {
		t.Errorf("se guardó clave=%q valor=%q, esperaba clave=nombre valor=Juan", mem.claveRecibida, mem.valorRecibido)
	}
	if mem.notaRecibida != "" {
		t.Error("no debería haberse guardado como nota libre: hay un patrón de nombre reconocido")
	}
}

func TestProcess_RecordarCiudad_SeGuardaComoHechoEstructurado(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordá que vivo en Córdoba")

	if mem.claveRecibida != "ciudad" || mem.valorRecibido != "Córdoba" {
		t.Errorf("se guardó clave=%q valor=%q, esperaba clave=ciudad valor=Córdoba", mem.claveRecibida, mem.valorRecibido)
	}
}

func TestProcess_RecordarAlgoSinPatron_SeGuardaComoNotaLibre(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordá que tengo reunión el jueves")

	if mem.notaRecibida != "tengo reunión el jueves" {
		t.Errorf("nota guardada = %q, esperaba %q", mem.notaRecibida, "tengo reunión el jueves")
	}
	if mem.claveRecibida != "" {
		t.Error("no debería haberse guardado como hecho estructurado: no hay patrón de nombre/ciudad")
	}
}

func TestProcess_CualEsMiNombre_ConHechoGuardado(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["nombre"] = "Juan"
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("cuál es mi nombre")

	if !contains(got, "Juan") {
		t.Errorf("respuesta = %q, esperaba que mencionara 'Juan'", got)
	}
}

func TestProcess_CualEsMiNombre_SinHechoGuardado(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("cuál es mi nombre")

	if got == "" {
		t.Error("esperaba una respuesta explicando que no sabe el nombre, no una vacía")
	}
}

func TestProcess_QueRecordas_ListaLasNotas(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.notas = []string{"2026-07-20: nota uno", "2026-07-21: nota dos"}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("qué recordás")

	if !contains(got, "2") {
		t.Errorf("respuesta = %q, esperaba que mencionara la cantidad de notas", got)
	}
}

func TestProcess_ComandoDeMemoria_SinMemoriaConfigurada(t *testing.T) {
	b := NewBrain(&manosFalsas{}, BrainOpciones{})

	got := b.Process("recordá que me llamo Juan")

	if got == "" {
		t.Error("esperaba un mensaje explicando que no hay memoria configurada")
	}
}

func TestProcess_ComandoDeMemoria_NoLlegaAHandsNiACoder(t *testing.T) {
	manos := &manosFalsas{respuesta: "no debería usarse"}
	coder := &coderFalso{respuestaProponer: "no debería usarse"}
	mem := nuevaMemoriaFalsa()
	b := NewBrain(manos, BrainOpciones{Coder: coder, Memoria: mem})

	b.Process("recordá que me llamo Juan")

	if manos.comandoRecibido != "" {
		t.Error("un comando de memoria no debería llegar a Hands.RunCommand")
	}
	if coder.peticionRecibida != "" {
		t.Error("un comando de memoria no debería llegar a CoderAgent.Proponer")
	}
}

// --- Tests: Fase 3, trato personalizado ---

func TestProcess_UsaNombreEnVezDeSenor_CuandoLoConoce(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["nombre"] = "Juan"
	manos := &manosFalsas{respuesta: "Abriendo, señor."}
	b := NewBrain(manos, BrainOpciones{Memoria: mem})

	got := b.Process("abrir chrome")

	if got != "Abriendo, Juan." {
		t.Errorf("respuesta = %q, esperaba que reemplazara 'señor' por 'Juan'", got)
	}
}

func TestProcess_MantieneSenor_CuandoNoConoceElNombre(t *testing.T) {
	manos := &manosFalsas{respuesta: "Abriendo, señor."}
	b := NewBrain(manos, BrainOpciones{})

	got := b.Process("abrir chrome")

	if got != "Abriendo, señor." {
		t.Errorf("respuesta = %q, esperaba que mantuviera 'señor' sin memoria configurada", got)
	}
}

func TestSaludar_UsaNombreCuandoLoConoce(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["nombre"] = "Ana"
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Saludar()

	if contains(got, "señor") {
		t.Errorf("Saludar() = %q, no debería contener 'señor' si conoce el nombre", got)
	}
	if !contains(got, "Ana") {
		t.Errorf("Saludar() = %q, esperaba que mencionara 'Ana'", got)
	}
}

func TestDespedirse_UsaNombreCuandoLoConoce(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["nombre"] = "Ana"
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Despedirse()

	if !contains(got, "Ana") {
		t.Errorf("Despedirse() = %q, esperaba que mencionara 'Ana'", got)
	}
}

// --- Tests: Fase 3, hechos nuevos (cumpleaños, trabajo, llamame) ---

func TestProcess_Llamame_GuardaComoNombre(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("llamame Juancito")

	if mem.claveRecibida != "nombre" || mem.valorRecibido != "Juancito" {
		t.Errorf("se guardó clave=%q valor=%q, esperaba clave=nombre valor=Juancito", mem.claveRecibida, mem.valorRecibido)
	}
}

func TestProcess_RecordarCumpleanos(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordá que mi cumpleaños es 15 de marzo")

	if mem.claveRecibida != "cumpleaños" || mem.valorRecibido != "15 de marzo" {
		t.Errorf("se guardó clave=%q valor=%q, esperaba clave=cumpleaños valor='15 de marzo'", mem.claveRecibida, mem.valorRecibido)
	}
}

func TestProcess_RecordarTrabajo(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordá que trabajo en una fábrica")

	if mem.claveRecibida != "trabajo" || mem.valorRecibido != "una fábrica" {
		t.Errorf("se guardó clave=%q valor=%q, esperaba clave=trabajo valor='una fábrica'", mem.claveRecibida, mem.valorRecibido)
	}
}

func TestProcess_CuandoEsMiCumpleanos_ConHecho(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["cumpleaños"] = "15 de marzo"
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("cuándo es mi cumpleaños")

	if !contains(got, "15 de marzo") {
		t.Errorf("respuesta = %q, esperaba que mencionara '15 de marzo'", got)
	}
}

func TestProcess_DondeTrabajo_ConHecho(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.hechos["trabajo"] = "una fábrica"
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("dónde trabajo")

	if !contains(got, "una fábrica") {
		t.Errorf("respuesta = %q, esperaba que mencionara 'una fábrica'", got)
	}
}

// --- Tests: Fase 3, recordatorios con hora ---

func TestProcess_RecordameConHora_ProgramaRecordatorio(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("recordame llamar a mamá a las 17")

	if len(mem.recordatorios) != 1 {
		t.Fatalf("se esperaba 1 recordatorio programado, hay %d", len(mem.recordatorios))
	}
	if mem.recordatorios[0].texto != "llamar a mamá" {
		t.Errorf("texto = %q, esperaba %q", mem.recordatorios[0].texto, "llamar a mamá")
	}
	if mem.recordatorios[0].momento.Hour() != 17 {
		t.Errorf("hora = %d, esperaba 17", mem.recordatorios[0].momento.Hour())
	}
	if !contains(got, "17") {
		t.Errorf("respuesta = %q, esperaba que confirmara la hora programada", got)
	}
}

func TestProcess_RecordameConHora_TienePrioridadSobreNotaLibre(t *testing.T) {
	// "recordame X a las Y" no debería terminar guardado como nota libre:
	// el chequeo de recordatorio con hora tiene que ganar primero.
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordame comprar pan a las 20")

	if len(mem.notas) != 0 {
		t.Errorf("no debería haberse guardado como nota libre, se guardaron %d notas", len(mem.notas))
	}
	if len(mem.recordatorios) != 1 {
		t.Errorf("se esperaba 1 recordatorio, hay %d", len(mem.recordatorios))
	}
}

func TestProcess_RecordameSinHora_SeGuardaComoNotaLibre(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("recordame comprar pan")

	if len(mem.recordatorios) != 0 {
		t.Errorf("no debería haberse programado un recordatorio sin hora, hay %d", len(mem.recordatorios))
	}
	if mem.notaRecibida != "comprar pan" {
		t.Errorf("nota = %q, esperaba %q", mem.notaRecibida, "comprar pan")
	}
}

// --- Tests: Fase 5, timers, cancelar/listar recordatorios, qué dijiste ---

func TestProcess_Timer_ProgramaRecordatorio(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("poné un timer de 5 minutos")

	if len(mem.recordatorios) != 1 {
		t.Fatalf("esperaba 1 recordatorio programado, hay %d", len(mem.recordatorios))
	}
	if !contains(got, "5m") {
		t.Errorf("respuesta = %q, esperaba que mencionara la duración", got)
	}
}

func TestProcess_Timer_TienePrioridadSobreRecordatorioConHora(t *testing.T) {
	// "avisame en 10 minutos" no debería confundirse con "avisame a las X".
	mem := nuevaMemoriaFalsa()
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("avisame en 10 minutos")

	if len(mem.recordatorios) != 1 {
		t.Fatalf("esperaba 1 recordatorio (timer), hay %d", len(mem.recordatorios))
	}
	if mem.recordatorios[0].texto != "¡Timer terminado!" {
		t.Errorf("texto = %q, esperaba el texto genérico de timer", mem.recordatorios[0].texto)
	}
}

func TestProcess_QueRecordatoriosTengo(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.recordatorios = []recordatorioFalso{{texto: "llamar a mamá"}, {texto: "comprar pan"}}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("qué recordatorios tengo")

	if !contains(got, "2") {
		t.Errorf("respuesta = %q, esperaba que mencionara la cantidad (2)", got)
	}
}

func TestProcess_CancelarRecordatorioEspecifico(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.recordatorios = []recordatorioFalso{{texto: "llamar a mamá"}, {texto: "comprar pan"}}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	b.Process("cancelá el recordatorio de mamá")

	if len(mem.recordatorios) != 1 {
		t.Fatalf("esperaba 1 recordatorio restante, hay %d", len(mem.recordatorios))
	}
	if mem.recordatorios[0].texto != "comprar pan" {
		t.Errorf("quedó %q, esperaba que sobreviviera 'comprar pan'", mem.recordatorios[0].texto)
	}
}

func TestProcess_CancelarTodosLosRecordatorios(t *testing.T) {
	mem := nuevaMemoriaFalsa()
	mem.recordatorios = []recordatorioFalso{{texto: "uno"}, {texto: "dos"}}
	b := NewBrain(&manosFalsas{}, BrainOpciones{Memoria: mem})

	got := b.Process("cancelá todos los recordatorios")

	if len(mem.recordatorios) != 0 {
		t.Errorf("esperaba 0 recordatorios restantes, hay %d", len(mem.recordatorios))
	}
	if !contains(got, "2") {
		t.Errorf("respuesta = %q, esperaba que mencionara cuántos canceló", got)
	}
}

func TestProcess_QueDijiste_RepiteUltimaRespuesta(t *testing.T) {
	manos := &manosFalsas{respuesta: "Abriendo, señor."}
	b := NewBrain(manos, BrainOpciones{})

	b.Process("abrir chrome")
	got := b.Process("qué dijiste")

	if got != "Abriendo, señor." {
		t.Errorf("respuesta = %q, esperaba que repitiera la respuesta anterior", got)
	}
}

func TestProcess_QueDijiste_SinNadaPrevio(t *testing.T) {
	b := NewBrain(&manosFalsas{}, BrainOpciones{})

	got := b.Process("qué dijiste")

	if got == "" {
		t.Error("esperaba un mensaje explicando que todavía no dijo nada, no una respuesta vacía")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
