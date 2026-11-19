package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ResumenSolicitud struct {
	ID                     int64     `json:"id"`
	TipoReferencia         string    `json:"tipoReferencia"`
	CodigoReferencia       string    `json:"codigoReferencia"`
	Anio                   int       `json:"anio"`
	SolicitanteID          int64     `json:"solicitanteId"`
	NombreSolicitante      string    `json:"nombreSolicitante"`
	FechaCompra            string    `json:"fechaCompra"`
	NumeroMemo             *string   `json:"numeroMemo"`
	FechaAsignacion        *string   `json:"fechaAsignacion"`
	NumeroCEA              *string   `json:"numeroCea"`
	EstadoRegistro         string    `json:"estadoRegistro"`
	CantidadTotalComprada  float64   `json:"cantidadTotalComprada"`
	CantidadTotalEntregada float64   `json:"cantidadTotalEntregada"`
	CantidadTotalPendiente float64   `json:"cantidadTotalPendiente"`
	EstadoCumplimiento     string    `json:"estadoCumplimiento"`
	PorcentajeEntregado    float64   `json:"porcentajeEntregado"`
	ValorTotalCompradoCLP  *int64    `json:"valorTotalCompradoClp"`
	ValorTotalEntregadoCLP *int64    `json:"valorTotalEntregadoClp"`
	ValorTotalPendienteCLP *int64    `json:"valorTotalPendienteClp"`
	Observaciones          *string   `json:"observaciones"`
	CreadoEn               time.Time `json:"creadoEn"`
	ActualizadoEn          time.Time `json:"actualizadoEn"`
}

type LineaSolicitud struct {
	ID                        int64   `json:"id"`
	MaterialID                int64   `json:"materialId"`
	CodigoMaterial            string  `json:"codigoMaterial"`
	NombreMaterial            string  `json:"nombreMaterial"`
	UnidadMedida              string  `json:"unidadMedida"`
	CantidadComprada          float64 `json:"cantidadComprada"`
	CantidadEntregada         float64 `json:"cantidadEntregada"`
	CantidadPendiente         float64 `json:"cantidadPendiente"`
	EstadoCumplimiento        string  `json:"estadoCumplimiento"`
	PrecioUnitarioAplicadoCLP *int64  `json:"precioUnitarioClp"`
	ValorCompradoCLP          *int64  `json:"valorCompradoClp"`
	ValorEntregadoCLP         *int64  `json:"valorEntregadoClp"`
	ValorPendienteCLP         *int64  `json:"valorPendienteClp"`
}

type LineaEntrega struct {
	ID                  int64   `json:"id"`
	SolicitudMaterialID int64   `json:"solicitudMaterialId"`
	MaterialID          int64   `json:"materialId"`
	NombreMaterial      string  `json:"nombreMaterial"`
	UnidadMedida        string  `json:"unidadMedida"`
	CantidadEntregada   float64 `json:"cantidadEntregada"`
}

type Entrega struct {
	ID              int64          `json:"id"`
	SolicitudID     int64          `json:"solicitudId"`
	FechaEntrega    string         `json:"fechaEntrega"`
	ChoferID        *int64         `json:"choferId"`
	NombreChofer    *string        `json:"nombreChofer"`
	VehiculoID      *int64         `json:"vehiculoId"`
	PatenteVehiculo *string        `json:"patenteVehiculo"`
	Estado          string         `json:"estado"`
	Observaciones   *string        `json:"observaciones"`
	MotivoAnulacion *string        `json:"motivoAnulacion,omitempty"`
	CreadoEn        time.Time      `json:"creadoEn"`
	ActualizadoEn   time.Time      `json:"actualizadoEn"`
	Materiales      []LineaEntrega `json:"materiales"`
}

type DetalleSolicitud struct {
	ResumenSolicitud
	Materiales []LineaSolicitud `json:"materiales"`
	Entregas   []Entrega        `json:"entregas"`
}

