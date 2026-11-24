# Deploy Report: HU-05 Verify Shipment Integrity

## Deployment Details

**Run URL:** https://github.com/JairoHGomezCastillo/hubx/actions/runs/45493671765

**Commit:** 5ffb09d1ca827dca7fdab4cb9f4373f91a7357b8 (branch: 005-verify-integrity)

**Environment:** staging

**Services and State:**
- hubx-api: live (deployed in 20s)
- hubx-gw: live (deployed in 20s)
- hubx-mfe: live (deployed in 20s)

**Migrations:** Applied successfully
```
{"time":"2026-11-24T21:42:07.77537058Z","level":"INFO","msg":"migrations up to date","applied":null,"total":1}
```

**Smoke Test Result:** 57 scenarios (57 passed)

## Curl Health Checks

**Command 1:**
```
curl -s https://hubx-gw-stg.onrender.com/health
```
**Result:** pass
**Excerpt:**
```
{"service":"hubx-gw","status":"ok"}
```

**Command 2:**
```
curl -s https://hubx-gw-stg.onrender.com/api/health
```
**Result:** pass
**Excerpt:**
```
{"status":"ok","service":"hubx-api"}
```

**Command 3:**
```
curl -s -w "\nStatus: %{http_code}\n" https://hubx-mfe-stg.onrender.com/
```
**Result:** pass
**Excerpt:**
```
Status: 200
```

## Summary

Deployment of HU-05 to staging completed successfully. All three services deployed and are live. Migrations were applied (baseline schema 0001, no new migrations for this story). All 57 smoke test scenarios passed, including browser-based E2E tests in Chromium. Public staging URLs confirm all three services are healthy and responding.

No failures or findings. The story branch is ready for PR review and merge to the release branch.
