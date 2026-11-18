<!--
SYNC IMPACT REPORT (material temporal de revisión; eliminar antes de confirmar el commit)
=========================================================================================
Version change: (sin versión / plantilla sin completar) → 1.0.0

Modified principles (título anterior → nuevo):
- Principio 1 de la plantilla (sin título) → I. Ledger Inmutable como Fuente de Verdad
- Principio 2 de la plantilla (sin título) →
  II. Arquitectura de Tres Piezas y Gateway Obligatorio
- Principio 3 de la plantilla (sin título) → III. Stack Tecnológico y Prohibiciones
- Principio 4 de la plantilla (sin título) → IV. Diseño Modular e Inyección por Constructores
- Principio 5 de la plantilla (sin título) → V. Test-First (NO NEGOCIABLE)

Added sections:
- VI. Seguridad No Negociable (nuevo principio)
- Proceso de Desarrollo (sección 2 de la plantilla, antes sin nombre)
- Integración y Despliegue (sección 3 de la plantilla, antes sin nombre)
- Procedimiento de enmienda, política de versionado y revisión de cumplimiento en Governance

Removed sections: Ninguna

Follow-up TODOs: Ninguno. La fecha de ratificación se fija en 2026-11-17 asumiendo
  adopción inicial, porque no existía una constitución previa. Confirmar si la adopción
  original ocurrió en una fecha anterior y corregir el metadato si es necesario.
-->

# hubx Constitution

## Core Principles

### I. Ledger Inmutable como Fuente de Verdad
El sistema es una plataforma de trazabilidad de carga aérea con aliados terrestres cuyo
estado canónico es un ledger de eventos encadenados por hash SHA-256 y firmados por el
actor que los emite. Todo evento DEBE ser inmutable: NO se actualiza ni se elimina; las
correcciones se registran como eventos nuevos. El sistema DEBE usar un ledger verificable
en lugar de blockchain on-chain durante el alcance temporal definido, y TODO acceso al
ledger DEBE realizarse a través de la interfaz LedgerRepository para permitir la
migración futura a una red real.

*Rationale*: la inmutabilidad y la firma por actor garantizan la auditabilidad de extremo
a extremo; la abstracción del repositorio evita acoplar el dominio a una tecnología de
persistencia concreta.

### II. Arquitectura de Tres Piezas y Gateway Obligatorio
El sistema DEBE estar compuesto por hubx-gw (gateway), hubx-api (API base) y hubx-mfe
(frontend). El frontend NO DEBE comunicarse directamente con la API: TODO el tráfico
DEBE pasar por el gateway. La API DEBE exigir el secreto compartido del gateway y DEBE
validar el JWT por sí misma, de forma independiente.

*Rationale*: el gateway concentra el control perimetral; la validación propia del JWT en
la API evita delegar la autenticación en un único punto de fallo.

### III. Stack Tecnológico y Prohibiciones
El backend DEBE implementarse en la versión estable más reciente de Go usando la
biblioteca estándar (net/http con ServeMux) para el enrutamiento; NO se DEBEN introducir
frameworks web externos. La persistencia DEBE ser MySQL. El frontend DEBE ser React con
TypeScript en modo estricto. La configuración DEBE gestionarse en el archivo
application.properties.

*Rationale*: restringir el stack reduce la superficie de dependencias y homogeniza el
mantenimiento, las pruebas y la incorporación de nuevos agentes.

### IV. Diseño Modular e Inyección por Constructores
El código Go DEBE organizarse en un módulo por funcionalidad bajo internal/<modulo>,
exponiendo una función Register(mux) que registre sus rutas. Las dependencias DEBEN
inyectarse manualmente mediante constructores que devuelven interfaces. Los errores
centinela del dominio DEBEN traducirse a respuestas HTTP en los handlers. El logging
DEBE realizarse con log/slog en formato JSON.

*Rationale*: la inversión de dependencias mediante interfaces hace el dominio testeable y
desacopla el transporte HTTP de la lógica de negocio.

### V. Test-First (NO NEGOCIABLE)
TDD es obligatorio para las pruebas unitarias, centradas en el comportamiento observable
del componente; el ciclo rojo-verde-refactorizar DEBE respetarse. Cada escenario Gherkin
(exitoso, borde y error) DEBE tener una prueba de integración contra la API. Ninguna
historia se da por terminada sin pruebas automatizadas de extremo a extremo con Playwright
en TypeScript, y todas DEBEN pasar en verde.

*Rationale*: escribir las pruebas primero fija el comportamiento esperado y evita que la
cobertura se convierta en una actividad posterior a la entrega.

### VI. Seguridad No Negociable
TODAS las entradas DEBEN validarse en el servidor y TODAS las consultas de datos DEBEN ser
parametrizadas. La autorización DEBE usar JWT con los roles operador, aliado y cliente, y
TODO acceso a recursos DEBE restringirse a los recursos propios. Los secretos DEBEN
suministrarse exclusivamente mediante variables de entorno. Las contraseñas DEBEN
hashearse con bcrypt. DEBEN aplicarse cabeceras de seguridad, CORS restrictivo y rate
limiting en el gateway. gosec y govulncheck DEBEN pasar sin hallazgos antes de fusionar.

*Rationale*: estos controles cubren las superficies de ataque críticas y su verificación
automatizada impide regresiones silenciosas.

## Proceso de Desarrollo

Las historias de usuario DEBEN escribirse en formato Como/Quiero/Para, ser lo más
pequeñas posible manteniendo valor, cumplir las 3C e INVEST e incluir criterios de
aceptación Gherkin BDD con caso exitoso, borde y error.

TODA funcionalidad DEBE recorrer las fases specify, clarify, plan, tasks, implement y
analyze. Los artefactos DEBEN residir en specs/NNN-nombre/. Las especificaciones DEBEN
describir el QUÉ y el POR QUÉ, NO el CÓMO.

El equipo DEBE organizarse en cinco agentes con carpetas propias: lider/orquestador,
backend-go, frontend-react, qa-seguridad y devops. Los commits DEBEN etiquetarse
[agent:rol]. QA DEBE escribir las pruebas de integración antes de la implementación y
DEBE revisar antes de integrar.

## Integración y Despliegue

Cada historia DEBE desarrollarse en una rama propia con PR obligatorio. La CI DEBE estar
en verde antes de cualquier fusión y master/main DEBE permanecer siempre desplegable.

El modelo de ramas DEBE seguir: de master sale una release; las funcionalidades van a una
release; las release se fusionan a master; los hotfix salen y van a master. La
configuración DEBE gestionarse por entorno.

## Governance

Esta constitución prevalece sobre cualquier otra práctica del proyecto; en caso de
conflicto, prevalece lo aquí definido.

Las enmiendas DEBEN documentarse en una propuesta, contar con aprobación e incluir un plan
de migración; el impacto DEBE registrarse en el Sync Impact Report de este archivo.

El versionado DEBE seguir Semantic Versioning: MAJOR para eliminaciones o redefiniciones
incompatibles de principios; MINOR para principios o secciones nuevos y ampliaciones
materiales; PATCH para aclaraciones, correcciones y refinamientos no semánticos.

TODO PR y revisión DEBEN verificar el cumplimiento de esta constitución. Toda complejidad
añadida DEBE justificarse explícitamente.

**Version**: 1.0.0 | **Ratified**: 2026-11-17 | **Last Amended**: 2026-11-17