const columnasResumenSolicitud = `
	solicitud_id, tipo_referencia, codigo_referencia, anio,
	solicitante_id, nombre_solicitante, fecha_compra, numero_memo,
	fecha_asignacion, numero_cea, estado_registro,
	cantidad_total_comprada,
	cantidad_total_entregada,
	cantidad_total_pendiente,
	estado_cumplimiento, porcentaje_entregado,
	valor_total_comprado_clp, valor_total_entregado_clp,
	valor_total_pendiente_clp, observaciones, creado_en, actualizado_en`

type escaner interface{ Scan(...any) error }

func escanearResumen(fila escaner) (ResumenSolicitud, error) {
	var x ResumenSolicitud
	var fechaCompra time.Time
	var memo, cea, observaciones sql.NullString
	var fechaAsignacion sql.NullTime
	var comprado, entregado, pendiente sql.NullInt64
	err := fila.Scan(
		&x.ID, &x.TipoReferencia, &x.CodigoReferencia, &x.Anio,
		&x.SolicitanteID, &x.NombreSolicitante, &fechaCompra, &memo,
		&fechaAsignacion, &cea, &x.EstadoRegistro,
		&x.CantidadTotalComprada, &x.CantidadTotalEntregada, &x.CantidadTotalPendiente,
		&x.EstadoCumplimiento, &x.PorcentajeEntregado,
		&comprado, &entregado, &pendiente, &observaciones, &x.CreadoEn, &x.ActualizadoEn,
	)
	if err != nil {
		return x, err
	}
	x.FechaCompra = fechaCompra.Format("2006-01-02")
	x.NumeroMemo = punteroTexto(memo)
	if fechaAsignacion.Valid {
		f := fechaAsignacion.Time.Format("2006-01-02")
		x.FechaAsignacion = &f
	}
	x.NumeroCEA = punteroTexto(cea)
	x.ValorTotalCompradoCLP = punteroEntero(comprado)
	x.ValorTotalEntregadoCLP = punteroEntero(entregado)
	x.ValorTotalPendienteCLP = punteroEntero(pendiente)
	x.Observaciones = punteroTexto(observaciones)
	return x, nil
}

func (s *Servidor) listarSolicitudes(w http.ResponseWriter, r *http.Request) {
	consulta := `SELECT ` + columnasResumenSolicitud + ` FROM vista_resumen_solicitudes WHERE 1 = 1`
	args := make([]any, 0, 4)
	busqueda := strings.TrimSpace(r.URL.Query().Get("busqueda"))
	if busqueda == "" {
		busqueda = strings.TrimSpace(r.URL.Query().Get("buscar"))
	}
	if buscar := busqueda; buscar != "" {
		consulta += ` AND (codigo_referencia LIKE ? OR nombre_solicitante LIKE ? OR numero_memo LIKE ? OR numero_cea LIKE ?)`
		patron := "%" + buscar + "%"
		args = append(args, patron, patron, patron, patron)
	}
	if anioTexto := strings.TrimSpace(r.URL.Query().Get("anio")); anioTexto != "" {
		anio, err := strconv.Atoi(anioTexto)
		if err != nil || anio < 2000 || anio > 2200 {
			escribirError(w, http.StatusBadRequest, "filtro_invalido", "El filtro anio no es valido", nil)
			return
		}
		consulta += ` AND anio = ?`
		args = append(args, anio)
	}
	if solicitanteTexto := strings.TrimSpace(r.URL.Query().Get("solicitanteId")); solicitanteTexto != "" {
		solicitanteID, err := strconv.ParseInt(solicitanteTexto, 10, 64)
		if err != nil || solicitanteID <= 0 {
			escribirError(w, http.StatusBadRequest, "filtro_invalido", "El filtro solicitanteId no es valido", nil)
			return
		}
		consulta += ` AND solicitante_id = ?`
		args = append(args, solicitanteID)
	}
	if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
		if estado != "sin_entrega" && estado != "parcial" && estado != "completa" && estado != "sobreentrega" && estado != "anulada" {
			escribirError(w, http.StatusBadRequest, "filtro_invalido", "El filtro estado no es valido", nil)
			return
		}
		consulta += ` AND estado_cumplimiento = ?`
		args = append(args, estado)
	}
	consulta += ` ORDER BY fecha_compra DESC, solicitud_id DESC LIMIT 1000`
	rows, err := s.db.QueryContext(r.Context(), consulta, args...)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	defer rows.Close()
	items := make([]ResumenSolicitud, 0)
	for rows.Next() {
		x, err := escanearResumen(rows)
		if err != nil {
			errorBaseDatos(w, s.logger, err)
			return
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, items)
}

