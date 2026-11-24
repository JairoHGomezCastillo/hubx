# Puesta en marcha de HUBX (macOS, con notas para Windows)

Cómo dejar una máquina lista para trabajar en HUBX y en qué estado está la base del proyecto. Es la sección 3 de
la guía del equipo hecha paso a paso y comprobada. Las historias van a una rama de integración que elige quien la trabaja (hoy `release/vX.Y.Z`, que sale de `main`; la primera, `release/sdd-hubx-system`, es la v0.1.0). La guía la
llama `release/v1`.

## 1. Herramientas

Las versiones son las que funcionaron el 2026-11-20. El repo declara `go 1.27.4` (antes 1.26.5, con 15 vulnerabilidades de la biblioteca estándar); hace falta ese Go o uno más
nuevo (`brew upgrade go`).

| Herramienta | Versión probada | Para qué |
|---|---|---|
| Git | cualquiera reciente | repo |
| Go | 1.27.4 | `hubx-api`, `hubx-gw`, `gosec`, `govulncheck` |
| Node | 26.13.1 LTS (npm 11) | `hubx-mfe`, `hubx-bender` |
| Docker Desktop | 29.x con Compose v5 | PostgreSQL, API y gateway locales |
| GNU Make | 3.81+ (el de macOS sirve) | `make up`, `make test-integration` |
| GitHub CLI (`gh`) | 2.105 | PRs (skill `/create-pr`) |
| gosec / govulncheck | última | seguridad (constitución, principio VI) |
| opencode + Spec Kit (`specify`) | última | flujo SDD con agentes |

### macOS (Homebrew)

```bash
# Homebrew, si no lo tienes: https://brew.sh
xcode-select --install            # trae git y make
brew install go node gh
brew install --cask docker        # abre Docker Desktop una vez para que arranque el daemon

# Herramientas de seguridad: compílalas con el mismo Go que tienes en el PATH
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc

# opencode y Spec Kit
brew install sst/tap/opencode     # o: curl -fsSL https://opencode.ai/install | bash
brew install uv && uv tool install specify-cli --from git+https://github.com/github/spec-kit.git
```

Comprueba:

```bash
go version; node -v; docker --version; docker compose version; make --version | head -1
gh --version | head -1; gosec -version; govulncheck -version; opencode --version; specify check
```

> Apple Silicon: todas las imágenes (`golang:1.27.4-alpine`, `postgres:17.7-alpine`, `distroless/static`) son
> multi-arquitectura; no hace falta configurar nada.

### Windows (lo que se hizo en la máquina prestada)

`winget install GoLang.Go OpenJS.NodeJS.LTS ezwinports.make GitHub.cli` y Docker Desktop. Hay que ejecutar `make`
**desde Git Bash**, no desde PowerShell. Después de instalar, reinicia la terminal o el IDE para que tome el nuevo
`PATH`. Si actualizas Go, vuelve a instalar `gosec` y `govulncheck`: compilados con otro Go, fallan con
"package requires newer Go version".

## 2. Clonar e iniciar sesión

```bash
git clone https://github.com/JairoHGomezCastillo/hubx.git && cd hubx
git switch <rama de integración>   # la de tu historia (hoy release/*)
gh auth login --hostname github.com --git-protocol https --web
npm --prefix hubx-bender ci && npm --prefix hubx-bender run browsers
npm --prefix hubx-mfe ci
```

El `.env` con los secretos locales **no está en el repo**. `make` lo genera solo, con valores aleatorios, la primera
vez. Si quieres regenerarlo, bórralo.

Variables de entorno requeridas (todas en `.env`, generadas por `scripts/env.sh`):

| Variable | Formato | Para qué |
|---|---|---|
| `POSTGRES_PASSWORD` | aleatoria | contraseña de PostgreSQL local |
| `JWT_SECRET` | 32 bytes en hex (64 caracteres) | firma y validación del JWT de sesión (HU-01). **Debe ser la misma** en `hubx-api` y `hubx-gw`; `compose.yaml` la inyecta a ambos |
| `GATEWAY_SECRET` | aleatoria | secreto compartido gateway → API |

Nunca se escriben secretos en el repo; `application.properties` deja `jwt.secret` vacío y el valor llega solo por entorno.

## 3. Levantar y probar

```bash
make help                 # lista de targets
make up                   # PostgreSQL -> migraciones -> API -> gateway; vuelve cuando /health responde
curl -s http://localhost:8080/health        # {"service":"hubx-gw","status":"ok"}
curl -s http://localhost:8080/api/health    # {"status":"ok","service":"hubx-api"}
make down                 # apaga (conserva la base de datos); make reset la borra

make test-integration     # pila aislada y limpia + hubx-bender por el gateway; al final borra todo
make test                 # pruebas unitarias
make security             # gosec + govulncheck en ambos servicios
make psql                 # consola SQL de la base local (requiere make up)
```

Resultado esperado de `make test-integration`: los 3 escenarios de `features/health.feature` (gateway, API por el
gateway y el `@e2e` en Chromium a través del MFE) más los de HU-01 Sign in (`features/HU01_sign_in.feature`, con su
`@e2e`). La primera vez descarga Chromium (`npm --prefix hubx-bender run browsers`) y construye la imagen del MFE.

