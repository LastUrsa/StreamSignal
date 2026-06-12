# Test Validation

This maintainer-facing document explains how StreamSignal treats automated tests and release validation. It is not an end-user guide.

## Baseline Checks

Every pull request and release candidate should pass:

```bash
go test ./...
go vet ./...
npm test --prefix frontend
npm run build --prefix frontend
npm audit --prefix frontend
govulncheck ./...
```

The frontend test suite includes mocked UI workflow coverage. It does not publish real posts.

Provider-backed workflow E2E runs separately because it creates real test-account posts and deletes them afterward:

```bash
STREAMSIGNAL_E2E=1 go test -tags=e2e ./internal/e2e -v -count=1
```

In GitHub Actions, `Provider Workflow E2E` runs on a schedule and as a release-build prerequisite when the test-account secrets are configured.

SIP contract smoke tests run through Newman for release validation, not on every pull request:

```bash
./scripts/test-sip-newman.sh
```

In GitHub Actions, `SIP Newman Smoke` is available manually and runs as a release-build prerequisite. It launches StreamSignal in service mode with an isolated `STREAMSIGNAL_ENV`, discovers the localhost SIP port, and runs the non-publishing Postman collection in `docs/streamsignal-sip-newman-smoke.postman_collection.json`.

If dependency manifests change, also confirm:

```bash
go mod tidy
npm install --prefix frontend
```

The tracked lockfiles should only change when the dependency update is intentional.

## Testing Philosophy

StreamSignal handles credentials and outbound posting, so tests should protect workflow behavior rather than chase a single repo-wide coverage number.

Good coverage for this project means:

- validation and normalization paths are covered
- publishing orchestration is covered for success, warning, skipped, and failure cases
- destination targeting is covered so posts go only where the user selected
- persistence behavior is covered when workflow state changes
- outbound adapters have request, response, and error-path tests
- frontend tests cover user-visible decisions and safety messaging

## Strongly Protected Areas

Keep targeted coverage around:

- Preview remaining network-safe except for explicit link-preview metadata paths
- Go Live and End Stream destination selection
- blank Home fields referenced by templates surfacing as preview warnings without blocking publish when rendered content remains valid
- Announcement Profiles applying Home fields, media defaults, and destination groups
- duplicate-publish warning and confirmation behavior
- Bluesky `Live Now` set, clear, duration, opt-out, and recovery behavior
- credential persistence through secret references rather than raw values
- non-production runtime data and credential namespace isolation
- service-mode launch parsing, SIP mode reporting, single-instance behavior, and environment-scoped instance locks
- SIP current-profile, active-profile announcement/end-stream execution, duplicate-guard confirmation, sanitized status metadata, last-run operation status, method/content-type rejection, local-host enforcement, and absent generic command/action endpoints
- Discord, Bluesky, and Mastodon publisher payloads
- remote image and preview URL validation
- settings and profile defaults that prefill Home sessions without overriding user edits unexpectedly

## Service Mode Validation

Service-mode changes should keep automated coverage around:

- `StreamSignal.exe` resolving to standalone mode
- `StreamSignal.exe --service` resolving to service mode and hidden startup
- `StreamSignal.exe --show` resolving to standalone startup when no instance owns the lock
- second-instance launch handling activating the existing window only when appropriate
- `/api/v1/app` reporting the configured runtime mode
- `/api/v1/profile/current`, `/api/v1/announce`, `/api/v1/announce/confirm`, `/api/v1/announce/status`, `/api/v1/end-stream`, and `/api/v1/end-stream/status` preserving the active-profile LivePanel workflow
- single-instance lock IDs staying separated by `STREAMSIGNAL_ENV`

Before release, validate the OS/window-manager behavior manually on Windows:

```bash
./scripts/build-prod-windows.sh --service-smoke --env dev
./scripts/test-sip-newman.sh
```

The service-mode smoke test builds the Windows executable and checks:

- `StreamSignal.exe --service` creates one process, starts SIP, reports mode `service`, and has no main window handle
- `StreamSignal.exe --show` reuses the service process and creates/restores a main window handle
- `StreamSignal.exe` reports mode `standalone`
- `StreamSignal.exe --service` while standalone mode is running does not create a duplicate process
- the test runs under an isolated `STREAMSIGNAL_ENV` so it does not use production app data or credentials

If the smoke script fails because the host has no interactive desktop session, repeat the same launch sequence manually on Windows before release.

## PR Review Guidance

When reviewing behavior changes, ask:

- What user-visible or workflow-risk behavior changed?
- Where is that behavior tested now?
- Does the test exercise the public behavior instead of an implementation detail?
- If coverage moved down, was equivalent protection added at a better layer?

Require targeted tests when a PR touches:

- `internal/app`: service and orchestration tests
- `internal/platforms/*`: adapter success and failure tests
- `internal/storage/*`: persistence round-trip tests
- `app.go` or Wails bindings: app-shell tests when behavior changes
- `frontend/src/App.tsx`: interaction tests for changed user paths

## Coverage Posture

Use coverage as a signal, not the only gate.

Recommended expectations:

- backend coverage should not regress materially in high-risk packages
- frontend statement coverage should stay high enough to exercise core workflows
- branch coverage matters most where UI decisions affect publishing or safety
- pure type-definition files do not need artificial tests

Avoid rewarding superficial tests that only assert static rendering or internal implementation details.
