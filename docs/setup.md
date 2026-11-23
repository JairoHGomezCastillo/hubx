# Puesta en marcha de HUBX (macOS, con notas para Windows)

Cómo dejar una máquina lista para trabajar en HUBX y en qué estado está la base del proyecto. Es la sección 3 de
la guía del equipo hecha paso a paso y comprobada. La rama de integración es `release/sdd-hubx-system`; la guía la
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
git switch release/sdd-hubx-system        # o la rama en la que vayas a trabajar
gh auth login --hostname github.com --git-protocol https --web
npm --prefix hubx-bender ci
npm --prefix hubx-mfe ci
```

El `.env` con los secretos locales **no está en el repo**. `make` lo genera solo, con valores aleatorios, la primera
vez. Si quieres regenerarlo, bórralo.

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

Resultado esperado de `make test-integration`: `2 passed` (`SMOKE_gateway_health` y
`SMOKE_api_health_through_gateway`), en unos 35 s la primera vez.

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
| `gosec` reporta G304 en `hubx-api/internal/config/config.go:102` | Hallazgo previo conocido, pendiente de B (ítem 5) |
| `npm --prefix X init -y` crea `package.json` en la raíz | Bug de npm 11; no uses `init` con `--prefix` |
| `open application.properties: permission denied` en el contenedor `migrate` | Archivos con permiso `600` en tu disco (pasa en macOS). Los Dockerfile ya copian con `--chmod=0644`; si persiste: `git ls-files -z \| xargs -0 chmod 644` |
| `govulncheck` reporta vulnerabilidades en `net/http` o `crypto/*` | Tu Go es anterior a 1.27.4: actualízalo y reinstala `gosec` y `govulncheck` |
| `make security` se detiene en `gosec` y no corre `govulncheck` | Esperado mientras siga el G304 de B; corre `govulncheck -C hubx-api ./...` aparte |

## 4. Estado de la base (puesto A)

| Ítem de la guía | Estado | Dónde |
|---|---|---|
| 1. Subir lo existente | Hecho (PR #1 fusionado); `promts/` fuera del repo | — |
| 2. `hubx-bender` con Playwright | Hecho (fusionado) | `hubx-bender/` |
| 3. `Makefile` y `make test-integration` | Hecho (ya en `release/sdd-hubx-system`) | `Makefile`, `scripts/`, `compose.yaml`, `*/Dockerfile` |
| 4. PostgreSQL y migraciones | Hecho (ya en `release/sdd-hubx-system`) | `hubx-api/migrations/`, `hubx-api/internal/migrate/`, `hubx-api/cmd/migrate/` |
| 9. Protección de ramas | Hecho: `protect-main` (PR + 1 aprobación) y `protect-release` (PR + 1 aprobación desde 2026-11-21; ningún agente fusiona); check obligatorio `agents-check` en ambas (2026-11-23); sin push directo | GitHub, Settings, Rules |
| 10. README con la guía y el backlog | Hecho | `README.md`, `docs/GUIA-EQUIPO-HUBX.md` |
| HU-01 Sign in | **Siguiente** (1 a 4 ya fusionados) | `specs/001-sign-in` |
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
