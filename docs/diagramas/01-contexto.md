# 01 · Contexto del sistema

HUBX es una plataforma de trazabilidad de carga aérea donde los participantes de un envío comparten
un historial único, confiable y verificable. Hay tres roles de negocio y un perímetro único de
entrada (el gateway).

## Actores y límites

```mermaid
flowchart TB
    subgraph Actores["Participantes del envío"]
        OP["Operador<br/>(operator)"]
        GP["Socio de tierra<br/>(ground partner)"]
        CL["Cliente<br/>(customer)"]
    end

    subgraph HUBX["Plataforma HUBX"]
        MFE["hubx-mfe<br/>Frontend React"]
        GW["hubx-gw<br/>Gateway · perímetro único"]
        API["hubx-api<br/>Dominio y ledger"]
        DB[("PostgreSQL")]
    end

    DOC["Evidencia en PDF (HU-10)"]
    TERCERO["Tercero sin acceso a HUBX"]

    OP --> MFE
    GP --> MFE
    CL --> MFE
    MFE -->|"solo /api"| GW
    GW -->|"reverse proxy"| API
    API --> DB
    OP -.->|"descarga y comparte"| DOC
    TERCERO -.->|"recalcula las huellas<br/>solo con el documento"| DOC

    classDef actor fill:#e0e7ff,stroke:#4f46e5,color:#1e1b4b;
    classDef svc fill:#dcfce7,stroke:#16a34a,color:#052e16;
    class OP,GP,CL,TERCERO actor;
    class GW,API,MFE,DB,DOC svc;
```

El tercero nunca entra a HUBX: verifica el documento por su cuenta. La evidencia prueba que el
historial es consistente consigo mismo; no prueba por sí sola que HUBX no lo reconstruyó entero, y
las huellas cubren los identificadores de las partes, no sus nombres.

## Qué puede hacer cada rol

Resumen de la matriz de permisos del backlog (regla transversal 6). Pendientes de implementar: listar
por etapa (HU-06) y reportar incidencias (HU-08).

```mermaid
flowchart LR
    subgraph OP["Operador"]
        O1["Crear envío"]
        O2["Registrar En vuelo /<br/>En aeropuerto destino"]
        O3["Asignar socio de tierra"]
        O4["Ver y verificar cualquier envío"]
        O5["Ver conteos por etapa,<br/>descargar evidencia"]
        O6["Listar por etapa<br/>(HU-06, pendiente)"]
    end
    subgraph GP["Socio de tierra"]
        G1["Registrar En reparto /<br/>Entregado (solo asignados)"]
        G2["Ver sus envíos asignados<br/>y verificar su integridad"]
        G3["Reportar incidencias<br/>(HU-08, pendiente)"]
    end
    subgraph CL["Cliente"]
        C1["Ver solo sus propios envíos"]
        C2["Verificar integridad<br/>de sus envíos"]
    end

    classDef op fill:#dbeafe,stroke:#2563eb,color:#0c2a66;
    classDef gp fill:#fef3c7,stroke:#d97706,color:#5a3a06;
    classDef cl fill:#ede9fe,stroke:#7c3aed,color:#3b0764;
    classDef pend fill:#f1f5f9,stroke:#94a3b8,color:#334155,stroke-dasharray:4 3;
    class O1,O2,O3,O4,O5 op;
    class G1,G2 gp;
    class C1,C2 cl;
    class O6,G3 pend;
```

## Reglas transversales clave

- **Sesión:** dura una hora; luego se pide iniciar sesión otra vez antes de cualquier operación.
- **Secuencia:** las etapas avanzan siempre en orden, sin retroceso ni repetición.
- **Inmutabilidad:** un avance no se modifica ni se borra.
- **Privacidad:** quien no puede ver un envío recibe la misma respuesta que si no existiera.
- **Listas:** se muestran de a 20, lo más reciente primero (salvo que una historia diga otra cosa; en
  HU-07 va primero el envío asignado hace más tiempo).
