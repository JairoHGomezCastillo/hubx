# 05 · Flujos agénticos

HUBX se construye con un equipo de cinco agentes (`.opencode/agents/`) sobre un ciclo
Spec-Driven Development (`.specify/` + Spec Kit). El agente por defecto es `hubx-lead`, único punto
de entrada y única voz que reporta al usuario.

## Equipo y fronteras por permisos

El aislamiento entre agentes se refuerza por permisos (denegación por defecto + allow-list), no solo
por convención. Cada frontera de la arquitectura tiene un guardrail agéntico que la hace cumplir.

```mermaid
flowchart TB
    LEAD["hubx-lead · primary<br/>orquesta, valida, consolida<br/>escribe specs/ y backlog/<br/>git commit: ask · push: denegado"]

    QA["qa-seguridad · subagent<br/>escribe hubx-bender/**<br/>gosec · govulncheck · curl · playwright"]
    BE["backend-go · subagent<br/>escribe hubx-api/** y hubx-gw/**<br/>go build/test/vet"]
    FE["frontend-react · subagent<br/>escribe hubx-mfe/**<br/>npm · npx"]
    DO["devops · subagent<br/>escribe infra · .github/ · deploy/<br/>docker · make"]

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

## Ciclo SDD por historia

El lead conduce las fases sin saltar ninguna y sin avanzar sin evidencia (constitución, proceso de
desarrollo). El `workflow.yml` de Spec Kit añade puertas de revisión humana entre fases.

```mermaid
flowchart LR
    S["specify<br/>spec.md: QUÉ y POR QUÉ"] --> RS{{"revisión humana<br/>(aprobar/rechazar)"}}
    RS -->|rechaza| ABORT1["aborta"]
    RS -->|aprueba| C["clarify<br/>máx. 3 preguntas"]
    C --> P["plan<br/>plan.md: el CÓMO"]
    P --> RP{{"revisión humana<br/>(aprobar/rechazar)"}}
    RP -->|rechaza| ABORT2["aborta"]
    RP -->|aprueba| T["tasks<br/>tasks.md: tareas atómicas"]
    T --> I["implement<br/>delegación en subagentes"]
    I --> A["analyze<br/>contraste vs spec + constitución"]

    classDef phase fill:#eef2ff,stroke:#6366f1,color:#312e81;
    classDef gate fill:#fef9c3,stroke:#ca8a04,color:#422006;
    class S,C,P,T,I,A phase;
    class RS,RP gate;
```

## Orquestación de la fase implement

Orden por defecto: QA escribe primero las pruebas que fallan (RED), backend implementa hasta GREEN,
QA verifica, luego frontend si aplica, y devops cierra con build y arranque. Cada subagente devuelve
un contrato JSON con `status` y evidencia; el lead no avanza si `status != ok`.

```mermaid
sequenceDiagram
    actor User as Usuario
    participant Lead as hubx-lead
    participant QA as qa-seguridad
    participant BE as backend-go
    participant FE as frontend-react
    participant DO as devops

    User->>Lead: historia (p. ej. HU-04)
    Lead->>Lead: diagnóstico de fase · specify→plan→tasks

    Lead->>QA: contrato JSON (fase implement)
    QA-->>Lead: pruebas en RED (evidencia de fallo)

    Lead->>BE: contrato JSON (implementar hasta GREEN)
    BE-->>Lead: status ok + go test pass

    Lead->>QA: contrato JSON (verificar)
    QA-->>Lead: GREEN + cobertura Gherkin + hallazgos (OWASP/CWE/CVE)

    opt la historia toca el MFE
        Lead->>FE: contrato JSON (conectar UI)
        FE-->>Lead: status ok + npm run build pass
    end

    Lead->>DO: contrato JSON (build, arranque, config)
    DO-->>Lead: status ok + evidencia /health

    Lead->>Lead: fase analyze (vs spec y constitución)
    Lead-->>User: resumen + evidencia + commit propuesto [agent:rol]
```

## Contrato de delegación

El lead envía a cada subagente un contrato explícito y exige un contrato de retorno. Así la
evidencia es verificable y las fronteras quedan declaradas en cada invocación.

```mermaid
flowchart LR
    subgraph IN["Contrato de entrada (lead → subagente)"]
        I1["story_id · phase · objective"]
        I2["acceptance (Gherkin)"]
        I3["constraints (principios)"]
        I4["owned_paths · forbidden"]
        I5["report_schema"]
    end
    subgraph OUT["Contrato de retorno (subagente → lead)"]
        O1["status: ok|blocked|failed"]
        O2["summary · artifacts"]
        O3["evidence (comando + resultado)"]
        O4["findings (severidad, OWASP, CWE, CVE)"]
        O5["decisions · blockers · next_agent"]
    end
    IN ==> OUT

    classDef io fill:#f1f5f9,stroke:#64748b,color:#1e293b;
    class I1,I2,I3,I4,I5,O1,O2,O3,O4,O5 io;
```

## Reglas de escalada

```mermaid
flowchart TB
    START["Durante la orquestación"] --> COND{"¿Condición de escalada?"}
    COND -->|"conflicto historia ↔ constitución"| ESC["DETENER y escalar al usuario"]
    COND -->|"subagente falla 3 veces seguidas"| ESC
    COND -->|"decisión de negocio no cubierta por el backlog"| ESC
    COND -->|"no"| CONT["continuar el flujo"]

    classDef stop fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef go fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class ESC stop;
    class CONT go;
```
