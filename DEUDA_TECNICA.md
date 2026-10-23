# Deuda técnica — Sistema de Cotizaciones

Registro de mejoras y pendientes detectados durante el desarrollo. Sirve como
memoria del equipo: qué falta pulir, por qué, y en qué archivos/líneas se tocaría.
No son errores que rompan el sistema, sino puntos a mejorar más adelante.

> Las rutas que empiezan con `cotizaciones-frontend/` viven en el **repo del
> frontend**; las que empiezan con `cotizaciones-backend/` en **este repo**. Los
> números de línea son aproximados (pueden moverse conforme cambie el código).

**Estado:** Pendiente = sin empezar · En progreso · Hecho

---

## 1. Validar que los campos obligatorios no se envíen vacíos

**Estado:** Pendiente
**Módulo:** Clientes (aplica también a futuros formularios)

**Descripción:** Reforzar la validación para que no se puedan guardar registros
con campos obligatorios vacíos, tanto en el formulario como en el backend (el
backend no debe confiar solo en la validación de la pantalla).

**Dónde se tocaría:**
- `cotizaciones-frontend/src/components/clientes/FormularioCliente.tsx` — función
  `validar()` (líneas 19-26). Hoy solo exige nombre y teléfono; revisar qué otros
  campos deben ser obligatorios y marcar el campo vacío.
- `cotizaciones-backend/internal/infrastructure/httpserver/handlers/cliente_handler.go`
  — método `Create` (validación de entrada); hoy solo valida `nombreComercial`.
- `cotizaciones-backend/internal/usecase/cliente_usecase.go` — método `Create`,
  si la regla es de negocio (ej. normalizar/rechazar vacíos).

---

## 2. Notificaciones con SweetAlert en la parte superior izquierda

**Estado:** Pendiente
**Módulo:** Transversal (empezando por Clientes)

**Descripción:** Sustituir los avisos actuales (mensajes de éxito/error hechos a
mano) por notificaciones tipo SweetAlert, mostradas como *toast* en la esquina
**superior izquierda**. Unifica el estilo de avisos en todo el sistema.

**Dónde se tocaría:**
- `cotizaciones-frontend/package.json` — agregar la dependencia (p. ej. `sweetalert2`).
- `cotizaciones-frontend/src/components/clientes/FormularioCliente.tsx` — banner de
  error (línea 52) y bloque de éxito (línea 54) se reemplazarían por SweetAlert.
- `cotizaciones-frontend/src/components/clientes/ListaClientes.tsx` — aviso de error
  de carga/eliminación y el modal de confirmación de borrado (podría volverse un
  SweetAlert de confirmación).
- Configurar la posición como *toast* `top-start` (superior izquierda).

> Nota: implica tocar UI, así que requiere aprobación explícita antes de aplicarse.

---

## 3. El dato del teléfono aparece "pegado" con el del correo

**Estado:** Pendiente
**Módulo:** Clientes (listado)

**Descripción:** En la fila del listado, el correo y el teléfono se muestran juntos
sin separación clara. Hay que decidir cómo presentarlos (apilarlos, separarlos con
una etiqueta, o darles su propia columna).

**Dónde se tocaría:**
- `cotizaciones-frontend/src/components/clientes/ListaClientes.tsx` — línea 68, el
  bloque `<div className="fila__datos">` con los dos `<span>` (correo y teléfono).
- `cotizaciones-frontend/src/styles/app.css` — regla de `.fila__datos` (línea 122);
  no tiene layout de columna como sí lo tienen `.fila__nombre`/`.fila__rfc`
  (línea 123). Darle `display:flex; flex-direction:column` separaría los datos.
- Si se decide columna aparte, ajustar también `.fila--cliente` (grid, línea 120).

---

## 4. Agregar botón de "Editar" junto al de "Eliminar"

**Estado:** Pendiente
**Módulo:** Clientes (requiere trabajo en backend + frontend)

**Descripción:** Permitir editar un cliente ya registrado. Hoy solo se puede crear
y eliminar. Implica agregar la operación de actualización de punta a punta.

**Dónde se tocaría:**

*Backend (nuevo endpoint `PUT /api/clientes/:id`):*
- `cotizaciones-backend/internal/domain/cliente_repository.go` — agregar método
  `Update` a la interfaz.
- `cotizaciones-backend/internal/infrastructure/postgres/cliente_repository.go` —
  implementar `Update` (UPDATE ... WHERE id).
- `cotizaciones-backend/internal/usecase/cliente_usecase.go` — método `Update`.
- `cotizaciones-backend/internal/infrastructure/httpserver/handlers/cliente_handler.go`
  — handler `Update`.
- `cotizaciones-backend/internal/infrastructure/httpserver/router.go` — registrar la
  ruta `PUT /api/clientes/:id` protegida.

*Frontend:*
- `cotizaciones-frontend/src/components/clientes/ListaClientes.tsx` — línea 68, junto
  al botón `.fila__borrar`, agregar un botón de editar (y ajustar el grid
  `.fila--cliente` en `app.css` línea 120 para la columna extra).
- `cotizaciones-frontend/src/components/clientes/FormularioCliente.tsx` — reutilizar
  el formulario para modo "editar" (precargar datos y llamar a actualizar).
- `cotizaciones-frontend/src/services/clientes.ts` — agregar `actualizarCliente(id, datos)`.

> Nota: implica tocar UI, así que requiere aprobación explícita antes de aplicarse.

---

_Última actualización: 2026-10-23_
