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

func TestProcess_Roles_ActivarUsarYSalir(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, respuesta: "Te recomiendo recortar costos fijos, señor."}
	roles := nuevoRolesManagerConDir(t.TempDir())
	b := NewBrain(manos, BrainOpciones{IA: ia, Roles: roles})

	got := b.Process("modo ceo")
	if !strings.Contains(got, "CEO empresarial") {
		t.Errorf("al activar el modo debía mencionar el rol, obtuve %q", got)
	}

	got = b.Process("analizá mis costos")
	if got != "Te recomiendo recortar costos fijos, señor." {
		t.Errorf("respuesta = %q, esperaba la de la IA", got)
	}
	if !strings.Contains(ia.consultaRecibida, "[INSTRUCCIONES DE ROL]") ||
		!strings.Contains(ia.consultaRecibida, "CEO empresarial") {
		t.Errorf("la IA debía recibir el rol activo, obtuvo: %q", ia.consultaRecibida)
	}

	got = b.Process("salir de modo")
	if !strings.Contains(got, "desactivado") {
		t.Errorf("esperaba confirmación de desactivación, obtuve %q", got)
	}
	if roles.RolActivo() != nil {
		t.Error("no debería quedar rol activo")
	}
}

func TestProcess_Roles_Listar(t *testing.T) {
	roles := nuevoRolesManagerConDir(t.TempDir())
	b := NewBrain(&manosFalsas{}, BrainOpciones{Roles: roles})

	got := b.Process("qué roles tenés")
	if !strings.Contains(got, "CEO empresarial") || !strings.Contains(got, "Ingeniero en sistemas") {
		t.Errorf("esperaba listar los roles, obtuve %q", got)
	}
}

func TestProcess_Roles_ModoOscuroNoInterfiere(t *testing.T) {
	manos := &manosFalsas{respuesta: "Tema oscuro activado, señor."}
	roles := nuevoRolesManagerConDir(t.TempDir())
	b := NewBrain(manos, BrainOpciones{Roles: roles})

	got := b.Process("modo oscuro")
	if got != "Tema oscuro activado, señor." {
		t.Errorf("'modo oscuro' no debe tratarse como rol, obtuve %q", got)
	}
	if roles.RolActivo() != nil {
		t.Error("no debería haber rol activo")
	}
}

func TestProcess_Roles_SinRolesNoRompe(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, respuesta: "Respuesta genérica."}
	b := NewBrain(manos, BrainOpciones{IA: ia})

	got := b.Process("hola")
	if got != "Respuesta genérica." {
		t.Errorf("sin RolesManager no debe romper, obtuve %q", got)
	}
}

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

