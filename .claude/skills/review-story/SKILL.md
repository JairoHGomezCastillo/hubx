---
name: review-story
description: Revisión independiente de un PR de historia de HUBX antes de aprobarlo. Ejecuta scripts/review-story.sh (destino, CI, zonas de agentes, spec sin tecnología, Gherkin literal del backlog, sin escenarios inventados, @e2e, independencia de las semillas, estructura Go, mocks, tidy, pruebas, seguridad y la suite dos veces sobre la misma base) y añade los juicios que una máquina no puede hacer. Termina con un veredicto, el prompt de corrección para el lead y publica la revisión en el PR (un comentario de resumen que se actualiza en cada re-revisión, más comentarios en línea). Úsalo cuando el usuario pida "revisa el PR", "revisa la historia" o invoque /review-story.
argument-hint: "<número de PR> [--fast]"
---

# Revisar una historia

Objetivo: decirle al humano si puede aprobar el PR de una historia, con evidencia, y si no, darle el prompt exacto para
que su `hubx-lead` lo corrija. **No corriges nada tú**: el arreglo lo hacen los agentes del equipo (el humano está
aprendiendo el flujo para su demo).

## 1. Checks mecánicos

```bash
bash scripts/review-story.sh <PR> [--fast]
```

- Corre en un worktree temporal y un proyecto Docker aislado (`hubx-review-<PR>`): no toca la copia del usuario ni su
  pila local. Si los puertos 8080/3000/5432 están ocupados (otro `make test-integration` o `make up`), lo avisa: no
  pares la pila de otro proceso sin preguntar.
- `--fast` salta la suite, el build del MFE y los escaneos; úsalo solo para una primera pasada.
- Reproduce su tabla tal cual en la respuesta. Nunca marques como aprobado algo que el script no ejecutó.

## 2. Juicios (lo que el script no puede decidir)

Lee el diff (`gh pr diff <PR>`), `specs/NNN-*/` y `pr.md`, y responde cada punto con evidencia (archivo:línea o comando):

1. **Reglas de negocio:** cada respuesta de clarify en `spec.md` debe venir del backlog o de una decisión humana
   registrada. Si un agente la dedujo de un esquema o la inventó, el humano debe confirmarla (comentario en el PR).
2. **`@e2e`:** es una copia literal de un escenario del backlog y recorre el navegador (MFE), no llama a la API.
3. **`qa-report.md`:** cada afirmación tiene evidencia; contrástala con el código (p. ej. "el gateway aplica CORS" →
   búscalo). Una afirmación falsa es un hallazgo aunque todo esté en verde.
4. **Autoría real:** cada commit `[agent:rol]` contiene trabajo de ese rol; si `pr.md` o el resumen del lead dicen
   que el lead "corrigió" algo de un subagente, señálalo.
5. **Alcance:** nada de otras historias ni de otros compañeros (CI, README de otro dueño) sin coordinación.
6. **Riesgos de entorno:** datos que la historia deja en staging (persistente), migraciones nuevas y su número,
   secretos, puertos.

## 3. Respuesta (en español)

1. Veredicto en una línea: **aprobable** / **aprobable con observaciones** / **no aprobable**.
2. La tabla del script.
3. Hallazgos de juicio, cada uno con evidencia y por qué importa (rúbrica, staging, constitución).
4. Si hay algo que corregir: un bloque con el **prompt para el lead** (en español), con los puntos numerados y el
   criterio de "hecho" (por ejemplo, "make test-integration en verde y la suite dos veces sobre la misma base").
5. Si es aprobable: recuerda fusionar con merge commit y que al fusionar en la release staging se despliega solo.

## 4. Publicar la revisión en el PR

Deja la revisión en el PR para que el equipo y el evaluador la vean, sin preguntar (el usuario lo pidió así):

1. Escribe en un archivo temporal (fuera del repo, `mktemp`) el **resumen**: el veredicto, la tabla del script, los
   hallazgos de juicio y, si hay correcciones, el prompt para el lead. Es el mismo contenido de la respuesta del
   paso 3, en markdown.
2. Escribe en otro archivo temporal los **hallazgos con archivo y línea** del diff (uno por línea, separados por
   tabulador: `ruta<TAB>línea<TAB>comentario`), solo los que aporten algo en esa línea concreta. Usa la línea del
   archivo en el commit de cabeza del PR.
3. Publica:
   ```bash
   bash scripts/review-comment.sh <PR> <resumen.md> [hallazgos.tsv]
   ```
   - Hay **un solo comentario de resumen por PR**: si ya existe uno de review-story del usuario, se **edita**; volver a
     revisar no deja comentarios repetidos.
   - Los hallazgos fuera del diff (GitHub no los acepta en línea) pasan al resumen.
   - Solo comenta: no aprueba ni pide cambios. El autor del PR no puede aprobar su propio PR y la aprobación es de un
     compañero.
4. Devuelve al usuario el enlace del comentario. Nunca incluyas secretos, URLs de bases de datos ni tokens en lo que
   publicas.
