# 02 · Arquitectura

Arquitectura de tres capas con gateway obligatorio (principio II). El frontend nunca habla directo
con la API: todo el tráfico pasa por el gateway. Stack fijado por el principio III: Go estándar
(`net/http` + `ServeMux`), PostgreSQL, React + TypeScript estricto.

## Componentes y flujo de tráfico

```mermaid
flowchart LR
    Browser["Navegador"]

    subgraph MFE["hubx-mfe · React + TS"]
        UI["Vistas y servicios<br/>(JWT solo en memoria)"]
        NGINX["nginx (stack local y despliegue)<br/>o proxy de Vite (desarrollo)<br/>/api → gateway, mismo origen"]
    end

    subgraph GW["hubx-gw · Gateway :8080"]
        GHEALTH["GET /health"]
        SEC["Cabeceras de seguridad → CORS →<br/>JWT → límite de peticiones"]
        PROXY["/api/* → reverse proxy<br/>StripPrefix(/api) · X-Gateway-Secret"]
    end

    subgraph API["hubx-api · :8081"]
        GUARD["Exige el secreto del gateway<br/>y valida el JWT por su cuenta"]
        MOD["internal/handlers/* → internal/*<br/>auth · shipment · ledger · evidence"]
        REPO["LedgerRepository<br/>(interfaz)"]
    end

    DB[("PostgreSQL")]

    Browser --> UI --> NGINX
    NGINX -->|"solo /api"| SEC --> PROXY
    PROXY -->|"API_URL por entorno"| GUARD --> MOD
    PROXY -.->|"API sin respuesta → 502"| NGINX
    MOD --> REPO --> DB
    MOD --> DB

    classDef done fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class UI,NGINX,GHEALTH,SEC,PROXY,GUARD,MOD,REPO,DB done;
```

**Estado actual del código:** el dominio, el ledger, la persistencia y la validación del JWT en ambos
servicios están publicados. Las cabeceras de seguridad, el CORS restrictivo (lista vacía por defecto) y
el límite de peticiones (120 por minuto por usuario, 10 por minuto por correo en el inicio de sesión)
son de la *(historia 011)*; viven en memoria por instancia del gateway.

## Rutas de la API

Vistas desde el navegador; el gateway quita `/api`. Contratos en `contracts/`.

| Ruta | Historia |
|---|---|
| `GET /health` (gateway), `GET /api/health` | Salud, sin sesión |
| `POST /api/auth/sign-in` | HU-01, sin sesión |
| `POST /api/shipments` | HU-02 |
| `POST /api/shipments/{code}/updates`, `GET /api/ground-partners` | HU-03 |
| `GET /api/shipments/{code}/traceability` | HU-04 |
| `GET /api/shipments/{code}/integrity` | HU-05 |
| `GET /api/assigned-shipments` | HU-07 |
| `GET /api/shipments/stage-counts` | HU-09 |
| `GET /api/shipments/{code}/evidence` | HU-10 |

HU-06 y HU-08 no están implementadas.

## Despliegue y configuración por entorno

```mermaid
flowchart TB
    subgraph Env["Entorno > application-perfil.properties > application.properties > valor por defecto"]
        P1["hubx-gw: server.port=8080, api.url,<br/>cors.allowed-origins, ratelimit.*"]
        P2["hubx-api: server.port=8081"]
        P3["Secretos (JWT_SECRET, DB_URL, GATEWAY_SECRET)<br/>SOLO por entorno, nunca en el repo"]
    end

    subgraph Run["Ejecución: Docker Compose local · Render + Neon en staging y producción"]
        MFE2["hubx-mfe (nginx)"]
        GW2["hubx-gw :8080"]
        API2["hubx-api :8081"]
        MIG["cmd/migrate<br/>antes de desplegar"]
        DB2[("PostgreSQL")]
    end

    BENDER["hubx-bender<br/>Workspace de QA (Cucumber + Playwright)<br/>NO es componente en ejecución"]

    P1 --> GW2
    P2 --> API2
    P3 --> GW2
    P3 --> API2
    MFE2 --> GW2 --> API2 --> DB2
    MIG --> DB2
    BENDER -.->|"API por el gateway"| GW2
    BENDER -.->|"@e2e en navegador"| MFE2

    classDef cfg fill:#fef9c3,stroke:#ca8a04,color:#422006;
    classDef run fill:#dbeafe,stroke:#2563eb,color:#0c2a66;
    classDef qa fill:#fee2e2,stroke:#dc2626,color:#450a0a,stroke-dasharray:4 3;
    class P1,P2,P3 cfg;
    class MFE2,GW2,API2,MIG,DB2 run;
    class BENDER qa;
```

`cmd/api` todavía lee el puerto directamente de `PORT` (pendiente en `memory.md`).

## Estructura interna de un servicio Go (principio IV)

El dominio vive en `internal/<dominio>` y no importa `net/http`; los handlers viven en
`internal/handlers/<dominio>`, exponen `Register(mux, ...)` y traducen los errores centinela a HTTP. Las
dependencias se inyectan por constructores.

```mermaid
flowchart TB
    MAIN["cmd/api/main.go<br/>slog JSON · ReadHeaderTimeout 5s<br/>configuración y cableado"]
    MUX["http.ServeMux<br/>rutas nativas ('GET /ruta')"]

    subgraph Handlers["internal/handlers/&lt;dominio&gt;"]
        REG["Register(mux, ...)"]
        H["Handler<br/>traduce errores centinela → HTTP"]
    end

    subgraph Dominio["internal/&lt;dominio&gt;"]
        SVC["Servicio de dominio (interfaz)"]
        REPOI["Repositorio (interfaz)<br/>implementación PostgreSQL"]
    end

    TEST["test/ (caja negra, TDD)<br/>mocks/ (generados con mockgen)"]

    MAIN --> MUX --> REG --> H --> SVC --> REPOI
    TEST -.->|"prueba comportamiento"| H
    TEST -.->|"prueba comportamiento"| SVC

    classDef go fill:#cffafe,stroke:#0891b2,color:#083344;
    class MAIN,MUX,REG,H,SVC,REPOI,TEST go;
```

> El gateway no tiene dominio: `internal/proxy` y `internal/middleware` y, con la *(historia 011)*,
> `internal/config`, `internal/ratelimit` e `internal/server` (la cadena de middlewares).
