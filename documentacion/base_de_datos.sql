-- ============================================================================
-- SISTEMA DE GESTION DE ARIDOS
-- MySQL 8.0.16 o superior
-- Script inicial para ejecutar completo en DBeaver.
--
-- Para usar otro nombre de base de datos, cambie "gestion_aridos" en las
-- sentencias CREATE DATABASE, USE y SHOW FULL TABLES.
-- ============================================================================

SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE DATABASE IF NOT EXISTS `gestion_aridos`
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `gestion_aridos`;

-- ============================================================================
-- 1. SOLICITANTES
-- ============================================================================

CREATE TABLE IF NOT EXISTS `solicitantes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno del solicitante',
    `nombre` VARCHAR(255) NOT NULL
        COMMENT 'Nombre de la persona, empresa o institucion',
    `tipo` VARCHAR(20) NOT NULL DEFAULT 'sin_clasificar'
        COMMENT 'persona, empresa, institucion o sin_clasificar',
    `activo` TINYINT(1) NOT NULL DEFAULT 1
        COMMENT 'Indica si puede usarse en nuevas solicitudes',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    KEY `ix_solicitantes_activo_nombre` (`activo`, `nombre`),
    KEY `ix_solicitantes_nombre` (`nombre`),

    CONSTRAINT `ck_solicitantes_nombre`
        CHECK (CHAR_LENGTH(TRIM(`nombre`)) > 0),
    CONSTRAINT `ck_solicitantes_tipo`
        CHECK (`tipo` IN ('persona', 'empresa', 'institucion', 'sin_clasificar')),
    CONSTRAINT `ck_solicitantes_activo`
        CHECK (`activo` IN (0, 1))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Personas, empresas e instituciones solicitantes de aridos';

-- ============================================================================
-- 2. MATERIALES
-- ============================================================================

CREATE TABLE IF NOT EXISTS `materiales` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno del material',
    `codigo` VARCHAR(30) NOT NULL
        COMMENT 'Codigo estable del material',
    `nombre` VARCHAR(100) NOT NULL
        COMMENT 'Nombre visible del material',
    `unidad_medida` VARCHAR(20) NOT NULL DEFAULT 'm3'
        COMMENT 'Unidad de medida, por ejemplo m3, tonelada o unidad',
    `precio_unitario_actual_clp` BIGINT UNSIGNED NULL
        COMMENT 'Precio actual por unidad en pesos chilenos',
    `activo` TINYINT(1) NOT NULL DEFAULT 1
        COMMENT 'Indica si puede usarse en nuevas solicitudes',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_materiales_codigo` (`codigo`),
    KEY `ix_materiales_activo_nombre` (`activo`, `nombre`),

    CONSTRAINT `ck_materiales_codigo`
        CHECK (CHAR_LENGTH(TRIM(`codigo`)) > 0),
    CONSTRAINT `ck_materiales_nombre`
        CHECK (CHAR_LENGTH(TRIM(`nombre`)) > 0),
    CONSTRAINT `ck_materiales_unidad`
        CHECK (CHAR_LENGTH(TRIM(`unidad_medida`)) > 0),
    CONSTRAINT `ck_materiales_activo`
        CHECK (`activo` IN (0, 1))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Catalogo de materiales o tipos de aridos';

-- ============================================================================
-- 3. SOLICITUDES
-- ============================================================================

