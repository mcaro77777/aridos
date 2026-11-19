package configuracion

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Configuracion contiene las opciones de ejecucion. Los secretos se aceptan
// solo por variables de entorno para evitar que terminen versionados.
type Configuracion struct {
	Puerto               string
	DSN                  string
	DirectorioCliente    string
	UsuarioAdministrador string
	ClaveAdministrador   string
	SecretoAutenticacion []byte
	DuracionSesion       time.Duration
	Produccion           bool
	OrigenPermitido      string
	TiempoLectura        time.Duration
	TiempoEscritura      time.Duration
	TiempoApagado        time.Duration
	MaxConexiones        int
	MaxConexionesOciosas int
	VidaConexion         time.Duration
}

// Cargar lee y valida la configuracion del proceso.
func Cargar() (Configuracion, error) {
	duracionSesion, err := duracion("SESSION_DURATION", 12*time.Hour)
	if err != nil {
		return Configuracion{}, err
	}
	lectura, err := duracion("HTTP_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Configuracion{}, err
	}
	escritura, err := duracion("HTTP_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Configuracion{}, err
	}
	apagado, err := duracion("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Configuracion{}, err
	}
	vidaConexion, err := duracion("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	if err != nil {
		return Configuracion{}, err
	}
	maxConexiones, err := entero("DB_MAX_OPEN_CONNS", 20)
	if err != nil {
		return Configuracion{}, err
	}
	maxOciosas, err := entero("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return Configuracion{}, err
	}

	c := Configuracion{
		Puerto:               valor("PORT", "8080"),
		DSN:                  strings.TrimSpace(os.Getenv("DATABASE_DSN")),
		DirectorioCliente:    valor("CLIENT_DIR", "ClientApp"),
		UsuarioAdministrador: strings.TrimSpace(os.Getenv("APP_ADMIN_USER")),
		ClaveAdministrador:   os.Getenv("APP_ADMIN_PASSWORD"),
		SecretoAutenticacion: []byte(os.Getenv("APP_AUTH_SECRET")),
		DuracionSesion:       duracionSesion,
		Produccion:           strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production"),
		OrigenPermitido:      strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/"),
		TiempoLectura:        lectura,
		TiempoEscritura:      escritura,
		TiempoApagado:        apagado,
		MaxConexiones:        maxConexiones,
		MaxConexionesOciosas: maxOciosas,
		VidaConexion:         vidaConexion,
	}

	var faltantes []string
	if c.DSN == "" {
		faltantes = append(faltantes, "DATABASE_DSN")
	}
	if c.UsuarioAdministrador == "" {
		faltantes = append(faltantes, "APP_ADMIN_USER")
	}
	if c.ClaveAdministrador == "" {
		faltantes = append(faltantes, "APP_ADMIN_PASSWORD")
	}
	if len(c.SecretoAutenticacion) == 0 {
		faltantes = append(faltantes, "APP_AUTH_SECRET")
	}
	if len(faltantes) > 0 {
		return Configuracion{}, fmt.Errorf("faltan variables obligatorias: %s", strings.Join(faltantes, ", "))
	}
	if len(c.ClaveAdministrador) < 12 {
		return Configuracion{}, errors.New("APP_ADMIN_PASSWORD debe tener al menos 12 caracteres")
	}
	if len(c.SecretoAutenticacion) < 32 {
		return Configuracion{}, errors.New("APP_AUTH_SECRET debe tener al menos 32 caracteres")
	}
	if c.Puerto == "" || strings.ContainsAny(c.Puerto, ":/ ") {
		return Configuracion{}, errors.New("PORT no es valido")
	}
	if c.MaxConexiones < 1 || c.MaxConexionesOciosas < 0 || c.MaxConexionesOciosas > c.MaxConexiones {
		return Configuracion{}, errors.New("configuracion del pool MySQL no valida")
	}
	return c, nil
}

func valor(nombre, predeterminado string) string {
	v := strings.TrimSpace(os.Getenv(nombre))
	if v == "" {
		return predeterminado
	}
	return v
}

func duracion(nombre string, predeterminada time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(nombre))
	if v == "" {
		return predeterminada, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s debe ser una duracion positiva (por ejemplo 12h): %q", nombre, v)
	}
	return d, nil
}

func entero(nombre string, predeterminado int) (int, error) {
	v := strings.TrimSpace(os.Getenv(nombre))
	if v == "" {
		return predeterminado, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un numero entero: %q", nombre, v)
	}
	return n, nil
}
