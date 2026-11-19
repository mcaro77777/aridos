import { DatePipe, DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { ReactiveFormsModule, UntypedFormArray, UntypedFormBuilder, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { forkJoin, finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { mensajeError } from '../../core/errores';
import { Chofer, Entrega, EntregaEntrada, SolicitudDetalle, Vehiculo } from '../../core/modelos';
import { EstadoVacioComponent } from '../../shared/estado-vacio.component';
import { IconoComponent } from '../../shared/icono.component';

@Component({
  selector: 'app-solicitud-detalle',
  imports: [RouterLink, ReactiveFormsModule, DatePipe, DecimalPipe, EstadoVacioComponent, IconoComponent],
  template: `
    <header class="encabezado-pagina">
      <div><a class="volver" routerLink="/solicitudes"><app-icono nombre="volver" />Volver a solicitudes</a><h1>{{ solicitud()?.codigoReferencia || 'Detalle de solicitud' }}</h1>@if (solicitud(); as s) { <p>{{ s.nombreSolicitante }} · Compra del {{ s.fechaCompra | date:'dd/MM/yyyy' }}</p> }</div>
      @if (solicitud(); as s) { <div class="acciones"><a class="boton boton--secundario" [routerLink]="['/solicitudes', s.id, 'editar']"><app-icono nombre="editar" />Editar</a>@if (s.estadoRegistro !== 'anulada') { <button class="boton" type="button" (click)="abrirEntrega()"><app-icono nombre="camion" />Registrar entrega</button> }</div> }
    </header>
    @if (cargando()) { <div class="cargando">Cargando la solicitud…</div> }
    @else if (error()) { <div class="alerta alerta--error" role="alert">{{ error() }}</div><button class="boton boton--secundario reintentar" (click)="cargar()">Reintentar</button> }
    @else { @if (solicitud(); as s) {
      @if (s.estadoRegistro === 'anulada') { <div class="alerta alerta--error aviso-anulada"><app-icono nombre="alerta" />Esta solicitud está anulada. Sus datos se conservan para trazabilidad.</div> }
      <section class="resumen">
        <article class="avance-total"><div><small>Avance total</small><strong>{{ s.porcentajeEntregado | number:'1.0-0' }}%</strong></div><div class="barra-progreso"><span [style.width.%]="s.porcentajeEntregado"></span></div><p><b>{{ s.cantidadTotalEntregada | number:'1.0-2' }}</b> entregado de {{ s.cantidadTotalComprada | number:'1.0-2' }}</p></article>
        <article class="dato"><small>Estado</small><span [class]="'insignia insignia--' + s.estadoCumplimiento">{{ etiquetaEstado(s.estadoCumplimiento) }}</span></article>
        <article class="dato"><small>Pendiente</small><strong>{{ s.cantidadTotalPendiente | number:'1.0-2' }}</strong></article>
        <article class="dato"><small>Valor de compra</small><strong>{{ s.valorTotalCompradoClp == null ? 'Sin información' : ('$' + (s.valorTotalCompradoClp | number:'1.0-0')) }}</strong></article>
      </section>
      <div class="columnas">
        <div>
          <section class="tarjeta seccion">
            <div class="tarjeta__cabecera"><div><h2>Avance por material</h2><small class="texto-suave">Balance calculado con entregas confirmadas</small></div></div>
            <div class="materiales">
              @for (material of s.materiales; track material.materialId) {
                <article class="material">
                  <div class="material__cabecera"><div><strong>{{ material.nombreMaterial }}</strong><small>{{ material.codigoMaterial }} · {{ material.unidadMedida }}</small></div><span [class]="'insignia insignia--' + material.estadoCumplimiento">{{ etiquetaEstado(material.estadoCumplimiento) }}</span></div>
                  <div class="material__cifras"><span><small>Comprado</small><b>{{ material.cantidadComprada | number:'1.0-2' }}</b></span><span><small>Entregado</small><b>{{ material.cantidadEntregada | number:'1.0-2' }}</b></span><span><small>Pendiente</small><b>{{ material.cantidadPendiente | number:'1.0-2' }}</b></span></div>
                  <div class="barra-progreso"><span [style.width.%]="porcentajeMaterial(material.cantidadComprada, material.cantidadEntregada)"></span></div>
                </article>
              }
            </div>
          </section>
          <section class="tarjeta seccion">
            <div class="tarjeta__cabecera"><div><h2>Historial de entregas</h2><small class="texto-suave">{{ s.entregas.length }} registro{{ s.entregas.length === 1 ? '' : 's' }}</small></div></div>
            @if (s.entregas.length === 0) { <app-estado-vacio titulo="Aún no hay entregas" detalle="Registra el primer despacho para comenzar a descontar el saldo." icono="camion" /> }
            @else {
              <div class="entregas">
                @for (entrega of s.entregas; track entrega.id) {
                  <article class="entrega" [class.entrega--anulada]="entrega.estado === 'anulada'">
                    <div class="entrega__cabecera"><span class="entrega__icono"><app-icono nombre="camion" /></span><div><strong>Entrega #{{ entrega.id }}</strong><small>{{ entrega.fechaEntrega | date:'dd/MM/yyyy' }} · {{ entrega.nombreChofer || 'Sin chofer' }} · {{ entrega.patenteVehiculo || 'Sin vehículo' }}</small></div><span [class]="'insignia insignia--' + entrega.estado">{{ entrega.estado === 'confirmada' ? 'Confirmada' : 'Anulada' }}</span></div>
                    <div class="entrega__materiales">@for (item of entrega.materiales; track item.materialId) { <span><b>{{ item.cantidadEntregada | number:'1.0-2' }}</b> {{ item.unidadMedida }} · {{ item.nombreMaterial }}</span> }</div>
                    @if (entrega.observaciones) { <p>{{ entrega.observaciones }}</p> }
                    @if (entrega.estado === 'confirmada') { <button class="boton boton--texto boton-anular" type="button" (click)="abrirAnulacionEntrega(entrega)">Anular entrega</button> }
                  </article>
                }
              </div>
            }
          </section>
        </div>
        <aside>
          <section class="tarjeta seccion datos">
            <div class="tarjeta__cabecera"><h2>Antecedentes</h2></div>
            <dl><div><dt>Tipo de referencia</dt><dd>{{ s.tipoReferencia === 'centro_costo' ? 'Centro de costo' : 'CIM' }}</dd></div><div><dt>Solicitante</dt><dd>{{ s.nombreSolicitante }}</dd></div><div><dt>Memo</dt><dd>{{ s.numeroMemo || '—' }}</dd></div><div><dt>Fecha de asignación</dt><dd>{{ s.fechaAsignacion ? (s.fechaAsignacion | date:'dd/MM/yyyy') : '—' }}</dd></div><div><dt>Número CEA</dt><dd>{{ s.numeroCea || '—' }}</dd></div></dl>
            @if (s.observaciones) { <div class="observaciones"><small>Observaciones</small><p>{{ s.observaciones }}</p></div> }
          </section>
          @if (s.estadoRegistro !== 'anulada') { <button class="boton boton--peligro anular-solicitud" type="button" (click)="abrirAnulacionSolicitud()">Anular solicitud</button> }
        </aside>
      </div>
    } }

    @if (modalEntrega()) {
      <div class="modal-fondo" role="presentation" (click)="cerrarEntrega($event)"><form class="modal modal-entrega" [formGroup]="formularioEntrega" (ngSubmit)="registrarEntrega()" role="dialog" aria-modal="true" aria-labelledby="titulo-entrega">
        <div class="modal__cabecera"><div><h2 id="titulo-entrega">Registrar entrega</h2><small class="texto-suave">Ingresa las cantidades efectivamente despachadas.</small></div><button class="cerrar-modal" type="button" (click)="modalEntrega.set(false)" aria-label="Cerrar"><app-icono nombre="cerrar" /></button></div>
        <div class="modal__cuerpo">
          @if (errorModal()) { <div class="alerta alerta--error">{{ errorModal() }}</div> }
          <div class="rejilla-formulario"><div class="campo"><label for="fechaEntrega">Fecha *</label><input id="fechaEntrega" type="date" formControlName="fechaEntrega" /></div><div class="campo"><label for="choferId">Chofer</label><select id="choferId" formControlName="choferId"><option [ngValue]="null">Sin especificar</option>@for (item of choferes(); track item.id) { <option [ngValue]="item.id">{{ item.nombre }}</option> }</select></div><div class="campo ancho-completo"><label for="vehiculoId">Vehículo</label><select id="vehiculoId" formControlName="vehiculoId"><option [ngValue]="null">Sin especificar</option>@for (item of vehiculos(); track item.id) { <option [ngValue]="item.id">{{ item.patente }}{{ item.descripcion ? ' · ' + item.descripcion : '' }}</option> }</select></div></div>
          <fieldset class="cantidades" formArrayName="materiales"><legend>Cantidades por material</legend>@for (grupo of materialesEntrega.controls; track grupo; let indice = $index) { <div class="cantidad" [formGroupName]="indice"><label [for]="'cantidad-entrega-' + indice"><span>{{ nombreMaterial(numero(grupo.value.materialId)) }}</span><small>Pendiente: {{ pendienteMaterial(numero(grupo.value.materialId)) | number:'1.0-2' }}</small></label><input [id]="'cantidad-entrega-' + indice" type="number" min="0" step="0.001" formControlName="cantidadEntregada" placeholder="0" /></div> }</fieldset>
          <div class="campo"><label for="observacionesEntrega">Observaciones</label><textarea id="observacionesEntrega" formControlName="observaciones" maxlength="1000"></textarea></div>
        </div>
        <div class="modal__acciones"><button class="boton boton--secundario" type="button" (click)="modalEntrega.set(false)">Cancelar</button><button class="boton" type="submit" [disabled]="guardando()">{{ guardando() ? 'Registrando…' : 'Confirmar entrega' }}</button></div>
      </form></div>
    }
    @if (modalAnulacion()) {
      <div class="modal-fondo"><form class="modal modal--pequeno" [formGroup]="formularioAnulacion" (ngSubmit)="confirmarAnulacion()" role="dialog" aria-modal="true" aria-labelledby="titulo-anulacion"><div class="modal__cabecera"><h2 id="titulo-anulacion">{{ entregaAnular() ? 'Anular entrega' : 'Anular solicitud' }}</h2><button class="cerrar-modal" type="button" (click)="modalAnulacion.set(false)" aria-label="Cerrar"><app-icono nombre="cerrar" /></button></div><div class="modal__cuerpo"><p>Esta acción conservará el registro, pero dejará de considerarlo en los totales.</p>@if (errorModal()) { <div class="alerta alerta--error">{{ errorModal() }}</div> }<div class="campo"><label for="motivo">Motivo de anulación *</label><textarea id="motivo" formControlName="motivo" maxlength="500" placeholder="Explica brevemente el motivo"></textarea></div></div><div class="modal__acciones"><button class="boton boton--secundario" type="button" (click)="modalAnulacion.set(false)">Cancelar</button><button class="boton boton--peligro" type="submit" [disabled]="guardando()">{{ guardando() ? 'Anulando…' : 'Confirmar anulación' }}</button></div></form>
      </div>
    }
  `,
  styleUrl: './solicitud-detalle.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SolicitudDetalleComponent implements OnInit {
  private readonly api = inject(ApiService); private readonly route = inject(ActivatedRoute); private readonly router = inject(Router); private readonly fb = inject(UntypedFormBuilder);
  readonly solicitud = signal<SolicitudDetalle | null>(null); readonly choferes = signal<Chofer[]>([]); readonly vehiculos = signal<Vehiculo[]>([]); readonly cargando = signal(true); readonly guardando = signal(false); readonly error = signal(''); readonly errorModal = signal('');
  readonly modalEntrega = signal(false); readonly modalAnulacion = signal(false); readonly entregaAnular = signal<Entrega | null>(null);
  readonly formularioEntrega = this.fb.group({ fechaEntrega: [new Date().toISOString().slice(0,10), Validators.required], choferId: [null], vehiculoId: [null], observaciones: [''], materiales: this.fb.array([]) });
  readonly formularioAnulacion = this.fb.group({ motivo: ['', [Validators.required, Validators.minLength(3)]] });
  get materialesEntrega(): UntypedFormArray { return this.formularioEntrega.controls['materiales'] as UntypedFormArray; }
  ngOnInit(): void { this.cargar(); }
  cargar(): void {
    const id = Number(this.route.snapshot.paramMap.get('id')); if (!Number.isInteger(id) || id <= 0) { void this.router.navigate(['/solicitudes']); return; }
    this.cargando.set(true); this.error.set('');
    forkJoin({ solicitud: this.api.solicitud(id), choferes: this.api.choferes(false), vehiculos: this.api.vehiculos(false) }).pipe(finalize(() => this.cargando.set(false))).subscribe({ next: ({solicitud,choferes,vehiculos}) => { this.solicitud.set(solicitud); this.choferes.set(choferes); this.vehiculos.set(vehiculos); }, error: (error) => this.error.set(mensajeError(error, 'No pudimos cargar esta solicitud.')) });
  }
  porcentajeMaterial(comprada:number,entregada:number):number { return comprada > 0 ? Math.min((entregada/comprada)*100,100) : 0; }
  etiquetaEstado(e:string):string { return ({sin_entrega:'Sin entrega',parcial:'Parcial',completa:'Completa',sobreentrega:'Sobreentrega',anulada:'Anulada'} as Record<string,string>)[e]??e; }
  nombreMaterial(id:number):string { return this.solicitud()?.materiales.find((m)=>m.materialId===id)?.nombreMaterial ?? 'Material'; }
  pendienteMaterial(id:number):number { return this.solicitud()?.materiales.find((m)=>m.materialId===id)?.cantidadPendiente ?? 0; }
  numero(valor: unknown): number { return Number(valor); }
  abrirEntrega():void { this.errorModal.set(''); this.materialesEntrega.clear(); this.solicitud()?.materiales.forEach((m)=>this.materialesEntrega.push(this.fb.group({materialId:[m.materialId],cantidadEntregada:[null,[Validators.min(.001)]]}))); this.formularioEntrega.patchValue({fechaEntrega:new Date().toISOString().slice(0,10),choferId:null,vehiculoId:null,observaciones:''}); this.modalEntrega.set(true); }
  cerrarEntrega(event:MouseEvent):void { if(event.target===event.currentTarget)this.modalEntrega.set(false); }
  registrarEntrega():void {
    const v=this.formularioEntrega.getRawValue(); const materiales=v.materiales.filter((m:{cantidadEntregada:number|null})=>Number(m.cantidadEntregada)>0).map((m:{materialId:number;cantidadEntregada:number})=>({materialId:Number(m.materialId),cantidadEntregada:Number(m.cantidadEntregada)}));
    if(this.formularioEntrega.invalid||materiales.length===0){this.errorModal.set('Ingresa una cantidad mayor que cero para al menos un material.');return;}
    const datos:EntregaEntrada={fechaEntrega:v.fechaEntrega,choferId:v.choferId?Number(v.choferId):null,vehiculoId:v.vehiculoId?Number(v.vehiculoId):null,observaciones:v.observaciones?.trim()||null,materiales}; this.guardando.set(true); const id=this.solicitud()?.id; if(!id)return;
    this.api.registrarEntrega(id,datos).pipe(finalize(()=>this.guardando.set(false))).subscribe({next:()=>{this.modalEntrega.set(false);this.cargar();},error:(e)=>this.errorModal.set(mensajeError(e,'No pudimos registrar la entrega.'))});
  }
  abrirAnulacionSolicitud():void { this.entregaAnular.set(null);this.formularioAnulacion.reset({motivo:''});this.errorModal.set('');this.modalAnulacion.set(true); }
  abrirAnulacionEntrega(entrega:Entrega):void { this.entregaAnular.set(entrega);this.formularioAnulacion.reset({motivo:''});this.errorModal.set('');this.modalAnulacion.set(true); }
  confirmarAnulacion():void { if(this.formularioAnulacion.invalid){this.formularioAnulacion.markAllAsTouched();this.errorModal.set('Indica el motivo de la anulación.');return;} const motivo=this.formularioAnulacion.getRawValue().motivo.trim(); const entrega=this.entregaAnular(); const id=this.solicitud()?.id; if(!id)return; this.guardando.set(true); const peticion=entrega?this.api.anularEntrega(entrega.id,motivo):this.api.anularSolicitud(id,motivo); peticion.pipe(finalize(()=>this.guardando.set(false))).subscribe({next:()=>{this.modalAnulacion.set(false);this.cargar();},error:(e)=>this.errorModal.set(mensajeError(e,'No pudimos anular el registro.'))}); }
}