func (s *Servidor) obtenerSolicitud(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	x, err := s.cargarDetalleSolicitud(r.Context(), id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, x)
}

func (s *Servidor) cargarDetalleSolicitud(ctx context.Context, id int64) (DetalleSolicitud, error) {
	resumen, err := escanearResumen(s.db.QueryRowContext(ctx, `SELECT `+columnasResumenSolicitud+` FROM vista_resumen_solicitudes WHERE solicitud_id = ?`, id))
	if err != nil {
		return DetalleSolicitud{}, err
	}
	detalle := DetalleSolicitud{ResumenSolicitud: resumen, Materiales: make([]LineaSolicitud, 0), Entregas: make([]Entrega, 0)}
	rows, err := s.db.QueryContext(ctx, `
		SELECT solicitud_material_id, material_id, codigo_material, nombre_material,
		       unidad_medida, cantidad_comprada,
		       cantidad_entregada, cantidad_pendiente,
		       estado_cumplimiento, precio_unitario_aplicado_clp,
		       valor_comprado_clp, valor_entregado_clp, valor_pendiente_clp
		FROM vista_saldos_solicitud_material
		WHERE solicitud_id = ? ORDER BY nombre_material, solicitud_material_id`, id)
	if err != nil {
		return DetalleSolicitud{}, err
	}
	for rows.Next() {
		var linea LineaSolicitud
		var precio, valorComprado, valorEntregado, valorPendiente sql.NullInt64
		if err := rows.Scan(&linea.ID, &linea.MaterialID, &linea.CodigoMaterial, &linea.NombreMaterial,
			&linea.UnidadMedida, &linea.CantidadComprada, &linea.CantidadEntregada,
			&linea.CantidadPendiente, &linea.EstadoCumplimiento, &precio,
			&valorComprado, &valorEntregado, &valorPendiente); err != nil {
			rows.Close()
			return DetalleSolicitud{}, err
		}
		linea.PrecioUnitarioAplicadoCLP = punteroEntero(precio)
		linea.ValorCompradoCLP = punteroEntero(valorComprado)
		linea.ValorEntregadoCLP = punteroEntero(valorEntregado)
		linea.ValorPendienteCLP = punteroEntero(valorPendiente)
		detalle.Materiales = append(detalle.Materiales, linea)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DetalleSolicitud{}, err
	}
	if err := rows.Close(); err != nil {
		return DetalleSolicitud{}, err
	}

	filasEntregas, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.solicitud_id, e.fecha_entrega, e.chofer_id, c.nombre,
		       e.vehiculo_id, v.patente, e.estado, e.observaciones,
		       e.creado_en, e.actualizado_en
		FROM entregas e
		LEFT JOIN choferes c ON c.id = e.chofer_id
		LEFT JOIN vehiculos v ON v.id = e.vehiculo_id
		WHERE e.solicitud_id = ?
		ORDER BY e.fecha_entrega DESC, e.id DESC`, id)
	if err != nil {
		return DetalleSolicitud{}, err
	}
	for filasEntregas.Next() {
		var entrega Entrega
		var fecha time.Time
		var choferID, vehiculoID sql.NullInt64
		var chofer, patente, observaciones sql.NullString
		if err := filasEntregas.Scan(&entrega.ID, &entrega.SolicitudID, &fecha, &choferID, &chofer,
			&vehiculoID, &patente, &entrega.Estado, &observaciones,
			&entrega.CreadoEn, &entrega.ActualizadoEn); err != nil {
			filasEntregas.Close()
			return DetalleSolicitud{}, err
		}
		entrega.FechaEntrega = fecha.Format("2006-01-02")
		entrega.ChoferID = punteroEntero(choferID)
		entrega.NombreChofer = punteroTexto(chofer)
		entrega.VehiculoID = punteroEntero(vehiculoID)
		entrega.PatenteVehiculo = punteroTexto(patente)
		entrega.Observaciones = punteroTexto(observaciones)
		entrega.MotivoAnulacion = extraerMotivoAnulacion(entrega.Observaciones)
		entrega.Materiales = make([]LineaEntrega, 0)
		detalle.Entregas = append(detalle.Entregas, entrega)
	}
	if err := filasEntregas.Err(); err != nil {
		filasEntregas.Close()
		return DetalleSolicitud{}, err
	}
	if err := filasEntregas.Close(); err != nil {
		return DetalleSolicitud{}, err
	}
	for i := range detalle.Entregas {
		detalle.Entregas[i].Materiales, err = s.cargarLineasEntrega(ctx, detalle.Entregas[i].ID)
		if err != nil {
			return DetalleSolicitud{}, err
		}
	}
	return detalle, nil
}

func (s *Servidor) cargarLineasEntrega(ctx context.Context, entregaID int64) ([]LineaEntrega, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT em.id, em.solicitud_material_id, sm.material_id, m.nombre,
		       m.unidad_medida, em.cantidad_entregada
		FROM entregas_materiales em
		INNER JOIN solicitudes_materiales sm ON sm.id = em.solicitud_material_id
		INNER JOIN materiales m ON m.id = sm.material_id
		WHERE em.entrega_id = ? ORDER BY m.nombre, em.id`, entregaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lineas := make([]LineaEntrega, 0)
	for rows.Next() {
		var x LineaEntrega
		if err := rows.Scan(&x.ID, &x.SolicitudMaterialID, &x.MaterialID, &x.NombreMaterial, &x.UnidadMedida, &x.CantidadEntregada); err != nil {
			return nil, err
		}
		lineas = append(lineas, x)
	}
	return lineas, rows.Err()
}

