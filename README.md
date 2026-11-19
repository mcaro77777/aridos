# Gestión de áridos

Aplicación web que reemplaza el libro Excel de entregas de áridos. La API está escrita en Go, el frontend en Angular 22 y los datos se guardan en MySQL. En producción, un único contenedor sirve la API y el Angular compilado.

## Estructura

```text
cmd/api/                         punto de entrada de la API Go
internal/                        lógica de negocio, acceso a datos y HTTP
web/                             código fuente Angular
ClientApp/                       Angular compilado servido por Go
documentacion/base_de_datos.sql  base, tablas, relaciones e índices MySQL
Dockerfile                       compilación reproducible para Easypanel
```

Los directorios `crm-api` y `crm-web` son referencias de arquitectura y no forman parte de esta aplicación.

## Requisitos

- Go 1.24 o superior compatible.
- Node.js 22.12 o superior y npm para trabajar en el frontend.
- MySQL 8.0.16 o superior.
- Docker, opcional pero recomendado para comprobar el mismo artefacto que se desplegará.

## 1. Crear la base de datos con DBeaver

El esquema versionado está en [`documentacion/base_de_datos.sql`](documentacion/base_de_datos.sql). Crea la base `gestion_aridos` y las tablas `solicitantes`, `materiales`, `solicitudes`, `solicitudes_materiales`, `choferes`, `vehiculos`, `entregas` y `entregas_materiales`.

1. Conéctate desde DBeaver a un servidor MySQL 8.
2. Abre `documentacion/base_de_datos.sql` en un editor SQL de esa conexión.
3. Ejecuta el archivo completo con **Ejecutar script SQL**. No ejecutes solamente la sentencia donde esté el cursor.
4. Actualiza el navegador de DBeaver y confirma que aparezca la base `gestion_aridos` con sus ocho tablas.

El script puede volver a ejecutarse porque usa `IF NOT EXISTS`. No crea usuarios MySQL ni incluye contraseñas; el usuario que use la API debe tener permisos sobre `gestion_aridos`.

Ejemplo ejecutado desde la consola, si se prefiere:

```bash
mysql -h HOST -u USUARIO -p < documentacion/base_de_datos.sql
```

## 2. Configuración

Copia `.env.example` como `.env` y reemplaza todos los valores que comienzan con `CAMBIAR_`. El archivo `.env` está excluido de Git y nunca debe subirse.

| Variable | Obligatoria | Descripción |
|---|---:|---|
| `DATABASE_DSN` | Sí | Conexión MySQL en formato del controlador Go: `usuario:clave@tcp(host:3306)/gestion_aridos?charset=utf8mb4&parseTime=true&loc=UTC`. |
| `APP_ADMIN_USER` | Sí | Nombre de acceso del administrador. La aplicación lo valida desde el entorno; no lo guarda en MySQL. |
| `APP_ADMIN_PASSWORD` | Sí | Clave larga y única del administrador, validada desde el entorno. |
| `APP_AUTH_SECRET` | Sí | Secreto aleatorio para proteger la autenticación; usa como mínimo 32 bytes. |
| `APP_ENV` | No | `development` en local o `production` en Easypanel. En producción habilita las protecciones de cookie correspondientes. |
| `APP_BASE_URL` | No | URL pública de la aplicación, por ejemplo `https://aridos.ejemplo.cl`. |
| `PORT` | No | Puerto HTTP; por defecto `8080`. |
| `CLIENT_DIR` | No | Carpeta del Angular compilado; por defecto `ClientApp`. En Docker es `/app/ClientApp`. |
| `SESSION_DURATION` | No | Duración de la sesión con sintaxis Go, por ejemplo `12h`. |

> La API lee variables de entorno. Un archivo `.env` no se carga automáticamente al ejecutar `go run`; expórtalo en tu terminal o configura esas variables en el IDE. Docker y Easypanel sí permiten cargarlas directamente.

## 3. Desarrollo local

Primero inicia la API con sus variables de entorno. En PowerShell:

