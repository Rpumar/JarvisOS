package core

import "time"

type EjecutorComandos interface {
	RunCommand(cmd string) string
}

type TurnoConversacion struct {
	Usuario   string
	Asistente string
}

type ConectorIA interface {
	Disponible() bool
	Consultar(prompt string, historial []TurnoConversacion) (string, error)
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
	SetVolumen(nivel int)
}

type BrainOpciones struct {
	IA             ConectorIA
	Memoria        MemoriaPersistente
	Prefs          RegistroPreferencias
	Skills         *SkillsManager
	Roles          *RolesManager
	Procedimientos *GestorProcedimientos
	MaxHistorialIA int
}
