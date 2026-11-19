#!/usr/bin/env python3
"""Genera SQL MySQL idempotente desde el bloque AA:AT de Hoja1/Hoja9."""

from __future__ import annotations

import argparse
import hashlib
import re
import sys
import unicodedata
from dataclasses import dataclass
from datetime import date, datetime
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Sequence

try:
    from openpyxl import load_workbook
    from openpyxl.utils.datetime import from_excel
except ImportError as exc:
    raise SystemExit(
        "Falta openpyxl. Ejecute: python -m pip install -r "
        "documentacion/requirements-importacion.txt"
    ) from exc


MATERIALES = (
    ("arena", "Arena", "m3", Decimal("38000")),
    ("grava", "Grava", "m3", Decimal("25000")),
    ("gravilla", "Gravilla", "m3", Decimal("31000")),
    ("estabilizado", "Estabilizado", "m3", Decimal("24000")),
)
CONTEOS = {"solicitudes": 330, "continuaciones": 55, "entregas": 166,
           "lineas_solicitud": 635, "lineas_entrega": 257}
TOTALES = {
    "comprado": {"arena": Decimal("2965"), "grava": Decimal("1282"),
                  "gravilla": Decimal("1348"), "estabilizado": Decimal("1206")},
    "entregado": {"arena": Decimal("1038"), "grava": Decimal("462"),
                   "gravilla": Decimal("632"), "estabilizado": Decimal("867")},
}
INVALIDOS = {"", "si", "sinf", "sinfo", "sininformacion", "sininfo",
             "sinpatente", "spatente", "noinforma", "noinformacion",
             "solicitante", "solicitantes", "chofer", "chofercamion",
             "operadorcamion", "patente"}


@dataclass(frozen=True)
class Solicitud:
    clave: str
    tipo: str
    codigo: str
    anio: int
    solicitante: str
    solicitante_clave: str
    fecha_compra: date
    memo: str | None
    fecha_asignacion: date | None
    cea: str | None
    marcador: str


@dataclass(frozen=True)
class LineaSolicitud:
    solicitud: str
    material: str
    cantidad: Decimal
    precio: Decimal


@dataclass(frozen=True)
class Entrega:
    clave: str
    solicitud: str
    fecha: date
    chofer_clave: str | None
    patente_normalizada: str | None
    marcador: str


@dataclass(frozen=True)
class LineaEntrega:
    entrega: str
    material: str
    cantidad: Decimal


@dataclass
class Datos:
    solicitudes: list[Solicitud]
    lineas_solicitud: list[LineaSolicitud]
    entregas: list[Entrega]
    lineas_entrega: list[LineaEntrega]
    solicitantes: dict[str, str]
    choferes: dict[str, str]
    vehiculos: dict[str, str]
    continuaciones: int


def texto(valor: object) -> str | None:
    if valor is None:
        return None
    if isinstance(valor, float) and valor.is_integer():
        valor = int(valor)
    salida = unicodedata.normalize("NFKC", str(valor)).replace("\u00a0", " ")
    salida = re.sub(r"\s+", " ", salida).strip()
    return salida or None


def clave(valor: str) -> str:
    return texto(valor).casefold()  # type: ignore[union-attr]


def clave_simple(valor: str | None) -> str:
    if not valor:
        return ""
    descompuesto = unicodedata.normalize("NFKD", valor.casefold())
    return "".join(c for c in descompuesto if c.isalnum() and not unicodedata.combining(c))


def nombre(valor: object) -> str | None:
    salida = texto(valor)
    if (not salida or clave_simple(salida) in INVALIDOS or salida in {".", "..", "-", "--"}
            or salida.isdigit() or len(salida) > 255):
        return None
    return salida


def patente(valor: object) -> tuple[str, str] | None:
    salida = texto(valor)
    if not salida or clave_simple(salida) in INVALIDOS:
        return None
    normalizada = re.sub(r"[\s.\-]", "", salida).upper()
    if (not re.fullmatch(r"[A-Z0-9]{4,10}", normalizada)
            or not re.search(r"[A-Z]", normalizada)
            or not re.search(r"[0-9]", normalizada)):
        return None
    return salida.upper(), normalizada


def leer_fecha(valor: object, epoch: datetime, contexto: str) -> date | None:
    if valor is None or texto(valor) is None:
        return None
    if isinstance(valor, datetime):
        return valor.date()
    if isinstance(valor, date):
        return valor
    if isinstance(valor, (int, float)) and not isinstance(valor, bool):
        convertido = from_excel(valor, epoch)
        return convertido.date() if isinstance(convertido, datetime) else convertido
    salida = texto(valor)
    for formato in ("%Y-%m-%d", "%d/%m/%Y", "%d-%m-%Y"):
        try:
            return datetime.strptime(salida, formato).date()  # type: ignore[arg-type]
        except ValueError:
            pass
    raise ValueError(f"{contexto}: fecha inválida {valor!r}")


