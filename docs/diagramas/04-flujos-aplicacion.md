# 04 · Flujos de la aplicación

Secuencias de las historias del backlog. Todas pasan por el gateway (`/api`); la API valida el JWT
por su cuenta (principio II) y aplica las reglas de negocio.

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
    MFE->>GW: POST /api/login
    GW->>API: POST /login (reverse proxy)
    API->>API: normaliza email · verifica bcrypt
    alt credenciales válidas
        API-->>MFE: token de sesión (JWT, 1 h) + rol
        MFE-->>U: acceso con funciones de su rol
    else inválidas o email no registrado
        API-->>MFE: "credenciales no válidas" (sin decir qué dato)
        MFE-->>U: mensaje genérico, sin acceso
    end
```

## HU-02 · Crear un envío

Solo operador. Peso entre 0.1 y 5000 kg, origen distinto de destino, cliente registrado con rol
cliente. Crear no registra avance todavía.

```mermaid
sequenceDiagram
    actor OP as Operador
    participant GW as hubx-gw
    participant API as hubx-api
    participant LR as LedgerRepository

    OP->>GW: POST /api/shipments (origen, destino, vuelo, peso, cliente)
    GW->>API: reverse proxy
    API->>API: valida JWT + rol operador
    API->>API: valida datos (peso, origen≠destino, cliente con rol cliente)
    alt datos válidos
        API->>LR: crea envío (etapa "Creado", código único)
        LR-->>API: envío creado
        API-->>OP: envío con código único · etapa "Creado"
    else datos inválidos
        API-->>OP: envío no creado + motivo del negocio
    else rol sin permiso
        API-->>OP: "no tiene permiso para crear envíos"
    end
```

## HU-03 · Registrar un avance

El responsable de la etapa registra su avance; se encadena al anterior por su huella. La secuencia
es estricta y el responsable debe ser el correcto.

```mermaid
sequenceDiagram
    actor R as Responsable (operador o socio)
    participant GW as hubx-gw
    participant API as hubx-api
    participant LR as LedgerRepository

    R->>GW: POST /api/shipments/{codigo}/updates (nueva etapa)
    GW->>API: reverse proxy
    API->>API: valida JWT + rol
    API->>LR: obtiene etapa actual y último avance
    alt avance válido (secuencia y responsable correctos)
        API->>API: calcula huella encadenada (hashPrevio = huella anterior)
        API->>LR: append(avance con responsable y fecha)
        API-->>R: envío avanza a la nueva etapa
    else fuera de secuencia
        API-->>R: "ese avance no sigue la secuencia" · etapa sin cambios
    else no es el responsable
        API-->>R: "no es responsable de ese avance" · etapa sin cambios
    end
```

## HU-04 / HU-05 · Ver trazabilidad y verificar integridad

```mermaid
sequenceDiagram
    actor P as Participante
    participant GW as hubx-gw
    participant API as hubx-api
    participant LR as LedgerRepository

    P->>GW: GET /api/shipments/{codigo}/trace  (o /verify)
    GW->>API: reverse proxy
    API->>API: valida JWT + visibilidad (propio/todos/asignados)
    alt autorizado y existe
        API->>LR: cadenaDe(codigo) / verificarIntegridad(codigo)
        LR-->>API: avances ordenados / resultado de integridad
        API-->>P: trazabilidad o veredicto (íntegra / primera alteración en avance K)
    else no existe o sin acceso
        API-->>P: "no existe un envío con ese código"
    end
```

## Camino del error de upstream (gateway)

Si la API no responde, el gateway traduce el fallo de transporte en un `502` explícito en vez de
dejar la conexión colgando.

```mermaid
sequenceDiagram
    participant MFE as hubx-mfe
    participant GW as hubx-gw
    participant API as hubx-api

    MFE->>GW: GET /api/...
    GW->>API: reverse proxy
    API--xGW: sin respuesta (caído / timeout)
    GW-->>MFE: 502 Bad Gateway (ErrorHandler)
```