CREATE TABLE IF NOT EXISTS `solicitudes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno de la solicitud',
    `tipo_referencia` VARCHAR(20) NOT NULL
        COMMENT 'cim o centro_costo',
    `codigo_referencia` VARCHAR(64) NOT NULL
        COMMENT 'Numero del CIM o codigo del centro de costo',
    `anio` SMALLINT UNSIGNED NOT NULL
        COMMENT 'Anio al que pertenece la referencia',
    `solicitante_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Solicitante asociado',
    `fecha_compra` DATE NOT NULL
        COMMENT 'Fecha de compra o autorizacion',
    `numero_memo` VARCHAR(64) NULL
        COMMENT 'Numero de memorando asociado',
    `fecha_asignacion` DATE NULL
        COMMENT 'Fecha asignada registrada en el origen',
    `numero_cea` VARCHAR(64) NULL
        COMMENT 'Numero del documento CEA asociado',
    `estado` VARCHAR(20) NOT NULL DEFAULT 'activa'
        COMMENT 'Estado administrativo: activa o anulada',
    `observaciones` TEXT NULL
        COMMENT 'Comentarios adicionales de la solicitud',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_solicitudes_referencia`
        (`tipo_referencia`, `codigo_referencia`, `anio`),
    KEY `ix_solicitudes_codigo` (`codigo_referencia`),
    KEY `ix_solicitudes_solicitante_fecha` (`solicitante_id`, `fecha_compra`),
    KEY `ix_solicitudes_anio_tipo` (`anio`, `tipo_referencia`),
    KEY `ix_solicitudes_fecha_compra` (`fecha_compra`),
    KEY `ix_solicitudes_estado_fecha` (`estado`, `fecha_compra`),

    CONSTRAINT `fk_solicitudes_solicitante`
        FOREIGN KEY (`solicitante_id`) REFERENCES `solicitantes` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `ck_solicitudes_tipo_referencia`
        CHECK (`tipo_referencia` IN ('cim', 'centro_costo')),
    CONSTRAINT `ck_solicitudes_codigo_referencia`
        CHECK (CHAR_LENGTH(TRIM(`codigo_referencia`)) > 0),
    CONSTRAINT `ck_solicitudes_anio`
        CHECK (`anio` BETWEEN 2000 AND 2200),
    CONSTRAINT `ck_solicitudes_estado`
        CHECK (`estado` IN ('activa', 'anulada'))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Solicitudes identificadas por CIM o centro de costo';

-- ============================================================================
-- 4. MATERIALES COMPRADOS POR SOLICITUD
-- ============================================================================

CREATE TABLE IF NOT EXISTS `solicitudes_materiales` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador de la linea de material comprado',
    `solicitud_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Solicitud a la que pertenece el material',
    `material_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Material comprado',
    `cantidad_comprada` DECIMAL(12,3) NOT NULL
        COMMENT 'Cantidad comprada o autorizada',
    `precio_unitario_aplicado_clp` BIGINT UNSIGNED NULL
        COMMENT 'Precio unitario aplicado en pesos chilenos',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_solicitudes_materiales` (`solicitud_id`, `material_id`),
    KEY `ix_solicitudes_materiales_material` (`material_id`),

    CONSTRAINT `fk_solicitudes_materiales_solicitud`
        FOREIGN KEY (`solicitud_id`) REFERENCES `solicitudes` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `fk_solicitudes_materiales_material`
        FOREIGN KEY (`material_id`) REFERENCES `materiales` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `ck_solicitudes_materiales_cantidad`
        CHECK (`cantidad_comprada` > 0)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Cantidades de materiales compradas en cada solicitud';

-- ============================================================================
-- 5. CHOFERES
-- ============================================================================

