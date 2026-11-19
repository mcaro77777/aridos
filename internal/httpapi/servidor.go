package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"aridos/internal/seguridad"
	"github.com/go-sql-driver/mysql"
)

const (
	nombreCookie = "aridos_sesion"
	versionAPI   = "1.0.0"
)

type Opciones struct {
	DB                   *sql.DB
	Logger               *slog.Logger
	Sesiones             *seguridad.Gestor
	UsuarioAdministrador string
	ClaveAdministrador   string
	CookieSegura         bool
	OrigenPermitido      string
	DirectorioCliente    string
}

// Servidor expone la API y el cliente Angular en el mismo origen.
type Servidor struct {
	db                   *sql.DB
	logger               *slog.Logger
	sesiones             *seguridad.Gestor
	usuarioAdministrador string
	claveAdministrador   string
	cookieSegura         bool
	origenPermitido      string
	directorioCliente    string
	manejador            http.Handler
}

func Nuevo(op Opciones) (*Servidor, error) {
	if op.DB == nil || op.Sesiones == nil {
		return nil, errors.New("DB y gestor de sesiones son obligatorios")
	}
	if op.Logger == nil {
		op.Logger = slog.Default()
	}
	s := &Servidor{
		db:                   op.DB,
		logger:               op.Logger,
		sesiones:             op.Sesiones,
		usuarioAdministrador: op.UsuarioAdministrador,
		claveAdministrador:   op.ClaveAdministrador,
		cookieSegura:         op.CookieSegura,
		origenPermitido:      strings.TrimRight(op.OrigenPermitido, "/"),
		directorioCliente:    op.DirectorioCliente,
	}
	s.manejador = s.rutas()
	return s, nil
}

func (s *Servidor) Handler() http.Handler { return s.manejador }

func (s *Servidor) rutas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/version", s.version)
	mux.HandleFunc("POST /api/auth/login", s.iniciarSesion)
	mux.HandleFunc("POST /api/auth/logout", s.cerrarSesion)
	mux.Handle("GET /api/auth/session", s.autenticar(http.HandlerFunc(s.sesionActual)))

	mux.Handle("GET /api/resumen", s.autenticar(http.HandlerFunc(s.obtenerResumen)))
	mux.Handle("GET /api/solicitantes", s.autenticar(http.HandlerFunc(s.listarSolicitantes)))
	mux.Handle("POST /api/solicitantes", s.autenticar(http.HandlerFunc(s.crearSolicitante)))
	mux.Handle("PATCH /api/solicitantes/{id}", s.autenticar(http.HandlerFunc(s.editarSolicitante)))
	mux.Handle("GET /api/materiales", s.autenticar(http.HandlerFunc(s.listarMateriales)))
	mux.Handle("POST /api/materiales", s.autenticar(http.HandlerFunc(s.crearMaterial)))
	mux.Handle("PATCH /api/materiales/{id}", s.autenticar(http.HandlerFunc(s.editarMaterial)))
	mux.Handle("GET /api/choferes", s.autenticar(http.HandlerFunc(s.listarChoferes)))
	mux.Handle("POST /api/choferes", s.autenticar(http.HandlerFunc(s.crearChofer)))
	mux.Handle("PATCH /api/choferes/{id}", s.autenticar(http.HandlerFunc(s.editarChofer)))
	mux.Handle("GET /api/vehiculos", s.autenticar(http.HandlerFunc(s.listarVehiculos)))
	mux.Handle("POST /api/vehiculos", s.autenticar(http.HandlerFunc(s.crearVehiculo)))
	mux.Handle("PATCH /api/vehiculos/{id}", s.autenticar(http.HandlerFunc(s.editarVehiculo)))

	mux.Handle("GET /api/solicitudes", s.autenticar(http.HandlerFunc(s.listarSolicitudes)))
	mux.Handle("POST /api/solicitudes", s.autenticar(http.HandlerFunc(s.crearSolicitud)))
	mux.Handle("GET /api/solicitudes/{id}", s.autenticar(http.HandlerFunc(s.obtenerSolicitud)))
	mux.Handle("PATCH /api/solicitudes/{id}", s.autenticar(http.HandlerFunc(s.editarSolicitud)))
	mux.Handle("POST /api/solicitudes/{id}/anular", s.autenticar(http.HandlerFunc(s.anularSolicitud)))
	mux.Handle("POST /api/solicitudes/{id}/entregas", s.autenticar(http.HandlerFunc(s.registrarEntrega)))
	mux.Handle("POST /api/entregas/{id}/anular", s.autenticar(http.HandlerFunc(s.anularEntrega)))

	// Nunca se entrega index.html para una ruta API inexistente.
	apiNoEncontrada := func(w http.ResponseWriter, _ *http.Request) {
		escribirError(w, http.StatusNotFound, "ruta_no_encontrada", "La ruta de la API no existe", nil)
	}
	mux.HandleFunc("/api", apiNoEncontrada)
	mux.HandleFunc("/api/", apiNoEncontrada)
	mux.Handle("/", s.spa())
	return s.recuperar(s.registrar(s.cabecerasSeguras(s.validarOrigen(mux))))
}

