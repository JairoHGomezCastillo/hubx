# Guía de equipo HUBX: construir en paralelo, una rama y un PR por historia

> **Nota (2026-11-20):** la rama de integración real es `release/sdd-hubx-system`; donde esta guía dice `release/v1`,
> usa ese nombre. La puesta en marcha comprobada de cada máquina está en [`docs/setup.md`](setup.md).

Entrega: **martes 24 de noviembre de 2026, 18:00**. Hoy es jueves 19. La demostración individual vale el 40 %.

## 0. Cómo leer esta guía

1. Todos leen las secciones 1 a 4 (reglas y preparación).
2. Cada integrante lee la suya en la sección 5 (A, B, C o D).
3. Todos siguen el ciclo de la sección 6 con cada historia.

A, B, C y D son puestos; cada quien escribe su nombre en la tabla de la sección 4. A es quien armó la base (Jairo).

---

## 1. ¿Es correcta la idea de repartirlo?

Sí, con tres condiciones:

- **La base debe estar en `main` antes de que los demás implementen.** Hoy faltan el `Makefile` con `make test-integration`, el proyecto `hubx-bender`, el esquema de base de datos y la conexión del lector de configuración. Sin eso nadie puede correr las pruebas de integración. Es la sección 2: se reparte entre los cuatro y se mide en horas.
- **Cada historia va en su propia rama y su propia carpeta `specs/NNN-...`**, y el número de la carpeta es el de la historia, para que no choquen.
- **La demostración es individual y en vivo.** Las cinco historias de la demostración (HU-06 a HU-10) **no se implementan antes**: se sortean ese día y cada integrante construye una desde su máquina. Por eso la línea base llega solo hasta HU-05 y todos deben dominar el ciclo completo (sección 3).

Lo que más puede salir mal: dos personas editando a la vez `cmd/api/main.go`, `go.mod` o `memory.md`. La sección 7 dice cómo evitarlo.

---

## 2. Puerta de entrada (reparto de la base)

Estas tareas se reparten porque son pequeñas y se pueden hacer a la vez. HU-01 a HU-05 empiezan a implementar cuando 1 a 4 estén en `release/v1`; las demás pueden terminar durante el día siguiente.

| # | Tarea | Dueño | Cómo se comprueba |
|---|---|---|---|
| 1 | Subir al remoto lo que ya existe: agentes, `AGENTS.md`, `memory.md`, constitución, backlog, `specs/000-smoke-config` | A | Todos clonan sin errores |
| 2 | Proyecto `hubx-bender`: `npm --prefix hubx-bender init -y`, `@playwright/test`, `playwright.config.ts` con `baseURL` por variable de entorno (gateway) y `health.spec.ts` | A | `npm --prefix hubx-bender run test` pasa con los servicios arriba |
| 3 | `Makefile` con `make test-integration` (levanta PostgreSQL, API y gateway, espera `/health`, corre bender y apaga todo), `make up` y `make down` | A | `make test-integration` termina en verde sin bloquear la terminal |
| 4 | PostgreSQL local (Docker Compose) y migraciones en SQL plano (`hubx-api/migrations/`) con runner de biblioteca estándar. `0001_baseline_schema.sql` con `users`, `shipments`, `updates` y datos de prueba (un operator, un ground partner, un customer, y shipments en distintos stages que las historias de la demostración puedan leer) | A | `make up` crea las tablas; el runner es idempotente |
| 5 | `main.go` de `hubx-api` y del gateway leyendo su `application.properties` (hoy leen `PORT` directo) | B | `server.port` del archivo cambia el puerto; `SERVER_PORT` lo sobrescribe |
| 6 | Base del MFE: cliente HTTP único hacia `/api`, token en memoria, rutas y layout | B | `npm --prefix hubx-mfe run typecheck`, `lint` y `build` en verde |
| 7 | Paquete del ledger: `LedgerRepository` y cálculo del fingerprint SHA-256 (lo usan HU-02, HU-03, HU-04 y HU-05) | C | Pruebas unitarias en verde; contrato en `contracts/` |
| 8 | GitHub Actions por servicio (pruebas, `gosec`, `govulncheck`, build del MFE) | D | Un PR de prueba corre los checks |
| 9 | Ramas `main` y `release/v1` con PR obligatorio, CI en verde y una aprobación | A (dueño del repo) | Nadie puede empujar directo |
| 10 | Esta guía y el backlog enlazados en el README | A | Todos pueden leerlos |

