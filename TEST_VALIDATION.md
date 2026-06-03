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
- duplicate-publish warning and confirmation behavior
- Bluesky `Live Now` set, clear, duration, opt-out, and recovery behavior
- credential persistence through secret references rather than raw values
- Discord, Bluesky, and Mastodon publisher payloads
- remote image and preview URL validation
- settings defaults that prefill fresh Home sessions without overriding user edits

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
