import { ChangeDetectionStrategy, Component, input } from '@angular/core';

@Component({
  selector: 'app-icono',
  template: `
    <svg class="icono" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      @switch (nombre()) {
        @case ('inicio') { <path d="M3 10.7 12 3l9 7.7V21a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1V10.7Z" /> }
        @case ('solicitudes') { <path d="M6 3h12a2 2 0 0 1 2 2v16H4V5a2 2 0 0 1 2-2Zm2 5h8M8 12h8M8 16h5" /> }
        @case ('catalogos') { <path d="M4 4h6v6H4V4Zm10 0h6v6h-6V4ZM4 14h6v6H4v-6Zm10 0h6v6h-6v-6Z" /> }
        @case ('salir') { <path d="M10 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h5m5-4 4-4-4-4m4 4H8" /> }
        @case ('menu') { <path d="M4 7h16M4 12h16M4 17h16" /> }
        @case ('cerrar') { <path d="m6 6 12 12M18 6 6 18" /> }
        @case ('mas') { <path d="M12 5v14M5 12h14" /> }
        @case ('buscar') { <path d="m21 21-4.4-4.4m2.4-5.1a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z" /> }
        @case ('volver') { <path d="m15 18-6-6 6-6" /> }
        @case ('camion') { <path d="M3 6h11v11H3V6Zm11 4h4l3 3v4h-7v-7ZM7 20a2 2 0 1 0 0-4 2 2 0 0 0 0 4Zm10 0a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z" /> }
        @case ('editar') { <path d="m4 20 4.2-1 10.7-10.7a2 2 0 0 0-2.8-2.8L5.4 16.2 4 20Z" /> }
        @case ('alerta') { <path d="M12 3 2.5 20h19L12 3Zm0 6v5m0 3v.1" /> }
        @case ('cheque') { <path d="m5 12 4 4L19 6" /> }
        @case ('calendario') { <path d="M5 4h14a2 2 0 0 1 2 2v14H3V6a2 2 0 0 1 2-2Zm2-2v4m10-4v4M3 9h18" /> }
        @default { <circle cx="12" cy="12" r="8" /> }
      }
    </svg>
  `,
  styles: [`.icono{display:block;width:1.25rem;height:1.25rem;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}`],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class IconoComponent {
  readonly nombre = input.required<string>();
}
