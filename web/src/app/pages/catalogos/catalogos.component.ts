import { DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';
import { ReactiveFormsModule, UntypedFormBuilder, Validators } from '@angular/forms';
import { finalize, forkJoin } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { mensajeError } from '../../core/errores';
import { Catalogo, Chofer, Material, Solicitante, Vehiculo } from '../../core/modelos';
import { EstadoVacioComponent } from '../../shared/estado-vacio.component';
import { IconoComponent } from '../../shared/icono.component';

type TipoCatalogo = 'solicitantes' | 'materiales' | 'choferes' | 'vehiculos';

@Component({
  selector: 'app-catalogos',
  imports: [ReactiveFormsModule, DecimalPipe, EstadoVacioComponent, IconoComponent],
  template: `
    <header class="encabezado-pagina"><div><h1>Catálogos</h1><p>Administra las opciones disponibles al registrar solicitudes y entregas.</p></div><button class="boton" type="button" (click)="abrirNuevo()"><app-icono nombre="mas" />Agregar {{ singular() }}</button></header>
    <nav class="pestanas" aria-label="Catálogos">
      @for (item of tipos; track item.clave) { <button type="button" [class.activa]="tipo() === item.clave" (click)="seleccionar(item.clave)"><span>{{ item.nombre }}</span><b>{{ cantidad(item.clave) }}</b></button> }
    </nav>
    <section class="tarjeta">
      <div class="herramientas"><div class="campo campo--buscar"><label for="buscar-catalogo">Buscar en {{ titulo().toLowerCase() }}</label><span><app-icono nombre="buscar" /><input id="buscar-catalogo" type="search" [value]="busqueda()" (input)="busqueda.set(valorInput($event))" placeholder="Escribe para filtrar" /></span></div><label class="interruptor"><input type="checkbox" [checked]="mostrarInactivos()" (change)="mostrarInactivos.set(checkInput($event))" /><span>Mostrar inactivos</span></label></div>
      @if (error()) { <div class="alerta alerta--error error-lista">{{ error() }} <button class="boton boton--texto" type="button" (click)="cargarTodo()">Reintentar</button></div> }
      @if (cargando()) { <div class="cargando">Cargando catálogos…</div> }
      @else if (filtrados().length === 0) { <app-estado-vacio [titulo]="busqueda() ? 'Sin coincidencias' : 'Este catálogo está vacío'" detalle="Agrega un registro o modifica la búsqueda." icono="catalogos" /> }
      @else {
        <div class="tabla-contenedor">
          @switch (tipo()) {
            @case ('solicitantes') {
              <table class="tabla tabla--adaptable"><thead><tr><th>Nombre</th><th>Tipo</th><th>Estado</th><th><span class="solo-lector">Acciones</span></th></tr></thead><tbody>@for (fila of solicitantesFiltrados(); track fila.id) { <tr><td data-etiqueta="Nombre"><strong>{{ fila.nombre }}</strong></td><td data-etiqueta="Tipo">{{ etiquetaTipo(fila.tipo) }}</td><td data-etiqueta="Estado"><span [class]="'insignia insignia--' + (fila.activo ? 'activo' : 'inactivo')">{{ fila.activo ? 'Activo' : 'Inactivo' }}</span></td><td data-etiqueta="Acciones" class="acciones-tabla"><button class="boton boton--texto" (click)="abrirEditar(fila)">Editar</button><button class="boton boton--texto" [class.peligro]="fila.activo" (click)="alternarActivo(fila)">{{ fila.activo ? 'Desactivar' : 'Activar' }}</button></td></tr> }</tbody></table>
            }
            @case ('materiales') {
              <table class="tabla tabla--adaptable"><thead><tr><th>Código</th><th>Material</th><th>Unidad</th><th class="numero">Precio actual</th><th>Estado</th><th><span class="solo-lector">Acciones</span></th></tr></thead><tbody>@for (fila of materialesFiltrados(); track fila.id) { <tr><td data-etiqueta="Código"><b>{{ fila.codigo }}</b></td><td data-etiqueta="Material">{{ fila.nombre }}</td><td data-etiqueta="Unidad">{{ fila.unidadMedida }}</td><td data-etiqueta="Precio actual" class="numero">{{ fila.precioUnitarioActualClp == null ? '—' : ('$' + (fila.precioUnitarioActualClp | number:'1.0-0')) }}</td><td data-etiqueta="Estado"><span [class]="'insignia insignia--' + (fila.activo ? 'activo' : 'inactivo')">{{ fila.activo ? 'Activo' : 'Inactivo' }}</span></td><td data-etiqueta="Acciones" class="acciones-tabla"><button class="boton boton--texto" (click)="abrirEditar(fila)">Editar</button><button class="boton boton--texto" [class.peligro]="fila.activo" (click)="alternarActivo(fila)">{{ fila.activo ? 'Desactivar' : 'Activar' }}</button></td></tr> }</tbody></table>
            }
            @case ('choferes') {
              <table class="tabla tabla--adaptable"><thead><tr><th>Nombre</th><th>Estado</th><th><span class="solo-lector">Acciones</span></th></tr></thead><tbody>@for (fila of choferesFiltrados(); track fila.id) { <tr><td data-etiqueta="Nombre"><strong>{{ fila.nombre }}</strong></td><td data-etiqueta="Estado"><span [class]="'insignia insignia--' + (fila.activo ? 'activo' : 'inactivo')">{{ fila.activo ? 'Activo' : 'Inactivo' }}</span></td><td data-etiqueta="Acciones" class="acciones-tabla"><button class="boton boton--texto" (click)="abrirEditar(fila)">Editar</button><button class="boton boton--texto" [class.peligro]="fila.activo" (click)="alternarActivo(fila)">{{ fila.activo ? 'Desactivar' : 'Activar' }}</button></td></tr> }</tbody></table>
            }
            @case ('vehiculos') {
              <table class="tabla tabla--adaptable"><thead><tr><th>Patente</th><th>Descripción</th><th>Estado</th><th><span class="solo-lector">Acciones</span></th></tr></thead><tbody>@for (fila of vehiculosFiltrados(); track fila.id) { <tr><td data-etiqueta="Patente"><strong>{{ fila.patente }}</strong></td><td data-etiqueta="Descripción">{{ fila.descripcion || '—' }}</td><td data-etiqueta="Estado"><span [class]="'insignia insignia--' + (fila.activo ? 'activo' : 'inactivo')">{{ fila.activo ? 'Activo' : 'Inactivo' }}</span></td><td data-etiqueta="Acciones" class="acciones-tabla"><button class="boton boton--texto" (click)="abrirEditar(fila)">Editar</button><button class="boton boton--texto" [class.peligro]="fila.activo" (click)="alternarActivo(fila)">{{ fila.activo ? 'Desactivar' : 'Activar' }}</button></td></tr> }</tbody></table>
            }
          }
        </div>
      }
    </section>
    @if (modal()) {
      <div class="modal-fondo"><form class="modal" [formGroup]="formulario" (ngSubmit)="guardar()" role="dialog" aria-modal="true" aria-labelledby="titulo-modal"><div class="modal__cabecera"><h2 id="titulo-modal">{{ editando() ? 'Editar' : 'Agregar' }} {{ singular() }}</h2><button class="cerrar-modal" type="button" (click)="modal.set(false)" aria-label="Cerrar"><app-icono nombre="cerrar" /></button></div><div class="modal__cuerpo">
        @if (errorModal()) { <div class="alerta alerta--error error-modal">{{ errorModal() }}</div> }
        @switch (tipo()) {
          @case ('solicitantes') { <div class="rejilla-formulario"><div class="campo ancho-completo"><label for="nombre">Nombre *</label><input id="nombre" type="text" formControlName="nombre" maxlength="150" /></div><div class="campo ancho-completo"><label for="tipo">Tipo</label><select id="tipo" formControlName="tipo"><option value="persona">Persona</option><option value="empresa">Empresa</option><option value="institucion">Institución</option><option value="sin_clasificar">Sin clasificar</option></select></div></div> }
          @case ('materiales') { <div class="rejilla-formulario"><div class="campo"><label for="codigo">Código *</label><input id="codigo" type="text" formControlName="codigo" maxlength="30" /></div><div class="campo"><label for="unidadMedida">Unidad *</label><input id="unidadMedida" type="text" formControlName="unidadMedida" maxlength="20" placeholder="m³, ton, unidad" /></div><div class="campo ancho-completo"><label for="nombreMaterial">Nombre *</label><input id="nombreMaterial" type="text" formControlName="nombre" maxlength="120" /></div><div class="campo ancho-completo"><label for="precioUnitarioActualClp">Precio unitario actual CLP</label><input id="precioUnitarioActualClp" type="number" formControlName="precioUnitarioActualClp" min="0" step="1" /></div></div> }
          @case ('choferes') { <div class="campo"><label for="nombreChofer">Nombre *</label><input id="nombreChofer" type="text" formControlName="nombre" maxlength="150" /></div> }
          @case ('vehiculos') { <div class="rejilla-formulario"><div class="campo"><label for="patente">Patente *</label><input id="patente" type="text" formControlName="patente" maxlength="12" placeholder="ABCD12" /></div><div class="campo"><label for="descripcion">Descripción</label><input id="descripcion" type="text" formControlName="descripcion" maxlength="150" placeholder="Camión tolva" /></div></div> }
        }
      </div><div class="modal__acciones"><button class="boton boton--secundario" type="button" (click)="modal.set(false)">Cancelar</button><button class="boton" type="submit" [disabled]="guardando()">{{ guardando() ? 'Guardando…' : 'Guardar' }}</button></div></form></div>
    }
  `,
  styleUrl: './catalogos.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CatalogosComponent implements OnInit {
  private readonly api=inject(ApiService);private readonly fb=inject(UntypedFormBuilder);
  readonly tipos:Array<{clave:TipoCatalogo;nombre:string}>=[{clave:'solicitantes',nombre:'Solicitantes'},{clave:'materiales',nombre:'Materiales'},{clave:'choferes',nombre:'Choferes'},{clave:'vehiculos',nombre:'Vehículos'}];
  readonly tipo=signal<TipoCatalogo>('solicitantes');readonly busqueda=signal('');readonly mostrarInactivos=signal(false);readonly cargando=signal(true);readonly guardando=signal(false);readonly error=signal('');readonly errorModal=signal('');readonly modal=signal(false);readonly editando=signal<Catalogo|null>(null);
  readonly solicitantes=signal<Solicitante[]>([]);readonly materiales=signal<Material[]>([]);readonly choferes=signal<Chofer[]>([]);readonly vehiculos=signal<Vehiculo[]>([]);
  readonly titulo=computed(()=>this.tipos.find((t)=>t.clave===this.tipo())?.nombre??'Catálogo');readonly singular=computed(()=>({solicitantes:'solicitante',materiales:'material',choferes:'chofer',vehiculos:'vehículo'} as Record<TipoCatalogo,string>)[this.tipo()]);
  readonly formulario=this.fb.group({nombre:['',Validators.required],tipo:['sin_clasificar'],codigo:[''],unidadMedida:[''],precioUnitarioActualClp:[null],patente:[''],descripcion:['']});
  readonly solicitantesFiltrados=computed(()=>this.filtrar(this.solicitantes(),(x)=>`${x.nombre} ${x.tipo}`));readonly materialesFiltrados=computed(()=>this.filtrar(this.materiales(),(x)=>`${x.codigo} ${x.nombre} ${x.unidadMedida}`));readonly choferesFiltrados=computed(()=>this.filtrar(this.choferes(),(x)=>x.nombre));readonly vehiculosFiltrados=computed(()=>this.filtrar(this.vehiculos(),(x)=>`${x.patente} ${x.descripcion??''}`));
  readonly filtrados=computed<Catalogo[]>(()=>({solicitantes:this.solicitantesFiltrados(),materiales:this.materialesFiltrados(),choferes:this.choferesFiltrados(),vehiculos:this.vehiculosFiltrados()} as Record<TipoCatalogo,Catalogo[]>)[this.tipo()]);
  ngOnInit():void{this.cargarTodo();}
  cargarTodo():void{this.cargando.set(true);this.error.set('');forkJoin({solicitantes:this.api.solicitantes(true),materiales:this.api.materiales(true),choferes:this.api.choferes(true),vehiculos:this.api.vehiculos(true)}).pipe(finalize(()=>this.cargando.set(false))).subscribe({next:(r)=>{this.solicitantes.set(r.solicitantes);this.materiales.set(r.materiales);this.choferes.set(r.choferes);this.vehiculos.set(r.vehiculos);},error:(e)=>this.error.set(mensajeError(e,'No pudimos cargar los catálogos.'))});}
  seleccionar(tipo:TipoCatalogo):void{this.tipo.set(tipo);this.busqueda.set('');}
  cantidad(tipo:TipoCatalogo):number{return({solicitantes:this.solicitantes().length,materiales:this.materiales().length,choferes:this.choferes().length,vehiculos:this.vehiculos().length})[tipo];}
  filtrar<T extends {activo:boolean}>(datos:T[],texto:(item:T)=>string):T[]{const q=this.busqueda().trim().toLocaleLowerCase('es');return datos.filter((x)=>(this.mostrarInactivos()||x.activo)&&(q===''||texto(x).toLocaleLowerCase('es').includes(q)));}
  valorInput(e:Event):string{return(e.target as HTMLInputElement).value;}checkInput(e:Event):boolean{return(e.target as HTMLInputElement).checked;}
  etiquetaTipo(tipo:string):string{return({persona:'Persona',empresa:'Empresa',institucion:'Institución',sin_clasificar:'Sin clasificar'} as Record<string,string>)[tipo]??tipo;}
  abrirNuevo():void{this.editando.set(null);this.errorModal.set('');this.formulario.reset({nombre:'',tipo:'sin_clasificar',codigo:'',unidadMedida:'',precioUnitarioActualClp:null,patente:'',descripcion:''});this.configurarValidadores();this.modal.set(true);}
  abrirEditar(item:Catalogo):void{this.editando.set(item);this.errorModal.set('');this.formulario.reset({nombre:'nombre'in item?item.nombre:'',tipo:'tipo'in item?item.tipo:'sin_clasificar',codigo:'codigo'in item?item.codigo:'',unidadMedida:'unidadMedida'in item?item.unidadMedida:'',precioUnitarioActualClp:'precioUnitarioActualClp'in item?item.precioUnitarioActualClp:null,patente:'patente'in item?item.patente:'',descripcion:'descripcion'in item?item.descripcion??'':''});this.configurarValidadores();this.modal.set(true);}
  configurarValidadores():void{for(const nombre of ['nombre','codigo','unidadMedida','patente'])this.formulario.get(nombre)?.clearValidators();if(this.tipo()==='vehiculos')this.formulario.get('patente')?.setValidators(Validators.required);else this.formulario.get('nombre')?.setValidators(Validators.required);if(this.tipo()==='materiales'){this.formulario.get('codigo')?.setValidators(Validators.required);this.formulario.get('unidadMedida')?.setValidators(Validators.required);}this.formulario.updateValueAndValidity();}
  guardar():void{if(this.formulario.invalid){this.formulario.markAllAsTouched();this.errorModal.set('Completa los campos obligatorios.');return;}const v=this.formulario.getRawValue();let datos:Record<string,unknown>;if(this.tipo()==='solicitantes')datos={nombre:v.nombre.trim(),tipo:v.tipo};else if(this.tipo()==='materiales')datos={codigo:v.codigo.trim(),nombre:v.nombre.trim(),unidadMedida:v.unidadMedida.trim(),precioUnitarioActualClp:v.precioUnitarioActualClp==null||v.precioUnitarioActualClp===''?null:Number(v.precioUnitarioActualClp)};else if(this.tipo()==='choferes')datos={nombre:v.nombre.trim()};else datos={patente:v.patente.trim().toUpperCase(),descripcion:v.descripcion?.trim()||null};this.guardando.set(true);const actual=this.editando();const peticion=actual?this.api.actualizar(this.tipo(),actual.id,datos):this.api.crear(this.tipo(),datos);peticion.pipe(finalize(()=>this.guardando.set(false))).subscribe({next:()=>{this.modal.set(false);this.cargarTodo();},error:(e)=>this.errorModal.set(mensajeError(e,'No pudimos guardar el registro.'))});}
  alternarActivo(item:Catalogo):void{this.api.actualizar(this.tipo(),item.id,{activo:!item.activo}).subscribe({next:()=>this.cargarTodo(),error:(e)=>this.error.set(mensajeError(e,'No pudimos cambiar el estado.'))});}
}
