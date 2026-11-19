-- VALIDACIÓN DE LA IMPORTACIÓN DE ÁRIDOS DESDE EXCEL
-- Generado por scripts/generar_importacion_excel.py; no editar a mano.
-- Libro: Entrega aridos Año 2026.xlsm
-- SHA-256: 72b07689496e34e6762ce230027c29901421e18ac0fa3a48768e02308bf65c08
SET NAMES utf8mb4;
USE gestion_aridos;

-- Estos cuatro controles deben devolver OK.
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
