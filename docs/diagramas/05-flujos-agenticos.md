# 05 · Flujos agénticos

HUBX se construye con un equipo de cinco agentes sobre un ciclo Spec-Driven Development (`.specify/` +
Spec Kit). La fuente de sus instrucciones es `.opencode/agents/`; `make agents-sync` genera las versiones de
Kiro y Claude Code. El agente por defecto es `hubx-lead`, único punto de entrada y única voz que reporta al
usuario.

## Equipo y fronteras por permisos

El aislamiento entre agentes se refuerza por permisos (denegación por defecto + allow-list), no solo
por convención. Además `make agents-zones` (CI) rechaza un commit `[agent:rol]` que toque archivos fuera de
la zona de ese rol.

```mermaid
flowchart TB
    LEAD["hubx-lead · primary<br/>orquesta, valida, consolida<br/>specs/ · backlog/ · contracts/ · memory.md · README · docs/<br/>commit, push de la rama de la historia y PR sin confirmación<br/>nunca: merge, force push, main, release, hotfix"]

    QA["qa-seguridad · subagent<br/>escribe hubx-bender/** y qa-report.md<br/>make test-integration · go test · gosec · govulncheck"]
    BE["backend-go · subagent<br/>escribe hubx-api/** y hubx-gw/**<br/>go build/test/vet/fmt/generate · go get: ask"]
    FE["frontend-react · subagent<br/>escribe hubx-mfe/**<br/>npm run/test · npm install: ask"]
    DO["devops · subagent<br/>Makefile · Dockerfiles · compose · .github/ · scripts/ · deploy/<br/>make · docker · despliega solo a staging"]

    LEAD -->|"subagent"| QA
    LEAD -->|"subagent"| BE
    LEAD -->|"subagent"| FE
    LEAD -->|"subagent"| DO

    classDef lead fill:#e0e7ff,stroke:#4f46e5,color:#1e1b4b;
    classDef qa fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef be fill:#cffafe,stroke:#0891b2,color:#083344;
    classDef fe fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef do fill:#fef3c7,stroke:#d97706,color:#5a3a06;
    class LEAD lead;
    class QA qa;
    class BE be;
    class FE fe;
    class DO do;
```

Crear una rama `release/*` nueva desde `main` pide confirmación humana. Ningún agente fusiona: es una decisión
humana siempre.

## Ciclo SDD por historia

El lead recibe la historia y su rama de integración (`release/vX.Y.Z[-nombre]`), ambas obligatorias, y conduce
las fases sin saltar ninguna y sin avanzar sin evidencia. El punto de revisión humana es el PR. (El
`workflow.yml` genérico de Spec Kit trae puertas de revisión entre fases; el flujo de `hubx-lead` no lo usa.)

```mermaid
flowchart LR
    IN["historia + rama de integración"] --> S["specify<br/>spec.md: QUÉ y POR QUÉ"]
    S --> C["clarify<br/>máx. 3 preguntas por ronda"]
    C --> P["plan<br/>plan.md y contrato: el CÓMO"]
    P --> T["tasks<br/>tasks.md: tareas por agente"]
    T --> I["implement<br/>delegación en subagentes"]
    I --> A["analyze<br/>contraste vs spec + constitución"]
    A --> PR["pr<br/>push de la rama y PR a la rama de integración"]
    PR --> H{{"revisión y merge humanos"}}

    classDef phase fill:#eef2ff,stroke:#6366f1,color:#312e81;
    classDef gate fill:#fef9c3,stroke:#ca8a04,color:#422006;
    class IN,S,C,P,T,I,A,PR phase;
    class H gate;
```

## Orquestación de la fase implement

QA escribe primero las pruebas que fallan (RED), backend y frontend implementan en paralelo contra el
contrato, QA verifica y devops despliega la rama a staging y lo registra en `deploy.md`. Cada subagente
devuelve un JSON con `status` y evidencia; el lead no avanza si `status` no es `ok` y hace un commit
`[agent:rol]` por entrega aceptada.

```mermaid
sequenceDiagram
    actor User as Usuario
    participant Lead as hubx-lead
    participant QA as qa-seguridad
    participant BE as backend-go
    participant FE as frontend-react
    participant DO as devops
    participant GH as GitHub

    User->>Lead: historia (p. ej. HU-06) y rama release
    Lead->>Lead: diagnóstico, specify, clarify, plan, tasks

    Lead->>QA: fase red
    QA-->>Lead: escenarios Cucumber y @e2e en RED (evidencia)

    par si no comparten archivos
        Lead->>BE: fase implement
        BE-->>Lead: status ok + go test
    and si la historia toca el MFE
        Lead->>FE: fase implement
        FE-->>Lead: status ok + npm run build y test
    end

    Lead->>QA: fase verify
    QA-->>Lead: suite completa, gosec, govulncheck, OWASP y hallazgos
    opt hay hallazgos
        Lead->>BE: fase fix (o al agente que corresponda)
        BE-->>Lead: corregido, QA verifica otra vez
    end

    Lead->>GH: push de la rama de la historia
    opt salvo modo demo pedido por el humano
        Lead->>DO: validación de despliegue
        DO->>GH: deploy.yml a staging
        DO-->>Lead: deploy.md con evidencia
    end

    Lead->>Lead: analyze, pr.md, memory.md, backlog
    Lead->>GH: abre el PR
    Lead-->>User: resumen + evidencia + URL del PR
    User->>GH: revisa y fusiona
```

## Contrato de delegación

El lead envía a cada subagente un contrato explícito y exige un contrato de retorno. Así la
evidencia es verificable y las fronteras quedan declaradas en cada invocación.

```mermaid
flowchart LR
    subgraph IN["Contrato de entrada (lead → subagente)"]
        I1["story_id · phase: red, verify, implement, fix, deploy"]
        I2["objective"]
        I3["acceptance (Gherkin)"]
        I4["findings (solo en fix)"]
        I5["owned_paths · forbidden"]
    end
    subgraph OUT["Contrato de retorno (subagente → lead)"]
        O1["status (obligatorio)"]
        O2["summary (obligatorio)"]
        O3["evidence: comando + resultado literal (obligatorio)"]
        O4["blockers (obligatorio)"]
    end
    IN ==> OUT

    classDef io fill:#f1f5f9,stroke:#64748b,color:#1e293b;
    class I1,I2,I3,I4,I5,O1,O2,O3,O4 io;
```

## Reglas de escalada

```mermaid
flowchart TB
    START["Durante la orquestación"] --> COND{"¿Condición de escalada?"}
    COND -->|"conflicto historia ↔ constitución"| ESC["DETENER y escalar al usuario"]
    COND -->|"subagente falla 3 veces seguidas"| ESC
    COND -->|"decisión de negocio sin cobertura"| ESC
    COND -->|"subagente bloqueado por permisos"| ESC
    COND -->|"falta la historia o su rama release"| ESC
    COND -->|"no"| CONT["continuar el flujo"]

    classDef stop fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef go fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class ESC stop;
    class CONT go;
```