type entradaLineaSolicitud struct {
	MaterialID                int64   `json:"materialId"`
	CantidadComprada          float64 `json:"cantidadComprada"`
	PrecioUnitarioAplicadoCLP *int64  `json:"precioUnitarioClp"`
}

type entradaSolicitud struct {
	TipoReferencia   string                  `json:"tipoReferencia"`
	CodigoReferencia string                  `json:"codigoReferencia"`
	Anio             int                     `json:"anio"`
	SolicitanteID    int64                   `json:"solicitanteId"`
	FechaCompra      string                  `json:"fechaCompra"`
	NumeroMemo       *string                 `json:"numeroMemo"`
	FechaAsignacion  *string                 `json:"fechaAsignacion"`
	NumeroCEA        *string                 `json:"numeroCea"`
	Observaciones    *string                 `json:"observaciones"`
	Materiales       []entradaLineaSolicitud `json:"materiales"`
}

type cambiosSolicitud struct {
	TipoReferencia   *string                  `json:"tipoReferencia"`
	CodigoReferencia *string                  `json:"codigoReferencia"`
	Anio             *int                     `json:"anio"`
	SolicitanteID    *int64                   `json:"solicitanteId"`
	FechaCompra      *string                  `json:"fechaCompra"`
	NumeroMemo       opcionalNulo[string]     `json:"numeroMemo"`
	FechaAsignacion  opcionalNulo[string]     `json:"fechaAsignacion"`
	NumeroCEA        opcionalNulo[string]     `json:"numeroCea"`
	Observaciones    opcionalNulo[string]     `json:"observaciones"`
	Materiales       *[]entradaLineaSolicitud `json:"materiales"`
}