def leer_cantidad(valor: object, contexto: str) -> Decimal | None:
    if valor is None or texto(valor) is None:
        return None
    try:
        salida = Decimal(str(valor))
    except (InvalidOperation, ValueError) as exc:
        raise ValueError(f"{contexto}: cantidad inválida {valor!r}") from exc
    if salida == 0:
        return None
    if salida < 0 or salida > Decimal("999999999.999"):
        raise ValueError(f"{contexto}: cantidad fuera de rango")
    if salida != salida.quantize(Decimal("0.001")):
        raise ValueError(f"{contexto}: más de tres decimales")
    return salida


def agregar_nombre(catalogo: dict[str, str], valor: object) -> None:
    salida = nombre(valor)
    if salida:
        catalogo.setdefault(clave(salida), salida)


def agregar_patente(catalogo: dict[str, str], valor: object) -> None:
    salida = patente(valor)
    if salida:
        catalogo.setdefault(salida[1], salida[0])


def comprobar_formulas(libro) -> None:
    for hoja in ("Hoja1", "Hoja9"):
        for fila, valores in enumerate(
            libro[hoja].iter_rows(min_row=13, min_col=27, max_col=46, values_only=True), 13
        ):
            if texto(valores[0]) and any(isinstance(v, str) and v.startswith("=") for v in valores):
                raise ValueError(f"{hoja} fila {fila}: AA:AT contiene una fórmula")


def leer_libro(ruta: Path) -> Datos:
    formulas = load_workbook(ruta, data_only=False, read_only=True)
    try:
        comprobar_formulas(formulas)
    finally:
        formulas.close()
    libro = load_workbook(ruta, data_only=True, read_only=True)
    solicitudes: list[Solicitud] = []
    lineas_solicitud: list[LineaSolicitud] = []
    entregas: list[Entrega] = []
    lineas_entrega: list[LineaEntrega] = []
    solicitantes: dict[str, str] = {}
    choferes: dict[str, str] = {}
    vehiculos: dict[str, str] = {}
    continuaciones = 0
    naturales: set[tuple[str, str, int]] = set()
    try:
        for hoja, tipo in (("Hoja1", "cim"), ("Hoja9", "centro_costo")):
            ultima: dict[str, Solicitud] = {}
            for fila, valores in enumerate(
                libro[hoja].iter_rows(min_row=13, min_col=27, max_col=46, values_only=True), 13
            ):
                codigo = texto(valores[0])
                if not codigo:
                    continue
                contexto = f"{hoja} fila {fila}"
                fecha_compra = leer_fecha(valores[11], libro.epoch, contexto + " fecha compra")
                fecha_entrega = leer_fecha(valores[12], libro.epoch, contexto + " fecha entrega")
                compradas = [leer_cantidad(valores[3+i], contexto + " compra " + MATERIALES[i][0])
                             for i in range(4)]
                entregadas = [leer_cantidad(valores[7+i], contexto + " entrega " + MATERIALES[i][0])
                              for i in range(4)]
                ref = clave(codigo)
                if fecha_compra:
                    solicitante = nombre(valores[1])
                    if not solicitante or not any(compradas):
                        raise ValueError(f"{contexto}: solicitud incompleta")
                    natural = (tipo, ref, fecha_compra.year)
                    if natural in naturales:
                        raise ValueError(f"{contexto}: clave natural duplicada")
                    naturales.add(natural)
                    id_fuente = f"{hoja.casefold()}-fila-{fila}"
                    actual = Solicitud(
                        id_fuente, tipo, codigo, fecha_compra.year, solicitante, clave(solicitante),
                        fecha_compra, texto(valores[17]),
                        leer_fecha(valores[18], libro.epoch, contexto + " fecha asignación"),
                        texto(valores[19]), f"[IMPORT_EXCEL_2026_SOLICITUD:{hoja}:{fila}]"
                    )
                    solicitudes.append(actual)
                    ultima[ref] = actual
                    agregar_nombre(solicitantes, solicitante)
                    for material, cantidad in zip(MATERIALES, compradas):
                        if cantidad is not None:
                            lineas_solicitud.append(
                                LineaSolicitud(id_fuente, material[0], cantidad, material[3]))
                else:
                    actual = ultima.get(ref)
                    if not actual:
                        raise ValueError(f"{contexto}: continuación sin base anterior")
                    if any(compradas):
                        raise ValueError(f"{contexto}: compra sin fecha")
                    continuaciones += 1
                if bool(fecha_entrega) != bool(any(entregadas)):
                    raise ValueError(f"{contexto}: fecha/cantidad de entrega incompleta")
                if fecha_entrega:
                    id_entrega = f"{hoja.casefold()}-fila-{fila}"
                    chofer = nombre(valores[13])
                    vehiculo = patente(valores[14])
                    agregar_nombre(choferes, chofer)
                    if vehiculo:
                        agregar_patente(vehiculos, vehiculo[0])
                    entregas.append(Entrega(
                        id_entrega, actual.clave, fecha_entrega, clave(chofer) if chofer else None,
                        vehiculo[1] if vehiculo else None,
                        f"[IMPORT_EXCEL_2026_ENTREGA:{hoja}:{fila}]"))
                    for material, cantidad in zip(MATERIALES, entregadas):
                        if cantidad is not None:
                            lineas_entrega.append(LineaEntrega(id_entrega, material[0], cantidad))
        for hoja, columna in (("Hoja6", 1), ("Hoja7", 3)):
            for (valor,) in libro[hoja].iter_rows(min_col=columna, max_col=columna, values_only=True):
                agregar_nombre(solicitantes, valor)
        for hoja, columna in (("Hoja7", 1), ("Hoja3", 6)):
            for (valor,) in libro[hoja].iter_rows(min_col=columna, max_col=columna, values_only=True):
                agregar_nombre(choferes, valor)
        for hoja, columna in (("Hoja7", 2), ("Hoja2", 14)):
            for (valor,) in libro[hoja].iter_rows(min_col=columna, max_col=columna, values_only=True):
                agregar_patente(vehiculos, valor)
    finally:
        libro.close()
    datos = Datos(solicitudes, lineas_solicitud, entregas, lineas_entrega,
                  solicitantes, choferes, vehiculos, continuaciones)
    validar(datos)
    return datos