Con `make up`, el MFE queda en http://localhost:3000 (nginx sirve el build y reenvía `/api` al gateway). Para ver el
navegador en los `@e2e`: `HEADED=1 npm --prefix hubx-bender run test:e2e`.

Probar el sign-in a mano (usuarios demo de la sección 4):

```bash
curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
# -> {"token":"<jwt>","user":{"email":"operator@hubx.co","role":"operator"}}
```

Pruebas del runner de migraciones contra PostgreSQL real (opcional; usan una base desechable):

```bash
make up
make psql   # dentro: CREATE DATABASE hubx_test; \q
HUBX_TEST_DB_URL="postgres://hubx:$(grep POSTGRES_PASSWORD .env | cut -d= -f2)@localhost:5432/hubx_test?sslmode=disable" \
  go -C hubx-api test -count=1 ./internal/migrate/
```

### Problemas conocidos

| Síntoma | Causa y solución |
|---|---|
| `port 8080 is busy` en `make test-integration` | Tienes `make up` corriendo: `make down` primero |
| PostgreSQL no arranca: puerto 5432 en uso (Postgres.app, Homebrew postgres) | `POSTGRES_PORT=5433 make up` o detén el otro PostgreSQL |
| `run make up to generate .env` | Ejecutaste `docker compose` directo; usa `make up` o `make env` |
| `govulncheck: package requires newer Go version` | Reinstala `govulncheck` y `gosec` con tu Go actual |
| `gosec` reporta G304 en `hubx-api/internal/config/config.go:105` | Hallazgo previo conocido, pendiente de B (ítem 5) |
| `npm --prefix X init -y` crea `package.json` en la raíz | Bug de npm 11; no uses `init` con `--prefix` |
| `open application.properties: permission denied` en el contenedor `migrate` | Archivos con permiso `600` en tu disco (pasa en macOS). Los Dockerfile ya copian con `--chmod=0644`; si persiste: `git ls-files -z \| xargs -0 chmod 644` |
| `govulncheck` reporta vulnerabilidades en `net/http` o `crypto/*` | Tu Go es anterior a 1.27.4: actualízalo y reinstala `gosec` y `govulncheck` |
| `make security` se detiene en `gosec` y no corre `govulncheck` | Esperado mientras siga el G304 de B; corre `govulncheck -C hubx-api ./...` aparte |

## 4. Estado de la base (puesto A)

| Ítem de la guía | Estado | Dónde |
|---|---|---|
| 1. Subir lo existente | Hecho (PR #1 fusionado); `promts/` fuera del repo | — |
| 2. `hubx-bender` con Playwright | Hecho (fusionado) | `hubx-bender/` |
| 3. `Makefile` y `make test-integration` | Hecho (v0.1.0) | `Makefile`, `scripts/`, `compose.yaml`, `*/Dockerfile` |
| 4. PostgreSQL y migraciones | Hecho (v0.1.0) | `hubx-api/migrations/`, `hubx-api/internal/migrate/`, `hubx-api/cmd/migrate/` |
| 9. Protección de ramas | Hecho: `protect-main` (PR + 1 aprobación) y `protect-release` (PR + 1 aprobación desde 2026-11-21; ningún agente fusiona); check obligatorio `agents-check` en ambas (2026-11-23); sin push directo | GitHub, Settings, Rules |
| 10. README con la guía y el backlog | Hecho | `README.md`, `docs/GUIA-EQUIPO-HUBX.md` |
| HU-01 Sign in | **Hecho** (en rama `001-sign-in`, pendiente de PR): `POST /api/auth/sign-in`, JWT 1 h, 9/9 integración | `specs/001-sign-in` |
| Despliegue | Pendiente (domingo 22) | `deploy/` |

Las decisiones están en `memory.md`: el driver pgx v5.12.0, el runner propio, el esquema, los usuarios de prueba, la
numeración de migraciones y la prueba de manipulación del ledger.

**Usuarios de prueba** (contraseña `Hubx-Demo-2026`): `operator@hubx.co` (operator), `rapido@hubx.co` y
`express@hubx.co` (ground partner), `ana@hubx.co` y `andina@hubx.co` (customer). Shipments `HBX-SEED-0001` a
`HBX-SEED-0004`, todos en Created.

## 5. Claude Code en la otra máquina

- La skill `/create-pr` viaja con el repo (`.claude/skills/create-pr/`). Pide la rama origen y la destino y crea el PR
  con título Conventional Commits en inglés y descripción en español. Requiere `gh auth login`.
- La memoria personal de Claude Code (`~/.claude/`) **no** viaja; el contexto del proyecto está en `AGENTS.md`,
  `memory.md` y este documento. Para retomar, pídele a Claude Code: "lee `docs/setup.md` y `memory.md` y sigamos con HU-01 de la guía".
- Los agentes no empujan (`AGENTS.md`): los `git push` los haces tú, por ejemplo con `! git push -u origin <rama>`.
