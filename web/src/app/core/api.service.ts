import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';
import {
  Chofer,
  Entrega,
  EntregaEntrada,
  Material,
  ResumenDashboard,
  Solicitante,
  SolicitudDetalle,
  SolicitudEntrada,
  SolicitudResumen,
  Vehiculo,
} from './modelos';

type RespuestaLista<T> = T[] | { datos: T[] } | { items: T[]; total?: number };

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly base = '/api';

  resumen(): Observable<ResumenDashboard> {
    return this.http.get<ResumenDashboard>(`${this.base}/resumen`);
  }

  solicitudes(filtros: Record<string, string | number | boolean | null | undefined> = {}): Observable<SolicitudResumen[]> {
    return this.http
      .get<RespuestaLista<SolicitudResumen>>(`${this.base}/solicitudes`, { params: this.parametros(filtros) })
      .pipe(map((respuesta) => this.extraerLista(respuesta)));
  }

  solicitud(id: number): Observable<SolicitudDetalle> {
    return this.http.get<SolicitudDetalle>(`${this.base}/solicitudes/${id}`);
  }

  crearSolicitud(datos: SolicitudEntrada): Observable<SolicitudDetalle> {
    return this.http.post<SolicitudDetalle>(`${this.base}/solicitudes`, datos);
  }

  actualizarSolicitud(id: number, datos: SolicitudEntrada): Observable<SolicitudDetalle> {
    return this.http.patch<SolicitudDetalle>(`${this.base}/solicitudes/${id}`, datos);
  }

  anularSolicitud(id: number, motivo: string): Observable<void> {
    return this.http.post<void>(`${this.base}/solicitudes/${id}/anular`, { motivo });
  }

  registrarEntrega(solicitudId: number, datos: EntregaEntrada): Observable<Entrega> {
    return this.http.post<Entrega>(`${this.base}/solicitudes/${solicitudId}/entregas`, datos);
  }

  anularEntrega(entregaId: number, motivo: string): Observable<void> {
    return this.http.post<void>(`${this.base}/entregas/${entregaId}/anular`, { motivo });
  }

  solicitantes(incluirInactivos = true): Observable<Solicitante[]> {
    return this.listar<Solicitante>('solicitantes', incluirInactivos);
  }

  materiales(incluirInactivos = true): Observable<Material[]> {
    return this.listar<Material>('materiales', incluirInactivos);
  }

  choferes(incluirInactivos = true): Observable<Chofer[]> {
    return this.listar<Chofer>('choferes', incluirInactivos);
  }

  vehiculos(incluirInactivos = true): Observable<Vehiculo[]> {
    return this.listar<Vehiculo>('vehiculos', incluirInactivos);
  }

  crear<T>(recurso: string, datos: Partial<T>): Observable<T> {
    return this.http.post<T>(`${this.base}/${recurso}`, datos);
  }

  actualizar<T>(recurso: string, id: number, datos: Partial<T>): Observable<T> {
    return this.http.patch<T>(`${this.base}/${recurso}/${id}`, datos);
  }

  private listar<T>(recurso: string, incluirInactivos: boolean): Observable<T[]> {
    return this.http
      .get<RespuestaLista<T>>(`${this.base}/${recurso}`, { params: { incluirInactivos } })
      .pipe(map((respuesta) => this.extraerLista(respuesta)));
  }

  private extraerLista<T>(respuesta: RespuestaLista<T>): T[] {
    if (Array.isArray(respuesta)) return respuesta;
    if ('datos' in respuesta) return respuesta.datos;
    return respuesta.items;
  }

  private parametros(filtros: Record<string, string | number | boolean | null | undefined>): HttpParams {
    let params = new HttpParams();
    for (const [clave, valor] of Object.entries(filtros)) {
      if (valor !== null && valor !== undefined && valor !== '') params = params.set(clave, String(valor));
    }
    return params;
  }
}
