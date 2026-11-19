package httpapi

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aridos/internal/seguridad"
	_ "github.com/go-sql-driver/mysql"
)

func servidorPrueba(t *testing.T) *Servidor {
	t.Helper()
	db, err := sql.Open("mysql", "usuario:clave@tcp(127.0.0.1:3306)/gestion_aridos")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	gestor, err := seguridad.NuevoGestor([]byte("01234567890123456789012345678901"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	directorio := t.TempDir()
	if err := os.WriteFile(filepath.Join(directorio, "index.html"), []byte("<html>cliente angular</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Nuevo(Opciones{
		DB: db, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Sesiones: gestor,
		UsuarioAdministrador: "admin", ClaveAdministrador: "clave-administrador",
		DirectorioCliente: directorio,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInicioYSesion(t *testing.T) {
	s := servidorPrueba(t)
	login := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"usuario":"admin","contrasena":"clave-administrador"}`))
	login.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, login)
	if w.Code != http.StatusOK {
		t.Fatalf("login respondio %d: %s", w.Code, w.Body.String())
	}
	respuesta := w.Result()
	cookies := respuesta.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Name != nombreCookie {
		t.Fatalf("cookie de sesion incorrecta: %+v", cookies)
	}

	sesion := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sesion.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, sesion)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"autenticado":true`) {
		t.Fatalf("sesion respondio %d: %s", w.Code, w.Body.String())
	}
}

func TestRutaAPINoUsaFallbackSPA(t *testing.T) {
	s := servidorPrueba(t)
	for _, ruta := range []string{"/api", "/api/no-existe"} {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "cliente angular") {
			t.Fatalf("%s: respuesta inesperada %d: %s", ruta, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "ruta_no_encontrada") {
			t.Fatalf("%s: error JSON no encontrado: %s", ruta, w.Body.String())
		}
	}
}

func TestFallbackSPAYCabeceras(t *testing.T) {
	s := servidorPrueba(t)
	r := httptest.NewRequest(http.MethodGet, "/solicitudes/42", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "cliente angular") {
		t.Fatalf("fallback respondio %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Frame-Options") != "DENY" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("faltan cabeceras seguras: %+v", w.Header())
	}
}

func TestOpcionalNuloDistingueAusenteYNull(t *testing.T) {
	type entrada struct {
		Valor opcionalNulo[int64] `json:"valor"`
	}
	var ausente entrada
	if !leerJSONPrueba(t, `{}`, &ausente) || ausente.Valor.Presente {
		t.Fatal("campo ausente marcado como presente")
	}
	var nulo entrada
	if !leerJSONPrueba(t, `{"valor":null}`, &nulo) || !nulo.Valor.Presente || nulo.Valor.Valor != nil {
		t.Fatal("campo null no se distinguio")
	}
	var valor entrada
	if !leerJSONPrueba(t, `{"valor":0}`, &valor) || !valor.Valor.Presente || valor.Valor.Valor == nil || *valor.Valor.Valor != 0 {
		t.Fatal("campo con cero no se preservo")
	}
}

func leerJSONPrueba(t *testing.T, cuerpo string, destino any) bool {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(cuerpo))
	w := httptest.NewRecorder()
	return leerJSON(w, r, destino)
}

func TestCantidadValida(t *testing.T) {
	for _, valida := range []float64{0.001, 1, 12.345, 999999999.999} {
		if !cantidadValida(valida) {
			t.Errorf("cantidad valida rechazada: %v", valida)
		}
	}
	for _, invalida := range []float64{0, -1, 1.0001, 1000000000} {
		if cantidadValida(invalida) {
			t.Errorf("cantidad invalida aceptada: %v", invalida)
		}
	}
}

func TestNormalizarSolicitudDerivaAnio(t *testing.T) {
	e := entradaSolicitud{FechaCompra: "2026-09-14", Anio: 2001}
	normalizarSolicitud(&e)
	if e.Anio != 2026 {
		t.Fatalf("anio no derivado desde fechaCompra: %d", e.Anio)
	}
}

func TestCompararLineasIgnoraOrdenYConservaPrecioCero(t *testing.T) {
	cero := int64(0)
	a := []entradaLineaSolicitud{
		{MaterialID: 1, CantidadComprada: 10, PrecioUnitarioAplicadoCLP: &cero},
		{MaterialID: 2, CantidadComprada: 5.5},
	}
	b := []entradaLineaSolicitud{
		{MaterialID: 2, CantidadComprada: 5.5},
		{MaterialID: 1, CantidadComprada: 10, PrecioUnitarioAplicadoCLP: &cero},
	}
	if !lineasSolicitudIguales(a, b) {
		t.Fatal("lineas equivalentes se consideraron distintas")
	}
	b[1].PrecioUnitarioAplicadoCLP = nil
	if lineasSolicitudIguales(a, b) {
		t.Fatal("NULL y precio cero se consideraron iguales")
	}
}