def validar(datos: Datos) -> None:
    reales = {"solicitudes": len(datos.solicitudes), "continuaciones": datos.continuaciones,
              "entregas": len(datos.entregas), "lineas_solicitud": len(datos.lineas_solicitud),
              "lineas_entrega": len(datos.lineas_entrega)}
    if reales != CONTEOS:
        raise ValueError(f"Conteos inesperados: esperados={CONTEOS}, obtenidos={reales}")
    for solicitud in datos.solicitudes:
        if len(solicitud.codigo) > 64:
            raise ValueError(
                f"{solicitud.clave}: código de referencia supera 64 caracteres"
            )
        if solicitud.memo is not None and len(solicitud.memo) > 64:
            raise ValueError(f"{solicitud.clave}: número de memo supera 64 caracteres")
        if solicitud.cea is not None and len(solicitud.cea) > 64:
            raise ValueError(f"{solicitud.clave}: número CEA supera 64 caracteres")
    compradas = {(x.solicitud, x.material): x.cantidad for x in datos.lineas_solicitud}
    acumuladas: dict[tuple[str, str], Decimal] = {}
    entregas = {x.clave: x for x in datos.entregas}
    for linea in datos.lineas_entrega:
        llave = (entregas[linea.entrega].solicitud, linea.material)
        if llave not in compradas:
            raise ValueError(f"Entrega de material no comprado: {llave}")
        acumuladas[llave] = acumuladas.get(llave, Decimal(0)) + linea.cantidad
    excesos = [k for k, v in acumuladas.items() if v > compradas[k]]
    if excesos:
        raise ValueError(f"Sobreentregas en el Excel: {excesos[:5]}")
    for clase, filas in (("comprado", datos.lineas_solicitud),
                         ("entregado", datos.lineas_entrega)):
        totales = {m[0]: Decimal(0) for m in MATERIALES}
        for fila in filas:
            totales[fila.material] += fila.cantidad
        if totales != TOTALES[clase]:
            raise ValueError(f"Totales {clase}s inesperados: {totales}")


def q(valor: str | None) -> str:
    if valor is None:
        return "NULL"
    return "'" + valor.replace("\\", "\\\\").replace("'", "''").replace("\x00", "") + "'"


def qfecha(valor: date | None) -> str:
    return "NULL" if valor is None else q(valor.isoformat())


def qdecimal(valor: Decimal) -> str:
    salida = format(valor, "f")
    return salida.rstrip("0").rstrip(".") if "." in salida else salida


def valores(salida: list[str], tabla: str, columnas: Sequence[str], filas: Sequence[Sequence[str]]) -> None:
    for inicio in range(0, len(filas), 200):
        bloque = filas[inicio:inicio+200]
        salida.append(f"INSERT INTO {tabla} ({', '.join(columnas)}) VALUES")
        salida.append(",\n".join("  (" + ", ".join(f) + ")" for f in bloque) + ";\n")


