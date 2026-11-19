import { DatePipe, DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { debounceTime, distinctUntilChanged, finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { mensajeError } from '../../core/errores';
import { Solicitante, SolicitudResumen } from '../../core/modelos';
import { EstadoVacioComponent } from '../../shared/estado-vacio.component';
import { IconoComponent } from '../../shared/icono.component';

@Component({
  selector: 'app-solicitudes',
  imports: [ReactiveFormsModule, RouterLink, DatePipe, DecimalPipe, EstadoVacioComponent, IconoComponent],
  template: `
    <header class="encabezado-pagina">
      <div><h1>Solicitudes</h1><p>Consulta las compras y controla cuánto falta por entregar.</p></div>
      <a class="boton" routerLink="/solicitudes/nueva"><app-icono nombre="mas" />Nueva solicitud</a>
    </header>
    <section class="tarjeta filtros" [formGroup]="filtros" aria-label="Filtros de solicitudes">
      <div class="campo campo--buscar"><label for="buscar">Buscar</label><span><app-icono nombre="buscar" /><input id="buscar" type="search" formControlName="buscar" placeholder="Código, solicitante o memo" /></span></div>
      <div class="campo"><label for="estado">Estado de entrega</label><select id="estado" formControlName="estado"><option value="">Todos</option><option value="sin_entrega">Sin entrega</option><option value="parcial">Parcial</option><option value="completa">Completa</option><option value="sobreentrega">Sobreentrega</option><option value="anulada">Anulada</option></select></div>
      <div class="campo"><label for="solicitante">Solicitante</label><select id="solicitante" formControlName="solicitanteId"><option value="">Todos</option>@for (item of solicitantes(); track item.id) { <option [value]="item.id">{{ item.nombre }}</option> }</select></div>
      <div class="campo"><label for="anio">Año</label><input id="anio" type="number" min="2000" max="2100" formControlName="anio" /></div>
      <button class="boton boton--secundario limpiar" type="button" (click)="limpiar()">Limpiar</button>
    </section>
    @if (error()) { <div class="alerta alerta--error separacion" role="alert">{{ error() }} <button class="boton boton--texto" type="button" (click)="cargar()">Reintentar</button></div> }
    <section class="tarjeta listado" aria-live="polite">
      <div class="tarjeta__cabecera"><div><h2>Registros</h2><small class="texto-suave">{{ solicitudes().length }} solicitud{{ solicitudes().length === 1 ? '' : 'es' }}</small></div></div>
      @if (cargando()) { <div class="cargando">Buscando solicitudes…</div> }
      @else if (solicitudes().length === 0) { <app-estado-vacio titulo="No encontramos solicitudes" detalle="Prueba con otros filtros o registra una solicitud nueva." /> }
      @else {
        <div class="tabla-contenedor">
          <table class="tabla tabla--adaptable">
            <thead><tr><th>Referencia</th><th>Solicitante</th><th>Compra</th><th class="numero">Comprado</th><th class="numero">Entregado</th><th>Avance</th><th>Estado</th></tr></thead>
            <tbody>
              @for (solicitud of solicitudes(); track solicitud.id) {
                <tr>
                  <td data-etiqueta="Referencia">
                    <div class="referencia-contenido">
                      <a class="referencia-enlace" [routerLink]="['/solicitudes', solicitud.id]" [attr.aria-label]="'Abrir detalle de la solicitud ' + solicitud.codigoReferencia">
                        <span>{{ solicitud.codigoReferencia }}</span><span class="referencia-accion">Ver detalle <b aria-hidden="true">→</b></span>
                      </a>
                      <small>{{ etiquetaTipo(solicitud.tipoReferencia) }}</small>
                    </div>
                  </td>
                  <td data-etiqueta="Solicitante">{{ solicitud.nombreSolicitante }}</td>
                  <td data-etiqueta="Compra">{{ solicitud.fechaCompra | date:'dd/MM/yyyy' }}</td>
                  <td data-etiqueta="Comprado" class="numero">{{ solicitud.cantidadTotalComprada | number:'1.0-2' }}</td>
                  <td data-etiqueta="Entregado" class="numero">{{ solicitud.cantidadTotalEntregada | number:'1.0-2' }}</td>
                  <td data-etiqueta="Avance"><div class="avance"><span>{{ solicitud.porcentajeEntregado | number:'1.0-0' }}%</span><div class="barra-progreso"><i [style.width.%]="solicitud.porcentajeEntregado"></i></div></div></td>
                  <td data-etiqueta="Estado"><span [class]="'insignia insignia--' + solicitud.estadoCumplimiento">{{ etiquetaEstado(solicitud.estadoCumplimiento) }}</span></td>
                </tr>
              }
            </tbody>
          </table>
        </div>
      }
    </section>
  `,
  styleUrl: './solicitudes.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SolicitudesComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  readonly solicitudes = signal<SolicitudResumen[]>([]);
  readonly solicitantes = signal<Solicitante[]>([]);
  readonly cargando = signal(true);
  readonly error = signal('');
  readonly filtros = this.fb.nonNullable.group({ buscar: '', estado: '', solicitanteId: '', anio: new Date().getFullYear() });
  private cargaActual = 0;

  ngOnInit(): void {
    this.api.solicitantes(false).subscribe({ next: (datos) => this.solicitantes.set(datos) });
    this.filtros.valueChanges.pipe(debounceTime(250), distinctUntilChanged((anterior, actual) => JSON.stringify(anterior) === JSON.stringify(actual))).subscribe(() => this.cargar());
    this.cargar();
  }
  cargar(): void {
    const carga = ++this.cargaActual;
    this.cargando.set(true); this.error.set('');
    const f = this.filtros.getRawValue();
    this.api.solicitudes({ busqueda: f.buscar.trim(), estado: f.estado, solicitanteId: f.solicitanteId, anio: f.anio }).pipe(finalize(() => { if (carga === this.cargaActual) this.cargando.set(false); })).subscribe({
      next: (datos) => { if (carga === this.cargaActual) this.solicitudes.set(datos); },
      error: (error) => { if (carga === this.cargaActual) { this.solicitudes.set([]); this.error.set(mensajeError(error, 'No pudimos cargar las solicitudes.')); } },
    });
  }
  limpiar(): void { this.filtros.reset({ buscar: '', estado: '', solicitanteId: '', anio: new Date().getFullYear() }); }
  etiquetaEstado(estado: string): string { return ({ sin_entrega: 'Sin entrega', parcial: 'Parcial', completa: 'Completa', sobreentrega: 'Sobreentrega', anulada: 'Anulada' } as Record<string,string>)[estado] ?? estado; }
  etiquetaTipo(tipo: string): string { return tipo === 'centro_costo' ? 'Centro de costo' : 'CIM'; }
}
