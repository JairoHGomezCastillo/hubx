## Story
HU-xx <title>. Spec: `specs/NNN-name/`

## Evidence
- make test-integration: <result>
- go test / npm test: <result>
- gosec / govulncheck: <result>
- speckit.analyze: <deviations or "none">
- qa-report.md: `specs/NNN-name/qa-report.md`

## Checklist
- [ ] Every backlog scenario has a test
- [ ] No secrets, no cross-service imports, MFE only via /api
- [ ] Only my own paths changed (git diff --stat attached)
- [ ] Backlog status updated for this story only

<details><summary>git diff --stat</summary>

```
<output>
```
</details>
