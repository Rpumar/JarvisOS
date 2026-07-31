package core

import "time"

type EjecutorComandos interface {
	RunCommand(cmd string) string
	Hablar(texto string) string
}

type TurnoConversacion struct {
	Usuario   string
	Asistente string
}

type ConectorIA interface {
	Disponible() bool
	Consultar(prompt string, historial []TurnoConversacion) (string, error)
}

type AgenteDeCodigo interface {
	Proponer(peticion string) (string, error)
	Confirmar() string
	Cancelar() string
	TienePropuestaPendiente() bool
}

type MemoriaPersistente interface {
	GuardarHecho(clave, valor string) error
	ObtenerHecho(clave string) (valor string, existe bool)
	AgregarNota(texto string) error
	ObtenerNotas() []string
	BuscarNotas(texto string) []string
	AgregarRecordatorio(texto string, momento time.Time) error
	AgregarRecordatorioConPeriodo(texto string, momento time.Time, periodo string) error
	ObtenerRecordatoriosPendientesTexto() []string
	CancelarRecordatorios(textoBusqueda string) (int, error)
	CrearLista(nombre string) error
	AgregarItemLista(nombreLista, item string) error
	MarcarItemLista(nombreLista, item string) (string, error)
	ObtenerListas() []string
	ObtenerLista(nombre string) (string, bool)
	EliminarLista(nombre string) error
}

type RegistroPreferencias interface {
	RegistrarApp(nombre string)
	RegistrarComando(comando string)
	SetUltimoProyecto(ruta string)
	SetNombre(nombre string)
	SetTema(tema string)
	SetVoz(activada bool)
	SetVolumen(nivel int)
}

type BrainOpciones struct {
	IA             ConectorIA
	Coder          AgenteDeCodigo
	Memoria        MemoriaPersistente
	IngAgente      IngAgente
	Prefs          RegistroPreferencias
	MaxHistorialIA int
}

type IngAgente interface {
	Disponible() bool
	Procesar(peticion string) string
	SetRespuestaUsuario(respuesta string)
	Reset()
	TieneTareaPendiente() bool
	PlanPendienteDescripcion() string
	ContinuarPlan() string
}
