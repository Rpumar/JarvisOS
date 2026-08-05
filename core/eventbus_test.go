package core

import "testing"

func TestBusDeEventosSuscripcionYPublicacion(t *testing.T) {
	b := NuevoBusDeEventos()
	s := b.Suscribir("test", "s1", 10, nil)

	b.Publicar(Evento{Tipo: "test", Origen: "origen", Payload: "hola"})

	select {
	case ev := <-s.Canal:
		if ev.Origen != "origen" || ev.Payload != "hola" {
			t.Errorf("evento recibido incorrecto: %+v", ev)
		}
	default:
		t.Error("el suscriptor no recibió el evento")
	}
}

func TestBusDeEventosFiltro(t *testing.T) {
	b := NuevoBusDeEventos()
	recibidos := 0
	s := b.Suscribir("test", "f1", 10, func(ev Evento) bool {
		return ev.Payload == "solo-esta"
	})
	defer func() { _ = s }()

	b.Publicar(Evento{Tipo: "test", Payload: "otro"})
	b.Publicar(Evento{Tipo: "test", Payload: "solo-esta"})

	select {
	case ev := <-s.Canal:
		if ev.Payload != "solo-esta" {
			t.Errorf("el filtro dejó pasar %q", ev.Payload)
		}
	default:
		t.Error("no llegó el evento que debería pasar el filtro")
	}
	_ = recibidos
}

func TestBusDeEventosTiposIndependientes(t *testing.T) {
	b := NuevoBusDeEventos()
	sA := b.Suscribir("tipoA", "a", 10, nil)
	sB := b.Suscribir("tipoB", "b", 10, nil)

	b.Publicar(Evento{Tipo: "tipoA", Payload: "paraA"})

	select {
	case <-sA.Canal:
	default:
		t.Error("suscriptor A no recibió su evento")
	}
	select {
	case ev := <-sB.Canal:
		t.Errorf("suscriptor B recibió evento de otro tipo: %+v", ev)
	default:
	}
}

func TestBusDeEventosCerrarSuscriptor(t *testing.T) {
	b := NuevoBusDeEventos()
	s := b.Suscribir("test", "c1", 10, nil)
	b.CerrarSuscriptor(s)

	select {
	case _, ok := <-s.Canal:
		if ok {
			t.Error("el canal debería estar cerrado")
		}
	default:
		t.Error("cerrarSuscriptor debería cerrar el canal")
	}
}

func TestBusDeEventosBufferLlenoNoBloquea(t *testing.T) {
	b := NuevoBusDeEventos()
	_ = b.Suscribir("test", "b1", 1, nil)

	b.Publicar(Evento{Tipo: "test", Payload: "1"})
	b.Publicar(Evento{Tipo: "test", Payload: "2"})
	b.Publicar(Evento{Tipo: "test", Payload: "3"})
}
