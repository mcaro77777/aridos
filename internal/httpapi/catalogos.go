package httpapi

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

type Solicitante struct {
	ID            int64     `json:"id"`
	Nombre        string    `json:"nombre"`
	Tipo          string    `json:"tipo"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"creadoEn"`
	ActualizadoEn time.Time `json:"actualizadoEn"`
}

type Material struct {
	ID                      int64     `json:"id"`
	Codigo                  string    `json:"codigo"`
	Nombre                  string    `json:"nombre"`
	UnidadMedida            string    `json:"unidadMedida"`
	PrecioUnitarioActualCLP *int64    `json:"precioUnitarioActualClp"`
	Activo                  bool      `json:"activo"`
	CreadoEn                time.Time `json:"creadoEn"`
	ActualizadoEn           time.Time `json:"actualizadoEn"`
}

type Chofer struct {
	ID            int64     `json:"id"`
	Nombre        string    `json:"nombre"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"creadoEn"`
	ActualizadoEn time.Time `json:"actualizadoEn"`
}

type Vehiculo struct {
	ID                 int64     `json:"id"`
	Patente            string    `json:"patente"`
	PatenteNormalizada string    `json:"patenteNormalizada"`
	Descripcion        *string   `json:"descripcion"`
	Activo             bool      `json:"activo"`
	CreadoEn           time.Time `json:"creadoEn"`
	ActualizadoEn      time.Time `json:"actualizadoEn"`
}

func incluirInactivos(r *http.Request) bool {
	v := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("incluirInactivos")))
	return v == "1" || v == "true" || v == "si"
}

