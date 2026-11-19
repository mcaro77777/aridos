import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  return auth.comprobar().pipe(
    map((autenticado) => autenticado ? true : router.createUrlTree(['/acceso'], { queryParams: { retorno: state.url } })),
  );
};
