# 03 · Dominio y ledger encadenado

El estado canónico de cada envío es un **ledger de avances** encadenados por huella SHA-256, cada uno con
su responsable identificado (principio I). Los avances **no están firmados criptográficamente**: la
identidad del responsable sale de la sesión autenticada, y las firmas por participante son una extensión
futura. El ledger es inmutable: nada se actualiza ni se borra (en esta versión tampoco hay correcciones).
Todo acceso al ledger pasa por la interfaz `LedgerRepository`.

## Modelo de dominio

Tablas `users`, `shipments` y `updates` (migración `0001`). `stage` y `assigned_partner_id` del envío son
una proyección del ledger para lecturas rápidas. Las incidencias (HU-08) no existen todavía.

```mermaid
classDiagram
    class Usuario {
        +id
        +email (normalizado, único)
        +nombre
        +rol: operator, ground partner, customer
        +passwordHash (bcrypt)
    }
    class Envio {
        +codigo (único)
        +origen
        +destino
        +vuelo
        +peso: 0.1 a 5000 kg
        +etapaActual (proyección)
        +clienteId
        +socioAsignadoId (proyección)
        +creadoPor
    }
    class Avance {
        +secuencia
        +etapa
        +responsableId
        +socioAsignadoId
        +fecha
        +huellaPrevia (SHA-256, nula en el primero)
        +huella (SHA-256)
    }
    class LedgerRepository {
        <<interface>>
        +Last(shipmentID)
        +List(shipmentID)
        +Append(avance)
        +PendingAssignments(partnerID, limit, offset)
    }

    Usuario "1" --> "*" Envio : es cliente de
    Envio "1" --> "*" Avance : tiene cadena
    Avance "0..1" --> "0..1" Avance : huellaPrevia encadena
    LedgerRepository ..> Avance : administra
    Usuario "1" --> "*" Avance : responsable
```

`Append` inserta el avance y actualiza la proyección del envío en una sola transacción. Un trigger de
PostgreSQL rechaza `UPDATE` y `DELETE` sobre `updates`.

## Etapas del envío (máquina de estados)

Las etapas avanzan siempre en orden, sin retroceso ni repetición (regla transversal 2). "Asignado a
socio" solo se alcanza asignando un socio de tierra desde "En aeropuerto destino".

```mermaid
stateDiagram-v2
    [*] --> Creado : operador crea el envío
    Creado --> EnVuelo : operador
    EnVuelo --> EnAeropuertoDestino : operador
    EnAeropuertoDestino --> AsignadoASocio : operador asigna socio de tierra
    AsignadoASocio --> EnReparto : socio asignado
    EnReparto --> Entregado : socio asignado
    Entregado --> [*]

    note right of Creado
        Crear el envío aún no registra un avance;
        el primer avance es "En vuelo".
    end note
    note right of AsignadoASocio
        Desde aquí el socio asignado es
        responsable de la entrega. Sin reasignación
        en esta versión.
    end note
```

## Encadenamiento e integridad (HU-05)

Cada avance guarda la huella del anterior. La huella es el SHA-256 de siete líneas `clave=valor`
(`contracts/hu03-record-update.md`): envío, secuencia, etapa, responsable, socio asignado, fecha y huella
previa. Cubre identificadores, no nombres. Verificar recalcula las huellas y señala el primer avance
alterado.

```mermaid
flowchart LR
    A0["Avance 1 · En vuelo<br/>huella previa: -<br/>huella: H1"]
    A1["Avance 2 · En aeropuerto<br/>huella previa: H1<br/>huella: H2"]
    A2["Avance 3 · Asignado a socio<br/>huella previa: H2<br/>huella: H3"]

    A0 -->|"H1"| A1 -->|"H2"| A2

    classDef ev fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class A0,A1,A2 ev;
```

```mermaid
flowchart TB
    START["Solicitud de verificación<br/>(participante autorizado)"] --> EMPTY{"¿Tiene avances?"}
    EMPTY -->|No| OK0["Cadena íntegra ·<br/>no hay avances que revisar"]
    EMPTY -->|Sí| LOOP["Recalcular huella de cada avance<br/>y comparar con la almacenada"]
    LOOP --> MATCH{"¿Secuencia, huella previa<br/>y huella coinciden en todos?"}
    MATCH -->|Sí| OKN["Cadena íntegra ·<br/>N avances revisados"]
    MATCH -->|No| BAD["Cadena NO íntegra ·<br/>primera alteración en el avance K"]

    classDef ok fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef bad fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    class OK0,OKN ok;
    class BAD bad;
```

## Privacidad en consultas (HU-04 / HU-05)

Quien no puede ver un envío recibe la misma respuesta que si no existiera, para no filtrar su
existencia. Una sola regla (`shipment.CanSee`) sirve a HU-03, HU-04, HU-05 y HU-10.

```mermaid
flowchart TB
    Q["Consulta por código de envío"] --> EXISTS{"¿Existe el envío?"}
    EXISTS -->|No| NF["404: no existe un envío con ese código"]
    EXISTS -->|Sí| AUTH{"¿El solicitante puede verlo?<br/>cliente: propio · operador: todos ·<br/>socio: asignados a él"}
    AUTH -->|Sí| SHOW["Muestra la trazabilidad o el veredicto"]
    AUTH -->|No| NF2["404: no existe un envío con ese código<br/>(misma respuesta que no existir)"]

    classDef neutral fill:#f1f5f9,stroke:#64748b,color:#1e293b;
    class NF,NF2,SHOW neutral;
```