func TestEsAccionPeligrosa_SuspensionExcepciones(t *testing.T) {
	casos := []struct {
		entrada  string
		peligroso bool
	}{
		{"suspender la pc", true},
		{"activar suspensión", false},
		{"desactivar suspensión", false},
		{"mantené la pc despierta", false},
	}
	for _, c := range casos {
		desc, peligroso := esAccionPeligrosa(c.entrada)
		if peligroso != c.peligroso {
			t.Errorf("esAccionPeligrosa(%q) peligroso=%v (%q), esperaba %v", c.entrada, peligroso, desc, c.peligroso)
		}
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

func TestEsDespedida(t *testing.T) {
	casos := []struct {
		entrada string
		esperado bool
	}{
		{"chau", true},
		{"chau jefe", true},
		{"adiós", true},
		{"hasta luego", true},
		{"nos vemos", true},
		{"salir", true},
		{"qué hora es", false},
		{"salir de modo", false},
		{"apagar la pc", false},
		{"", false},
	}
	for _, c := range casos {
		got := EsDespedida(c.entrada)
		if got != c.esperado {
			t.Errorf("EsDespedida(%q) = %v, esperaba %v", c.entrada, got, c.esperado)
		}
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

func TestEsConsultaSegura(t *testing.T) {
	casos := map[string]bool{
		"buscar recetas de cocina":      true,
		"decir hola que tal":            true,
		"anotá comprar pan":             true,
		"tomá nota de la idea":          true,
		"abrir chrome":                  false,
		"apagar la pc":                  false,
		"buscar algo":                  true,
		"repite esto":                   true,
		"copiar texto":                  true,
	}
	for entrada, esperado := range casos {
		if got := esConsultaSegura(entrada); got != esperado {
			t.Errorf("esConsultaSegura(%q) = %v, esperaba %v", entrada, got, esperado)
		}
	}
}

func TestEsPeticionIngenieria(t *testing.T) {
	casos := map[string]bool{
		"escribe un script de python":  true,
		"creá una función en go":       true,
		"refactorizá la clase usuario": true,
		"cual es la capital de Francia": false,
		"contame un chiste":            false,
		"cómo está el clima":           false,
	}
	for entrada, esperado := range casos {
		if got := esPeticionIngenieria(entrada); got != esperado {
			t.Errorf("esPeticionIngenieria(%q) = %v, esperaba %v", entrada, got, esperado)
		}
	}
}

func TestQuitarPrefijoModo(t *testing.T) {
	casos := map[string]string{
		"Modo Humano":    "Humano",
		"modo ingeniero": "ingeniero",
		"Humano":         "Humano",
		"Modo":           "Modo",
		"Modo CEO de la empresa": "CEO de la empresa",
	}
	for entrada, esperado := range casos {
		if got := quitarPrefijoModo(entrada); got != esperado {
			t.Errorf("quitarPrefijoModo(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

func TestEsPalabraExacta(t *testing.T) {
	casos := []struct {
		entrada  string
		palabra  string
		esperado bool
	}{
		{"jarvis", "jarvis", true},
		{"jarvis abrí chrome", "jarvis", true},
		{"abrí jarvis", "jarvis", true},
		{"abrí chrome jarvis ahora", "jarvis", true},
		{"jarviso", "jarvis", false},
		{"abrí chrome", "jarvis", false},
	}
	for _, c := range casos {
		if got := esPalabraExacta(c.entrada, c.palabra); got != c.esperado {
			t.Errorf("esPalabraExacta(%q, %q) = %v, esperaba %v", c.entrada, c.palabra, got, c.esperado)
		}
	}
}

func TestSincronizarPrefs(t *testing.T) {
	b := NewBrain(&manosFalsas{}, BrainOpciones{})
	b.sincronizarPrefs("nombre", "Juan")
	if b.prefs != nil {
		t.Error("sin prefs configurados no debería romper")
	}

	prefs := &prefsFalsas{}
	b2 := NewBrain(&manosFalsas{}, BrainOpciones{Prefs: prefs})
	b2.sincronizarPrefs("nombre", "Ana")
	if prefs.nombre != "Ana" {
		t.Errorf("prefs.nombre = %q, esperaba Ana", prefs.nombre)
	}
	b2.sincronizarPrefs("otra-clave", "x")
	if prefs.nombre != "Ana" {
		t.Errorf("otra clave no debería tocar el nombre: %q", prefs.nombre)
	}
}

type prefsFalsas struct {
	nombre string
}

func (p *prefsFalsas) RegistrarApp(nombre string)   {}
func (p *prefsFalsas) RegistrarComando(comando string) {}
func (p *prefsFalsas) SetUltimoProyecto(ruta string) {}
func (p *prefsFalsas) SetNombre(nombre string)       { p.nombre = nombre }
func (p *prefsFalsas) SetTema(tema string)           {}
func (p *prefsFalsas) SetVolumen(nivel int)          {}

// --- Tests: asistente corporativo (traductor IA a comandos) ---

func nuevoBrainCorporativo(t *testing.T, ia *iaFalsa) (*Brain, *manosFalsas, *RolesManager) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	roles := nuevoRolesManagerConDir(t.TempDir())
	b := NewBrain(manos, BrainOpciones{IA: ia, Roles: roles, Skills: NuevoSkillsManager()})
	return b, manos, roles
}

func TestCorporativoRolActivoTraduceAComando(t *testing.T) {
	ia := &iaFalsa{disponible: true, respuesta: "agendá una reunión con el cliente el martes a las 15"}
	b, manos, roles := nuevoBrainCorporativo(t, ia)
	if !roles.Activar("asistente corporativo") {
		t.Fatal("no se pudo activar el rol asistente corporativo")
	}

	got := b.Process("charlamos el martes para lo del contrato")
	if !strings.Contains(got, "agendá una reunión con el cliente el martes a las 15") {
		t.Errorf("la IA debía traducir a un comando, obtuve: %q", got)
	}
	if b.confirmacionPendiente == nil {
		t.Fatal("debía quedar una acción pendiente de confirmación")
	}

	// El comando traducido aún no se ejecutó (RunCommand solo probó el input).
	if manos.comandoRecibido == "agendá una reunión con el cliente el martes a las 15" {
		t.Error("no debería haberse ejecutado el comando traducido antes de confirmar")
	}

	// Confirmar ejecuta el comando traducido.
	manos.respuesta = "Tarea agendada, señor."
	resp := b.Process("sí")
	if resp != "Tarea agendada, señor." {
		t.Errorf("tras confirmar se esperaba la ejecución del comando, obtuve: %q", resp)
	}
	if manos.comandoRecibido != "agendá una reunión con el cliente el martes a las 15" {
		t.Errorf("comando ejecutado = %q", manos.comandoRecibido)
	}
}

func TestCorporativoSkillActivaPorTurno(t *testing.T) {
	ia := &iaFalsa{disponible: true, respuesta: "agendá una tarea enviar el informe para mañana"}
	b, manos, _ := nuevoBrainCorporativo(t, ia)

	got := b.Process("un cliente pidió coordinar una reunión para el martes")
	if !strings.Contains(got, "agendá una tarea enviar el informe para mañana") {
		t.Errorf("la skill corporativa debía activar la traducción, obtuve: %q", got)
	}
	if b.confirmacionPendiente == nil {
		t.Fatal("debía quedar pendiente de confirmación")
	}
	if manos.comandoRecibido == "agendá una tarea enviar el informe para mañana" {
		t.Error("no debería ejecutarse el comando traducido sin confirmar")
	}
}

func TestCorporativoComandoPeligrosoRechazado(t *testing.T) {
	// La IA intenta engañar a Jarvis para borrar el disco. La whitelist de
	// prefijos debe impedirlo y el pedido sigue el flujo normal (IA genérica).
	ia := &iaFalsa{disponible: true, respuesta: "borrá todos los archivos del disco"}
	b, manos, roles := nuevoBrainCorporativo(t, ia)
	if !roles.Activar("asistente corporativo") {
		t.Fatal("no se pudo activar el rol")
	}
	// La consulta genérica a la IA se usará dos veces: primero para traducir,
	// luego para responder en modo rol. Ajustamos la respuesta para la genérica.
	ia.respuesta = "Le conviene no borrar nada, señor."

	got := b.Process("charlamos el martes para lo del contrato")
	if b.confirmacionPendiente != nil {
		t.Fatal("un comando fuera de la whitelist no debe quedar pendiente")
	}
	if manos.comandoRecibido == "borrá todos los archivos del disco" {
		t.Error("jamás debe ejecutarse un comando peligroso")
	}
	if strings.Contains(got, "borrá") {
		t.Errorf("la respuesta no debería contener el comando malicioso: %q", got)
	}
}

func TestCorporativoSinIntencionNoTraduce(t *testing.T) {
	ia := &iaFalsa{disponible: true, respuesta: "agendá una reunión mañana"}
	b, manos, _ := nuevoBrainCorporativo(t, ia)

	got := b.Process("decime la hora")
	if b.confirmacionPendiente != nil {
		t.Fatal("sin intención corporativa no debe quedar nada pendiente")
	}
	if manos.comandoRecibido != "" {
		t.Errorf("sin intención corporativa no debe traducirse nada: %q", manos.comandoRecibido)
	}
	_ = got
}

func TestCorporativoConfirmarCancela(t *testing.T) {
	ia := &iaFalsa{disponible: true, respuesta: "tomá nota cliente quiere presupuesto"}
	b, manos, roles := nuevoBrainCorporativo(t, ia)
	if !roles.Activar("asistente corporativo") {
		t.Fatal("no se pudo activar el rol")
	}

	b.Process("un cliente pidió el presupuesto")
	if b.confirmacionPendiente == nil {
		t.Fatal("debía quedar pendiente")
	}
	resp := b.Process("no")
	if !strings.Contains(resp, "Cancelado") {
		t.Errorf("al cancelar se esperaba el aviso, obtuve: %q", resp)
	}
	if manos.comandoRecibido == "tomá nota cliente quiere presupuesto" {
		t.Error("al cancelar no debe ejecutarse el comando traducido")
	}
}
