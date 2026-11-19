package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"
)

type entradaLineaEntrega struct {
	MaterialID        int64   `json:"materialId"`
	CantidadEntregada float64 `json:"cantidadEntregada"`
}

type entradaEntrega struct {
	FechaEntrega  string                `json:"fechaEntrega"`
	ChoferID      *int64                `json:"choferId"`
	VehiculoID    *int64                `json:"vehiculoId"`
	Observaciones *string               `json:"observaciones"`
	Materiales    []entradaLineaEntrega `json:"materiales"`
}

type lineaEntregaResuelta struct {
	SolicitudMaterialID int64
	MaterialID          int64
	CantidadEntregada   float64
}

func cantidadValida(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 999999999.999 {
		return false
	}
	return math.Abs(v*1000-math.Round(v*1000)) < 0.0000001
}

func validarEntrega(e *entradaEntrega) error {
	e.FechaEntrega = recortar(e.FechaEntrega)
	e.Observaciones = limpiarPuntero(e.Observaciones)
	if !fechaValida(e.FechaEntrega) {
		return &validacionError{"fechaEntrega debe usar el formato AAAA-MM-DD"}
	}
	if e.ChoferID != nil && *e.ChoferID <= 0 {
		return &validacionError{"choferId no es valido"}
	}
	if e.VehiculoID != nil && *e.VehiculoID <= 0 {
		return &validacionError{"vehiculoId no es valido"}
	}
	if len(e.Materiales) == 0 {
		return &validacionError{"debe indicar al menos un material entregado"}
	}
	vistos := make(map[int64]bool, len(e.Materiales))
	for _, linea := range e.Materiales {
		if linea.MaterialID <= 0 {
			return &validacionError{"materialId no es valido"}
		}
		if vistos[linea.MaterialID] {
			return &validacionError{"un material no puede repetirse en la entrega"}
		}
		vistos[linea.MaterialID] = true
		if !cantidadValida(linea.CantidadEntregada) {
			return &validacionError{"cantidadEntregada debe ser positiva y tener hasta 3 decimales"}
		}
	}
	return nil
}

func (s *Servidor) registrarEntrega(w http.ResponseWriter, r *http.Request) {
	solicitudID, ok := idRuta(w, r)
	if !ok {
		return
	}
	var e entradaEntrega
	if !leerJSON(w, r, &e) {
		return
	}
	if err := validarEntrega(&e); err != nil {
		escribirValidacion(w, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var estadoSolicitud string
	err = tx.QueryRowContext(r.Context(), `SELECT estado FROM solicitudes WHERE id = ? FOR UPDATE`, solicitudID).Scan(&estadoSolicitud)
	if err == nil && estadoSolicitud != "activa" {
		err = &errorNegocio{estado: http.StatusConflict, codigo: "solicitud_anulada", mensaje: "No se pueden registrar entregas en una solicitud anulada"}
	}
	if err == nil {
		err = validarCatalogoActivo(r.Context(), tx, "choferes", e.ChoferID, "chofer_inactivo", "El chofer no existe o esta inactivo")
	}
	if err == nil {
		err = validarCatalogoActivo(r.Context(), tx, "vehiculos", e.VehiculoID, "vehiculo_inactivo", "El vehiculo no existe o esta inactivo")
	}
	lineas := make([]lineaEntregaResuelta, 0, len(e.Materiales))
	if err == nil {
		for _, linea := range e.Materiales {
			var solicitudMaterialID int64
			var comprada float64
			err = tx.QueryRowContext(r.Context(), `
				SELECT id, cantidad_comprada
				FROM solicitudes_materiales
				WHERE solicitud_id = ? AND material_id = ? FOR UPDATE`, solicitudID, linea.MaterialID).
				Scan(&solicitudMaterialID, &comprada)
			if errors.Is(err, sql.ErrNoRows) {
				err = &errorNegocio{estado: http.StatusUnprocessableEntity, codigo: "material_ajeno", mensaje: "Un material no pertenece a la solicitud"}
			}
			if err != nil {
				break
			}
			var entregada float64
			err = tx.QueryRowContext(r.Context(), `
				SELECT COALESCE(SUM(em.cantidad_entregada), 0)
				FROM entregas_materiales em
				INNER JOIN entregas en ON en.id = em.entrega_id
				WHERE em.solicitud_material_id = ? AND en.estado = 'confirmada'`, solicitudMaterialID).Scan(&entregada)
			if err != nil {
				break
			}
			pendiente := comprada - entregada
			if linea.CantidadEntregada-pendiente > 0.0000001 {
				err = &errorNegocio{
					estado: http.StatusConflict, codigo: "sobreentrega",
					mensaje: "La cantidad indicada supera el saldo pendiente",
					detalles: map[string]any{
						"materialId": linea.MaterialID, "cantidadComprada": comprada,
						"cantidadYaEntregada": entregada, "cantidadPendiente": math.Max(0, pendiente),
						"cantidadSolicitada": linea.CantidadEntregada,
					},
				}
				break
			}
			lineas = append(lineas, lineaEntregaResuelta{SolicitudMaterialID: solicitudMaterialID, MaterialID: linea.MaterialID, CantidadEntregada: linea.CantidadEntregada})
		}
	}
	var entregaID int64
	if err == nil {
		var resultado sql.Result
		resultado, err = tx.ExecContext(r.Context(), `
			INSERT INTO entregas (solicitud_id, fecha_entrega, chofer_id, vehiculo_id, observaciones)
			VALUES (?, ?, ?, ?, ?)`, solicitudID, e.FechaEntrega, valorEnteroOpcional(e.ChoferID),
			valorEnteroOpcional(e.VehiculoID), textoSinLimite(e.Observaciones))
		if err == nil {
			entregaID, err = resultado.LastInsertId()
		}
	}
	if err == nil {
		var stmt *sql.Stmt
		stmt, err = tx.PrepareContext(r.Context(), `INSERT INTO entregas_materiales (entrega_id, solicitud_material_id, cantidad_entregada) VALUES (?, ?, ?)`)
		if err == nil {
			for _, linea := range lineas {
				if _, err = stmt.ExecContext(r.Context(), entregaID, linea.SolicitudMaterialID, linea.CantidadEntregada); err != nil {
					break
				}
			}
			_ = stmt.Close()
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = tx.Rollback()
		var negocio *errorNegocio
		if errors.As(err, &negocio) {
			escribirError(w, negocio.estado, negocio.codigo, negocio.mensaje, negocio.detalles)
			return
		}
		errorBaseDatos(w, s.logger, err)
		return
	}
	entrega, err := s.cargarEntrega(r.Context(), entregaID)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusCreated, entrega)
}

func validarCatalogoActivo(ctx context.Context, tx *sql.Tx, tabla string, id *int64, codigo, mensaje string) error {
	if id == nil {
		return nil
	}
	consulta := `SELECT id FROM ` + tabla + ` WHERE id = ? AND activo = 1`
	var encontrado int64
	if err := tx.QueryRowContext(ctx, consulta, *id).Scan(&encontrado); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &errorNegocio{estado: http.StatusUnprocessableEntity, codigo: codigo, mensaje: mensaje}
		}
		return err
	}
	return nil
}

