# 04 · Flujos de la aplicación

Secuencias de las historias implementadas. Todas pasan por el gateway (`/api`), que valida el JWT; la API
exige el secreto del gateway, valida el JWT otra vez (principio II) y aplica las reglas de negocio. HU-06 y
HU-08 no están implementadas.

## HU-01 · Iniciar sesión

El correo ignora mayúsculas y espacios. Si fallan las credenciales, el mensaje no revela qué dato
estuvo mal. La sesión dura una hora.

```mermaid
sequenceDiagram
    actor U as Usuario
    participant MFE as hubx-mfe
    participant GW as hubx-gw
    participant API as hubx-api

    U->>MFE: email + contraseña
    MFE->>GW: POST /api/auth/sign-in
    GW->>GW: límite por correo, 10/min (historia 011)
    alt límite superado
        GW-->>MFE: 429 + Retry-After
    else dentro del límite
        GW->>API: POST /auth/sign-in (reverse proxy)
        API->>API: normaliza email · verifica bcrypt
        alt credenciales válidas
            API-->>MFE: 200 token (JWT, 1 h) + rol
            MFE-->>U: acceso con funciones de su rol
        else inválidas o email no registrado
            API-->>MFE: 401 INVALID_CREDENTIALS (sin decir qué dato)
            MFE-->>U: mensaje genérico, sin acceso
        else faltan datos
            API-->>MFE: 400 MISSING_REQUIRED_FIELDS
        end
    end
```

## HU-02 · Crear un envío

Solo operador. Peso entre 0.1 y 5000 kg, origen distinto de destino, cliente registrado con rol
cliente. Crear no registra avance todavía, así que no toca el ledger.

```mermaid
sequenceDiagram
    actor OP as Operador
    participant GW as hubx-gw
    participant API as hubx-api
    participant DB as PostgreSQL

    OP->>GW: POST /api/shipments (origen, destino, vuelo, peso, cliente)
    GW->>GW: valida JWT
    GW->>API: reverse proxy
    API->>API: valida JWT y rol operador
    API->>API: valida datos (peso, origen≠destino, cliente con rol cliente)
    alt datos válidos
        API->>DB: inserta el envío (etapa Created, código HBX único)
        DB-->>API: envío creado
        API-->>OP: 201 envío con código único, etapa Created
    else datos inválidos
        API-->>OP: envío no creado + motivo del negocio
    else rol sin permiso
        API-->>OP: 403 sin permiso para crear envíos
    end
```

## HU-03 · Registrar un avance

El responsable de la etapa registra su avance; se encadena al anterior por su huella. La secuencia
es estricta y el responsable debe ser el correcto. Al asignar, el socio se elige con
`GET /api/ground-partners`.

```mermaid
sequenceDiagram
    actor R as Responsable (operador o socio)
    participant GW as hubx-gw
    participant API as hubx-api
    participant LR as LedgerRepository

    R->>GW: POST /api/shipments/{code}/updates (stage, assignedPartnerId)
    GW->>GW: valida JWT
    GW->>API: reverse proxy
    API->>API: valida JWT, visibilidad, secuencia y responsable
    alt avance válido
        API->>LR: Last(shipmentID)
        LR-->>API: último avance
        API->>LR: Append(avance con responsable, fecha y huella previa)
        LR->>LR: huella SHA-256 + proyección de etapa, en una transacción
        API-->>R: 201 el envío avanza a la nueva etapa
    else fuera de secuencia
        API-->>R: 409 OUT_OF_SEQUENCE, etapa sin cambios
    else no es el responsable, no existe o no puede verlo
        API-->>R: 403 NOT_RESPONSIBLE, etapa sin cambios
    end
```

## HU-04 / HU-05 · Ver trazabilidad y verificar integridad

```mermaid
sequenceDiagram
    actor P as Participante
    participant GW as hubx-gw
    participant API as hubx-api
    participant LR as LedgerRepository

    P->>GW: GET /api/shipments/{code}/traceability o /integrity
    GW->>GW: valida JWT
    GW->>API: reverse proxy
    API->>API: valida JWT + visibilidad (propio/todos/asignados)
    alt autorizado y existe
        API->>LR: List(shipmentID)
        LR-->>API: avances del más antiguo al más reciente
        API->>API: integridad, recalcula cada huella
        API-->>P: 200 trazabilidad o veredicto (íntegra / primera alteración en avance K)
    else no existe o sin acceso
        API-->>P: 404 no existe un envío con ese código
    end
```

## HU-07 / HU-09 / HU-10 · Asignados, conteos y evidencia

```mermaid
sequenceDiagram
    actor S as Socio de tierra
    actor OP as Operador
    participant GW as hubx-gw
    participant API as hubx-api

    S->>GW: GET /api/assigned-shipments?offset=N
    GW->>API: reverse proxy
    API-->>S: 200 pendientes propios, de a 20, el asignado hace más tiempo primero

    OP->>GW: GET /api/shipments/stage-counts
    GW->>API: reverse proxy
    API-->>OP: 200 total de las seis etapas en orden

    OP->>GW: GET /api/shipments/{code}/evidence
    GW->>API: reverse proxy
    API-->>OP: 200 PDF con datos, avances, huellas e instrucciones

    Note over S,API: otro rol recibe 403 FORBIDDEN sin datos
```

## Caminos de rechazo del gateway

Si la API no responde, el gateway traduce el fallo de transporte en un `502` explícito en vez de
dejar la conexión colgando. Sin sesión válida la petición no llega a la API. El `429` es de la
*(historia 011)*.

```mermaid
sequenceDiagram
    participant MFE as hubx-mfe
    participant GW as hubx-gw
    participant API as hubx-api

    MFE->>GW: GET /api/... sin JWT válido
    GW-->>MFE: 401
    MFE->>GW: GET /api/... por encima de 120/min del usuario
    GW-->>MFE: 429 RATE_LIMITED + Retry-After
    MFE->>GW: GET /api/...
    GW->>API: reverse proxy
    API--xGW: sin respuesta (caído)
    GW-->>MFE: 502 Bad Gateway (ErrorHandler)
```
