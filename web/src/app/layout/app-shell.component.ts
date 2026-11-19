import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { AuthService } from '../core/auth.service';
import { IconoComponent } from '../shared/icono.component';

@Component({
  selector: 'app-shell',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, IconoComponent],
  template: `
    <div class="shell" [class.shell--abierto]="menuAbierto()">
      <a class="saltar" href="#contenido">Saltar al contenido</a>
      <header class="barra-movil">
        <button class="boton-icono" type="button" (click)="alternarMenu()" aria-label="Abrir navegación"><app-icono nombre="menu" /></button>
        <a routerLink="/resumen" class="marca marca--movil" (click)="cerrarMenu()"><span class="marca__simbolo">A</span><span>Control de Áridos</span></a>
      </header>
      <aside class="lateral" aria-label="Navegación principal">
        <div class="lateral__cabecera">
          <a routerLink="/resumen" class="marca" (click)="cerrarMenu()">
            <span class="marca__simbolo">A</span><span><strong>Control de Áridos</strong><small>Gestión de entregas</small></span>
          </a>
          <button class="boton-icono lateral__cerrar" type="button" (click)="cerrarMenu()" aria-label="Cerrar navegación"><app-icono nombre="cerrar" /></button>
        </div>
        <nav class="navegacion">
          <span class="navegacion__etiqueta">Operación</span>
          <a routerLink="/resumen" routerLinkActive="activo" (click)="cerrarMenu()"><app-icono nombre="inicio" /><span>Resumen</span></a>
          <a routerLink="/solicitudes" routerLinkActive="activo" (click)="cerrarMenu()"><app-icono nombre="solicitudes" /><span>Solicitudes</span></a>
          <span class="navegacion__etiqueta">Configuración</span>
          <a routerLink="/catalogos" routerLinkActive="activo" (click)="cerrarMenu()"><app-icono nombre="catalogos" /><span>Catálogos</span></a>
        </nav>
        <div class="usuario">
          <span class="usuario__avatar">{{ inicialUsuario() }}</span>
          <span class="usuario__datos"><strong>{{ auth.sesion()?.nombre || auth.sesion()?.usuario || 'Usuario' }}</strong><small>Sesión activa</small></span>
          <button class="boton-icono" type="button" (click)="salir()" aria-label="Cerrar sesión"><app-icono nombre="salir" /></button>
        </div>
      </aside>
      @if (menuAbierto()) { <button class="velo" type="button" (click)="cerrarMenu()" aria-label="Cerrar navegación"></button> }
      <main id="contenido" class="contenido" tabindex="-1"><router-outlet /></main>
    </div>
  `,
  styleUrl: './app-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppShellComponent {
  readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  readonly menuAbierto = signal(false);
  inicialUsuario(): string { return (this.auth.sesion()?.nombre || this.auth.sesion()?.usuario || 'U').trim().charAt(0).toUpperCase(); }
  alternarMenu(): void { this.menuAbierto.update((valor) => !valor); }
  cerrarMenu(): void { this.menuAbierto.set(false); }
  salir(): void { this.auth.salir().subscribe(() => void this.router.navigate(['/acceso'])); }
}
