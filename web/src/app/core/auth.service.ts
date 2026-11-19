import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';
import { Sesion } from './modelos';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  readonly sesion = signal<Sesion | null>(null);
  readonly comprobada = signal(false);

  comprobar(): Observable<boolean> {
    if (this.comprobada()) return of(Boolean(this.sesion()?.autenticado));
    return this.http.get<Sesion>('/api/auth/session', { withCredentials: true }).pipe(
      tap((sesion) => {
        this.sesion.set(sesion);
        this.comprobada.set(true);
      }),
      map((sesion) => Boolean(sesion.autenticado)),
      catchError(() => {
        this.sesion.set(null);
        this.comprobada.set(true);
        return of(false);
      }),
    );
  }

  ingresar(usuario: string, contrasena: string): Observable<Sesion> {
    return this.http
      .post<Sesion>('/api/auth/login', { usuario, contrasena }, { withCredentials: true })
      .pipe(tap((sesion) => { this.sesion.set({ ...sesion, autenticado: true }); this.comprobada.set(true); }));
  }

  salir(): Observable<void> {
    return this.http.post<void>('/api/auth/logout', {}, { withCredentials: true }).pipe(
      tap(() => { this.sesion.set(null); this.comprobada.set(true); }),
      catchError(() => { this.sesion.set(null); return of(void 0); }),
    );
  }

  limpiar(): void {
    this.sesion.set(null);
    this.comprobada.set(true);
  }
}
