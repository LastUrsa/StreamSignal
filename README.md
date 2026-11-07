# StreamSignal

StreamSignal is a local-first desktop app for streamers and VTubers who want one repeatable **Go Live** and **End Stream** workflow across Discord, Bluesky, and Mastodon.

It keeps app data on the local machine, stores runtime secrets in the OS credential store, and posts only to the destinations selected for the current session.

## Features

- Publish Go Live announcements to Discord, Bluesky, and Mastodon.
- Preview destination-specific messages before sending.
- Post optional End Stream messages per destination.
- Set, clear, and recover Bluesky `Live Now` status.
- Reuse Announcement Profiles for repeated stream types.
- Target individual destinations or Destination Groups.
- Control the active profile from LivePanel through StreamSignal's localhost-only SIP API.

## Quickstart

1. Open **Destinations** and add each platform you want to publish to.
2. Use Discord webhooks, Bluesky app passwords, and Mastodon access tokens from accounts you control.
3. Create optional **Profiles** for repeated stream types.
4. On **Home**, enter session details, choose destinations or a group, and add optional media.
5. Generate previews and resolve any warnings.
6. Click **Go Live**, then **End Stream** when the stream is over.

## App Areas

- **Home**: session details, destination selection, previews, Go Live, End Stream, and recovery awareness.
- **Destinations**: platform setup, testing, templates, and Destination Groups.
- **Profiles**: reusable announcement presets.
- **Settings**: defaults, Bluesky Live Now recovery, and legal notices.
- **Logs**: debug-only workflow validation in non-production builds.

## Stack

- Go backend
- Wails desktop shell
- React + TypeScript frontend
- SQLite for local data
- Windows Credential Manager and macOS Keychain for runtime secrets

## Release support

Version 1.0 targets Windows x64 and macOS universal. Windows requires WebView2; macOS packaging is built on a macOS runner. Release builds are currently unsigned/not notarized; installation may require OS approval. Installer and DMG downloads are published on [GitHub Releases](https://github.com/LastUrsa/StreamSignal/releases).

Provider credentials and destination permissions are required for posting. A configured credential reference does not prove remote authentication. Unknown publication outcomes should be inspected by operation ID before retrying.

## Development

Use Go 1.26.10+, Node.js 22+, and Wails CLI 2.13.0 (matching the module).

Install frontend dependencies:

```bash
npm install --prefix frontend
```

Run the app in Wails development mode:

```bash
wails dev
```

Run a sandboxed local instance with separate app data and credential storage:

```bash
STREAMSIGNAL_ENV=dev wails dev
```

Build the Windows app from WSL:

```bash
./scripts/build-prod-windows.sh
```

Build and launch a sandboxed local review build:

```bash
./scripts/build-prod-windows.sh --launch --env dev
```

Run the Windows service-mode smoke test:

```bash
./scripts/build-prod-windows.sh --service-smoke --env dev
./scripts/test-sip-newman.sh
```

Production uses the normal `StreamSignal` app data folder and `streamsignal` credential namespace. Any non-production `STREAMSIGNAL_ENV` value uses its own namespaced store, such as `StreamSignal-dev` and `streamsignal-dev`.

StreamSignal supports three launch forms:

```bash
StreamSignal.exe
StreamSignal.exe --service
StreamSignal.exe --show
```

`--service` starts backend services and the SIP server without showing the main window. `--show` asks an existing instance to restore and focus its window, or launches normally when no instance is running.

## Quality Gates

Run these before release:

```bash
go test ./...
go vet ./...
npm test --prefix frontend
npm run build --prefix frontend
npm audit --prefix frontend
govulncheck ./...
./scripts/build-prod-windows.sh --service-smoke --env dev
```

Provider-backed workflow E2E tests are on demand because they create and delete real test-account posts. They also run as part of the tag-driven release build before artifacts are produced:

```bash
STREAMSIGNAL_E2E=1 go test -tags=e2e ./internal/e2e -v -count=1
```

## Docs

- [SIP API Reference](./docs/sip-api-reference.md)
- [Security Policy](./SECURITY.md)
- [Release Notes](./RELEASE_NOTES.md)
- [Release Readiness Checklist](./RELEASE_READINESS_CHECKLIST.md)
- [Test Validation](./TEST_VALIDATION.md)
- [Security Audit](./SECURITY_AUDIT.md)
- [Integration Limitations](./INTEGRATION_LIMITATIONS.md)
- [Brand Usage](./BRAND_USAGE.md)
- [macOS Release Test Plan](./MACOS_RELEASE_TEST_PLAN.md)


## SIP Troubleshooting and Operation Metadata

Adds capability-advertised SIP operations and diagnostics, correlated safe operation logs, per-destination publication/follow-up results, credential readiness, opt-in asynchronous publishing, and durable idempotency/restart recovery. Existing synchronous clients remain supported. Publishing operations use a five-minute server write deadline; asynchronous work has a separate 4.5-minute execution budget. The operation journal is `sip-operations.json` beside the selected app database (including the isolated dev store), retains up to 256 operations/14 days, and never contains request bodies, rendered content, credentials or receipt URLs. A failed journal write prevents new publishing; interrupted operations are unknown and never automatically replayed. Idempotency compares the route and exact request body bytes. A key is only protected while its operation remains in retained history.

See [SIP Observability v1](./SIP_OBSERVABILITY_V1.md) for the common headers, operation/error schema, retry rules, capabilities, and endpoint examples. Legacy `error` fields remain display-only; clients should log the fixed `message`/`errorCode` and operational IDs instead.

Build and validate with Go 1.26.10 or newer. Frontend build/test dependencies and Wails have been updated for security advisories.

Credential readiness checks only local configuration and the last recorded attempt. A rejected state is historical evidence, not a fresh authentication test; inspect `lastAttemptAt` after changing credentials. No readiness request publishes a post.

See [1.0 release validation](RELEASE_VALIDATION.md) for the verified checks, packaging gates, and manual-validation boundaries.
