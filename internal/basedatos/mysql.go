package basedatos

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var tablasObligatorias = []string{
	"choferes", "entregas", "entregas_materiales", "materiales",
	"solicitantes", "solicitudes", "solicitudes_materiales", "vehiculos",
}

var vistasObligatorias = []string{
	"vista_resumen_solicitudes", "vista_saldos_solicitud_material",
}

// Abrir crea el pool, comprueba la conexion y valida que el script SQL haya
// sido ejecutado completo antes de aceptar trafico.
func Abrir(ctx context.Context, dsn string, maxAbiertas, maxOciosas int, vidaMaxima time.Duration) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_DSN no valido: %w", err)
	}
	if strings.TrimSpace(cfg.DBName) == "" {
		return nil, fmt.Errorf("DATABASE_DSN debe indicar la base de datos")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	if _, ok := cfg.Params["charset"]; !ok {
		cfg.Params["charset"] = "utf8mb4"
	}
	if _, ok := cfg.Params["time_zone"]; !ok {
		cfg.Params["time_zone"] = "'+00:00'"
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("abrir MySQL: %w", err)
	}
	db.SetMaxOpenConns(maxAbiertas)
	db.SetMaxIdleConns(maxOciosas)
	db.SetConnMaxLifetime(vidaMaxima)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectar a MySQL: %w", err)
	}
	if err := verificarEsquema(ctx, db, cfg.DBName); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func verificarEsquema(ctx context.Context, db *sql.DB, esquema string) error {
	rows, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME, TABLE_TYPE
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?`, esquema)
	if err != nil {
		return fmt.Errorf("verificar esquema: %w", err)
	}
	defer rows.Close()
	tablas := make(map[string]bool)
	vistas := make(map[string]bool)
	for rows.Next() {
		var nombre, tipo string
		if err := rows.Scan(&nombre, &tipo); err != nil {
			return fmt.Errorf("leer esquema: %w", err)
		}
		if tipo == "BASE TABLE" {
			tablas[nombre] = true
		} else if tipo == "VIEW" {
			vistas[nombre] = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("leer esquema: %w", err)
	}
	var faltantes []string
	for _, nombre := range tablasObligatorias {
		if !tablas[nombre] {
			faltantes = append(faltantes, "tabla "+nombre)
		}
	}
	for _, nombre := range vistasObligatorias {
		if !vistas[nombre] {
			faltantes = append(faltantes, "vista "+nombre)
		}
	}
	if len(faltantes) > 0 {
		sort.Strings(faltantes)
		return fmt.Errorf("esquema incompleto en %q; faltan: %s", esquema, strings.Join(faltantes, ", "))
	}
	return nil
}
