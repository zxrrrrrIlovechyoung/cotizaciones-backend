# Guía de desarrollo — Sistema de Cotizaciones

Bienvenido/a. Este documento te explica, de cero, qué es el proyecto, cómo
levantarlo en tu máquina, cómo está organizado y cómo trabajamos. Está pensado
para alguien que se integra por primera vez.

> Documento relacionado: [`DEUDA_TECNICA.md`](./DEUDA_TECNICA.md) lista las
> mejoras y pendientes conocidos.

---

## 1. ¿Qué es?

Un sistema para que una empresa (un taller) registre **clientes** y elabore
**cotizaciones** (presupuestos con sus partidas, totales e IVA). Tiene inicio de
sesión y un área de cuenta del usuario.

## 2. Arquitectura general

Son **dos repositorios independientes** más una base de datos en contenedor:

| Componente | Tecnología | Puerto |
|---|---|---|
| Backend (API REST) | Go · Fiber v3 · pgx (PostgreSQL) · JWT · bcrypt | 3001 |
| Frontend (SPA) | React 19 · TypeScript · Vite · React Router · Axios | 5173 |
| Base de datos | PostgreSQL 16 en Docker (`postgres:16-alpine`) | 5433 → 5432 |

El backend sigue una **arquitectura hexagonal** (capas `domain` → `usecase` →
`infrastructure`): la lógica de negocio no conoce ni a Postgres ni a HTTP; esas
son implementaciones que se inyectan desde el exterior.

## 3. Requisitos previos

- **Go** (1.25+)
- **Node.js** (22+) y **pnpm** (se activa con `corepack`)
- **Docker Desktop** (para la base de datos)
- Opcional: un cliente de PostgreSQL (psql o pgAdmin) para inspeccionar la BD

## 4. Puesta en marcha (paso a paso)

### 4.1 Variables de entorno

Cada repo trae un `.env.example`. Copia a `.env`:

- **Backend** (`cotizaciones-backend/.env`): copia del ejemplo y deja
  `DB_PORT=5433` (el contenedor se expone en 5433 porque el 5432 suele estar
  ocupado por un PostgreSQL local).
- **Frontend** (`cotizaciones-frontend/.env`): `VITE_API_URL=http://localhost:3001/api`.

### 4.2 Base de datos (Docker)

Desde `cotizaciones-backend/`:

```bash
docker compose up -d
```

Esto levanta el contenedor `cotizaciones-postgres`. **Al primer arranque** (volumen
vacío) ejecuta automáticamente los archivos de `migrations/` en orden.

Datos de conexión: host `localhost`, puerto `5433`, base `cotizaciones_db`,
usuario `cotizaciones_user`, contraseña `cotizaciones_pass`.

> Si agregas una **migración nueva** y el contenedor ya tenía datos, no se aplica
> sola; córrela a mano:
> ```bash
> docker exec -i cotizaciones-postgres psql -U cotizaciones_user -d cotizaciones_db < migrations/00X_tu_migracion.sql
> ```

### 4.3 Backend

Desde `cotizaciones-backend/`:

```bash
go mod download
go run ./cmd/api
```

Debe imprimir que escucha en el puerto 3001. Prueba: `curl http://localhost:3001/api/health`.

### 4.4 Frontend

Desde `cotizaciones-frontend/`:

```bash
pnpm install
pnpm dev
```

Abre `http://localhost:5173`.

### 4.5 Usuario de prueba

`admin@cotizaciones.com` / `123456` (sembrado por la migración `001_init.sql`).

---

## 5. Estructura del backend

```
cmd/api/            -> punto de entrada (main.go): arma e inyecta todo
cmd/hashpw/         -> utilidad para generar hash bcrypt de una contraseña
internal/
  domain/           -> entidades y "puertos" (interfaces). Sin dependencias externas.
  usecase/          -> casos de uso (lógica de negocio). Depende solo de domain.
  infrastructure/
    httpserver/     -> Fiber: router, handlers (HTTP) y middleware (JWT)
    postgres/       -> implementación de los repositorios con pgx
    security/       -> bcrypt (hasher) y JWT
migrations/         -> *.sql en orden (001, 002, 003, ...)
```

