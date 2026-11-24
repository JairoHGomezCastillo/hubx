# 06 · DevSecOps y CI/CD

El agente `devops` define todo el pipeline como **pipeline as code** versionado en `.github/**`,
con seguridad integrada (shift-left) y puertas que bloquean el despliegue. La constitución exige que
`gosec` y `govulncheck` pasen sin hallazgos antes de merge, CI verde antes de cualquier merge, y
`master`/`main` siempre desplegable.

> *(Pendiente):* el pipeline aún no existe en `.github/`. Este documento describe el objetivo
> definido en `.opencode/agents/devops.md`.

## Pipeline as code (etapas y puertas)

```mermaid
flowchart LR
    PR["Push / Pull Request"] --> BT["Build & Test<br/>go build/test · npm run build"]
    BT --> SAST["SAST<br/>gosec → CWE / OWASP"]
    SAST --> SCA["SCA<br/>govulncheck → CVE / GHSA"]
    SCA --> SEC["Escaneo de secretos<br/>historial + diff"]
    SEC --> IMG["Escaneo de imágenes<br/>vulnerabilidades del contenedor"]
    IMG --> GATE{"¿Todas las puertas<br/>en verde?"}
    GATE -->|No| BLOCK["Bloquea merge / deploy"]
    GATE -->|Sí| DEPLOY["Despliegue<br/>(master siempre desplegable)"]

    classDef stage fill:#eff6ff,stroke:#3b82f6,color:#0c2a66;
    classDef sec fill:#fef2f2,stroke:#ef4444,color:#450a0a;
    classDef ok fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef block fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    class BT,DEPLOY stage;
    class SAST,SCA,SEC,IMG sec;
    class GATE ok;
    class BLOCK block;
```

## Puertas de calidad y seguridad

Cada etapa produce un artefacto o reporte. Los secretos del pipeline vienen del gestor de secretos
del CI (nunca en el YAML) y el workflow usa mínimo privilegio.

```mermaid
flowchart TB
    subgraph Gates["Puertas que fallan el build"]
        G1["gosec: hallazgos → CWE + categoría OWASP"]
        G2["govulncheck: vulnerabilidad → CVE (+ GHSA, versión corregida)"]
        G3["escaneo de secretos: credenciales en repo/historial"]
        G4["escaneo de imágenes: base mínima · usuario no-root · sin secretos en capas"]
    end
    RESULT{"¿Alguna en rojo<br/>o omitida?"}
    G1 --> RESULT
    G2 --> RESULT
    G3 --> RESULT
    G4 --> RESULT
    RESULT -->|Sí| NO["status != ok · deploy bloqueado"]
    RESULT -->|No| YES["avanza a deploy"]

    classDef gate fill:#fef9c3,stroke:#ca8a04,color:#422006;
    classDef no fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef yes fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class G1,G2,G3,G4 gate;
    class NO no;
    class YES yes;
```

## Modelo de ramas (constitución · Integración y Despliegue)

`release` sale de `master`; las `feature` entran a `release`; `release` entra a `master`; los
hotfixes nacen de `master` y vuelven a `master`. Cada feature con PR obligatorio y CI verde.

```mermaid
gitGraph
    commit id: "master base"
    branch release/sdd-hubx
    checkout release/sdd-hubx
    commit id: "release abierta"
    branch feature/HU-xx
    checkout feature/HU-xx
    commit id: "implementa HU"
    commit id: "CI verde + PR"
    checkout release/sdd-hubx
    merge feature/HU-xx
    checkout main
    merge release/sdd-hubx tag: "desplegable"
    branch hotfix/urgente
    checkout hotfix/urgente
    commit id: "fix"
    checkout main
    merge hotfix/urgente
```

## Relación con los agentes de seguridad

`qa-seguridad` y el pipeline de `devops` comparten las mismas herramientas y la misma clasificación
de hallazgos (OWASP Top 10, CWE, CVE), de modo que lo que QA revisa manualmente es lo que el pipeline
verifica de forma automática.

```mermaid
flowchart LR
    QA["qa-seguridad<br/>revisión manual + SAST/SCA/DAST<br/>clasifica OWASP · CWE · CVE"]
    PIPE["pipeline devops<br/>SAST · SCA · secretos · imágenes<br/>mismas puertas, automáticas"]
    CONST["Constitución · principio VI<br/>gosec + govulncheck sin hallazgos antes de merge"]

    CONST --> QA
    CONST --> PIPE
    QA -.->|"mismos estándares"| PIPE

    classDef qa fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef pipe fill:#fef3c7,stroke:#d97706,color:#5a3a06;
    classDef const fill:#e0e7ff,stroke:#4f46e5,color:#1e1b4b;
    class QA qa;
    class PIPE pipe;
    class CONST const;
```