def generar_sql(datos: Datos, digest: str, libro: str) -> str:
    salida = [
        "-- IMPORTACIÓN IDEMPOTENTE DE ÁRIDOS DESDE EXCEL",
        "-- Generado por scripts/generar_importacion_excel.py; no editar a mano.",
        f"-- Libro: {libro}", f"-- SHA-256: {digest}",
        "-- Fuente: AA:AT de Hoja1 y Hoja9. No se importaron fórmulas.",
        "-- 330 solicitudes, 166 entregas, 635 líneas compradas, 257 líneas entregadas.",
        "-- Los precios históricos se guardan por solicitud; el precio actual NO se modifica.",
        "", "SET NAMES utf8mb4;", "USE gestion_aridos;", "",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_solicitantes;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_choferes;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_vehiculos;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_solicitudes;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_solicitudes_materiales;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_entregas;",
        "DROP TEMPORARY TABLE IF EXISTS tmp_import_entregas_materiales;", "",
        "CREATE TEMPORARY TABLE tmp_import_solicitantes (nombre_clave VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PRIMARY KEY, nombre VARCHAR(255) NOT NULL) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_choferes (nombre_clave VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PRIMARY KEY, nombre VARCHAR(255) NOT NULL) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_vehiculos (patente_normalizada VARCHAR(20) PRIMARY KEY, patente VARCHAR(20) NOT NULL) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_solicitudes (clave VARCHAR(40) PRIMARY KEY, tipo_referencia VARCHAR(20) NOT NULL, codigo_referencia VARCHAR(64) NOT NULL, anio SMALLINT UNSIGNED NOT NULL, solicitante_clave VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL, fecha_compra DATE NOT NULL, numero_memo VARCHAR(64), fecha_asignacion DATE, numero_cea VARCHAR(64), marcador VARCHAR(120) NOT NULL, UNIQUE KEY uq_tmp_solicitud(tipo_referencia,codigo_referencia,anio)) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_solicitudes_materiales (solicitud_clave VARCHAR(40) NOT NULL, material_codigo VARCHAR(30) NOT NULL, cantidad DECIMAL(12,3) NOT NULL, precio_aplicado BIGINT UNSIGNED NOT NULL, PRIMARY KEY(solicitud_clave,material_codigo)) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_entregas (clave VARCHAR(40) PRIMARY KEY, solicitud_clave VARCHAR(40) NOT NULL, fecha_entrega DATE NOT NULL, chofer_clave VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin, patente_normalizada VARCHAR(20), marcador VARCHAR(120) NOT NULL) ENGINE=InnoDB;",
        "CREATE TEMPORARY TABLE tmp_import_entregas_materiales (entrega_clave VARCHAR(40) NOT NULL, material_codigo VARCHAR(50) NOT NULL, cantidad DECIMAL(12,3) NOT NULL, PRIMARY KEY(entrega_clave,material_codigo)) ENGINE=InnoDB;", ""
    ]
    valores(salida, "tmp_import_solicitantes", ("nombre_clave", "nombre"),
            [(q(k), q(v)) for k, v in sorted(datos.solicitantes.items())])
    valores(salida, "tmp_import_choferes", ("nombre_clave", "nombre"),
            [(q(k), q(v)) for k, v in sorted(datos.choferes.items())])
    valores(salida, "tmp_import_vehiculos", ("patente_normalizada", "patente"),
            [(q(k), q(v)) for k, v in sorted(datos.vehiculos.items())])
    valores(salida, "tmp_import_solicitudes",
            ("clave", "tipo_referencia", "codigo_referencia", "anio", "solicitante_clave",
             "fecha_compra", "numero_memo", "fecha_asignacion", "numero_cea", "marcador"),
            [(q(x.clave), q(x.tipo), q(x.codigo), str(x.anio), q(x.solicitante_clave),
              qfecha(x.fecha_compra), q(x.memo), qfecha(x.fecha_asignacion), q(x.cea), q(x.marcador))
             for x in datos.solicitudes])
    valores(salida, "tmp_import_solicitudes_materiales",
            ("solicitud_clave", "material_codigo", "cantidad", "precio_aplicado"),
            [(q(x.solicitud), q(x.material), qdecimal(x.cantidad), qdecimal(x.precio))
             for x in datos.lineas_solicitud])
    valores(salida, "tmp_import_entregas",
            ("clave", "solicitud_clave", "fecha_entrega", "chofer_clave",
             "patente_normalizada", "marcador"),
            [(q(x.clave), q(x.solicitud), qfecha(x.fecha), q(x.chofer_clave),
              q(x.patente_normalizada), q(x.marcador)) for x in datos.entregas])
    valores(salida, "tmp_import_entregas_materiales",
            ("entrega_clave", "material_codigo", "cantidad"),
            [(q(x.entrega), q(x.material), qdecimal(x.cantidad)) for x in datos.lineas_entrega])
    salida.extend(SQL_PROCEDIMIENTO.strip("\n").splitlines())
    return "\n".join(salida) + "\n"


