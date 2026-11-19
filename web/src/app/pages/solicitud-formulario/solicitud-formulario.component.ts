import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { ReactiveFormsModule, UntypedFormArray, UntypedFormBuilder, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { forkJoin, finalize, of } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { mensajeError } from '../../core/errores';
import { Material, Solicitante, SolicitudEntrada } from '../../core/modelos';
import { IconoComponent } from '../../shared/icono.component';

@Component({
  selector: 'app-solicitud-formulario',
  imports: [ReactiveFormsModule, RouterLink, IconoComponent],
  template: `
    <header class="encabezado-pagina">
      <div><a class="volver" [routerLink]="id() ? ['/solicitudes', id()] : ['/solicitudes']"><app-icono nombre="volver" />Volver</a><h1>{{ id() ? 'Editar solicitud' : 'Nueva solicitud' }}</h1><p>{{ id() ? 'Actualiza los antecedentes generales de la compra.' : 'Registra la compra y los materiales que deberán entregarse.' }}</p></div>
    </header>
    @if (cargando()) { <div class="cargando">Preparando formulario…</div> }
    @else {
      <form [formGroup]="formulario" (ngSubmit)="guardar()" novalidate>
        @if (error()) { <div class="alerta alerta--error" role="alert">{{ error() }}</div> }
        <section class="tarjeta bloque">
          <div class="tarjeta__cabecera"><div><span class="paso">01</span><h2>Identificación de la solicitud</h2></div></div>
          <div class="tarjeta__contenido rejilla-formulario">
            <div class="campo"><label for="tipoReferencia">Tipo de referencia *</label><select id="tipoReferencia" formControlName="tipoReferencia"><option value="cim">CIM</option><option value="centro_costo">Centro de costo</option></select></div>
            <div class="campo"><label for="codigoReferencia">Código de referencia *</label><input id="codigoReferencia" type="text" formControlName="codigoReferencia" maxlength="50" placeholder="Ej.: CIM-2026-014" [class.invalido]="mostrarError('codigoReferencia')" />@if (mostrarError('codigoReferencia')) { <small class="error-campo">Ingresa el código.</small> }</div>
            <div class="campo"><label for="solicitanteId">Solicitante *</label><select id="solicitanteId" formControlName="solicitanteId" [class.invalido]="mostrarError('solicitanteId')"><option [ngValue]="null">Selecciona una opción</option>@for (item of solicitantes(); track item.id) { <option [ngValue]="item.id">{{ item.nombre }}</option> }</select>@if (mostrarError('solicitanteId')) { <small class="error-campo">Selecciona el solicitante.</small> }</div>
            <div class="campo"><label for="fechaCompra">Fecha de compra *</label><input id="fechaCompra" type="date" formControlName="fechaCompra" [class.invalido]="mostrarError('fechaCompra')" /></div>
            <div class="campo"><label for="numeroMemo">Número de memo</label><input id="numeroMemo" type="text" formControlName="numeroMemo" maxlength="50" /></div>
            <div class="campo"><label for="fechaAsignacion">Fecha de asignación</label><input id="fechaAsignacion" type="date" formControlName="fechaAsignacion" /></div>
            <div class="campo"><label for="numeroCea">Número CEA</label><input id="numeroCea" type="text" formControlName="numeroCea" maxlength="50" /></div>
            <div class="campo ancho-completo"><label for="observaciones">Observaciones</label><textarea id="observaciones" formControlName="observaciones" maxlength="1000" placeholder="Información adicional de la solicitud"></textarea></div>
          </div>
        </section>
        <section class="tarjeta bloque">
          <div class="tarjeta__cabecera"><div><span class="paso">02</span><h2>Materiales comprados</h2><small class="texto-suave">Agrega al menos un material con su cantidad.</small></div><button class="boton boton--secundario" type="button" (click)="agregarMaterial()"><app-icono nombre="mas" />Agregar material</button></div>
          <div class="materiales" formArrayName="materiales">
            @for (grupo of materialesArray.controls; track grupo; let indice = $index) {
              <div class="fila-material" [formGroupName]="indice">
                <span class="numero-fila">{{ indice + 1 }}</span>
                <div class="campo"><label [for]="'material-' + indice">Material *</label><select [id]="'material-' + indice" formControlName="materialId"><option [ngValue]="null">Selecciona</option>@for (item of materialesCatalogo(); track item.id) { <option [ngValue]="item.id">{{ item.codigo }} · {{ item.nombre }} ({{ item.unidadMedida }})</option> }</select></div>
                <div class="campo"><label [for]="'cantidad-' + indice">Cantidad comprada *</label><input [id]="'cantidad-' + indice" type="number" formControlName="cantidadComprada" min="0.001" step="0.001" /></div>
                <div class="campo"><label [for]="'precio-' + indice">Precio unitario CLP</label><input [id]="'precio-' + indice" type="number" formControlName="precioUnitarioClp" min="0" step="1" /></div>
                <button class="quitar" type="button" (click)="quitarMaterial(indice)" [disabled]="materialesArray.length === 1" aria-label="Quitar material"><app-icono nombre="cerrar" /></button>
              </div>
            }
          </div>
        </section>
        <div class="pie-formulario"><a class="boton boton--secundario" [routerLink]="id() ? ['/solicitudes', id()] : ['/solicitudes']">Cancelar</a><button class="boton" type="submit" [disabled]="guardando()">{{ guardando() ? 'Guardando…' : (id() ? 'Guardar cambios' : 'Crear solicitud') }}</button></div>
      </form>
    }
  `,
  styleUrl: './solicitud-formulario.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SolicitudFormularioComponent implements OnInit {
  private readonly fb = inject(UntypedFormBuilder);
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  readonly id = signal<number | null>(null);
  readonly solicitantes = signal<Solicitante[]>([]);
  readonly materialesCatalogo = signal<Material[]>([]);
  readonly cargando = signal(true);
  readonly guardando = signal(false);
  readonly enviado = signal(false);
  readonly error = signal('');
  readonly formulario = this.fb.group({
    tipoReferencia: ['cim', Validators.required], codigoReferencia: ['', Validators.required], solicitanteId: [null, Validators.required],
    fechaCompra: [new Date().toISOString().slice(0, 10), Validators.required], numeroMemo: [''], fechaAsignacion: [''], numeroCea: [''], observaciones: [''],
    materiales: this.fb.array([]),
  });
  get materialesArray(): UntypedFormArray { return this.formulario.controls['materiales'] as UntypedFormArray; }

  ngOnInit(): void {
    const parametro = this.route.snapshot.paramMap.get('id');
    const id = parametro ? Number(parametro) : null; this.id.set(id !== null && Number.isFinite(id) ? id : null);
    forkJoin({
      solicitantes: this.api.solicitantes(false),
      materiales: this.api.materiales(false),
      solicitud: this.id() ? this.api.solicitud(this.id() as number) : of(null),
    }).pipe(finalize(() => this.cargando.set(false))).subscribe({
      next: ({ solicitantes, materiales, solicitud }) => {
        this.solicitantes.set(solicitantes); this.materialesCatalogo.set(materiales);
        if (solicitud) {
          const detalle = solicitud;
          this.formulario.patchValue({ tipoReferencia: detalle.tipoReferencia, codigoReferencia: detalle.codigoReferencia, solicitanteId: detalle.solicitanteId, fechaCompra: detalle.fechaCompra?.slice(0,10), numeroMemo: detalle.numeroMemo ?? '', fechaAsignacion: detalle.fechaAsignacion?.slice(0,10) ?? '', numeroCea: detalle.numeroCea ?? '', observaciones: detalle.observaciones ?? '' });
          detalle.materiales.forEach((m) => this.materialesArray.push(this.crearGrupoMaterial(m.materialId, m.cantidadComprada, m.precioUnitarioClp)));
        }
        if (this.materialesArray.length === 0) this.agregarMaterial();
      },
      error: (error) => this.error.set(mensajeError(error, 'No pudimos preparar el formulario.')),
    });
  }
  crearGrupoMaterial(materialId: number | null = null, cantidad = 1, precio: number | null = null) { return this.fb.group({ materialId: [materialId, Validators.required], cantidadComprada: [cantidad, [Validators.required, Validators.min(.001)]], precioUnitarioClp: [precio, Validators.min(0)] }); }
  agregarMaterial(): void { this.materialesArray.push(this.crearGrupoMaterial()); }
  quitarMaterial(indice: number): void { if (this.materialesArray.length > 1) this.materialesArray.removeAt(indice); }
  mostrarError(nombre: string): boolean { const control = this.formulario.get(nombre); return Boolean(control?.invalid && (control.touched || this.enviado())); }
  guardar(): void {
    this.enviado.set(true); this.error.set('');
    if (this.formulario.invalid) { this.formulario.markAllAsTouched(); this.error.set('Revisa los campos marcados antes de guardar.'); return; }
    const v = this.formulario.getRawValue();
    const datos: SolicitudEntrada = { tipoReferencia: v.tipoReferencia, codigoReferencia: v.codigoReferencia.trim(), solicitanteId: Number(v.solicitanteId), fechaCompra: v.fechaCompra, numeroMemo: v.numeroMemo?.trim() || null, fechaAsignacion: v.fechaAsignacion || null, numeroCea: v.numeroCea?.trim() || null, observaciones: v.observaciones?.trim() || null, materiales: v.materiales.map((m: {materialId:number;cantidadComprada:number;precioUnitarioClp:number|null|string}) => ({ materialId: Number(m.materialId), cantidadComprada: Number(m.cantidadComprada), precioUnitarioClp: m.precioUnitarioClp === null || m.precioUnitarioClp === '' ? null : Number(m.precioUnitarioClp) })) };
    this.guardando.set(true);
    const peticion = this.id() ? this.api.actualizarSolicitud(this.id() as number, datos) : this.api.crearSolicitud(datos);
    peticion.pipe(finalize(() => this.guardando.set(false))).subscribe({ next: (solicitud) => void this.router.navigate(['/solicitudes', solicitud.id]), error: (error) => this.error.set(mensajeError(error, 'No pudimos guardar la solicitud.')) });
  }
}
