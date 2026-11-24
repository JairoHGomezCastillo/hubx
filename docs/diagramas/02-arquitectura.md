# 02 · Arquitectura

Arquitectura de tres capas con gateway obligatorio (principio II). El frontend nunca habla directo
con la API: todo el tráfico pasa por el gateway. Stack fijado por el principio III: Go estándar
(`net/http` + `ServeMux`), MySQL, React + TypeScript estricto.

## Componentes y flujo de tráfico

```mermaid
flowchart LR
    Browser["Navegador"]

    subgraph MFE["hubx-mfe · React + TS"]
        UI["Componentes<br/>(estados: carga/éxito/fallo)"]
        VITE["Proxy de Vite (dev)<br/>/api → gateway"]
    end

    subgraph GW["hubx-gw · Gateway :8080"]
        GHEALTH["GET /health"]
        PROXY["/api/* → reverse proxy<br/>StripPrefix(/api)"]
        SEC["Cabeceras seguridad · CORS ·<br/>rate limiting (pendiente)"]
    end

    subgraph API["hubx-api · :8081"]
        AHEALTH["GET /health"]
        MOD["internal/módulos<br/>Register(mux) (pendiente)"]
        REPO["LedgerRepository<br/>(interfaz, pendiente)"]
    end

    DB[("MySQL<br/>(pendiente)")]

    Browser --> UI --> VITE
    VITE -->|"solo /api"| PROXY
    PROXY -->|"API_URL por entorno"| MOD
    PROXY -.->|"error upstream → 502"| GW
    MOD --> REPO --> DB

    classDef done fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef pend fill:#f1f5f9,stroke:#94a3b8,color:#334155,stroke-dasharray:4 3;
    class UI,VITE,GHEALTH,PROXY,AHEALTH done;
    class SEC,MOD,REPO,DB pend;
```

**Estado actual del código:** solo `/health` en gateway y API, y el reverse proxy funcionan
hoy. Los módulos de dominio, el `LedgerRepository`, la persistencia MySQL y los controles de
seguridad del gateway están definidos por la constitución pero aún no implementados.

## Despliegue y configuración por entorno

```mermaid
flowchart TB
    subgraph Env["Variables de entorno · application.properties"]
        P1["hubx-gw: PORT=8080, API_URL"]
        P2["hubx-api: PORT=8081"]
        P3["Secretos (JWT, DB, secreto compartido)<br/>SOLO por entorno, nunca en el repo"]
    end

    subgraph Run["Ejecución"]
        GW2["hubx-gw :8080"]
        API2["hubx-api :8081"]
        DB2[("MySQL")]
    end

    BENDER["hubx-bender<br/>Workspace de QA (Playwright/TS)<br/>NO es componente en ejecución"]

    P1 --> GW2
    P2 --> API2
    P3 --> GW2
    P3 --> API2
    GW2 --> API2 --> DB2
    BENDER -.->|"pruebas E2E vía gateway"| GW2

    classDef cfg fill:#fef9c3,stroke:#ca8a04,color:#422006;
    classDef run fill:#dbeafe,stroke:#2563eb,color:#0c2a66;
    classDef qa fill:#fee2e2,stroke:#dc2626,color:#450a0a,stroke-dasharray:4 3;
    class P1,P2,P3 cfg;
    class GW2,API2,DB2 run;
    class BENDER qa;
```

## Estructura interna de un servicio Go (principio IV)

Cada módulo vive bajo `internal/<módulo>`, expone `Register(mux)` y se inyecta por constructores
que devuelven interfaces. El `main` se mantiene mínimo.

```mermaid
flowchart TB
    MAIN["cmd/&lt;svc&gt;/main.go<br/>slog JSON · ReadHeaderTimeout 5s<br/>lee entorno · llama Register()"]
    MUX["http.ServeMux<br/>rutas nativas Go 1.22+ ('GET /ruta')"]

    subgraph Modulo["internal/&lt;módulo&gt;"]
        REG["Register(mux)"]
        H["Handler (interfaz)<br/>traduce errores centinela → HTTP"]
        SVC["Servicio de dominio (interfaz)"]
        REPOI["Repositorio (interfaz)"]
    end

    TEST["&lt;módulo&gt;_test.go<br/>table-driven, junto al código (TDD)"]

    MAIN --> MUX --> REG --> H --> SVC --> REPOI
    TEST -.->|"prueba comportamiento"| H

    classDef go fill:#cffafe,stroke:#0891b2,color:#083344;
    class MAIN,MUX,REG,H,SVC,REPOI,TEST go;
```

> El módulo `health` ya sigue este patrón: `NewHandler()` devuelve `http.Handler`, `Register(mux)`
> monta `GET /health`, y acepta solo GET (405 en otros métodos).
