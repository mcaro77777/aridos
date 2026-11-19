import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { finalize } from 'rxjs';
import { AuthService } from '../../core/auth.service';
import { mensajeError } from '../../core/errores';

@Component({
  selector: 'app-acceso',
  imports: [ReactiveFormsModule],
  template: `
    <main class="acceso">
      <section class="presentacion" aria-labelledby="titulo-presentacion">
        <div class="presentacion__contenido">
          <span class="sello">Control operacional</span>
          <h1 id="titulo-presentacion">Cada entrega,<br><em>bajo control.</em></h1>
          <p>Administra solicitudes, materiales y despachos de áridos en un solo lugar, con información clara y trazable.</p>
          <div class="linea"><span></span><small>Gestión municipal · 2026</small></div>
        </div>
      </section>
      <section class="panel-acceso" aria-labelledby="titulo-acceso">
        <form class="formulario" [formGroup]="formulario" (ngSubmit)="ingresar()">
          <div class="marca"><span>A</span><strong>Control de Áridos</strong></div>
          <div class="formulario__titulo"><small>Bienvenido</small><h2 id="titulo-acceso">Inicia sesión</h2><p>Ingresa tus credenciales para continuar.</p></div>
          @if (error()) { <div class="alerta alerta--error" role="alert">{{ error() }}</div> }
          <div class="campo">
            <label for="usuario">Usuario</label>
            <input id="usuario" type="text" formControlName="usuario" autocomplete="username" placeholder="Tu usuario" [class.invalido]="enviado() && formulario.controls.usuario.invalid" />
            @if (enviado() && formulario.controls.usuario.invalid) { <small class="error-campo">Ingresa tu usuario.</small> }
          </div>
          <div class="campo">
            <label for="contrasena">Contraseña</label>
            <input id="contrasena" type="password" formControlName="contrasena" autocomplete="current-password" placeholder="Tu contraseña" [class.invalido]="enviado() && formulario.controls.contrasena.invalid" />
            @if (enviado() && formulario.controls.contrasena.invalid) { <small class="error-campo">Ingresa tu contraseña.</small> }
          </div>
          <button class="boton boton--acceso" type="submit" [disabled]="guardando()">{{ guardando() ? 'Ingresando…' : 'Ingresar al sistema' }}</button>
          <p class="ayuda">Si no puedes ingresar, solicita acceso al administrador del sistema.</p>
        </form>
      </section>
    </main>
  `,
  styleUrl: './acceso.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AccesoComponent {
  private readonly fb = inject(FormBuilder);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  readonly guardando = signal(false);
  readonly enviado = signal(false);
  readonly error = signal('');
  readonly formulario = this.fb.nonNullable.group({ usuario: ['', Validators.required], contrasena: ['', Validators.required] });

  ingresar(): void {
    this.enviado.set(true); this.error.set('');
    if (this.formulario.invalid) { this.formulario.markAllAsTouched(); return; }
    this.guardando.set(true);
    const { usuario, contrasena } = this.formulario.getRawValue();
    this.auth.ingresar(usuario.trim(), contrasena).pipe(finalize(() => this.guardando.set(false))).subscribe({
      next: () => void this.router.navigateByUrl(this.route.snapshot.queryParamMap.get('retorno') || '/resumen'),
      error: (error) => this.error.set(mensajeError(error, 'Usuario o contraseña incorrectos.')),
    });
  }
}