```powershell
$env:APP_ENV = 'development'
$env:DATABASE_DSN = 'usuario_aridos:clave@tcp(localhost:3306)/gestion_aridos?charset=utf8mb4&parseTime=true&loc=UTC'
$env:APP_ADMIN_USER = 'admin'
$env:APP_ADMIN_PASSWORD = 'una-clave-local-larga'
$env:APP_AUTH_SECRET = 'un-secreto-local-de-al-menos-32-bytes'
$env:PORT = '8080'
go run ./cmd/api
```

En otra terminal inicia Angular:

```powershell
Set-Location web
npm ci
npm start
```

Abre `http://localhost:4200`. El servidor de desarrollo redirige `/api` a `http://localhost:8080` mediante `web/proxy.conf.json`.

## 4. Pruebas y compilación local

```powershell
go test ./...

Set-Location web
npm ci
npm run typecheck
npm run build
Set-Location ..

go build -trimpath -o aridos-api.exe ./cmd/api
```

`npm run build` recrea `ClientApp`. Esa carpeta se conserva en el repositorio para que el binario Go también pueda ejecutarse fuera de Docker; el Dockerfile siempre vuelve a compilarla desde `web`.

Para ejecutar el binario local, define las variables anteriores y luego usa:

```powershell
./aridos-api.exe
```

La comprobación de salud está disponible en `GET /api/health`.

## 5. Docker

Construye y ejecuta la imagen desde la raíz del repositorio:

```bash
docker build -t gestion-aridos:local .
docker run --rm --name gestion-aridos -p 8080:8080 --env-file .env gestion-aridos:local
```

Abre `http://localhost:8080` y comprueba `http://localhost:8080/api/health`.

El Dockerfile usa varias etapas: instala dependencias y compila Angular, compila un binario Go estático y copia únicamente el binario y `ClientApp` a una imagen Alpine. El proceso final se ejecuta con el usuario sin privilegios `aridos` y escucha en el puerto `8080`.

Si MySQL se ejecuta en tu equipo y la aplicación está dentro de Docker, `localhost` apunta al contenedor. En Windows o macOS usa `host.docker.internal` como host en `DATABASE_DSN`; en una red Docker usa el nombre del servicio MySQL.

## 6. Despliegue en Easypanel / Hostinger

Configura el servicio así:

| Opción | Valor |
|---|---|
| Repositorio | `mcaro77777/aridos` |
| Rama | `main` |
| Ruta de compilación | `/` |
| Método | Dockerfile del repositorio |
| Puerto interno | `8080` |
| Ruta de salud | `/api/health` |

En la sección de entorno agrega `DATABASE_DSN`, `APP_ADMIN_USER`, `APP_ADMIN_PASSWORD`, `APP_AUTH_SECRET`, `APP_ENV=production`, `APP_BASE_URL=https://tu-dominio` y, si se desea cambiar, `SESSION_DURATION`. `PORT=8080` y `CLIENT_DIR=/app/ClientApp` ya están definidos de forma segura en la imagen.

Para `DATABASE_DSN`, usa el nombre interno del servicio MySQL de Easypanel, no la IP pública ni `localhost`. La aplicación y MySQL deben compartir la misma red. No es necesario publicar el puerto `3306` a Internet.

Después de asociar el dominio y HTTPS, vuelve a desplegar si cambiaste `APP_BASE_URL`. Verifica primero `/api/health` y luego inicia sesión en la aplicación.

## Seguridad de despliegue

- Genera `APP_AUTH_SECRET` con un generador criptográfico y no lo reutilices en otros sistemas. Por ejemplo: `openssl rand -base64 48`.
- Usa claves distintas para MySQL y el administrador de la aplicación.
- Guarda secretos únicamente en las variables protegidas de Easypanel; no los escribas en el Dockerfile, en Git ni en capturas.
- Mantén MySQL accesible solo por la red interna y habilita copias de seguridad periódicas.
- Define credenciales administrativas definitivas antes de entregar acceso y usa siempre HTTPS en producción.
- El Excel original, proyectos de referencia, dependencias y archivos `.env` están excluidos tanto del repositorio como del contexto de Docker.