**Cómo agregar un módulo nuevo (receta):** crea la entidad y la interfaz del
repositorio en `domain`, el caso de uso en `usecase`, la implementación en
`postgres`, el handler en `httpserver/handlers`, registra la ruta en
`httpserver/router.go`, y conéctalo en `cmd/api/main.go`.

## 6. Estructura del frontend

```
src/
  views/        -> pantallas (LoginView, CotizacionesView, ClientesView, MiCuentaView, ...)
  components/   -> componentes de UI por módulo (cotizaciones/, clientes/, AppHeader, ...)
  services/     -> acceso a la API (api.ts base de axios + un archivo por módulo)
  stores/       -> estado global (auth.tsx: sesión, login, logout)
  data/          -> tipos y datos estáticos de apoyo
  styles/       -> variables.css (tokens de diseño) y app.css
```

El token JWT se guarda en `localStorage` y `services/api.ts` lo adjunta en cada
petición; si la API responde 401 (sesión expirada) limpia la sesión y manda al login.

## 7. Módulos y endpoints

Todos los de datos requieren el token (cabecera `Authorization: Bearer <token>`).

| Módulo | Método y ruta | Qué hace |
|---|---|---|
| Salud | `GET /api/health` | Comprobación de vida (público) |
| Auth | `POST /api/auth/login` | Inicia sesión y devuelve el token (público, con límite de intentos) |
| Auth | `GET /api/auth/me` | Perfil del usuario del token |
| Auth | `PUT /api/auth/password` | Cambia la contraseña (verifica la actual) |
| Clientes | `GET /api/clientes` | Lista los clientes |
| Clientes | `POST /api/clientes` | Crea un cliente |
| Clientes | `DELETE /api/clientes/:id` | Elimina un cliente |
| Cotizaciones | `GET /api/cotizaciones` | Lista las cotizaciones con sus partidas (sin las eliminadas) |
| Cotizaciones | `POST /api/cotizaciones` | Crea una cotización (el backend calcula folio y totales) |
| Cotizaciones | `DELETE /api/cotizaciones/:id` | Borrado **suave** (marca 'Eliminada') |

El **logout** y la pantalla **Mi cuenta** viven solo en el frontend (el JWT es sin
estado: cerrar sesión = descartar el token).

Detalles de negocio útiles:
- Las **cotizaciones** tienen una relación 1→N con **partidas** (dos tablas).
- Los **totales** (costo, subtotal, IVA, total, utilidad) y el **folio** correlativo
  los calcula el backend; nunca se confía en lo que manda el cliente.
- El **asesor** de una cotización se toma del usuario logueado (del token).

## 8. Cómo trabajamos (convenciones)

- **El frontend manda:** primero se entiende qué pide la pantalla y el backend se
  adapta a eso (exponiendo el JSON con los mismos nombres de campo que consume la UI).
- **Por módulo, en dos pasos:** (1) backend + pruebas, confirmar el escenario;
  (2) conectar el frontend. Se hace **un commit al terminar el backend** y **otro al
  terminar el frontend**, para poder volver un paso atrás si algo se rompe.
- **No se modifica la UI** (etiquetas/maquetación) sin acuerdo; sí se puede cambiar
  la lógica/los `services`.
- Lo que queda pendiente se anota en [`DEUDA_TECNICA.md`](./DEUDA_TECNICA.md).

## 9. Verificación y pruebas

**Backend:**
```bash
go build ./...     # compila
go vet ./...       # análisis estático
go test ./...      # pruebas (los casos de uso se prueban con "fakes", sin BD)
```

**Frontend:**
```bash
pnpm type-check    # revisa tipos (tsc)
pnpm build         # build de producción (más estricto que type-check)
```

> Observabilidad: cuando el backend responde un error 500, registra en su **log**
> (consola del servidor) el método, la ruta y la causa real, pero al cliente solo
> le responde un mensaje genérico. Nunca se registran datos sensibles.

## 10. Estado actual

- ✅ Autenticación (login, perfil, cambiar contraseña, logout)
- ✅ Clientes (listar, crear, eliminar) — conectado de punta a punta
- ✅ Cotizaciones (listar, crear, eliminar) — conectado de punta a punta
- ✅ Mi cuenta (perfil + sesión + seguridad)
- ⏳ Configuración — pantalla reservada, aún sin construir
- Pendientes técnicos detallados en `DEUDA_TECNICA.md`

---

_Última actualización: 2026-10-23_
