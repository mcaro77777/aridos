export type TipoSolicitante = 'persona' | 'empresa' | 'institucion' | 'sin_clasificar';
export type EstadoEntrega = 'confirmada' | 'anulada';
export type EstadoCumplimiento = 'sin_entrega' | 'parcial' | 'completa' | 'sobreentrega' | 'anulada';

export interface Sesion {
  autenticado: boolean;
  usuario: string;
  nombre?: string;
}

export interface Solicitante {
  id: number;
  nombre: string;
  tipo: TipoSolicitante;
  activo: boolean;
}

export interface Material {
  id: number;
  codigo: string;
  nombre: string;
  unidadMedida: string;
  precioUnitarioActualClp: number | null;
  activo: boolean;
}

export interface Chofer {
  id: number;
  nombre: string;
  activo: boolean;
}

export interface Vehiculo {
  id: number;
  patente: string;
  patenteNormalizada?: string;
  descripcion: string | null;
  activo: boolean;
}

export interface SolicitudMaterial {
  id?: number;
  materialId: number;
  codigoMaterial?: string;
  nombreMaterial: string;
  unidadMedida: string;
  cantidadComprada: number;
  cantidadEntregada: number;
  cantidadPendiente: number;
  precioUnitarioClp: number | null;
  valorTotalClp?: number | null;
  estadoCumplimiento: EstadoCumplimiento;
}

export interface EntregaMaterial {
  id?: number;
  materialId: number;
  nombreMaterial: string;
  unidadMedida: string;
  cantidadEntregada: number;
}

export interface Entrega {
  id: number;
  fechaEntrega: string;
  choferId: number | null;
  nombreChofer: string | null;
  vehiculoId: number | null;
  patenteVehiculo: string | null;
  estado: EstadoEntrega;
  observaciones: string | null;
  motivoAnulacion?: string | null;
  materiales: EntregaMaterial[];
}

export interface SolicitudResumen {
  id: number;
  tipoReferencia: 'cim' | 'centro_costo';
  codigoReferencia: string;
  anio: number;
  solicitanteId: number;
  nombreSolicitante: string;
  fechaCompra: string;
  numeroMemo: string | null;
  fechaAsignacion: string | null;
  numeroCea: string | null;
  observaciones?: string | null;
  estadoRegistro: 'activa' | 'anulada';
  estadoCumplimiento: EstadoCumplimiento;
  cantidadTotalComprada: number;
  cantidadTotalEntregada: number;
  cantidadTotalPendiente: number;
  porcentajeEntregado: number;
  valorTotalCompradoClp?: number | null;
}

export interface SolicitudDetalle extends SolicitudResumen {
  materiales: SolicitudMaterial[];
  entregas: Entrega[];
}

export interface ResumenDashboard {
  solicitudesActivas: number;
  solicitudesPendientes: number;
  solicitudesCompletas: number;
  entregasMes: number;
  cantidadComprada: number;
  cantidadEntregada: number;
  porcentajeGeneral: number;
  solicitudesRecientes: SolicitudResumen[];
  materiales: Array<{
    materialId: number;
    nombreMaterial: string;
    unidadMedida: string;
    cantidadComprada: number;
    cantidadEntregada: number;
    cantidadPendiente: number;
    porcentajeEntregado: number;
  }>;
}

export interface MaterialSolicitudEntrada {
  materialId: number;
  cantidadComprada: number;
  precioUnitarioClp: number | null;
}

export interface SolicitudEntrada {
  tipoReferencia: 'cim' | 'centro_costo';
  codigoReferencia: string;
  solicitanteId: number;
  fechaCompra: string;
  numeroMemo: string | null;
  fechaAsignacion: string | null;
  numeroCea: string | null;
  observaciones: string | null;
  materiales: MaterialSolicitudEntrada[];
}

export interface EntregaEntrada {
  fechaEntrega: string;
  choferId: number | null;
  vehiculoId: number | null;
  observaciones: string | null;
  materiales: Array<{ materialId: number; cantidadEntregada: number }>;
}

export interface ErrorApi {
  error?: { codigo?: string; mensaje?: string; detalles?: unknown } | string;
  mensaje?: string;
}

export type Catalogo = Solicitante | Material | Chofer | Vehiculo;
