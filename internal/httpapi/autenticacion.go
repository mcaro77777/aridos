package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"aridos/internal/seguridad"
)

type claveContexto string

const claveUsuario claveContexto = "usuario"

type credenciales struct {
	Usuario    string `json:"usuario"`
	Contrasena string `json:"contrasena"`
}

func (s *Servidor) iniciarSesion(w http.ResponseWriter, r *http.Request) {
	var entrada credenciales
	if !leerJSON(w, r, &entrada) {
		return
	}
	usuarioValido := seguridad.CredencialesIguales(strings.TrimSpace(entrada.Usuario), s.usuarioAdministrador)
	claveValida := seguridad.CredencialesIguales(entrada.Contrasena, s.claveAdministrador)
	if !usuarioValido || !claveValida {
		time.Sleep(150 * time.Millisecond)
		escribirError(w, http.StatusUnauthorized, "credenciales_invalidas", "Usuario o contrasena incorrectos", nil)
		return
	}
	token, expira, err := s.sesiones.Crear(s.usuarioAdministrador)
	if err != nil {
		s.logger.Error("crear sesion", "error", err)
		escribirError(w, http.StatusInternalServerError, "error_interno", "No fue posible iniciar sesion", nil)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: nombreCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: s.cookieSegura, SameSite: http.SameSiteLaxMode,
		Expires: expira, MaxAge: int(time.Until(expira).Seconds()),
	})
	escribirJSON(w, http.StatusOK, map[string]any{"autenticado": true, "usuario": s.usuarioAdministrador, "expiraEn": expira})
}

func (s *Servidor) cerrarSesion(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: nombreCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: s.cookieSegura, SameSite: http.SameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servidor) sesionActual(w http.ResponseWriter, r *http.Request) {
	usuario, _ := r.Context().Value(claveUsuario).(string)
	escribirJSON(w, http.StatusOK, map[string]any{"autenticado": true, "usuario": usuario})
}

func (s *Servidor) autenticar(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(nombreCookie)
		if err != nil {
			escribirError(w, http.StatusUnauthorized, "sesion_requerida", "Debe iniciar sesion", nil)
			return
		}
		usuario, err := s.sesiones.Validar(cookie.Value)
		if err != nil {
			escribirError(w, http.StatusUnauthorized, "sesion_invalida", "La sesion no es valida o vencio", nil)
			return
		}
		ctx := context.WithValue(r.Context(), claveUsuario, usuario)
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}
