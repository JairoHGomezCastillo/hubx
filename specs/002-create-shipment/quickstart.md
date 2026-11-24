# Quickstart: Create a shipment (HU-02)

Prerequisites: `make up` (PostgreSQL with migration 0001, API, gateway, MFE).

```bash
op() { curl -s -X POST http://localhost:8080/api/auth/sign-in -H 'Content-Type: application/json' \
  -d "{\"email\":\"$1\",\"password\":\"Hubx-Demo-2026\"}" | jq -r .token; }
create() { curl -s -w ' %{http_code}\n' -X POST http://localhost:8080/api/shipments \
  -H "Authorization: Bearer $1" -H 'Content-Type: application/json' -d "$2"; }
T=$(op operator@hubx.co)

create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}'   # 201 Created
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":0.1,"customerEmail":"andina@hubx.co"}'   # 201
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":5000,"customerEmail":"andina@hubx.co"}'  # 201
create "$T" '{"origin":"Bogotá","destination":" bogotá ","flight":"AV-204","weightKg":1,"customerEmail":"andina@hubx.co"}' # 400 SAME_ORIGIN_DESTINATION
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":0,"customerEmail":"andina@hubx.co"}'     # 400 WEIGHT_OUT_OF_RANGE
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":5001,"customerEmail":"andina@hubx.co"}'  # 400 WEIGHT_OUT_OF_RANGE
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":1,"customerEmail":"nobody@hubx.co"}'     # 422 CUSTOMER_NOT_FOUND
create "$T" '{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":1,"customerEmail":"rapido@hubx.co"}'     # 422 CUSTOMER_NOT_CUSTOMER_ROLE
create "$T" '{"origin":"Bogotá","destination":"Miami","weightKg":1,"customerEmail":"andina@hubx.co"}'                       # 400 MISSING_REQUIRED_FIELDS
create "$(op rapido@hubx.co)" '{}'                                                                                          # 403 FORBIDDEN
create "$(op ana@hubx.co)" '{}'                                                                                             # 403 FORBIDDEN
```

Browser: http://localhost:3000, sign in as operator@hubx.co, press `Create shipment`, fill the form, see
`Shipment created`, `Code: HBX-...` and `Stage: Created`.
