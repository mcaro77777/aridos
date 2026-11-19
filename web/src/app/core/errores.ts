import { HttpErrorResponse } from '@angular/common/http';
import { ErrorApi } from './modelos';

export function mensajeError(error: unknown, alternativo = 'No fue posible completar la operación.'): string {
  if (error instanceof HttpErrorResponse) {
    const cuerpo = error.error as ErrorApi | string | null;
    if (typeof cuerpo === 'string' && cuerpo.trim()) return cuerpo;
    if (cuerpo && typeof cuerpo === 'object') {
      if (typeof cuerpo.error === 'string') return cuerpo.error;
      if (cuerpo.error && typeof cuerpo.error === 'object' && cuerpo.error.mensaje) return cuerpo.error.mensaje;
      if (cuerpo.mensaje) return cuerpo.mensaje;
    }
    if (error.status === 0) return 'No hay conexión con el servidor. Revisa tu red e intenta nuevamente.';
    if (error.status === 403) return 'No tienes permisos para realizar esta acción.';
    if (error.status === 404) return 'El registro solicitado ya no existe.';
  }
  return alternativo;
}
