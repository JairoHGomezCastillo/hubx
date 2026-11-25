# 06 · DevSecOps y CI/CD

El agente `devops` mantiene el pipeline como código en `.github/workflows/`. La constitución exige que
`gosec` y `govulncheck` pasen sin hallazgos antes de merge, CI verde antes de cualquier merge, y
`main` siempre desplegable.

| Workflow | Cuándo | Qué hace |
|---|---|---|
| `ci.yml` | push y PR | build, pruebas, gosec, govulncheck e integración |
| `agents.yml` | push y PR | `agents-check`; `agents-zones` en PR que no van a `main` |
| `main-source.yml` | PR a `main` | solo `release/vX.Y.Z[-nombre]` o `hotfix/vX.Y.Z[-nombre]` |
| `images.yml` | push | imágenes en GHCR etiquetadas con el SHA |
| `deploy-staging.yml` | PR fusionado en una rama distinta de `main` | llama a `deploy.yml` (staging) |
| `deploy.yml` | manual o llamado | despliega las imágenes de un commit a un entorno |
| `release.yml` | release o hotfix fusionado en `main` | tag, GitHub Release, sincronización y producción |

## Pipeline as code (etapas y puertas)

```mermaid
flowchart LR
    PR["Push / Pull Request"] --> API["api<br/>build · test · gosec · govulncheck"]
    PR --> GW["gw<br/>build · test · gosec · govulncheck"]
    PR --> MFE["mfe<br/>build · lint · typecheck · test"]
    API --> INT["integration<br/>make test-integration"]
    GW --> INT
    MFE --> INT
    PR --> AG["agents-check · agents-zones"]
    PR --> SRC["release-source<br/>(solo PR a main)"]
    INT --> GATE{"¿Todo en verde?"}
    AG --> GATE
    SRC --> GATE
    GATE -->|No| BLOCK["No se fusiona"]
    GATE -->|Sí| MERGE["Aprobación y merge humanos"]
    PR --> IMG["images<br/>ghcr.io/.../hubx-servicio:SHA"]

    classDef stage fill:#eff6ff,stroke:#3b82f6,color:#0c2a66;
    classDef sec fill:#fef2f2,stroke:#ef4444,color:#450a0a;
    classDef ok fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef block fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    class MFE,INT,IMG,MERGE stage;
    class API,GW,AG,SRC sec;
    class GATE ok;
    class BLOCK block;
```

Las imágenes se construyen solo en push. El escaneo de secretos y el de imágenes no existen en los workflows.

## Entornos y promoción

Se construye una vez: los entornos despliegan imágenes, nunca ramas. La imagen probada en staging es la que se
promueve.

```mermaid
flowchart TB
    subgraph Staging["Staging (Render + rama staging de Neon)"]
        M1["PR fusionado en rama de integración<br/>o cualquier ref a mano"] --> R1["reinicia la base desde su padre<br/>(si RESET_DB_FROM_PARENT=true)"]
        R1 --> MG1["migraciones"] --> D1["despliega imágenes del SHA<br/>API, luego gateway y MFE"] --> S1["warm-up y smoke<br/>(suite salvo @needs-secret, @empty-db<br/>y @needs-limited-gateway)"]
    end
    subgraph Produccion["Producción (Render + Neon)"]
        M2["release o hotfix fusionado en main"] --> TAG["tag vX.Y.Z + GitHub Release"]
        TAG --> PROM{"promotable:<br/>¿el árbol de la cabeza del PR es igual<br/>al del commit fusionado?"}
        PROM -->|No| STOP["no se promueve<br/>producción queda igual"]
        PROM -->|"Sí, con DEPLOY_PRODUCTION=true"| MG2["migraciones"] --> D2["despliega imágenes de la cabeza del PR"] --> S2["warm-up y smoke<br/>(solo @smoke, sin escribir)"]
        TAG --> SYNC["hotfix: PR de sincronización<br/>a cada rama de integración abierta"]
    end

    classDef stage fill:#eff6ff,stroke:#3b82f6,color:#0c2a66;
    classDef block fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    class M1,R1,MG1,D1,S1,M2,TAG,MG2,D2,S2,SYNC,PROM stage;
    class STOP block;
```

Sin `DEPLOY_PRODUCTION=true`, los trabajos `promotable` y `deploy` no se ejecutan. Una release pasa por staging
antes; un hotfix solo si alguien lo despliega a mano. Los agentes no pueden desplegar a producción. La exclusión de
`@needs-limited-gateway` es de la *(historia 011)*.

## Puertas de calidad y seguridad

Los secretos del pipeline vienen de los entornos de GitHub (nunca en el YAML) y los workflows declaran
permisos mínimos.

```mermaid
flowchart TB
    subgraph Gates["Puertas que fallan el build"]
        G1["gosec en hubx-api y hubx-gw"]
        G2["govulncheck en hubx-api y hubx-gw"]
        G3["pruebas unitarias Go y MFE, lint y typecheck"]
        G4["make test-integration: API por el gateway y @e2e en navegador"]
        G5["agents-check y agents-zones"]
    end
    RESULT{"¿Alguna en rojo?"}
    G1 --> RESULT
    G2 --> RESULT
    G3 --> RESULT
    G4 --> RESULT
    G5 --> RESULT
    RESULT -->|Sí| NO["no se fusiona"]
    RESULT -->|No| YES["revisión humana"]

    classDef gate fill:#fef9c3,stroke:#ca8a04,color:#422006;
    classDef no fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef yes fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class G1,G2,G3,G4,G5 gate;
    class NO no;
    class YES yes;
```

## Modelo de ramas (constitución · Integración y Despliegue)

`main` es la única rama permanente. `release/vX.Y.Z[-nombre]` sale de `main`; las ramas de historia `NNN-nombre`
entran a la release por PR; la release entra a `main`; los hotfixes nacen de `main` y vuelven a `main`. Las ramas
se borran al fusionarse, y CI crea el tag (nunca a mano). No hay backports.

```mermaid
gitGraph
    commit id: "main"
    branch release/vX.Y.Z-nombre
    checkout release/vX.Y.Z-nombre
    commit id: "release abierta"
    branch NNN-historia
    checkout NNN-historia
    commit id: "entregas por agente"
    commit id: "CI verde + PR"
    checkout release/vX.Y.Z-nombre
    merge NNN-historia
    checkout main
    merge release/vX.Y.Z-nombre tag: "vX.Y.Z"
    branch hotfix/vX.Y.W
    checkout hotfix/vX.Y.W
    commit id: "fix"
    checkout main
    merge hotfix/vX.Y.W tag: "vX.Y.W"
```

## Relación con los agentes de seguridad

`qa-seguridad` y el pipeline usan las mismas herramientas: lo que QA ejecuta antes del PR es lo que CI repite.
QA añade la revisión con el OWASP API Security Top 10 en `qa-report.md`.

```mermaid
flowchart LR
    QA["qa-seguridad<br/>suite, gosec, govulncheck<br/>OWASP API Top 10 en qa-report.md"]
    PIPE["CI<br/>suite, gosec, govulncheck<br/>mismas puertas, automáticas"]
    CONST["Constitución · principio VI<br/>gosec + govulncheck sin hallazgos antes de merge"]

    CONST --> QA
    CONST --> PIPE
    QA -.->|"mismas herramientas"| PIPE

    classDef qa fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    classDef pipe fill:#fef3c7,stroke:#d97706,color:#5a3a06;
    classDef const fill:#e0e7ff,stroke:#4f46e5,color:#1e1b4b;
    class QA qa;
    class PIPE pipe;
    class CONST const;
```
