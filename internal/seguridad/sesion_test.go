package seguridad

import (
	"errors"
	"testing"
	"time"
)

func TestSesionCrearYValidar(t *testing.T) {
	g, err := NuevoGestor([]byte("01234567890123456789012345678901"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	g.reloj = func() time.Time { return instante }
	token, expira, err := g.Crear("administrador")
	if err != nil {
		t.Fatal(err)
	}
	if expira != instante.Add(time.Hour) {
		t.Fatalf("expiracion inesperada: %v", expira)
	}
	usuario, err := g.Validar(token)
	if err != nil || usuario != "administrador" {
		t.Fatalf("sesion no valida: usuario=%q err=%v", usuario, err)
	}
}

func TestSesionRechazaManipulacionYVencimiento(t *testing.T) {
	g, _ := NuevoGestor([]byte("01234567890123456789012345678901"), time.Minute)
	instante := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	g.reloj = func() time.Time { return instante }
	token, _, _ := g.Crear("admin")
	if _, err := g.Validar("x" + token); !errors.Is(err, ErrTokenInvalido) {
		t.Fatalf("se esperaba token invalido, recibido %v", err)
	}
	g.reloj = func() time.Time { return instante.Add(time.Minute) }
	if _, err := g.Validar(token); !errors.Is(err, ErrTokenVencido) {
		t.Fatalf("se esperaba token vencido, recibido %v", err)
	}
}

func TestCredencialesIguales(t *testing.T) {
	if !CredencialesIguales("una-clave", "una-clave") {
		t.Fatal("credenciales iguales resultaron diferentes")
	}
	if CredencialesIguales("una-clave", "otra-clave") {
		t.Fatal("credenciales diferentes resultaron iguales")
	}
}
