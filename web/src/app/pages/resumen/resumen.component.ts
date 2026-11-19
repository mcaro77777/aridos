import { DatePipe, DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { mensajeError } from '../../core/errores';
import { ResumenDashboard } from '../../core/modelos';
import { EstadoVacioComponent } from '../../shared/estado-vacio.component';
import { IconoComponent } from '../../shared/icono.component';

@Component({
  selector: 'app-resumen',
  imports: [RouterLink, DatePipe, DecimalPipe, EstadoVacioComponent, IconoComponent],
  template: `
    <header class="encabezado-pagina">
      <div><h1>Resumen operacional</h1><p>Una vista clara del avance de solicitudes y entregas.</p></div>
      <a class="boton" routerLink="/solicitudes/nueva"><app-icono nombre="mas" />Nueva solicitud</a>
    </header>
    @if (cargando()) { <div class="cargando" role="status">Preparando resumen…</div> }
    @else if (error()) {
      <div class="alerta alerta--error" role="alert">{{ error() }}</div>
      <button class="boton boton--secundario reintentar" type="button" (click)="cargar()">Reintentar</button>
    } @else { @if (resumen(); as datos) {
      <section class="metricas" aria-label="Indicadores principales">
        <article class="metrica"><span class="metrica__icono"><app-icono nombre="solicitudes" /></span><div><small>Solicitudes activas</small><strong>{{ datos.solicitudesActivas | number:'1.0-0' }}</strong><em>{{ datos.solicitudesPendientes }} con entrega pendiente</em></div></article>
        <article class="metrica"><span class="metrica__icono metrica__icono--verde"><app-icono nombre="cheque" /></span><div><small>Solicitudes completas</small><strong>{{ datos.solicitudesCompletas | number:'1.0-0' }}</strong><em>Finalizadas a la fecha</em></div></article>
        <article class="metrica"><span class="metrica__icono metrica__icono--dorado"><app-icono nombre="camion" /></span><div><small>Entregas este mes</small><strong>{{ datos.entregasMes | number:'1.0-0' }}</strong><em>Despachos confirmados</em></div></article>
        <article class="metrica metrica--destacada"><div><small>Avance general</small><strong>{{ datos.porcentajeGeneral | number:'1.0-0' }}%</strong><div class="barra-progreso"><span [style.width.%]="datos.porcentajeGeneral"></span></div><em>{{ datos.cantidadEntregada | number:'1.0-2' }} de {{ datos.cantidadComprada | number:'1.0-2' }} entregado</em></div></article>
      </section>
      <div class="columnas">
        <section class="tarjeta">
          <div class="tarjeta__cabecera"><div><h2>Avance por material</h2><small class="texto-suave">Cantidades compradas y despachadas</small></div></div>
          @if (datos.materiales.length === 0) { <app-estado-vacio titulo="Sin materiales todavía" detalle="Cuando registres solicitudes, aquí verás su avance." icono="catalogos" /> }
          @else {
            <div class="materiales">
              @for (material of datos.materiales; track material.materialId) {
                <article class="material">
                  <div class="material__titulo"><div><strong>{{ material.nombreMaterial }}</strong><small>{{ material.cantidadEntregada | number:'1.0-2' }} / {{ material.cantidadComprada | number:'1.0-2' }} {{ material.unidadMedida }}</small></div><b>{{ material.porcentajeEntregado | number:'1.0-0' }}%</b></div>
                  <div class="barra-progreso"><span [style.width.%]="material.porcentajeEntregado"></span></div>
                  <small class="texto-suave">Pendiente: {{ material.cantidadPendiente | number:'1.0-2' }} {{ material.unidadMedida }}</small>
                </article>
              }
            </div>
          }
        </section>
        <section class="tarjeta">
          <div class="tarjeta__cabecera"><div><h2>Solicitudes recientes</h2><small class="texto-suave">Últimos registros ingresados</small></div><a routerLink="/solicitudes">Ver todas</a></div>
          @if (datos.solicitudesRecientes.length === 0) { <app-estado-vacio titulo="Aún no hay solicitudes" detalle="Crea la primera para comenzar el seguimiento." /> }
          @else {
            <div class="recientes">
              @for (solicitud of datos.solicitudesRecientes; track solicitud.id) {
                <a class="reciente" [routerLink]="['/solicitudes', solicitud.id]">
                  <span class="reciente__fecha"><b>{{ solicitud.fechaCompra | date:'dd' }}</b><small>{{ solicitud.fechaCompra | date:'MMM' }}</small></span>
                  <span><strong>{{ solicitud.codigoReferencia }}</strong><small>{{ solicitud.nombreSolicitante }}</small></span>
                  <span class="insignia" [class]="'insignia insignia--' + solicitud.estadoCumplimiento">{{ etiquetaEstado(solicitud.estadoCumplimiento) }}</span>
                </a>
              }
            </div>
          }
        </section>
      </div>
    } }
  `,
  styleUrl: './resumen.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ResumenComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly resumen = signal<ResumenDashboard | null>(null);
  readonly cargando = signal(true);
  readonly error = signal('');
  ngOnInit(): void { this.cargar(); }
  cargar(): void {
    this.cargando.set(true); this.error.set('');
    this.api.resumen().pipe(finalize(() => this.cargando.set(false))).subscribe({ next: (datos) => this.resumen.set(datos), error: (error) => this.error.set(mensajeError(error, 'No pudimos cargar el resumen.')) });
  }
  etiquetaEstado(estado: string): string { return ({ sin_entrega: 'Sin entrega', parcial: 'Parcial', completa: 'Completa', sobreentrega: 'Sobreentrega', anulada: 'Anulada' } as Record<string, string>)[estado] ?? estado; }
}