SQL_PROCEDIMIENTO = r"""
DROP PROCEDURE IF EXISTS sp_importar_excel_aridos_2026;
DELIMITER $$
CREATE PROCEDURE sp_importar_excel_aridos_2026()
BEGIN
  DECLARE EXIT HANDLER FOR SQLEXCEPTION
  BEGIN
    ROLLBACK;
    DO RELEASE_LOCK('importacion_excel_aridos_2026');
    RESIGNAL;
  END;
  IF COALESCE(GET_LOCK('importacion_excel_aridos_2026',30),0) <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='No se obtuvo el bloqueo de importacion';
  END IF;
  START TRANSACTION;

  INSERT INTO solicitantes(nombre,tipo,activo)
  SELECT t.nombre,'sin_clasificar',1 FROM tmp_import_solicitantes t
  WHERE NOT EXISTS (SELECT 1 FROM solicitantes s
    WHERE CAST(LOWER(REGEXP_REPLACE(TRIM(s.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.nombre_clave AS BINARY));
  IF EXISTS (SELECT 1 FROM tmp_import_solicitantes t JOIN solicitantes s
      ON CAST(LOWER(REGEXP_REPLACE(TRIM(s.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.nombre_clave AS BINARY)
      GROUP BY t.nombre_clave HAVING COUNT(*)<>1) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Solicitante ausente o duplicado por nombre normalizado';
  END IF;

  INSERT INTO choferes(nombre,activo)
  SELECT t.nombre,1 FROM tmp_import_choferes t
  WHERE NOT EXISTS (SELECT 1 FROM choferes c
    WHERE CAST(LOWER(REGEXP_REPLACE(TRIM(c.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.nombre_clave AS BINARY));
  IF EXISTS (SELECT 1 FROM tmp_import_choferes t JOIN choferes c
      ON CAST(LOWER(REGEXP_REPLACE(TRIM(c.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.nombre_clave AS BINARY)
      GROUP BY t.nombre_clave HAVING COUNT(*)<>1) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Chofer ausente o duplicado por nombre normalizado';
  END IF;

  INSERT INTO vehiculos(patente,descripcion,activo)
  SELECT t.patente,NULL,1 FROM tmp_import_vehiculos t
  WHERE NOT EXISTS (SELECT 1 FROM vehiculos v WHERE v.patente_normalizada=t.patente_normalizada);

  -- No se asigna ni actualiza precio_unitario_actual_clp.
  INSERT INTO materiales(codigo,nombre,unidad_medida,activo) VALUES
    ('arena','Arena','m3',1),('grava','Grava','m3',1),
    ('gravilla','Gravilla','m3',1),('estabilizado','Estabilizado','m3',1)
  ON DUPLICATE KEY UPDATE nombre=VALUES(nombre),unidad_medida=VALUES(unidad_medida),activo=1;

  IF EXISTS (SELECT 1 FROM tmp_import_solicitudes t JOIN solicitudes s
      ON s.tipo_referencia=t.tipo_referencia AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio
      JOIN solicitantes so ON so.id=s.solicitante_id
      WHERE NOT(s.fecha_compra<=>t.fecha_compra AND s.numero_memo<=>t.numero_memo
        AND s.fecha_asignacion<=>t.fecha_asignacion AND s.numero_cea<=>t.numero_cea
        AND CAST(LOWER(REGEXP_REPLACE(TRIM(so.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.solicitante_clave AS BINARY))) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Una solicitud existente difiere del Excel';
  END IF;

  INSERT INTO solicitudes(tipo_referencia,codigo_referencia,anio,solicitante_id,
      fecha_compra,numero_memo,fecha_asignacion,numero_cea,estado,observaciones)
  SELECT t.tipo_referencia,t.codigo_referencia,t.anio,so.id,t.fecha_compra,t.numero_memo,
      t.fecha_asignacion,t.numero_cea,'activa',t.marcador
  FROM tmp_import_solicitudes t JOIN solicitantes so
    ON CAST(LOWER(REGEXP_REPLACE(TRIM(so.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.solicitante_clave AS BINARY)
  WHERE NOT EXISTS (SELECT 1 FROM solicitudes s
    WHERE s.tipo_referencia=t.tipo_referencia
      AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio);
  UPDATE solicitudes s JOIN tmp_import_solicitudes t
    ON s.tipo_referencia=t.tipo_referencia AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio
  SET s.observaciones=CONCAT_WS(CHAR(10),NULLIF(s.observaciones,''),t.marcador)
  WHERE LOCATE(t.marcador,COALESCE(s.observaciones,''))=0;

  IF EXISTS (SELECT 1 FROM tmp_import_solicitudes_materiales tm
      JOIN tmp_import_solicitudes t ON t.clave=tm.solicitud_clave
      JOIN solicitudes s ON s.tipo_referencia=t.tipo_referencia AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio
      JOIN materiales m ON m.codigo=tm.material_codigo
      JOIN solicitudes_materiales sm ON sm.solicitud_id=s.id AND sm.material_id=m.id
      WHERE sm.cantidad_comprada<>tm.cantidad OR
        (sm.precio_unitario_aplicado_clp IS NOT NULL AND sm.precio_unitario_aplicado_clp<>tm.precio_aplicado)) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Una linea comprada existente difiere del Excel';
  END IF;
  INSERT INTO solicitudes_materiales(solicitud_id,material_id,cantidad_comprada,precio_unitario_aplicado_clp)
  SELECT s.id,m.id,tm.cantidad,tm.precio_aplicado FROM tmp_import_solicitudes_materiales tm
    JOIN tmp_import_solicitudes t ON t.clave=tm.solicitud_clave
    JOIN solicitudes s ON s.tipo_referencia=t.tipo_referencia AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio
    JOIN materiales m ON m.codigo=tm.material_codigo
  WHERE NOT EXISTS (SELECT 1 FROM solicitudes_materiales sm0
    WHERE sm0.solicitud_id=s.id AND sm0.material_id=m.id);
  UPDATE solicitudes_materiales sm JOIN solicitudes s ON s.id=sm.solicitud_id
    JOIN tmp_import_solicitudes t ON s.tipo_referencia=t.tipo_referencia AND s.codigo_referencia=t.codigo_referencia AND s.anio=t.anio
    JOIN tmp_import_solicitudes_materiales tm ON tm.solicitud_clave=t.clave
    JOIN materiales m ON m.id=sm.material_id AND m.codigo=tm.material_codigo
  SET sm.precio_unitario_aplicado_clp=tm.precio_aplicado WHERE sm.precio_unitario_aplicado_clp IS NULL;

  IF EXISTS (SELECT 1 FROM tmp_import_entregas t
      JOIN tmp_import_solicitudes ts ON ts.clave=t.solicitud_clave
      JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
      JOIN entregas e ON e.solicitud_id=s.id AND LOCATE(t.marcador,COALESCE(e.observaciones,''))>0
      GROUP BY t.clave HAVING COUNT(*)>1) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Marca de entrega duplicada';
  END IF;
  IF EXISTS (SELECT 1 FROM tmp_import_entregas t
      JOIN tmp_import_solicitudes ts ON ts.clave=t.solicitud_clave
      JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
      JOIN entregas e ON e.solicitud_id=s.id AND LOCATE(t.marcador,COALESCE(e.observaciones,''))>0
      LEFT JOIN choferes c ON t.chofer_clave IS NOT NULL AND
        CAST(LOWER(REGEXP_REPLACE(TRIM(c.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.chofer_clave AS BINARY)
      LEFT JOIN vehiculos v ON v.patente_normalizada=t.patente_normalizada
      WHERE e.fecha_entrega<>t.fecha_entrega OR e.estado<>'confirmada'
        OR NOT(e.chofer_id<=>c.id) OR NOT(e.vehiculo_id<=>v.id)) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Una entrega existente difiere del Excel';
  END IF;

  -- Suma lo confirmado previamente y sólo las entregas nuevas del lote.
  IF EXISTS (SELECT sm.id FROM tmp_import_entregas_materiales tem
      JOIN tmp_import_entregas te ON te.clave=tem.entrega_clave
      JOIN tmp_import_solicitudes ts ON ts.clave=te.solicitud_clave
      JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
      JOIN materiales m ON m.codigo=tem.material_codigo
      JOIN solicitudes_materiales sm ON sm.solicitud_id=s.id AND sm.material_id=m.id
      LEFT JOIN (SELECT em.solicitud_material_id,SUM(em.cantidad_entregada) cantidad
        FROM entregas_materiales em JOIN entregas e0 ON e0.id=em.entrega_id AND e0.estado='confirmada'
        GROUP BY em.solicitud_material_id) p ON p.solicitud_material_id=sm.id
      LEFT JOIN entregas ex ON ex.solicitud_id=s.id AND LOCATE(te.marcador,COALESCE(ex.observaciones,''))>0
      LEFT JOIN entregas_materiales emx
        ON emx.entrega_id=ex.id AND emx.solicitud_material_id=sm.id
      GROUP BY sm.id,sm.cantidad_comprada,p.cantidad
      HAVING COALESCE(p.cantidad,0)+SUM(CASE WHEN emx.id IS NULL THEN tem.cantidad ELSE 0 END)>sm.cantidad_comprada) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='La importacion produciria una sobreentrega';
  END IF;

  INSERT INTO entregas(solicitud_id,fecha_entrega,chofer_id,vehiculo_id,estado,observaciones)
  SELECT s.id,t.fecha_entrega,c.id,v.id,'confirmada',t.marcador FROM tmp_import_entregas t
    JOIN tmp_import_solicitudes ts ON ts.clave=t.solicitud_clave
    JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
    LEFT JOIN choferes c ON t.chofer_clave IS NOT NULL AND
      CAST(LOWER(REGEXP_REPLACE(TRIM(c.nombre),'[[:space:]]+',' ')) AS BINARY)=CAST(t.chofer_clave AS BINARY)
    LEFT JOIN vehiculos v ON v.patente_normalizada=t.patente_normalizada
  WHERE NOT EXISTS (SELECT 1 FROM entregas e WHERE e.solicitud_id=s.id
    AND LOCATE(t.marcador,COALESCE(e.observaciones,''))>0);

  IF EXISTS (SELECT 1 FROM tmp_import_entregas_materiales tem
      JOIN tmp_import_entregas te ON te.clave=tem.entrega_clave
      JOIN tmp_import_solicitudes ts ON ts.clave=te.solicitud_clave
      JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
      JOIN entregas e ON e.solicitud_id=s.id AND LOCATE(te.marcador,COALESCE(e.observaciones,''))>0
      JOIN materiales m ON m.codigo=tem.material_codigo
      JOIN solicitudes_materiales sm ON sm.solicitud_id=s.id AND sm.material_id=m.id
      JOIN entregas_materiales em ON em.entrega_id=e.id AND em.solicitud_material_id=sm.id
      WHERE em.cantidad_entregada<>tem.cantidad) THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Una linea entregada existente difiere del Excel';
  END IF;
  INSERT INTO entregas_materiales(entrega_id,solicitud_material_id,cantidad_entregada)
  SELECT e.id,sm.id,tem.cantidad FROM tmp_import_entregas_materiales tem
    JOIN tmp_import_entregas te ON te.clave=tem.entrega_clave
    JOIN tmp_import_solicitudes ts ON ts.clave=te.solicitud_clave
    JOIN solicitudes s ON s.tipo_referencia=ts.tipo_referencia AND s.codigo_referencia=ts.codigo_referencia AND s.anio=ts.anio
    JOIN entregas e ON e.solicitud_id=s.id AND LOCATE(te.marcador,COALESCE(e.observaciones,''))>0
    JOIN materiales m ON m.codigo=tem.material_codigo
    JOIN solicitudes_materiales sm ON sm.solicitud_id=s.id AND sm.material_id=m.id
  WHERE NOT EXISTS (SELECT 1 FROM entregas_materiales em0
    WHERE em0.entrega_id=e.id AND em0.solicitud_material_id=sm.id);

  COMMIT;
  DO RELEASE_LOCK('importacion_excel_aridos_2026');
END$$
DELIMITER ;
CALL sp_importar_excel_aridos_2026();
DROP PROCEDURE IF EXISTS sp_importar_excel_aridos_2026;
SELECT 'Importacion terminada' resultado,
 (SELECT COUNT(*) FROM tmp_import_solicitudes) solicitudes,
 (SELECT COUNT(*) FROM tmp_import_entregas) entregas,
 (SELECT COUNT(*) FROM tmp_import_solicitudes_materiales) lineas_compradas,
 (SELECT COUNT(*) FROM tmp_import_entregas_materiales) lineas_entregadas;
"""


