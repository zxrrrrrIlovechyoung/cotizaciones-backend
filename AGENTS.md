# Sistema de cotizaciones

## Frontend local

- Antes de arrancar Vite, comprueba `http://127.0.0.1:5173/login`. Si responde, reutiliza el servidor existente y evita reiniciarlo.
- Para un arranque estable en Windows, ejecuta `node node_modules/vite/bin/vite.js --host 127.0.0.1 --strictPort` desde `FRONTEND` en una terminal de PowerShell que se mantenga abierta.
- Un proceso iniciado desde una sesión temporal del agente puede cerrarse cuando termina esa sesión. Si solo puedes arrancarlo de forma temporal, indícalo al usuario y evita prometer que seguirá disponible.
- Para diagnosticar el acceso, comprueba `/login` y `/api/health` en el puerto `5173`. Si falla `/login`, revisa el proceso y la salida de Vite. Si solo falla `/api/health`, comprueba la API en el puerto `3001` y el proxy de Vite.