Recomendación: hacer 1 a 8 con el flujo SDD y los agentes (spec `100-foundation`), porque sirve también como evidencia de la rúbrica.

Dos decisiones que deben tomarse en esta etapa y quedar en `memory.md`:
- **Migraciones:** sin dependencias nuevas, con un runner propio (la constitución limita las dependencias a driver de PostgreSQL, JWT y bcrypt). Usar una herramienta externa es una decisión humana.
- **Plataforma de despliegue gratuita y PostgreSQL gestionada:** decídanlo entre todos hoy; el despliegue es de A (sección 9).

---

## 3. Preparación en la máquina de cada integrante (todos, hoy)

Haz esto en orden y confirma cada punto antes de seguir.

1. **Herramientas.** Git, Go (última estable), Node LTS, Docker, `gosec` y `govulncheck`. Comprueba con `go version`, `node -v`, `docker --version`, `gosec -version` y `govulncheck -version`.
2. **Clonar.** `git clone <url> hubx`, entrar y `git switch release/v1`.
3. **opencode.** Instalar y conectar un modelo gratuito (`/connect`). Abrir `opencode` en la raíz: debe arrancar con el agente `hubx-lead`.
4. **Spec Kit.** `specify check` pasa y los comandos `/speckit.*` aparecen en opencode.
   - *Opcional, Kiro o Claude Code:* los mismos agentes existen para las dos herramientas, con un modelo por rol (tabla en `README.md`, "Multi-agent architecture"). Arranca con `kiro-cli chat --agent hubx-lead` o con `claude --agent hubx-lead` (en Claude Code acepta una vez el aviso de confianza de la carpeta para que se apliquen los permisos de cada rol). Los agentes se editan solo en `.opencode/agents/`; después ejecuta `make agents-sync`.
5. **Servicios.** `make up`, y después:
   ```bash
   curl -s http://localhost:8080/health
   curl -s http://localhost:8080/api/health
   ```
   Ambos responden `ok`.
6. **Pruebas.** `make test-integration` termina en verde.
7. **Leer, en este orden:** `AGENTS.md`, la constitución (`.specify/memory/constitution.md`), las reglas transversales y tu historia en `backlog/backlog.md`, y `memory.md`.

Si algo falla, no sigas: avisa al equipo con el comando y la salida literal.

---

## 4. Reparto

Solo la línea base (HU-01 a HU-05) y lo necesario para implementarla. Las historias HU-06 a HU-10 quedan reservadas para la demostración en vivo.

| Puesto | Nombre | Base (sección 2) | Historia de línea base | Otros entregables |
|---|---|---|---|---|
| A | Jairo | 1, 2, 3, 4, 9, 10 | HU-01 Sign in | Despliegue |
| B | _escribe aquí_ | 5, 6 | HU-02 Create a shipment | README y descripción de la arquitectura multiagente |
| C | _escribe aquí_ | 7 | HU-03 Record a shipment update | Prueba de manipulación del ledger |
| D | _escribe aquí_ | 8 | HU-04 View traceability y HU-05 Verify integrity | Despliegue (apoya a A) y consolidación de la evidencia de QA y seguridad |

La carga de C es la historia más densa (HU-03) más el ledger; la de D son dos historias de lectura que comparten la cadena, y por eso se agrupan.

**Orden de fusión:** HU-01 primero (define el inicio de sesión y la validación del JWT). HU-02 a HU-05 avanzan en paralelo hasta el plan, las tareas y las pruebas en rojo, y **se implementan cuando HU-01 y los ítems 1 a 4 de la sección 2 estén fusionados** en `release/v1`. HU-02 y HU-03 necesitan además el ledger (ítem 7).

**Las cinco historias de la demostración (no se tocan en `release/v1` ni en `main`):**

| Historia | Qué lee de la línea base |
|---|---|
| HU-06 List shipments by stage | `shipments` y su stage actual |
| HU-07 View my assigned shipments | `shipments` y el ground partner asignado |
| HU-08 Report an incident | `shipments`; añade su propia migración |
| HU-09 View the shipment count by stage | `shipments` |
| HU-10 Download the traceability evidence | La cadena de `updates` (HU-04) |