def generar_validacion(digest: str, libro: str) -> str:
    encabezado = f"""-- VALIDACIÓN DE LA IMPORTACIÓN DE ÁRIDOS DESDE EXCEL
-- Generado por scripts/generar_importacion_excel.py; no editar a mano.
-- Libro: {libro}
-- SHA-256: {digest}
SET NAMES utf8mb4;
USE gestion_aridos;

"""
    cuerpo = r"""-- Estos cuatro controles deben devolver OK.
SELECT 'SOLICITUDES_IMPORTADAS' control,330 esperado,COUNT(*) valor_real,
 IF(COUNT(*)=330,'OK','REVISAR') estado FROM solicitudes
 WHERE observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%';
SELECT 'ENTREGAS_IMPORTADAS' control,166 esperado,COUNT(*) valor_real,
 IF(COUNT(*)=166,'OK','REVISAR') estado FROM entregas
 WHERE observaciones LIKE '%IMPORT_EXCEL_2026_ENTREGA:%';
SELECT 'LINEAS_COMPRADAS' control,635 esperado,COUNT(*) valor_real,
 IF(COUNT(*)=635,'OK','REVISAR') estado
 FROM solicitudes_materiales sm JOIN solicitudes s ON s.id=sm.solicitud_id
 WHERE s.observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%';
SELECT 'LINEAS_ENTREGADAS' control,257 esperado,COUNT(*) valor_real,
 IF(COUNT(*)=257,'OK','REVISAR') estado
 FROM entregas_materiales em JOIN entregas e ON e.id=em.entrega_id
 WHERE e.observaciones LIKE '%IMPORT_EXCEL_2026_ENTREGA:%';

-- Totales comprobados contra el libro; las ocho filas deben devolver OK.
SELECT CONCAT('COMPRADO_',UPPER(m.codigo)) control,
 CASE m.codigo WHEN 'arena' THEN 2965 WHEN 'grava' THEN 1282
  WHEN 'gravilla' THEN 1348 WHEN 'estabilizado' THEN 1206 END esperado,
 SUM(sm.cantidad_comprada) valor_real,
 IF(SUM(sm.cantidad_comprada)=CASE m.codigo WHEN 'arena' THEN 2965 WHEN 'grava' THEN 1282
  WHEN 'gravilla' THEN 1348 WHEN 'estabilizado' THEN 1206 END,'OK','REVISAR') estado
FROM solicitudes_materiales sm JOIN solicitudes s ON s.id=sm.solicitud_id
JOIN materiales m ON m.id=sm.material_id
WHERE s.observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%'
GROUP BY m.id,m.codigo
UNION ALL
SELECT CONCAT('ENTREGADO_',UPPER(m.codigo)),
 CASE m.codigo WHEN 'arena' THEN 1038 WHEN 'grava' THEN 462
  WHEN 'gravilla' THEN 632 WHEN 'estabilizado' THEN 867 END,
 SUM(em.cantidad_entregada),
 IF(SUM(em.cantidad_entregada)=CASE m.codigo WHEN 'arena' THEN 1038 WHEN 'grava' THEN 462
  WHEN 'gravilla' THEN 632 WHEN 'estabilizado' THEN 867 END,'OK','REVISAR')
FROM entregas_materiales em JOIN entregas e ON e.id=em.entrega_id
JOIN solicitudes_materiales sm ON sm.id=em.solicitud_material_id
JOIN materiales m ON m.id=sm.material_id
WHERE e.observaciones LIKE '%IMPORT_EXCEL_2026_ENTREGA:%'
GROUP BY m.id,m.codigo;

-- El precio histórico va en la línea comprada, nunca en el precio vigente del catálogo.
SELECT m.codigo,
 CASE m.codigo WHEN 'arena' THEN 38000 WHEN 'grava' THEN 25000
  WHEN 'gravilla' THEN 31000 WHEN 'estabilizado' THEN 24000 END precio_esperado,
 MIN(sm.precio_unitario_aplicado_clp) precio_minimo,
 MAX(sm.precio_unitario_aplicado_clp) precio_maximo,
 IF(MIN(sm.precio_unitario_aplicado_clp)=CASE m.codigo WHEN 'arena' THEN 38000
     WHEN 'grava' THEN 25000 WHEN 'gravilla' THEN 31000 WHEN 'estabilizado' THEN 24000 END
    AND MAX(sm.precio_unitario_aplicado_clp)=CASE m.codigo WHEN 'arena' THEN 38000
     WHEN 'grava' THEN 25000 WHEN 'gravilla' THEN 31000 WHEN 'estabilizado' THEN 24000 END,
    'OK','REVISAR') estado
FROM solicitudes_materiales sm JOIN solicitudes s ON s.id=sm.solicitud_id
JOIN materiales m ON m.id=sm.material_id
WHERE s.observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%'
GROUP BY m.id,m.codigo ORDER BY m.codigo;

-- Cada una de las consultas siguientes debe devolver cero filas.
-- Sobreentregas confirmadas:
SELECT s.tipo_referencia,s.codigo_referencia,s.anio,m.codigo,
 sm.cantidad_comprada,SUM(em.cantidad_entregada) cantidad_entregada
FROM solicitudes_materiales sm JOIN solicitudes s ON s.id=sm.solicitud_id
JOIN materiales m ON m.id=sm.material_id
JOIN entregas_materiales em ON em.solicitud_material_id=sm.id
JOIN entregas e ON e.id=em.entrega_id AND e.estado='confirmada'
GROUP BY s.id,sm.id,m.id HAVING SUM(em.cantidad_entregada)>sm.cantidad_comprada;

-- Solicitudes sin detalle:
SELECT s.id,s.tipo_referencia,s.codigo_referencia,s.anio
FROM solicitudes s LEFT JOIN solicitudes_materiales sm ON sm.solicitud_id=s.id
WHERE s.observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%'
GROUP BY s.id HAVING COUNT(sm.id)=0;

-- Entregas sin detalle o con estado incorrecto:
SELECT e.id,e.fecha_entrega,e.estado,COUNT(em.id) lineas
FROM entregas e LEFT JOIN entregas_materiales em ON em.entrega_id=e.id
WHERE e.observaciones LIKE '%IMPORT_EXCEL_2026_ENTREGA:%'
GROUP BY e.id HAVING COUNT(em.id)=0 OR e.estado<>'confirmada';

-- Marcas de origen repetidas:
SELECT REGEXP_SUBSTR(observaciones,'\\[IMPORT_EXCEL_2026_ENTREGA:[^]]+\\]') marca,
 COUNT(*) repeticiones FROM entregas
WHERE observaciones LIKE '%IMPORT_EXCEL_2026_ENTREGA:%'
GROUP BY marca HAVING COUNT(*)>1;

-- Informativo: faltantes opcionales que ya estaban vacíos en Excel.
SELECT SUM(numero_memo IS NULL) solicitudes_sin_memo,
 SUM(fecha_asignacion IS NULL) solicitudes_sin_fecha_asignacion,
 SUM(numero_cea IS NULL) solicitudes_sin_cea
FROM solicitudes WHERE observaciones LIKE '%IMPORT_EXCEL_2026_SOLICITUD:%';

-- Revisión visible del precio actual, que el importador no modifica.
SELECT codigo,nombre,unidad_medida,precio_unitario_actual_clp FROM materiales
WHERE codigo IN('arena','grava','gravilla','estabilizado') ORDER BY codigo;
"""
    return encabezado + cuerpo


