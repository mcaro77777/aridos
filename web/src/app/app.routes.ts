import { Routes } from '@angular/router';
import { authGuard } from './core/auth.guard';

export const routes: Routes = [
  {
    path: 'acceso',
    loadComponent: () => import('./pages/acceso/acceso.component').then((m) => m.AccesoComponent),
  },
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () => import('./layout/app-shell.component').then((m) => m.AppShellComponent),
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'resumen' },
      {
        path: 'resumen',
        loadComponent: () => import('./pages/resumen/resumen.component').then((m) => m.ResumenComponent),
      },
      {
        path: 'solicitudes',
        loadComponent: () => import('./pages/solicitudes/solicitudes.component').then((m) => m.SolicitudesComponent),
      },
      {
        path: 'solicitudes/nueva',
        loadComponent: () => import('./pages/solicitud-formulario/solicitud-formulario.component').then((m) => m.SolicitudFormularioComponent),
      },
      {
        path: 'solicitudes/:id/editar',
        loadComponent: () => import('./pages/solicitud-formulario/solicitud-formulario.component').then((m) => m.SolicitudFormularioComponent),
      },
      {
        path: 'solicitudes/:id',
        loadComponent: () => import('./pages/solicitud-detalle/solicitud-detalle.component').then((m) => m.SolicitudDetalleComponent),
      },
      {
        path: 'catalogos',
        loadComponent: () => import('./pages/catalogos/catalogos.component').then((m) => m.CatalogosComponent),
      },
    ],
  },
  { path: '**', redirectTo: 'resumen' },
];