Son cinco y los integrantes son cuatro: el día de la demostración se sortea y sobra una.

**Calendario**

| Día | Qué pasa |
|---|---|
| Jue 19 | Sección 2 (ítems 1 a 4 primero) y sección 3 para todos. B, C y D hacen `specify` → `tasks` de su historia |
| Vie 20 | HU-01 fusionada; el resto implementa HU-02 a HU-05 |
| Sáb 21 | Línea base completa en `release/v1`. CI en verde |
| Dom 22 | `release/v1` → `main` y primer despliegue. Evidencia consolidada |
| Lun 23 | Ensayo cronometrado en ramas desechables con HU-06 a HU-10. README y descripción multiagente cerrados |
| Mar 24 | Revisión final y entrega con margen, antes de las 18:00 |

---

## 5. Guía por integrante

El ciclo detallado está en la sección 6; aquí está lo propio de cada puesto. Cuando termines tu parte, ayuda a revisar los PR de los demás.

### A (Jairo): base, HU-01 y despliegue

1. Ítems 1, 2, 3, 4, 9 y 10 de la sección 2, en ese orden. Avisa al equipo cuando 1 a 4 estén en `release/v1`.
2. **HU-01 Sign in** (`specs/001-sign-in`): login, hash con bcrypt, JWT de una hora con el rol, middleware que valida el JWT y expone la identidad a los demás módulos, y el secreto compartido gateway-API. Todos dependen de esto: fusiónalo en cuanto el CI esté en verde y escribe "HU-01 fusionada, rebasen".
3. **Despliegue** (con D): `deploy/`, secretos cargados a mano en la plataforma (único paso manual permitido), humo contra la URL pública y `specs/NNN-*/deploy.md`.
4. Archivos tuyos: `Makefile`, `deploy/`, `hubx-api/migrations/`, helpers de `hubx-bender`, `internal/auth`.

### B: config, base del MFE y HU-02

1. Ítems 5 y 6 de la sección 2 (lector de configuración conectado en API y gateway, y base del MFE).
2. **HU-02 Create a shipment** (`specs/002-create-shipment`): solo el operator crea; la creación registra el primer update (stage Created) con huella y responsable, usando el paquete del ledger de C.
3. **README** (instalación y ejecución) y **descripción de la arquitectura multiagente**: los cinco agentes, su función, el flujo del lead y las pruebas hechas; pide a cada integrante su aporte.
4. Archivos tuyos: `internal/config` y `main.go` de ambos servicios (solo las líneas de configuración), `internal/shipments`, `hubx-mfe/src/` base y sus pantallas.

### C: ledger y HU-03

1. Ítem 7 de la sección 2: `LedgerRepository` y el fingerprint SHA-256. Acuerda con B y D, en el contrato, los campos del update y la fórmula antes de implementar. Es lo primero que debes fusionar.
2. **HU-03 Record a shipment update** (`specs/003-record-update`): secuencia de stages, matriz de permisos del backlog, identidad del responsable desde la sesión, encadenamiento e inmutabilidad. Es la historia con más reglas: dedícale la clarificación.
3. **Prueba de manipulación:** alterar un update por SQL debe romper la verificación (la ejecutas con D tras HU-05).
4. Archivos tuyos: `internal/ledger`, `internal/updates`.

### D: CI, HU-04, HU-05 y evidencia

1. Ítem 8 de la sección 2: workflows de GitHub Actions por servicio, con filtros de ruta, caché, permisos mínimos y acciones fijadas por versión.
2. **HU-04 View a shipment's traceability** (`specs/004-view-traceability`): cadena completa en orden, con visibilidad por rol.
3. **HU-05 Verify the integrity of the chain** (`specs/005-verify-integrity`): recalcula las huellas de punta a punta y detecta la manipulación.
4. **Evidencia:** reúne los `qa-report.md` y los resultados de `gosec`, `govulncheck` y OWASP de todas las historias en un solo índice; apoya a A en el despliegue.
5. Archivos tuyos: `.github/`, `internal/traceability`, `internal/integrity`.

---

## 6. Ciclo de cada historia (igual para todos)

Antes de empezar:

```bash
git switch release/v1 && git pull
```

**1. Rama y spec con el número de tu historia.** En opencode (agente `hubx-lead`):

```
Trabaja la historia HU-0X del backlog. Crea la spec con el número 0X (--number 0X) y el nombre corto <nombre>.
```

El número sale siempre del backlog (HU-0X → `00X`). El lead ejecuta `.specify/scripts/bash/create-new-feature.sh` con `--allow-existing-branch --number 00X --short-name <nombre>` y crea él mismo la rama con `git switch -c`, porque el script de Spec Kit 1.4 crea la carpeta pero no la rama. Si el script renumera (otra carpeta usa ese número), el lead se detiene. Comprueba que la rama y la carpeta se llaman `0X-nombre` (por ejemplo `002-create-shipment`); si no, corrígelo antes de seguir. Nunca dos personas con el mismo número.

**2. Flujo.** `specify` → `clarify` → `plan` → `tasks`. Tras cada paso, lee lo producido antes de continuar. Revisa que `spec.md` abra con `Source: backlog HU-0X`, no hable de tecnología y referencie (no copie) las reglas transversales.

**3. Pruebas primero.** El lead delega en `qa-seguridad` (`red`): `hubx-bender/features/HU0X_<nombre>.feature` con los escenarios del backlog copiados tal cual (Cucumber), sus pasos en Playwright contra el gateway y al menos un escenario `@e2e` con el camino feliz en el navegador. Todo en rojo y por la razón correcta (constitución 1.4.0).

**4. Implementación.** `backend-go` (dominio en `internal/<dominio>`, handlers en `internal/handlers/<dominio>`, pruebas en `test/` y mocks generados con gomock en `mocks/`) y `frontend-react` con TDD contra el contrato del plan. La historia no está terminada si su `@e2e` está en rojo, aunque la API esté en verde. Luego `qa-seguridad` (`verify`): suite, `gosec`, `govulncheck` y tabla OWASP en `qa-report.md`. Con hallazgos, vuelven como `fix`.

**5. Analyze.** `/speckit.analyze` y corrige las desviaciones.

**6. Commits.** Uno por entrega de subagente, con el formato `[agent:role] HU-0X resumen`. El lead los hace solo, sin pedirte confirmación; los revisas en el PR:

```
[agent:qa-seguridad] HU-02 integration tests (red)
[agent:backend-go] HU-02 create shipment module
```

**7. PR hacia `release/v1`** (no hacia `main`). El PR es el resultado final de la historia. El lead escribe la descripción en `specs/0X-<name>/pr.md` con la evidencia real, empuja la rama de la historia y abre el PR (`gh pr create --base release/v1 --body-file specs/0X-<name>/pr.md`) sin pedirte nada; te da la URL. El título sigue Conventional Commits con gitmoji y la marca del agente al final: `<tipo>(<alcance>): <emoji> <descripción corta> [agent:hubx-lead]`, por ejemplo `feat(HU-02): ✨ create shipment module [agent:hubx-lead]` (tipos y emojis en `.opencode/agents/hubx-lead.md`, paso 10). **Ningún agente fusiona**: fusionar feature → release y release/hotfix → `main` es siempre una decisión humana, y el ruleset de release exige una aprobación. La historia queda en revisión mientras el PR está abierto y se da por terminada cuando se fusiona.

Plantilla de la descripción del PR (también en `.github/pull_request_template.md`):

```
## Story
HU-0X <title>. Spec: specs/0X-<name>/

## Evidence
- make test-integration: <result>
- go test / npm test: <result>
- gosec / govulncheck: <result>
- speckit.analyze: <deviations or "none">
- qa-report.md: specs/0X-<name>/qa-report.md

## Checklist
- [ ] Every backlog scenario has a test
- [ ] No secrets, no cross-service imports, MFE only via /api
- [ ] Only my own paths changed (git diff --stat attached)
- [ ] Backlog status updated for this story only
```

**8. Revisión cruzada.** Cada PR lo revisa otra persona: comprueba el checklist y ejecuta la historia en su máquina (`git switch <rama> && make test-integration`). Con CI en verde y aprobación, se fusiona.

**9. Después de fusionar:** GitHub borra la rama automáticamente; sincroniza `release/v1` (`git switch release/v1 && git pull`) y borra tu rama local (`git branch -d <tu-rama>`).

