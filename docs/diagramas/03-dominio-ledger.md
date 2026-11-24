# 03 · Dominio y ledger encadenado

El estado canónico de cada envío es un **ledger de avances** encadenados por hash SHA-256 y firmados
por el responsable que los emite (principio I). Es inmutable: nada se actualiza ni se borra; las
correcciones serían avances nuevos. Todo acceso al ledger pasa por la interfaz `LedgerRepository`.

## Modelo de dominio

```mermaid
classDiagram
    class Usuario {
        +id
        +email
        +rol: operator|partner|customer
        +passwordHash (bcrypt)
    }
    class Envio {
        +codigo (único)
        +origen
        +destino
        +vuelo
        +peso: 0.1..5000 kg
        +etapaActual
        +clienteId
        +socioAsignadoId
    }
    class Avance {
        +secuencia
        +etapa
        +responsableId
        +fecha
        +hashPrevio (SHA-256)
        +huella (SHA-256)
    }
    class Incidencia {
        +descripcion (<=500)
        +autorId
        +fecha
    }
    class LedgerRepository {
        <<interface>>
        +append(avance) Avance
        +cadenaDe(codigoEnvio) Avance[]
        +verificarIntegridad(codigoEnvio) Resultado
    }

    Usuario "1" --> "*" Envio : es cliente de
    Envio "1" --> "*" Avance : tiene cadena
    Envio "1" --> "*" Incidencia : registra
    Avance "0..1" --> "1" Avance : hashPrevio encadena
    LedgerRepository ..> Avance : administra
    Usuario "1" --> "*" Avance : responsable
```

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

Cada avance guarda la huella del anterior. Verificar recalcula las huellas y señala el primer avance
alterado.

```mermaid
flowchart LR
    A0["Avance 1 · En vuelo<br/>hashPrevio: ∅<br/>huella: H1"]
    A1["Avance 2 · En aeropuerto<br/>hashPrevio: H1<br/>huella: H2"]
    A2["Avance 3 · Asignado a socio<br/>hashPrevio: H2<br/>huella: H3"]

    A0 -->|"H1"| A1 -->|"H2"| A2

    classDef ev fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class A0,A1,A2 ev;
```

```mermaid
flowchart TB
    START["Solicitud de verificación<br/>(participante autorizado)"] --> EMPTY{"¿Tiene avances?"}
    EMPTY -->|No| OK0["Cadena íntegra ·<br/>no hay avances que revisar"]
    EMPTY -->|Sí| LOOP["Recalcular huella de cada avance<br/>y comparar con la almacenada"]
    LOOP --> MATCH{"¿Todas coinciden<br/>y el encadenamiento es válido?"}
    MATCH -->|Sí| OKN["Cadena íntegra ·<br/>N avances revisados"]
    MATCH -->|No| BAD["Cadena NO íntegra ·<br/>primera alteración en el avance K"]

    classDef ok fill:#dcfce7,stroke:#16a34a,color:#052e16;
    classDef bad fill:#fee2e2,stroke:#dc2626,color:#450a0a;
    class OK0,OKN ok;
    class BAD bad;
```

## Privacidad en consultas (HU-04 / HU-05)

Quien no puede ver un envío recibe la misma respuesta que si no existiera, para no filtrar su
existencia.

```mermaid
flowchart TB
    Q["Consulta por código de envío"] --> EXISTS{"¿Existe el envío?"}
    EXISTS -->|No| NF["'No existe un envío con ese código'"]
    EXISTS -->|Sí| AUTH{"¿El solicitante puede verlo?<br/>cliente: propio · operador: todos ·<br/>socio: asignados a él"}
    AUTH -->|Sí| SHOW["Muestra la trazabilidad"]
    AUTH -->|No| NF2["'No existe un envío con ese código'<br/>(misma respuesta que no existir)"]

    classDef neutral fill:#f1f5f9,stroke:#64748b,color:#1e293b;
    class NF,NF2,SHOW neutral;
```
