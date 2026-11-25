# Quickstart: Download the traceability evidence (HU-10)

Validation guide only; implementation details live in `tasks.md`. See
[the contract](../../contracts/hu10-download-evidence.md) for the endpoint and [the data model](./data-model.md) for
the document content.

## Prerequisites

- Docker with Compose, Go 1.27.4 and Node 24.
- `make test-integration` builds the full stack (PostgreSQL + migrations + API + gateway + MFE) in an isolated
  project, runs Cucumber (API and `@e2e`) and tears it down. No separate `make up` is required.

## Commands

```text
bash scripts/release-version.sh release/v0.10.0-demo   # version=v0.10.0
go -C hubx-api generate ./...                          # regenerate gomock mocks
make test                                              # Go unit tests + MFE unit tests
make test-integration                                  # API scenarios + @e2e browser happy path
make security                                          # gosec + govulncheck
make agents-zones                                      # commit zones
```

## Expected outcomes

- Unit tests: new `evidence` domain and handler packages pass in `test/` with generated mocks in `mocks/`.
- `make test-integration`: all HU-10 scenarios pass, including the `@e2e` download from the Dashboard-driven page in
  Chromium; older stories stay green.
- Security: `gosec` and `govulncheck` report no findings; `go.mod` is unchanged (no new dependency).
- Manual check (optional, against `make up`): sign in as `operator@hubx.co`, open a shipment with three updates, click
  `Download evidence`, open the saved PDF and follow `How to verify the fingerprints` with a plain text editor; the
  recomputed SHA-256 values match the fingerprints printed in the document.

## Demo mode

The human requested no staging deploy validation for this story. No `deploy.md` and no devops delegation: staging is
deployed and smoke-tested automatically when the PR is merged. `pr.md` records this decision.