---

## 7. Evitar conflictos

| Punto de choque | Regla |
|---|---|
| `hubx-api/cmd/api/main.go` | Una línea por módulo: `<modulo>.Register(mux)`. En un conflicto, conserva ambas líneas |
| `go.mod` / `go.sum` | Dependencias nuevas solo con confirmación humana y justificadas en el plan; si chocan, `go mod tidy` tras resolver |
| Migraciones | Una por historia, `NNNN_nombre.sql`. La `0001` es de A. Reserven números antes y anótenlos en `memory.md` |
| `backlog/backlog.md` | Cada PR cambia solo el estado de su historia |
| `memory.md` | Solo se agregan líneas al final de su sección; en un conflicto se conservan ambas |
| `contracts/` | Un archivo por historia; si cambias el contrato de otro, avisa antes |
| `hubx-bender` | Un archivo por historia (`HU02_*.spec.ts`); los helpers comunes los toca solo A |
| Tras fusionar HU-01 | `git fetch && git rebase origin/release/v1` y vuelve a correr `make test-integration` |

---

## 8. Evidencia que exige la rúbrica

Cada historia deja en el repo: `specs/NNN-*/spec.md`, `plan.md`, `tasks.md` (tareas marcadas `- [x]` por el lead, con su evidencia), `qa-report.md` y, si aplica, `deploy.md`. A nivel de repo:

- Instrucciones de instalación y ejecución en el `README` (A).
- URL desplegada y evidencia de pruebas y seguridad (A, con los informes de todos).
- Descripción de la arquitectura multiagente: los cinco agentes, su función, el flujo del lead y las pruebas hechas (todos aportan; A consolida).
- Trazabilidad backlog → spec → PR.

---

## 9. Despliegue (A, con apoyo de todos)

1. Un entorno gratuito para los tres servicios y PostgreSQL gestionada. Variables de entorno en la plataforma: `DB_URL`, `JWT_SECRET`, `GATEWAY_SECRET`, `API_URL` y `CORS_ALLOWED_ORIGINS`.
2. `release/v1` → `main` con PR, cuando HU-01 a HU-05 estén en verde.
3. Humo contra la URL pública: inicio de sesión, crear un shipment, registrar un update y verificar la cadena. Resultado literal en `deploy.md`.
4. `main` siempre desplegable: desde aquí, todo cambio entra por PR.

---

## 10. Ensayo de la demostración (lunes 23)

Cada uno construye **una historia en vivo, desde su propio ordenador, en 5 a 10 minutos**, y la historia sale por sorteo entre HU-06 y HU-10. Preparación:

1. Ensaya cronometrado con una de HU-06 a HU-10, **en una rama desechable que nunca se fusiona**, para que las cinco sigan pendientes en `release/v1` y `main`. Borra la rama al terminar.
2. Tu máquina debe arrancar sin pasos manuales: `git pull`, `make up`, `opencode`.
3. Ten a mano qué mostrar en cada fase (spec sin tecnología, pruebas en rojo, verde, evidencia).
4. Prepara respuestas: por qué la spec no menciona la tecnología, qué detectó `analyze`, por qué QA es un agente separado, qué hace el lead y por qué no escribe código.
5. Plan B si el modelo gratuito falla o se limita: ten un modelo alternativo conectado y sabe qué parte puedes hacer a mano.

---

## 11. Si algo se atasca

| Síntoma | Qué hacer |
|---|---|
| El agente edita fuera de su zona | Rechaza la entrega (`git diff --stat`) y devuélvela al subagente indicando la ruta prohibida |
| El modelo gratuito ignora una regla | Cita la regla literal de la constitución y repite el paso; si falla tres veces, escala al equipo |
| `make test-integration` falla solo en tu máquina | Compara versiones (`go version`, `node -v`) y puertos ocupados (8080, 8081, 5432) |
| Dos personas usaron el mismo número de spec | La del PR más antiguo conserva el número; la otra renombra rama, carpeta y referencias |
| Conflicto en `main.go` | Conserva ambas líneas `Register`; compila y prueba |
| El CI falla por `gosec` o `govulncheck` | No lo desactives: corrige el hallazgo o justifícalo por escrito en el PR |