func validarSolicitud(e entradaSolicitud) error {
	e.TipoReferencia = recortar(e.TipoReferencia)
	if e.TipoReferencia != "cim" && e.TipoReferencia != "centro_costo" {
		return &validacionError{"tipoReferencia debe ser cim o centro_costo"}
	}
	if err := validarLargo("codigoReferencia", e.CodigoReferencia, 1, 64); err != nil {
		return err
	}
	if e.SolicitanteID <= 0 {
		return &validacionError{"solicitanteId es obligatorio"}
	}
	if !fechaValida(e.FechaCompra) {
		return &validacionError{"fechaCompra debe usar el formato AAAA-MM-DD"}
	}
	if e.Anio < 2000 || e.Anio > 2200 {
		return &validacionError{"el anio de fechaCompra debe estar entre 2000 y 2200"}
	}
	if e.FechaAsignacion != nil && !fechaValida(*e.FechaAsignacion) {
		return &validacionError{"fechaAsignacion debe usar el formato AAAA-MM-DD"}
	}
	if err := validarTextoOpcional("numeroMemo", e.NumeroMemo, 64); err != nil {
		return err
	}
	if err := validarTextoOpcional("numeroCea", e.NumeroCEA, 64); err != nil {
		return err
	}
	if len(e.Materiales) == 0 {
		return &validacionError{"debe indicar al menos un material"}
	}
	vistos := make(map[int64]bool, len(e.Materiales))
	for _, linea := range e.Materiales {
		if linea.MaterialID <= 0 {
			return &validacionError{"cada materialId debe ser valido"}
		}
		if vistos[linea.MaterialID] {
			return &validacionError{"un material no puede repetirse en la solicitud"}
		}
		vistos[linea.MaterialID] = true
		if !cantidadValida(linea.CantidadComprada) {
			return &validacionError{"cantidadComprada debe ser positiva y tener hasta 3 decimales"}
		}
		if linea.PrecioUnitarioAplicadoCLP != nil && *linea.PrecioUnitarioAplicadoCLP < 0 {
			return &validacionError{"precioUnitarioClp no puede ser negativo"}
		}
	}
	return nil
}

func validarTextoOpcional(nombre string, valor *string, maximo int) error {
	if valor != nil && len([]rune(strings.TrimSpace(*valor))) > maximo {
		return &validacionError{fmt.Sprintf("%s no puede superar %d caracteres", nombre, maximo)}
	}
	return nil
}

