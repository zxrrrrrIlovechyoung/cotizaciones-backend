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

## 5. Reforzar la validación de campos en Cotizaciones

**Estado:** Pendiente
**Módulo:** Cotizaciones

**Descripción:** El formulario ya valida por pasos en el frontend, pero el backend
valida poco (solo que haya nombre y al menos una partida). Hay que reforzar la
validación del lado del servidor para no confiar únicamente en la pantalla
(cantidades/precios mayores a cero, formato de correo, campos obligatorios, etc.).

**Dónde se tocaría:**
- `cotizaciones-frontend/src/components/cotizaciones/FormularioCotizacion.tsx` —
  funciones `validarPaso0` (línea 69), `validarPaso1` (línea 82) y `validarPaso2`
  (línea 97); revisar que cubran todos los campos obligatorios.
- `cotizaciones-backend/internal/infrastructure/httpserver/handlers/cotizacion_handler.go`
  — método `Create`: validar la entrada antes de guardar.
- `cotizaciones-backend/internal/usecase/cotizacion_usecase.go` — reglas de negocio
  (ej. precio de venta no menor al de compra, cantidades positivas).

---

## 6. Implementar la función de "Imprimir / PDF"

**Estado:** Pendiente
**Módulo:** Cotizaciones (detalle)

**Descripción:** En el detalle de una cotización existe el botón **Imprimir / PDF**,
pero todavía no hace nada (no tiene evento). Falta generar el documento imprimible
o el PDF de la cotización.

**Dónde se tocaría:**
- `cotizaciones-frontend/src/views/CotizacionesView.tsx` — línea 74, el botón
  `.topbar__print`: agregarle el evento `onClick` que dispare la impresión / PDF.
- Definir cómo se genera el PDF (en el frontend con una librería, o en el backend
  con un endpoint que arme el documento).

---

## 7. Implementar el botón de "Editar" de una cotización

**Estado:** Pendiente
**Módulo:** Cotizaciones (requiere backend + frontend)

**Descripción:** En el detalle existe el botón **Editar**, pero no tiene función.
Falta permitir modificar una cotización ya creada (recalculando totales).

**Dónde se tocaría:**

*Backend (nuevo endpoint `PUT /api/cotizaciones/:id`):*
- `cotizaciones-backend/internal/domain/cotizacion_repository.go` — agregar `Update`.
- `cotizaciones-backend/internal/infrastructure/postgres/cotizacion_repository.go` —
  implementar `Update` (actualizar cabecera y reemplazar partidas).
- `cotizaciones-backend/internal/usecase/cotizacion_usecase.go` — método `Update`
  (recalcular totales).
- `cotizaciones-backend/internal/infrastructure/httpserver/handlers/cotizacion_handler.go`
  y `.../router.go` — handler y ruta.

*Frontend:*
- `cotizaciones-frontend/src/views/CotizacionesView.tsx` — línea 78, el botón
  `.topbar__edit`: agregar el evento que abra el formulario en modo "editar".
- `cotizaciones-frontend/src/components/cotizaciones/FormularioCotizacion.tsx` —
  reutilizar el formulario para precargar y actualizar una cotización existente.
- `cotizaciones-frontend/src/services/cotizaciones.ts` — agregar `actualizarCotizacion(id, datos)`.

> Nota: los puntos 6 y 7 implican tocar UI, así que requieren aprobación explícita
> antes de aplicarse.

---

_Última actualización: 2026-10-23_