def digest_archivo(ruta: Path) -> str:
    sha = hashlib.sha256()
    with ruta.open("rb") as archivo:
        for bloque in iter(lambda: archivo.read(1024 * 1024), b""):
            sha.update(bloque)
    return sha.hexdigest()


def main() -> int:
    raiz = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--excel", type=Path, default=raiz / "Entrega aridos Año 2026.xlsm")
    parser.add_argument("--salida", type=Path,
                        default=raiz / "documentacion" / "poblar_datos_excel_2026.sql")
    parser.add_argument("--validacion", type=Path,
                        default=raiz / "documentacion" / "validar_importacion_excel_2026.sql")
    args = parser.parse_args()
    if not args.excel.is_file():
        raise SystemExit(f"No existe el libro: {args.excel}")
    datos = leer_libro(args.excel)
    digest = digest_archivo(args.excel)
    args.salida.parent.mkdir(parents=True, exist_ok=True)
    args.validacion.parent.mkdir(parents=True, exist_ok=True)
    args.salida.write_text(generar_sql(datos, digest, args.excel.name), encoding="utf-8", newline="\n")
    args.validacion.write_text(generar_validacion(digest, args.excel.name),
                               encoding="utf-8", newline="\n")
    print(f"Carga: {args.salida}")
    print(f"Validación: {args.validacion}")
    print(f"SHA-256: {digest}")
    print(f"Conteos: {len(datos.solicitudes)} solicitudes; {datos.continuaciones} continuaciones; "
          f"{len(datos.entregas)} entregas; {len(datos.lineas_solicitud)} líneas compradas; "
          f"{len(datos.lineas_entrega)} líneas entregadas")
    print(f"Catálogos: {len(datos.solicitantes)} solicitantes; {len(datos.choferes)} choferes; "
          f"{len(datos.vehiculos)} vehículos")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ValueError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
