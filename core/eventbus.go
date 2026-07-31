package core

import "fmt"

type Evento struct {
	Tipo    string
	Origen  string
	Payload any
}

type Suscriptor struct {
	ID      string
	Canal   chan Evento
	Cerrar  chan struct{}
	Filtro  func(Evento) bool
}

type BusDeEventos struct {
	subs    map[string][]*Suscriptor
	abierto chan struct{}
}

var busGlobal *BusDeEventos

func init() {
	busGlobal = NuevoBusDeEventos()
}

func BusGlobal() *BusDeEventos {
	return busGlobal
}

func NuevoBusDeEventos() *BusDeEventos {
	return &BusDeEventos{
		subs:    make(map[string][]*Suscriptor),
		abierto: make(chan struct{}),
	}
}

func (b *BusDeEventos) Suscribir(tipo string, id string, buffer int, filtro func(Evento) bool) *Suscriptor {
	s := &Suscriptor{
		ID:     id,
		Canal:  make(chan Evento, buffer),
		Cerrar: make(chan struct{}),
		Filtro:  filtro,
	}
	b.subs[tipo] = append(b.subs[tipo], s)
	return s
}

func (b *BusDeEventos) Publicar(evento Evento) {
	for _, s := range b.subs[evento.Tipo] {
		if s.Filtro != nil && !s.Filtro(evento) {
			continue
		}
		select {
		case s.Canal <- evento:
		default:
			fmt.Printf("[EVENTBUS] Buffer lleno para suscriptor %s/%s, descartando evento\n", evento.Tipo, s.ID)
		}
	}
}

func (b *BusDeEventos)CerrarSuscriptor(s *Suscriptor) {
	close(s.Cerrar)
	close(s.Canal)
}
