---
name: create-pr
description: Crea un pull request en GitHub entre dos ramas ya empujadas. Solo pide la rama origen y la destino; genera el título en inglés con Conventional Commits y la descripción en español siguiendo la plantilla de PR de GitHub (o la del repositorio si existe). Úsalo cuando el usuario pida "crea el PR", "abre un pull request" o invoque /create-pr.
argument-hint: "[rama-origen] [rama-destino]"
---

# Crear un pull request

Objetivo: el usuario solo indica **rama origen** y **rama destino**; todo lo demás lo deduces del repositorio.
No empujas ramas (en HUBX los agentes no hacen `git push`) y no inventas resultados de pruebas.

## 1. Resolver `gh`

```bash
GH="$(command -v gh || echo '/c/Program Files/GitHub CLI/gh.exe')"
"$GH" auth status
```

Si no hay sesión, pide al usuario que ejecute `! "<ruta de gh>" auth login --hostname github.com --git-protocol https --web` y detente.

## 2. Pedir las ramas

- Si `$ARGUMENTS` trae dos nombres, úsalos como `origen destino`.
- Si no, pregunta con **una sola** llamada a AskUserQuestion con dos preguntas:
  - Origen: opciones = rama actual (recomendada) y las ramas remotas más recientes
    (`git for-each-ref --sort=-committerdate --format='%(refname:short)' refs/remotes/origin | head`).
  - Destino: opciones = la rama release que indique el usuario (en HUBX, `release/vX.Y.Z[-nombre]`, dato obligatorio de cada historia; ver
    `memory.md`), la rama por defecto (`"$GH" repo view --json defaultBranchRef -q .defaultBranchRef.name`) y `main`.
- No hagas más preguntas: título y descripción se generan sin pedir confirmación.

## 3. Validar antes de crear

```bash
git fetch -q origin
git rev-parse --verify -q "origin/$SRC" >/dev/null   # origen empujado
git rev-parse --verify -q "origin/$DST" >/dev/null   # destino existe
git rev-list --count "origin/$DST..origin/$SRC"      # > 0 commits
git rev-list --count "origin/$SRC..$SRC" 2>/dev/null # commits locales sin empujar
"$GH" pr list --head "$SRC" --base "$DST" --state open --json number,url
```

- Origen no está en `origin`: di al usuario `! git push -u origin <origen>` y detente.
- Cero commits de diferencia: informa que no hay nada que proponer y detente.
- Hay commits locales sin empujar: avisa (el PR no los incluirá) y sigue.
- Ya existe un PR abierto para ese par: muestra su URL y detente; solo lo actualizas con `"$GH" pr edit` si el
  usuario lo pide.

## 4. Reunir el contenido

```bash
git log --no-merges --format='%h %s%n%b' "origin/$DST..origin/$SRC"
git diff --stat "origin/$DST...origin/$SRC"
git diff "origin/$DST...origin/$SRC" -- . ':(exclude)*.lock' ':(exclude)*-lock.json' ':(exclude)go.sum' | head -800
```

Busca además: historias (`HU-\d+`) en ramas y commits, carpetas `specs/NNN-*` tocadas, `qa-report.md` con resultados,
issues (`#123`) mencionados. Solo cita evidencia que exista en el repo o en los commits.

## 5. Título (inglés, Conventional Commits)

Formato: `<type>(<scope>)<!>: <summary>`

- `type`: el del cambio dominante: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`, `style`,
  `revert`. Si los commits ya siguen Conventional Commits, respeta el tipo mayoritario.
- `scope`: opcional y corto: la historia (`hu-02`), el servicio (`api`, `gw`, `mfe`, `bender`) o el área
  (`foundation`, `deps`, `infra`).
- `!` solo si hay un cambio incompatible (contrato HTTP, esquema o config que rompe a otros).
- `summary`: imperativo, minúsculas, sin punto final, que el título completo no supere 72 caracteres.

Ejemplos: `feat(hu-02): create shipments as operator`, `build(foundation): add compose stack and migration runner`.

## 6. Descripción (español)

1. Si el repo tiene plantilla, úsala respetando sus encabezados y rellena cada sección en español:
   `.github/pull_request_template.md`, `.github/PULL_REQUEST_TEMPLATE.md`, `docs/pull_request_template.md`,
   `pull_request_template.md` o `.github/PULL_REQUEST_TEMPLATE/*.md` (elige la que corresponda al cambio).
2. Si no hay plantilla, usa esta (estructura habitual de GitHub). Quita las secciones que no apliquen, salvo
   Descripción, Cambios y Pruebas:

```markdown
## Descripción
<Qué cambia y por qué, en 2-4 frases. Si hay historia: "Historia: HU-0X <título>. Spec: `specs/0X-nombre/`".>

## Tipo de cambio
- [ ] Corrección de error (cambio que no rompe nada y soluciona un problema)
- [ ] Nueva funcionalidad (cambio que no rompe nada y agrega funcionalidad)
- [ ] Cambio incompatible (rompe o modifica funcionalidad existente)
- [ ] Documentación
- [ ] Infraestructura, CI o dependencias

## Cambios principales
- <un punto por cambio relevante, con la ruta entre comillas invertidas>

## ¿Cómo se ha probado?
- <comando>: <resultado real tomado de commits, qa-report.md o la sesión>
- Lo que no se ha ejecutado se marca como `pendiente`, nunca como aprobado.

## Checklist
- [ ] Mi código sigue las convenciones del proyecto
- [ ] Revisé mi propio código
- [ ] Agregué pruebas que demuestran que la corrección o la funcionalidad funcionan
- [ ] Las pruebas unitarias y de integración pasan localmente
- [ ] Actualicé la documentación correspondiente
- [ ] No incluyo secretos ni credenciales

## Issues relacionados
Closes #<n>   <!-- solo si existe -->

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

- Marca con `[x]` solo lo que puedas comprobar (por ejemplo, el tipo de cambio por los archivos tocados, "agregué
  pruebas" si el diff incluye pruebas). Lo demás queda `[ ]` para que lo marque el autor.
- En HUBX agrega, después de "¿Cómo se ha probado?", la evidencia que pide la guía si aplica: `make test-integration`,
  `go test` / `npm test`, `gosec` / `govulncheck` y la ruta del `qa-report.md`, con el resultado real o `pendiente`.
- Nombres de código, rutas y comandos van en inglés y entre comillas invertidas; la prosa en español.
- No escribas `@usuario` salvo que sea un login real que deba ser notificado.

## 7. Crear el PR

Escribe el cuerpo en un archivo temporal (no en el repo) y crea el PR:

```bash
BODY="$(mktemp)"; cat > "$BODY" <<'EOF'
...
EOF
"$GH" pr create --base "$DST" --head "$SRC" --title "$TITLE" --body-file "$BODY"
rm -f "$BODY"
```

Si el usuario pidió borrador, añade `--draft`. Al terminar, responde con la URL del PR, el título y un resumen de una
línea; no repitas la descripción completa.