func (s *Servidor) cabecerasSeguras(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		if s.cookieSegura {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		siguiente.ServeHTTP(w, r)
	})
}

func (s *Servidor) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancelar := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancelar()
	if err := s.db.PingContext(ctx); err != nil {
		s.logger.Error("health de MySQL fallo", "error", err)
		escribirJSON(w, http.StatusServiceUnavailable, map[string]any{"estado": "no_disponible", "baseDatos": "sin_conexion"})
		return
	}
	escribirJSON(w, http.StatusOK, map[string]any{"estado": "disponible", "baseDatos": "conectada"})
}

func (s *Servidor) version(w http.ResponseWriter, _ *http.Request) {
	escribirJSON(w, http.StatusOK, map[string]string{"version": versionAPI})
}

func (s *Servidor) recuperar(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recuperado := recover(); recuperado != nil {
				s.logger.Error("panic recuperado", "panic", recuperado, "metodo", r.Method, "ruta", r.URL.Path, "pila", string(debug.Stack()))
				escribirError(w, http.StatusInternalServerError, "error_interno", "Ocurrio un error interno", nil)
			}
		}()
		siguiente.ServeHTTP(w, r)
	})
}

type respuestaEstado struct {
	http.ResponseWriter
	estado int
}

func (w *respuestaEstado) WriteHeader(estado int) {
	w.estado = estado
	w.ResponseWriter.WriteHeader(estado)
}

func (s *Servidor) registrar(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		rw := &respuestaEstado{ResponseWriter: w, estado: http.StatusOK}
		siguiente.ServeHTTP(rw, r)
		s.logger.Info("solicitud HTTP", "metodo", r.Method, "ruta", r.URL.Path, "estado", rw.estado, "duracion_ms", time.Since(inicio).Milliseconds(), "remoto", r.RemoteAddr)
	})
}

func (s *Servidor) validarOrigen(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			origen := strings.TrimRight(r.Header.Get("Origin"), "/")
			if origen != "" && !s.origenValido(origen, r.Host) {
				escribirError(w, http.StatusForbidden, "origen_no_permitido", "El origen de la solicitud no esta permitido", nil)
				return
			}
		}
		siguiente.ServeHTTP(w, r)
	})
}

func (s *Servidor) origenValido(origen, host string) bool {
	if s.origenPermitido != "" {
		return origen == s.origenPermitido
	}
	u, err := url.Parse(origen)
	return err == nil && strings.EqualFold(u.Host, host) && (u.Scheme == "http" || u.Scheme == "https")
}

