package httpapi

import "net/http"

type ResumenMaterial struct {
	MaterialID          int64   `json:"materialId"`
	NombreMaterial      string  `json:"nombreMaterial"`
	UnidadMedida        string  `json:"unidadMedida"`
	CantidadComprada    float64 `json:"cantidadComprada"`
	CantidadEntregada   float64 `json:"cantidadEntregada"`
	CantidadPendiente   float64 `json:"cantidadPendiente"`
	PorcentajeEntregado float64 `json:"porcentajeEntregado"`
}

type ResumenGeneral struct {
	SolicitudesActivas    int64              `json:"solicitudesActivas"`
	SolicitudesPendientes int64              `json:"solicitudesPendientes"`
	SolicitudesCompletas  int64              `json:"solicitudesCompletas"`
	EntregasMes           int64              `json:"entregasMes"`
	CantidadComprada      float64            `json:"cantidadComprada"`
	CantidadEntregada     float64            `json:"cantidadEntregada"`
	PorcentajeGeneral     float64            `json:"porcentajeGeneral"`
	SolicitudesRecientes  []ResumenSolicitud `json:"solicitudesRecientes"`
	Materiales            []ResumenMaterial  `json:"materiales"`
}

func (s *Servidor) obtenerResumen(w http.ResponseWriter, r *http.Request) {
	var x ResumenGeneral
	err := s.db.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(estado_registro = 'activa'), 0),
		       COALESCE(SUM(estado_registro = 'activa' AND estado_cumplimiento IN ('sin_entrega', 'parcial')), 0),
		       COALESCE(SUM(estado_registro = 'activa' AND estado_cumplimiento = 'completa'), 0),
		       COALESCE(SUM(CASE WHEN estado_registro = 'activa' THEN cantidad_total_comprada ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN estado_registro = 'activa' THEN cantidad_total_entregada ELSE 0 END), 0)
		FROM vista_resumen_solicitudes`).Scan(
		&x.SolicitudesActivas, &x.SolicitudesPendientes, &x.SolicitudesCompletas,
		&x.CantidadComprada, &x.CantidadEntregada,
	)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	if x.CantidadComprada > 0 {
		x.PorcentajeGeneral = x.CantidadEntregada / x.CantidadComprada * 100
	}
	if err = s.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM entregas
		WHERE estado = 'confirmada'
		  AND fecha_entrega >= DATE_FORMAT(UTC_DATE(), '%Y-%m-01')
		  AND fecha_entrega < DATE_ADD(DATE_FORMAT(UTC_DATE(), '%Y-%m-01'), INTERVAL 1 MONTH)`).Scan(&x.EntregasMes); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+columnasResumenSolicitud+` FROM vista_resumen_solicitudes ORDER BY fecha_compra DESC, solicitud_id DESC LIMIT 5`)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	x.SolicitudesRecientes = make([]ResumenSolicitud, 0)
	for rows.Next() {
		solicitud, scanErr := escanearResumen(rows)
		if scanErr != nil {
			rows.Close()
			errorBaseDatos(w, s.logger, scanErr)
			return
		}
		x.SolicitudesRecientes = append(x.SolicitudesRecientes, solicitud)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		errorBaseDatos(w, s.logger, err)
		return
	}
	if err := rows.Close(); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	filasMateriales, err := s.db.QueryContext(r.Context(), `
		SELECT material_id, nombre_material, unidad_medida,
		       SUM(cantidad_comprada),
		       SUM(cantidad_entregada),
		       SUM(cantidad_pendiente),
		       CASE WHEN SUM(cantidad_comprada) = 0 THEN 0
		            ELSE ROUND(SUM(cantidad_entregada) / SUM(cantidad_comprada) * 100, 2)
		       END
		FROM vista_saldos_solicitud_material
		WHERE estado_solicitud = 'activa'
		GROUP BY material_id, nombre_material, unidad_medida
		ORDER BY nombre_material`)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	x.Materiales = make([]ResumenMaterial, 0)
	for filasMateriales.Next() {
		var material ResumenMaterial
		if err := filasMateriales.Scan(&material.MaterialID, &material.NombreMaterial, &material.UnidadMedida,
			&material.CantidadComprada, &material.CantidadEntregada, &material.CantidadPendiente, &material.PorcentajeEntregado); err != nil {
			filasMateriales.Close()
			errorBaseDatos(w, s.logger, err)
			return
		}
		x.Materiales = append(x.Materiales, material)
	}
	if err := filasMateriales.Err(); err != nil {
		filasMateriales.Close()
		errorBaseDatos(w, s.logger, err)
		return
	}
	if err := filasMateriales.Close(); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, x)
}