CREATE TABLE IF NOT EXISTS `choferes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno del chofer',
    `nombre` VARCHAR(255) NOT NULL
        COMMENT 'Nombre completo del chofer',
    `activo` TINYINT(1) NOT NULL DEFAULT 1
        COMMENT 'Indica si puede seleccionarse en nuevas entregas',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    KEY `ix_choferes_activo_nombre` (`activo`, `nombre`),
    KEY `ix_choferes_nombre` (`nombre`),

    CONSTRAINT `ck_choferes_nombre`
        CHECK (CHAR_LENGTH(TRIM(`nombre`)) > 0),
    CONSTRAINT `ck_choferes_activo`
        CHECK (`activo` IN (0, 1))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Catalogo de choferes';

-- ============================================================================
-- 6. VEHICULOS
-- ============================================================================

CREATE TABLE IF NOT EXISTS `vehiculos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno del vehiculo',
    `patente` VARCHAR(20) NOT NULL
        COMMENT 'Patente visible del vehiculo',
    `patente_normalizada` VARCHAR(20)
        GENERATED ALWAYS AS (
            UPPER(
                REPLACE(
                    REPLACE(
                        REPLACE(TRIM(`patente`), '-', ''),
                        ' ',
                        ''
                    ),
                    '.',
                    ''
                )
            )
        ) STORED
        COMMENT 'Patente sin separadores, usada para evitar duplicados',
    `descripcion` VARCHAR(255) NULL
        COMMENT 'Marca, modelo, empresa u otra referencia',
    `activo` TINYINT(1) NOT NULL DEFAULT 1
        COMMENT 'Indica si puede seleccionarse en nuevas entregas',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_vehiculos_patente_normalizada` (`patente_normalizada`),
    KEY `ix_vehiculos_activo_patente` (`activo`, `patente`),

    CONSTRAINT `ck_vehiculos_patente`
        CHECK (CHAR_LENGTH(TRIM(`patente`)) > 0),
    CONSTRAINT `ck_vehiculos_activo`
        CHECK (`activo` IN (0, 1))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Catalogo de vehiculos y camiones';

-- ============================================================================
-- 7. ENTREGAS
-- ============================================================================

CREATE TABLE IF NOT EXISTS `entregas` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador interno de la entrega',
    `solicitud_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Solicitud atendida por la entrega',
    `fecha_entrega` DATE NOT NULL
        COMMENT 'Fecha efectiva de entrega',
    `chofer_id` BIGINT UNSIGNED NULL
        COMMENT 'Chofer responsable de la entrega',
    `vehiculo_id` BIGINT UNSIGNED NULL
        COMMENT 'Vehiculo utilizado en la entrega',
    `estado` VARCHAR(20) NOT NULL DEFAULT 'confirmada'
        COMMENT 'Estado de la entrega: confirmada o anulada',
    `observaciones` TEXT NULL
        COMMENT 'Comentarios adicionales de la entrega',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',
    `actualizado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de ultima modificacion en UTC',

    PRIMARY KEY (`id`),
    KEY `ix_entregas_solicitud_estado_fecha`
        (`solicitud_id`, `estado`, `fecha_entrega`),
    KEY `ix_entregas_fecha_estado` (`fecha_entrega`, `estado`),
    KEY `ix_entregas_chofer` (`chofer_id`),
    KEY `ix_entregas_vehiculo` (`vehiculo_id`),

    CONSTRAINT `fk_entregas_solicitud`
        FOREIGN KEY (`solicitud_id`) REFERENCES `solicitudes` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `fk_entregas_chofer`
        FOREIGN KEY (`chofer_id`) REFERENCES `choferes` (`id`)
        ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT `fk_entregas_vehiculo`
        FOREIGN KEY (`vehiculo_id`) REFERENCES `vehiculos` (`id`)
        ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT `ck_entregas_estado`
        CHECK (`estado` IN ('confirmada', 'anulada'))
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Entregas parciales o completas asociadas a solicitudes';

-- ============================================================================
-- 8. MATERIALES INCLUIDOS EN CADA ENTREGA
-- ============================================================================

CREATE TABLE IF NOT EXISTS `entregas_materiales` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
        COMMENT 'Identificador de la linea de material entregado',
    `entrega_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Entrega a la que pertenece la cantidad',
    `solicitud_material_id` BIGINT UNSIGNED NOT NULL
        COMMENT 'Material comprado contra el cual se descuenta la entrega',
    `cantidad_entregada` DECIMAL(12,3) NOT NULL
        COMMENT 'Cantidad efectivamente entregada',
    `creado_en` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        COMMENT 'Fecha y hora de creacion en UTC',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_entregas_materiales`
        (`entrega_id`, `solicitud_material_id`),
    KEY `ix_entregas_materiales_solicitud_material`
        (`solicitud_material_id`, `entrega_id`),

    CONSTRAINT `fk_entregas_materiales_entrega`
        FOREIGN KEY (`entrega_id`) REFERENCES `entregas` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `fk_entregas_materiales_solicitud_material`
        FOREIGN KEY (`solicitud_material_id`) REFERENCES `solicitudes_materiales` (`id`)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT `ck_entregas_materiales_cantidad`
        CHECK (`cantidad_entregada` > 0)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Cantidades de materiales incluidas en cada entrega';

-- ============================================================================
-- DATOS INICIALES
-- Los precios quedan en NULL hasta confirmar la tarifa correcta.
-- Cambie la unidad si el negocio no trabaja en metros cubicos.
-- ============================================================================

INSERT IGNORE INTO `materiales`
    (`codigo`, `nombre`, `unidad_medida`, `precio_unitario_actual_clp`, `activo`)
VALUES
    ('arena',        'Arena',        'm3', NULL, 1),
    ('grava',        'Grava',        'm3', NULL, 1),
    ('gravilla',     'Gravilla',     'm3', NULL, 1),
    ('estabilizado', 'Estabilizado', 'm3', NULL, 1);

-- ============================================================================
-- VISTA: SALDO POR MATERIAL Y SOLICITUD
-- Solo considera entregas confirmadas y con la misma solicitud de origen.
-- ============================================================================

CREATE OR REPLACE SQL SECURITY INVOKER VIEW `vista_saldos_solicitud_material` AS
SELECT
    `sm`.`id` AS `solicitud_material_id`,
    `sm`.`solicitud_id`,
    `s`.`tipo_referencia`,
    `s`.`codigo_referencia`,
    `s`.`anio`,
    `s`.`estado` AS `estado_solicitud`,
    `sm`.`material_id`,
    `m`.`codigo` AS `codigo_material`,
    `m`.`nombre` AS `nombre_material`,
    `m`.`unidad_medida`,
    `sm`.`cantidad_comprada`,
    COALESCE(`entregado`.`cantidad_entregada`, 0.000) AS `cantidad_entregada`,
    `sm`.`cantidad_comprada`
        - COALESCE(`entregado`.`cantidad_entregada`, 0.000)
        AS `cantidad_pendiente`,
    CASE
        WHEN `s`.`estado` = 'anulada' THEN 'anulada'
        WHEN COALESCE(`entregado`.`cantidad_entregada`, 0.000) = 0
            THEN 'sin_entrega'
        WHEN COALESCE(`entregado`.`cantidad_entregada`, 0.000)
             > `sm`.`cantidad_comprada`
            THEN 'sobreentrega'
        WHEN COALESCE(`entregado`.`cantidad_entregada`, 0.000)
             = `sm`.`cantidad_comprada`
            THEN 'completa'
        ELSE 'parcial'
    END AS `estado_cumplimiento`,
    `sm`.`precio_unitario_aplicado_clp`,
    CASE
        WHEN `sm`.`precio_unitario_aplicado_clp` IS NULL THEN NULL
        ELSE ROUND(
            `sm`.`cantidad_comprada` * `sm`.`precio_unitario_aplicado_clp`,
            0
        )
    END AS `valor_comprado_clp`,
    CASE
        WHEN `sm`.`precio_unitario_aplicado_clp` IS NULL THEN NULL
        ELSE ROUND(
            COALESCE(`entregado`.`cantidad_entregada`, 0.000)
                * `sm`.`precio_unitario_aplicado_clp`,
            0
        )
    END AS `valor_entregado_clp`,
    CASE
        WHEN `sm`.`precio_unitario_aplicado_clp` IS NULL THEN NULL
        ELSE ROUND(
            (
                `sm`.`cantidad_comprada`
                - COALESCE(`entregado`.`cantidad_entregada`, 0.000)
            ) * `sm`.`precio_unitario_aplicado_clp`,
            0
        )
    END AS `valor_pendiente_clp`
FROM `solicitudes_materiales` AS `sm`
INNER JOIN `solicitudes` AS `s`
    ON `s`.`id` = `sm`.`solicitud_id`
INNER JOIN `materiales` AS `m`
    ON `m`.`id` = `sm`.`material_id`
LEFT JOIN (
    SELECT
        `em`.`solicitud_material_id`,
        SUM(`em`.`cantidad_entregada`) AS `cantidad_entregada`
    FROM `entregas_materiales` AS `em`
    INNER JOIN `entregas` AS `e`
        ON `e`.`id` = `em`.`entrega_id`
       AND `e`.`estado` = 'confirmada'
    INNER JOIN `solicitudes_materiales` AS `sm_validacion`
        ON `sm_validacion`.`id` = `em`.`solicitud_material_id`
       AND `sm_validacion`.`solicitud_id` = `e`.`solicitud_id`
    GROUP BY `em`.`solicitud_material_id`
) AS `entregado`
    ON `entregado`.`solicitud_material_id` = `sm`.`id`;

-- ============================================================================
-- VISTA: RESUMEN GENERAL POR SOLICITUD
-- ============================================================================

CREATE OR REPLACE SQL SECURITY INVOKER VIEW `vista_resumen_solicitudes` AS
SELECT
    `s`.`id` AS `solicitud_id`,
    `s`.`tipo_referencia`,
    `s`.`codigo_referencia`,
    `s`.`anio`,
    `s`.`solicitante_id`,
    `sol`.`nombre` AS `nombre_solicitante`,
    `s`.`fecha_compra`,
    `s`.`numero_memo`,
    `s`.`fecha_asignacion`,
    `s`.`numero_cea`,
    `s`.`estado` AS `estado_registro`,
    COALESCE(SUM(`v`.`cantidad_comprada`), 0.000)
        AS `cantidad_total_comprada`,
    COALESCE(SUM(`v`.`cantidad_entregada`), 0.000)
        AS `cantidad_total_entregada`,
    COALESCE(SUM(`v`.`cantidad_pendiente`), 0.000)
        AS `cantidad_total_pendiente`,
    CASE
        WHEN `s`.`estado` = 'anulada' THEN 'anulada'
        WHEN COALESCE(
            MAX(CASE WHEN `v`.`cantidad_pendiente` < 0 THEN 1 ELSE 0 END),
            0
        ) = 1 THEN 'sobreentrega'
        WHEN COALESCE(SUM(`v`.`cantidad_entregada`), 0.000) = 0
            THEN 'sin_entrega'
        WHEN COUNT(`v`.`material_id`) > 0
         AND MAX(CASE WHEN `v`.`cantidad_pendiente` <> 0 THEN 1 ELSE 0 END) = 0
            THEN 'completa'
        ELSE 'parcial'
    END AS `estado_cumplimiento`,
    CASE
        WHEN COALESCE(SUM(`v`.`cantidad_comprada`), 0.000) = 0 THEN 0.00
        ELSE ROUND(
            COALESCE(SUM(`v`.`cantidad_entregada`), 0.000)
                / SUM(`v`.`cantidad_comprada`) * 100,
            2
        )
    END AS `porcentaje_entregado`,
    CASE
        WHEN COUNT(`v`.`material_id`) = 0 THEN NULL
        WHEN SUM(
            CASE
                WHEN `v`.`precio_unitario_aplicado_clp` IS NULL THEN 1
                ELSE 0
            END
        ) > 0 THEN NULL
        ELSE SUM(`v`.`valor_comprado_clp`)
    END AS `valor_total_comprado_clp`,
    CASE
        WHEN COUNT(`v`.`material_id`) = 0 THEN NULL
        WHEN SUM(
            CASE
                WHEN `v`.`precio_unitario_aplicado_clp` IS NULL THEN 1
                ELSE 0
            END
        ) > 0 THEN NULL
        ELSE SUM(`v`.`valor_entregado_clp`)
    END AS `valor_total_entregado_clp`,
    CASE
        WHEN COUNT(`v`.`material_id`) = 0 THEN NULL
        WHEN SUM(
            CASE
                WHEN `v`.`precio_unitario_aplicado_clp` IS NULL THEN 1
                ELSE 0
            END
        ) > 0 THEN NULL
        ELSE SUM(`v`.`valor_pendiente_clp`)
    END AS `valor_total_pendiente_clp`,
    `s`.`observaciones`,
    `s`.`creado_en`,
    `s`.`actualizado_en`
FROM `solicitudes` AS `s`
INNER JOIN `solicitantes` AS `sol`
    ON `sol`.`id` = `s`.`solicitante_id`
LEFT JOIN `vista_saldos_solicitud_material` AS `v`
    ON `v`.`solicitud_id` = `s`.`id`
GROUP BY
    `s`.`id`,
    `s`.`tipo_referencia`,
    `s`.`codigo_referencia`,
    `s`.`anio`,
    `s`.`solicitante_id`,
    `sol`.`nombre`,
    `s`.`fecha_compra`,
    `s`.`numero_memo`,
    `s`.`fecha_asignacion`,
    `s`.`numero_cea`,
    `s`.`estado`,
    `s`.`observaciones`,
    `s`.`creado_en`,
    `s`.`actualizado_en`;

-- ============================================================================
-- COMPROBACION RAPIDA
-- ============================================================================

SELECT
    `codigo`,
    `nombre`,
    `unidad_medida`,
    `precio_unitario_actual_clp`,
    `activo`
FROM `materiales`
ORDER BY `id`;

SHOW FULL TABLES FROM `gestion_aridos`;