func (s *Servidor) spa() http.Handler {
	raiz, err := filepath.Abs(s.directorioCliente)
	if err != nil {
		raiz = s.directorioCliente
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			escribirError(w, http.StatusMethodNotAllowed, "metodo_no_permitido", "Metodo no permitido", nil)
			return
		}
		ruta := strings.TrimPrefix(filepath.Clean(filepath.FromSlash(r.URL.Path)), string(filepath.Separator))
		archivo := filepath.Join(raiz, ruta)
		relativa, relErr := filepath.Rel(raiz, archivo)
		if relErr != nil || relativa == ".." || strings.HasPrefix(relativa, ".."+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		if info, statErr := os.Stat(archivo); statErr == nil && !info.IsDir() {
			if tipo := mime.TypeByExtension(filepath.Ext(archivo)); tipo != "" {
				w.Header().Set("Content-Type", tipo)
			}
			if strings.Contains(filepath.Base(archivo), ".") && filepath.Ext(archivo) != ".html" {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, archivo)
			return
		}
		// Una ruta con extension es un recurso faltante, no una ruta Angular.
		if filepath.Ext(ruta) != "" {
			http.NotFound(w, r)
			return
		}
		indice := filepath.Join(raiz, "index.html")
		if _, statErr := os.Stat(indice); statErr != nil {
			http.Error(w, "Cliente web no disponible", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indice)
	})
}

type errorAPI struct {
	Error detalleError `json:"error"`
}

type detalleError struct {
	Codigo   string         `json:"codigo"`
	Mensaje  string         `json:"mensaje"`
	Detalles map[string]any `json:"detalles,omitempty"`
}

func escribirError(w http.ResponseWriter, estado int, codigo, mensaje string, detalles map[string]any) {
	escribirJSON(w, estado, errorAPI{Error: detalleError{Codigo: codigo, Mensaje: mensaje, Detalles: detalles}})
}

func escribirJSON(w http.ResponseWriter, estado int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}

func leerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		mensaje := "El cuerpo JSON no es valido"
		var maximo *http.MaxBytesError
		if errors.As(err, &maximo) {
			mensaje = "El cuerpo excede 1 MiB"
		}
		escribirError(w, http.StatusBadRequest, "json_invalido", mensaje, nil)
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		escribirError(w, http.StatusBadRequest, "json_invalido", "Solo se permite un valor JSON", nil)
		return false
	}
	return true
}

// opcionalNulo distingue en un PATCH entre campo ausente, campo null y campo
// con valor. encoding/json no ofrece esa distincion usando solo **T.
type opcionalNulo[T any] struct {
	Presente bool
	Valor    *T
}

func (o *opcionalNulo[T]) UnmarshalJSON(datos []byte) error {
	o.Presente = true
	if string(datos) == "null" {
		o.Valor = nil
		return nil
	}
	var valor T
	if err := json.Unmarshal(datos, &valor); err != nil {
		return err
	}
	o.Valor = &valor
	return nil
}

func idRuta(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		escribirError(w, http.StatusBadRequest, "id_invalido", "El identificador no es valido", nil)
		return 0, false
	}
	return id, true
}

func errorBaseDatos(w http.ResponseWriter, logger *slog.Logger, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		escribirError(w, http.StatusNotFound, "no_encontrado", "El registro no existe", nil)
		return
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062:
			escribirError(w, http.StatusConflict, "registro_duplicado", "Ya existe un registro con esos datos", nil)
			return
		case 1451, 1452:
			escribirError(w, http.StatusConflict, "referencia_invalida", "El registro esta relacionado con datos inexistentes o en uso", nil)
			return
		case 3819:
			escribirError(w, http.StatusUnprocessableEntity, "datos_invalidos", "Los datos no cumplen las reglas del negocio", nil)
			return
		}
	}
	logger.Error("operacion de base de datos fallo", "error", err)
	escribirError(w, http.StatusInternalServerError, "error_base_datos", "No fue posible completar la operacion", nil)
}

func textoOpcional(v *string, maximo int) any {
	if v == nil {
		return nil
	}
	limpio := strings.TrimSpace(*v)
	if limpio == "" {
		return nil
	}
	if len([]rune(limpio)) > maximo {
		return nil
	}
	return limpio
}

func fechaValida(fecha string) bool {
	_, err := time.Parse("2006-01-02", fecha)
	return err == nil
}

func recortar(s string) string { return strings.TrimSpace(s) }

func validarLargo(nombre, valor string, minimo, maximo int) error {
	n := len([]rune(strings.TrimSpace(valor)))
	if n < minimo || n > maximo {
		return fmt.Errorf("%s debe tener entre %d y %d caracteres", nombre, minimo, maximo)
	}
	return nil
}
