import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { IconoComponent } from './icono.component';

@Component({
  selector: 'app-estado-vacio',
  imports: [IconoComponent],
  template: `
    <div class="estado-vacio" role="status">
      <span class="estado-vacio__icono"><app-icono [nombre]="icono()" /></span>
      <strong>{{ titulo() }}</strong>
      <p>{{ detalle() }}</p>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EstadoVacioComponent {
  readonly titulo = input.required<string>();
  readonly detalle = input.required<string>();
  readonly icono = input('solicitudes');
}
