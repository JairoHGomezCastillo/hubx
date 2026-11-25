# Diagramas y flujos de HUBX

Documentación visual del proyecto en dos planos: la **arquitectura de la aplicación** y los
**flujos agénticos** que la construyen. Todos los diagramas están en
[Mermaid](https://mermaid.js.org/), por lo que GitHub, el IDE y la mayoría de visores Markdown
los renderizan sin herramientas externas.

> Fuente de verdad: `.specify/memory/constitution.md` (autoridad máxima) y `backlog/backlog.md`.
> Si un diagrama y la constitución discrepan, prevalece la constitución y este directorio debe
> corregirse.

## Índice

| Documento | Contenido |
|---|---|
| [01-contexto.md](./01-contexto.md) | Contexto del sistema: actores, roles y límites |
| [02-arquitectura.md](./02-arquitectura.md) | Componentes, despliegue y estructura interna de los servicios |
| [03-dominio-ledger.md](./03-dominio-ledger.md) | Modelo de dominio, etapas del envío y ledger encadenado |
| [04-flujos-aplicacion.md](./04-flujos-aplicacion.md) | Secuencias de las historias (inicio de sesión, crear envío, trazabilidad, verificación, conteos, evidencia) |
| [05-flujos-agenticos.md](./05-flujos-agenticos.md) | Equipo de agentes, ciclo SDD y orquestación de una historia |
| [06-devsecops-cicd.md](./06-devsecops-cicd.md) | CI/CD, puertas de seguridad, entornos y modelo de ramas |

## Convenciones

- Los diagramas describen el **estado real** del repositorio, revisado el 2026-11-25 contra el código, los
  contratos, los workflows y la constitución. Lo que aún no existe se marca como *(pendiente)*.
- Lo marcado *(historia 011)* está en la rama `011-gateway-hardening` y aún no se ha publicado.
- Lenguaje del negocio (del backlog): envío, etapa, avance/actualización, huella, responsable,
  socio de tierra. En el código los nombres están en inglés (`shipment`, `stage`, `update`, `fingerprint`).
- Los tres servicios en ejecución son `hubx-gw`, `hubx-api` y `hubx-mfe`. `hubx-bender` es el
  workspace de QA y **no** es un componente en ejecución.

## Cómo editar

1. Edita el bloque ` ```mermaid ` correspondiente.
2. Previsualiza en el IDE o en GitHub antes de confirmar.
3. Si cambias una regla de negocio o un principio, actualiza primero la constitución/backlog y
   luego el diagrama.