func (s *Servidor) cargarEntrega(ctx context.Context, id int64) (Entrega, error) {
	var entrega Entrega
	var fecha time.Time
	var choferID, vehiculoID sql.NullInt64
	var chofer, patente, observaciones sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.solicitud_id, e.fecha_entrega, e.chofer_id, c.nombre,
		       e.vehiculo_id, v.patente, e.estado, e.observaciones,
		       e.creado_en, e.actualizado_en
		FROM entregas e
		LEFT JOIN choferes c ON c.id = e.chofer_id
		LEFT JOIN vehiculos v ON v.id = e.vehiculo_id
		WHERE e.id = ?`, id).Scan(&entrega.ID, &entrega.SolicitudID, &fecha, &choferID, &chofer,
		&vehiculoID, &patente, &entrega.Estado, &observaciones, &entrega.CreadoEn, &entrega.ActualizadoEn)
	if err != nil {
		return entrega, err
	}
	entrega.FechaEntrega = fecha.Format("2006-01-02")
	entrega.ChoferID = punteroEntero(choferID)
	entrega.NombreChofer = punteroTexto(chofer)
	entrega.VehiculoID = punteroEntero(vehiculoID)
	entrega.PatenteVehiculo = punteroTexto(patente)
	entrega.Observaciones = punteroTexto(observaciones)
	entrega.MotivoAnulacion = extraerMotivoAnulacion(entrega.Observaciones)
	entrega.Materiales, err = s.cargarLineasEntrega(ctx, id)
	return entrega, err
}

func (s *Servidor) anularEntrega(w http.ResponseWriter, r *http.Request) {
	entregaID, ok := idRuta(w, r)
	if !ok {
		return
	}
	var entrada entradaAnulacion
	if !leerJSON(w, r, &entrada) {
		return
	}
	entrada.Motivo = strings.TrimSpace(entrada.Motivo)
	if len([]rune(entrada.Motivo)) < 3 || len([]rune(entrada.Motivo)) > 500 {
		escribirValidacion(w, &validacionError{"motivo debe tener entre 3 y 500 caracteres"})
		return
	}
	var solicitudID int64
	if err := s.db.QueryRowContext(r.Context(), `SELECT solicitud_id FROM entregas WHERE id = ?`, entregaID).Scan(&solicitudID); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var ignorar string
	if err = tx.QueryRowContext(r.Context(), `SELECT estado FROM solicitudes WHERE id = ? FOR UPDATE`, solicitudID).Scan(&ignorar); err == nil {
		var estado string
		err = tx.QueryRowContext(r.Context(), `SELECT estado FROM entregas WHERE id = ? AND solicitud_id = ? FOR UPDATE`, entregaID, solicitudID).Scan(&estado)
		if err == nil && estado == "confirmada" {
			_, err = tx.ExecContext(r.Context(), `
				UPDATE entregas
				SET estado = 'anulada',
				    observaciones = CONCAT_WS(CHAR(10), NULLIF(TRIM(observaciones), ''), CONCAT('[ANULACION] ', ?))
				WHERE id = ?`, entrada.Motivo, entregaID)
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = tx.Rollback()
		errorBaseDatos(w, s.logger, err)
		return
	}
	entrega, err := s.cargarEntrega(r.Context(), entregaID)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, entrega)
}

func extraerMotivoAnulacion(observaciones *string) *string {
	if observaciones == nil {
		return nil
	}
	lineas := strings.Split(*observaciones, "\n")
	for i := len(lineas) - 1; i >= 0; i-- {
		if motivo := strings.TrimSpace(strings.TrimPrefix(lineas[i], "[ANULACION] ")); motivo != strings.TrimSpace(lineas[i]) && motivo != "" {
			return &motivo
		}
	}
	return nil
}

type errorNegocio struct {
	estado   int
	codigo   string
	mensaje  string
	detalles map[string]any
}

func (e *errorNegocio) Error() string { return e.mensaje }