func (s *Servidor) crearSolicitud(w http.ResponseWriter, r *http.Request) {
	var e entradaSolicitud
	if !leerJSON(w, r, &e) {
		return
	}
	normalizarSolicitud(&e)
	if err := validarSolicitud(e); err != nil {
		escribirValidacion(w, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	resultado, err := tx.ExecContext(r.Context(), `
		INSERT INTO solicitudes
		(tipo_referencia, codigo_referencia, anio, solicitante_id, fecha_compra,
		 numero_memo, fecha_asignacion, numero_cea, observaciones)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.TipoReferencia, e.CodigoReferencia, e.Anio, e.SolicitanteID, e.FechaCompra,
		textoOpcional(e.NumeroMemo, 64), fechaOpcional(e.FechaAsignacion),
		textoOpcional(e.NumeroCEA, 64), textoSinLimite(e.Observaciones))
	if err == nil {
		var id int64
		id, err = resultado.LastInsertId()
		if err == nil {
			err = insertarLineasSolicitud(r.Context(), tx, id, e.Materiales)
		}
		if err == nil {
			err = tx.Commit()
		}
		if err == nil {
			detalle, cargarErr := s.cargarDetalleSolicitud(r.Context(), id)
			if cargarErr != nil {
				errorBaseDatos(w, s.logger, cargarErr)
				return
			}
			escribirJSON(w, http.StatusCreated, detalle)
			return
		}
	}
	_ = tx.Rollback()
	errorBaseDatos(w, s.logger, err)
}

func insertarLineasSolicitud(ctx context.Context, tx *sql.Tx, solicitudID int64, lineas []entradaLineaSolicitud) error {
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO solicitudes_materiales (solicitud_id, material_id, cantidad_comprada, precio_unitario_aplicado_clp) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, linea := range lineas {
		if _, err := stmt.ExecContext(ctx, solicitudID, linea.MaterialID, linea.CantidadComprada, valorEnteroOpcional(linea.PrecioUnitarioAplicadoCLP)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Servidor) editarSolicitud(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	var cambios cambiosSolicitud
	if !leerJSON(w, r, &cambios) {
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	actual, estado, err := cargarSolicitudParaEditar(r.Context(), tx, id)
	if err != nil {
		_ = tx.Rollback()
		errorBaseDatos(w, s.logger, err)
		return
	}
	if estado == "anulada" {
		_ = tx.Rollback()
		escribirError(w, http.StatusConflict, "solicitud_anulada", "Una solicitud anulada no se puede editar", nil)
		return
	}
	lineasOriginales := append([]entradaLineaSolicitud(nil), actual.Materiales...)
	aplicarCambiosSolicitud(&actual, cambios)
	reemplazarLineas := false
	if cambios.Materiales != nil {
		actual.Materiales = *cambios.Materiales
		reemplazarLineas = !lineasSolicitudIguales(lineasOriginales, actual.Materiales)
	}
	normalizarSolicitud(&actual)
	if err = validarSolicitud(actual); err != nil {
		_ = tx.Rollback()
		escribirValidacion(w, err)
		return
	}
	_, err = tx.ExecContext(r.Context(), `
		UPDATE solicitudes SET tipo_referencia = ?, codigo_referencia = ?, anio = ?,
		solicitante_id = ?, fecha_compra = ?, numero_memo = ?, fecha_asignacion = ?,
		numero_cea = ?, observaciones = ? WHERE id = ?`,
		actual.TipoReferencia, actual.CodigoReferencia, actual.Anio, actual.SolicitanteID,
		actual.FechaCompra, textoOpcional(actual.NumeroMemo, 64), fechaOpcional(actual.FechaAsignacion),
		textoOpcional(actual.NumeroCEA, 64), textoSinLimite(actual.Observaciones), id)
	if err == nil && reemplazarLineas {
		var entregas int
		err = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM entregas WHERE solicitud_id = ?`, id).Scan(&entregas)
		if err == nil && entregas > 0 {
			err = &conflictoError{"materiales_con_entregas", "No se pueden reemplazar materiales porque la solicitud tiene entregas registradas"}
		}
		if err == nil {
			_, err = tx.ExecContext(r.Context(), `DELETE FROM solicitudes_materiales WHERE solicitud_id = ?`, id)
		}
		if err == nil {
			err = insertarLineasSolicitud(r.Context(), tx, id, actual.Materiales)
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = tx.Rollback()
		var conflicto *conflictoError
		if errors.As(err, &conflicto) {
			escribirError(w, http.StatusConflict, conflicto.codigo, conflicto.mensaje, nil)
			return
		}
		errorBaseDatos(w, s.logger, err)
		return
	}
	detalle, err := s.cargarDetalleSolicitud(r.Context(), id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, detalle)
}

func cargarSolicitudParaEditar(ctx context.Context, tx *sql.Tx, id int64) (entradaSolicitud, string, error) {
	var e entradaSolicitud
	var estado string
	var memo, cea, observaciones sql.NullString
	var fechaCompra time.Time
	var fechaAsignacion sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT tipo_referencia, codigo_referencia, anio, solicitante_id, fecha_compra,
		numero_memo, fecha_asignacion, numero_cea, observaciones, estado
		FROM solicitudes WHERE id = ? FOR UPDATE`, id).
		Scan(&e.TipoReferencia, &e.CodigoReferencia, &e.Anio, &e.SolicitanteID, &fechaCompra,
			&memo, &fechaAsignacion, &cea, &observaciones, &estado)
	if err != nil {
		return e, estado, err
	}
	e.FechaCompra = fechaCompra.Format("2006-01-02")
	e.NumeroMemo = punteroTexto(memo)
	if fechaAsignacion.Valid {
		fecha := fechaAsignacion.Time.Format("2006-01-02")
		e.FechaAsignacion = &fecha
	}
	e.NumeroCEA = punteroTexto(cea)
	e.Observaciones = punteroTexto(observaciones)
	rows, err := tx.QueryContext(ctx, `SELECT material_id, cantidad_comprada, precio_unitario_aplicado_clp FROM solicitudes_materiales WHERE solicitud_id = ? ORDER BY id FOR UPDATE`, id)
	if err != nil {
		return e, estado, err
	}
	defer rows.Close()
	for rows.Next() {
		var linea entradaLineaSolicitud
		var precio sql.NullInt64
		if err := rows.Scan(&linea.MaterialID, &linea.CantidadComprada, &precio); err != nil {
			return e, estado, err
		}
		linea.PrecioUnitarioAplicadoCLP = punteroEntero(precio)
		e.Materiales = append(e.Materiales, linea)
	}
	return e, estado, rows.Err()
}

func aplicarCambiosSolicitud(e *entradaSolicitud, c cambiosSolicitud) {
	if c.TipoReferencia != nil {
		e.TipoReferencia = *c.TipoReferencia
	}
	if c.CodigoReferencia != nil {
		e.CodigoReferencia = *c.CodigoReferencia
	}
	if c.Anio != nil {
		e.Anio = *c.Anio
	}
	if c.SolicitanteID != nil {
		e.SolicitanteID = *c.SolicitanteID
	}
	if c.FechaCompra != nil {
		e.FechaCompra = *c.FechaCompra
	}
	if c.NumeroMemo.Presente {
		e.NumeroMemo = c.NumeroMemo.Valor
	}
	if c.FechaAsignacion.Presente {
		e.FechaAsignacion = c.FechaAsignacion.Valor
	}
	if c.NumeroCEA.Presente {
		e.NumeroCEA = c.NumeroCEA.Valor
	}
	if c.Observaciones.Presente {
		e.Observaciones = c.Observaciones.Valor
	}
}

func lineasSolicitudIguales(a, b []entradaLineaSolicitud) bool {
	if len(a) != len(b) {
		return false
	}
	porMaterial := make(map[int64]entradaLineaSolicitud, len(a))
	for _, linea := range a {
		porMaterial[linea.MaterialID] = linea
	}
	for _, linea := range b {
		anterior, ok := porMaterial[linea.MaterialID]
		if !ok || math.Abs(anterior.CantidadComprada-linea.CantidadComprada) > 0.0000001 || !enterosOpcionalesIguales(anterior.PrecioUnitarioAplicadoCLP, linea.PrecioUnitarioAplicadoCLP) {
			return false
		}
	}
	return true
}

func enterosOpcionalesIguales(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func normalizarSolicitud(e *entradaSolicitud) {
	e.TipoReferencia = recortar(e.TipoReferencia)
	e.CodigoReferencia = recortar(e.CodigoReferencia)
	e.FechaCompra = recortar(e.FechaCompra)
	e.NumeroMemo = limpiarPuntero(e.NumeroMemo)
	e.FechaAsignacion = limpiarPuntero(e.FechaAsignacion)
	e.NumeroCEA = limpiarPuntero(e.NumeroCEA)
	e.Observaciones = limpiarPuntero(e.Observaciones)
	if fecha, err := time.Parse("2006-01-02", e.FechaCompra); err == nil {
		e.Anio = fecha.Year()
	}
}

func (s *Servidor) anularSolicitud(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
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
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var estado string
	err = tx.QueryRowContext(r.Context(), `SELECT estado FROM solicitudes WHERE id = ? FOR UPDATE`, id).Scan(&estado)
	if err == nil && estado != "anulada" {
		_, err = tx.ExecContext(r.Context(), `
			UPDATE solicitudes
			SET estado = 'anulada',
			    observaciones = CONCAT_WS(CHAR(10), NULLIF(TRIM(observaciones), ''), CONCAT('[ANULACION] ', ?))
			WHERE id = ?`, entrada.Motivo, id)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = tx.Rollback()
		errorBaseDatos(w, s.logger, err)
		return
	}
	detalle, err := s.cargarDetalleSolicitud(r.Context(), id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, detalle)
}

type entradaAnulacion struct {
	Motivo string `json:"motivo"`
}

func fechaOpcional(v *string) any {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return strings.TrimSpace(*v)
}

func textoSinLimite(v *string) any {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return strings.TrimSpace(*v)
}

func valorEnteroOpcional(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

type conflictoError struct {
	codigo  string
	mensaje string
}

func (e *conflictoError) Error() string { return e.mensaje }