func punteroEntero(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func punteroTexto(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// Solicitantes.

func (s *Servidor) listarSolicitantes(w http.ResponseWriter, r *http.Request) {
	consulta := `SELECT id, nombre, tipo, activo, creado_en, actualizado_en FROM solicitantes`
	if !incluirInactivos(r) {
		consulta += ` WHERE activo = 1`
	}
	consulta += ` ORDER BY nombre, id`
	rows, err := s.db.QueryContext(r.Context(), consulta)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	defer rows.Close()
	items := make([]Solicitante, 0)
	for rows.Next() {
		var x Solicitante
		if err := rows.Scan(&x.ID, &x.Nombre, &x.Tipo, &x.Activo, &x.CreadoEn, &x.ActualizadoEn); err != nil {
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

type entradaSolicitante struct {
	Nombre string `json:"nombre"`
	Tipo   string `json:"tipo"`
	Activo *bool  `json:"activo,omitempty"`
}

type cambiosSolicitante struct {
	Nombre *string `json:"nombre"`
	Tipo   *string `json:"tipo"`
	Activo *bool   `json:"activo"`
}

func validarSolicitante(nombre, tipo string) error {
	if err := validarLargo("nombre", nombre, 1, 255); err != nil {
		return err
	}
	switch tipo {
	case "persona", "empresa", "institucion", "sin_clasificar":
		return nil
	default:
		return &validacionError{"tipo debe ser persona, empresa, institucion o sin_clasificar"}
	}
}

func (s *Servidor) crearSolicitante(w http.ResponseWriter, r *http.Request) {
	var e entradaSolicitante
	if !leerJSON(w, r, &e) {
		return
	}
	e.Nombre, e.Tipo = recortar(e.Nombre), recortar(e.Tipo)
	if e.Tipo == "" {
		e.Tipo = "sin_clasificar"
	}
	if err := validarSolicitante(e.Nombre, e.Tipo); err != nil {
		escribirValidacion(w, err)
		return
	}
	activo := true
	if e.Activo != nil {
		activo = *e.Activo
	}
	resultado, err := s.db.ExecContext(r.Context(), `INSERT INTO solicitantes (nombre, tipo, activo) VALUES (?, ?, ?)`, e.Nombre, e.Tipo, activo)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	id, _ := resultado.LastInsertId()
	x, err := s.buscarSolicitante(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusCreated, x)
}

func (s *Servidor) editarSolicitante(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	actual, err := s.buscarSolicitante(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var e cambiosSolicitante
	if !leerJSON(w, r, &e) {
		return
	}
	if e.Nombre != nil {
		actual.Nombre = recortar(*e.Nombre)
	}
	if e.Tipo != nil {
		actual.Tipo = recortar(*e.Tipo)
	}
	if e.Activo != nil {
		actual.Activo = *e.Activo
	}
	if err := validarSolicitante(actual.Nombre, actual.Tipo); err != nil {
		escribirValidacion(w, err)
		return
	}
	_, err = s.db.ExecContext(r.Context(), `UPDATE solicitantes SET nombre = ?, tipo = ?, activo = ? WHERE id = ?`, actual.Nombre, actual.Tipo, actual.Activo, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	actual, err = s.buscarSolicitante(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, actual)
}

func (s *Servidor) buscarSolicitante(r *http.Request, id int64) (Solicitante, error) {
	var x Solicitante
	err := s.db.QueryRowContext(r.Context(), `SELECT id, nombre, tipo, activo, creado_en, actualizado_en FROM solicitantes WHERE id = ?`, id).
		Scan(&x.ID, &x.Nombre, &x.Tipo, &x.Activo, &x.CreadoEn, &x.ActualizadoEn)
	return x, err
}

// Materiales.

func (s *Servidor) listarMateriales(w http.ResponseWriter, r *http.Request) {
	consulta := `SELECT id, codigo, nombre, unidad_medida, precio_unitario_actual_clp, activo, creado_en, actualizado_en FROM materiales`
	if !incluirInactivos(r) {
		consulta += ` WHERE activo = 1`
	}
	consulta += ` ORDER BY nombre, id`
	rows, err := s.db.QueryContext(r.Context(), consulta)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	defer rows.Close()
	items := make([]Material, 0)
	for rows.Next() {
		var x Material
		var precio sql.NullInt64
		if err := rows.Scan(&x.ID, &x.Codigo, &x.Nombre, &x.UnidadMedida, &precio, &x.Activo, &x.CreadoEn, &x.ActualizadoEn); err != nil {
			errorBaseDatos(w, s.logger, err)
			return
		}
		x.PrecioUnitarioActualCLP = punteroEntero(precio)
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, items)
}

type entradaMaterial struct {
	Codigo                  string `json:"codigo"`
	Nombre                  string `json:"nombre"`
	UnidadMedida            string `json:"unidadMedida"`
	PrecioUnitarioActualCLP *int64 `json:"precioUnitarioActualClp"`
	Activo                  *bool  `json:"activo,omitempty"`
}

type cambiosMaterial struct {
	Codigo                  *string             `json:"codigo"`
	Nombre                  *string             `json:"nombre"`
	UnidadMedida            *string             `json:"unidadMedida"`
	PrecioUnitarioActualCLP opcionalNulo[int64] `json:"precioUnitarioActualClp"`
	Activo                  *bool               `json:"activo"`
}

func validarMaterial(x Material) error {
	if err := validarLargo("codigo", x.Codigo, 1, 30); err != nil {
		return err
	}
	if err := validarLargo("nombre", x.Nombre, 1, 100); err != nil {
		return err
	}
	if err := validarLargo("unidadMedida", x.UnidadMedida, 1, 20); err != nil {
		return err
	}
	if x.PrecioUnitarioActualCLP != nil && *x.PrecioUnitarioActualCLP < 0 {
		return &validacionError{"precioUnitarioActualClp no puede ser negativo"}
	}
	return nil
}

func (s *Servidor) crearMaterial(w http.ResponseWriter, r *http.Request) {
	var e entradaMaterial
	if !leerJSON(w, r, &e) {
		return
	}
	x := Material{Codigo: recortar(e.Codigo), Nombre: recortar(e.Nombre), UnidadMedida: recortar(e.UnidadMedida), PrecioUnitarioActualCLP: e.PrecioUnitarioActualCLP, Activo: true}
	if x.UnidadMedida == "" {
		x.UnidadMedida = "m3"
	}
	if e.Activo != nil {
		x.Activo = *e.Activo
	}
	if err := validarMaterial(x); err != nil {
		escribirValidacion(w, err)
		return
	}
	resultado, err := s.db.ExecContext(r.Context(), `INSERT INTO materiales (codigo, nombre, unidad_medida, precio_unitario_actual_clp, activo) VALUES (?, ?, ?, ?, ?)`, x.Codigo, x.Nombre, x.UnidadMedida, valorEnteroOpcional(x.PrecioUnitarioActualCLP), x.Activo)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	id, _ := resultado.LastInsertId()
	x, err = s.buscarMaterial(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusCreated, x)
}

func (s *Servidor) editarMaterial(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	x, err := s.buscarMaterial(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var e cambiosMaterial
	if !leerJSON(w, r, &e) {
		return
	}
	if e.Codigo != nil {
		x.Codigo = recortar(*e.Codigo)
	}
	if e.Nombre != nil {
		x.Nombre = recortar(*e.Nombre)
	}
	if e.UnidadMedida != nil {
		x.UnidadMedida = recortar(*e.UnidadMedida)
	}
	if e.PrecioUnitarioActualCLP.Presente {
		x.PrecioUnitarioActualCLP = e.PrecioUnitarioActualCLP.Valor
	}
	if e.Activo != nil {
		x.Activo = *e.Activo
	}
	if err := validarMaterial(x); err != nil {
		escribirValidacion(w, err)
		return
	}
	_, err = s.db.ExecContext(r.Context(), `UPDATE materiales SET codigo = ?, nombre = ?, unidad_medida = ?, precio_unitario_actual_clp = ?, activo = ? WHERE id = ?`, x.Codigo, x.Nombre, x.UnidadMedida, valorEnteroOpcional(x.PrecioUnitarioActualCLP), x.Activo, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	x, err = s.buscarMaterial(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, x)
}

func (s *Servidor) buscarMaterial(r *http.Request, id int64) (Material, error) {
	var x Material
	var precio sql.NullInt64
	err := s.db.QueryRowContext(r.Context(), `SELECT id, codigo, nombre, unidad_medida, precio_unitario_actual_clp, activo, creado_en, actualizado_en FROM materiales WHERE id = ?`, id).
		Scan(&x.ID, &x.Codigo, &x.Nombre, &x.UnidadMedida, &precio, &x.Activo, &x.CreadoEn, &x.ActualizadoEn)
	x.PrecioUnitarioActualCLP = punteroEntero(precio)
	return x, err
}

// Choferes.

func (s *Servidor) listarChoferes(w http.ResponseWriter, r *http.Request) {
	consulta := `SELECT id, nombre, activo, creado_en, actualizado_en FROM choferes`
	if !incluirInactivos(r) {
		consulta += ` WHERE activo = 1`
	}
	consulta += ` ORDER BY nombre, id`
	rows, err := s.db.QueryContext(r.Context(), consulta)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	defer rows.Close()
	items := make([]Chofer, 0)
	for rows.Next() {
		var x Chofer
		if err := rows.Scan(&x.ID, &x.Nombre, &x.Activo, &x.CreadoEn, &x.ActualizadoEn); err != nil {
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

type entradaChofer struct {
	Nombre string `json:"nombre"`
	Activo *bool  `json:"activo,omitempty"`
}

type cambiosChofer struct {
	Nombre *string `json:"nombre"`
	Activo *bool   `json:"activo"`
}

func (s *Servidor) crearChofer(w http.ResponseWriter, r *http.Request) {
	var e entradaChofer
	if !leerJSON(w, r, &e) {
		return
	}
	nombre := recortar(e.Nombre)
	if err := validarLargo("nombre", nombre, 1, 255); err != nil {
		escribirValidacion(w, err)
		return
	}
	activo := true
	if e.Activo != nil {
		activo = *e.Activo
	}
	resultado, err := s.db.ExecContext(r.Context(), `INSERT INTO choferes (nombre, activo) VALUES (?, ?)`, nombre, activo)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	id, _ := resultado.LastInsertId()
	x, err := s.buscarChofer(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusCreated, x)
}

func (s *Servidor) editarChofer(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	x, err := s.buscarChofer(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var e cambiosChofer
	if !leerJSON(w, r, &e) {
		return
	}
	if e.Nombre != nil {
		x.Nombre = recortar(*e.Nombre)
	}
	if e.Activo != nil {
		x.Activo = *e.Activo
	}
	if err := validarLargo("nombre", x.Nombre, 1, 255); err != nil {
		escribirValidacion(w, err)
		return
	}
	_, err = s.db.ExecContext(r.Context(), `UPDATE choferes SET nombre = ?, activo = ? WHERE id = ?`, x.Nombre, x.Activo, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	x, err = s.buscarChofer(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, x)
}

func (s *Servidor) buscarChofer(r *http.Request, id int64) (Chofer, error) {
	var x Chofer
	err := s.db.QueryRowContext(r.Context(), `SELECT id, nombre, activo, creado_en, actualizado_en FROM choferes WHERE id = ?`, id).
		Scan(&x.ID, &x.Nombre, &x.Activo, &x.CreadoEn, &x.ActualizadoEn)
	return x, err
}

// Vehiculos.

func (s *Servidor) listarVehiculos(w http.ResponseWriter, r *http.Request) {
	consulta := `SELECT id, patente, patente_normalizada, descripcion, activo, creado_en, actualizado_en FROM vehiculos`
	if !incluirInactivos(r) {
		consulta += ` WHERE activo = 1`
	}
	consulta += ` ORDER BY patente, id`
	rows, err := s.db.QueryContext(r.Context(), consulta)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	defer rows.Close()
	items := make([]Vehiculo, 0)
	for rows.Next() {
		var x Vehiculo
		var descripcion sql.NullString
		if err := rows.Scan(&x.ID, &x.Patente, &x.PatenteNormalizada, &descripcion, &x.Activo, &x.CreadoEn, &x.ActualizadoEn); err != nil {
			errorBaseDatos(w, s.logger, err)
			return
		}
		x.Descripcion = punteroTexto(descripcion)
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, items)
}

type entradaVehiculo struct {
	Patente     string  `json:"patente"`
	Descripcion *string `json:"descripcion"`
	Activo      *bool   `json:"activo,omitempty"`
}

type cambiosVehiculo struct {
	Patente     *string              `json:"patente"`
	Descripcion opcionalNulo[string] `json:"descripcion"`
	Activo      *bool                `json:"activo"`
}

func validarVehiculo(x Vehiculo) error {
	if err := validarLargo("patente", x.Patente, 1, 20); err != nil {
		return err
	}
	if x.Descripcion != nil && len([]rune(strings.TrimSpace(*x.Descripcion))) > 255 {
		return &validacionError{"descripcion no puede superar 255 caracteres"}
	}
	return nil
}

func (s *Servidor) crearVehiculo(w http.ResponseWriter, r *http.Request) {
	var e entradaVehiculo
	if !leerJSON(w, r, &e) {
		return
	}
	x := Vehiculo{Patente: recortar(e.Patente), Descripcion: limpiarPuntero(e.Descripcion), Activo: true}
	if e.Activo != nil {
		x.Activo = *e.Activo
	}
	if err := validarVehiculo(x); err != nil {
		escribirValidacion(w, err)
		return
	}
	resultado, err := s.db.ExecContext(r.Context(), `INSERT INTO vehiculos (patente, descripcion, activo) VALUES (?, ?, ?)`, x.Patente, textoOpcional(x.Descripcion, 255), x.Activo)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	id, _ := resultado.LastInsertId()
	x, err = s.buscarVehiculo(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusCreated, x)
}

func (s *Servidor) editarVehiculo(w http.ResponseWriter, r *http.Request) {
	id, ok := idRuta(w, r)
	if !ok {
		return
	}
	x, err := s.buscarVehiculo(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	var e cambiosVehiculo
	if !leerJSON(w, r, &e) {
		return
	}
	if e.Patente != nil {
		x.Patente = recortar(*e.Patente)
	}
	if e.Descripcion.Presente {
		x.Descripcion = limpiarPuntero(e.Descripcion.Valor)
	}
	if e.Activo != nil {
		x.Activo = *e.Activo
	}
	if err := validarVehiculo(x); err != nil {
		escribirValidacion(w, err)
		return
	}
	_, err = s.db.ExecContext(r.Context(), `UPDATE vehiculos SET patente = ?, descripcion = ?, activo = ? WHERE id = ?`, x.Patente, textoOpcional(x.Descripcion, 255), x.Activo, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	x, err = s.buscarVehiculo(r, id)
	if err != nil {
		errorBaseDatos(w, s.logger, err)
		return
	}
	escribirJSON(w, http.StatusOK, x)
}

func (s *Servidor) buscarVehiculo(r *http.Request, id int64) (Vehiculo, error) {
	var x Vehiculo
	var descripcion sql.NullString
	err := s.db.QueryRowContext(r.Context(), `SELECT id, patente, patente_normalizada, descripcion, activo, creado_en, actualizado_en FROM vehiculos WHERE id = ?`, id).
		Scan(&x.ID, &x.Patente, &x.PatenteNormalizada, &descripcion, &x.Activo, &x.CreadoEn, &x.ActualizadoEn)
	x.Descripcion = punteroTexto(descripcion)
	return x, err
}

func limpiarPuntero(v *string) *string {
	if v == nil {
		return nil
	}
	limpio := strings.TrimSpace(*v)
	if limpio == "" {
		return nil
	}
	return &limpio
}

type validacionError struct{ mensaje string }

func (e *validacionError) Error() string { return e.mensaje }

func escribirValidacion(w http.ResponseWriter, err error) {
	escribirError(w, http.StatusUnprocessableEntity, "datos_invalidos", err.Error(), nil)
}
