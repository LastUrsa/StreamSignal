# StreamSignal SIP Prototype

This document captures the StreamSignal SIP-v1.1 architecture and scope.

## Scope

The prototype implements these SIP-v1.1 endpoints:

```text
GET  /api/v1/app
GET  /api/v1/health
GET  /api/v1/capabilities
GET  /api/v1/status
GET  /api/v1/profiles
GET  /api/v1/profile/current
POST /api/v1/profile
POST /api/v1/announce
POST /api/v1/announce/confirm
GET  /api/v1/announce/status
POST /api/v1/end-stream
GET  /api/v1/end-stream/status
```

Generic action or command endpoints are intentionally not implemented.

## Hosting

StreamSignal starts a localhost-only HTTP SIP server during Wails startup in both standalone and service mode.

The server binds to `127.0.0.1` and uses the StreamSignal SIP port range:

```text
47020-47029
```

It tries `47020` first, then the next available port in the range. The server is stopped during Wails shutdown.

## Runtime Modes

StreamSignal can be launched as a normal desktop app or as a background service module:

```text
StreamSignal.exe
StreamSignal.exe --service
StreamSignal.exe --show
```

Normal launches open the main window and report SIP mode `standalone`. `--service` starts the same backend services without showing the main window and reports SIP mode `service`. `--show` restores and focuses an existing instance, or starts standalone mode when no instance is already running.

Only one StreamSignal process runs per runtime environment. Non-production `STREAMSIGNAL_ENV` values use separate app data, credential namespaces, and single-instance locks from production.

## Security Posture

The SIP prototype is intentionally local and conservative:

- the listener binds to `127.0.0.1` only
- non-local `Host` headers are rejected
- CORS headers are not emitted
- profile activation requires `Content-Type: application/json`
- request bodies are limited to 1 MiB
- responses include `X-Content-Type-Options: nosniff`
- responses include `Cache-Control: no-store`
- the HTTP server uses read, write, idle, and header timeouts

SIP-v1.1 does not include authentication. This implementation relies on same-machine locality and exposes only explicit StreamSignal business operations, not generic command execution.

## Architecture

SIP code lives in `internal/sip`.

That package owns:

- HTTP routing and JSON responses
- SIP response models
- SIP health, capabilities, status, and profile translation
- in-memory active SIP profile state
- SIP announcement and end-stream execution and last-run status

The SIP layer uses existing StreamSignal services. It does not replace or redesign announcement profiles, settings, recovery, publishing, or frontend state.

## Profile Activation

`POST /api/v1/profile` activates an existing `AnnouncementProfile` by name.

Activation:

- matches existing announcement profiles case-insensitively by name
- marks the selected profile as used
- stores the active SIP profile in backend/SIP memory
- makes the active profile visible through `GET /api/v1/status` and `GET /api/v1/profile/current`
- exposes safe, display-oriented drawer metadata through `GET /api/v1/status`, including the selected destination group, enabled destination counts, enabled platforms, destination-template summary text, image summary text, and profile defaults

Activation does not:

- publish announcements
- call Go Live
- call End Stream
- call Force Go Live
- edit destination configuration
- live-synchronize the React Home form
- expose raw destination config, credentials, webhook URLs, tokens, secret references, or raw image URLs/data through SIP status

The Home screen's compose state is currently React-local state. For this prototype, SIP profile activation updates backend/SIP state and usage metadata only.

## Announcement Execution

`POST /api/v1/announce` triggers the existing Go Live execution path using the active SIP profile's default announcement fields.

Execution:

- requires an active SIP profile
- converts profile defaults into a `domain.Announcement`
- calls the same execution service used by the standalone UI
- preserves existing validation, duplicate guard, destination resolution, publish, and Live Now behavior
- records the most recent SIP announcement result for `GET /api/v1/announce/status`

Execution does not:

- provide a generic command endpoint
- call Force Go Live automatically
- replace the standalone UI workflow

## Duplicate Guard Confirmation

When `POST /api/v1/announce` hits the existing duplicate guard, SIP returns `requiresConfirmation`, a `confirmationId`, and the guard message. LivePanel can surface that warning and call `POST /api/v1/announce/confirm` if the user chooses to continue.

Confirmation:

- uses the exact announcement snapshot that triggered the duplicate guard
- calls the existing `ForceGoLive` path used by the standalone UI
- is single-use
- expires after five minutes
- replaces any previous pending announcement confirmation when a new guarded announcement attempt occurs

Confirmation does not:

- expose a generic force endpoint
- bypass validation or destination execution behavior outside the existing Force Go Live workflow
- apply to End Stream

## End Stream Execution

`POST /api/v1/end-stream` triggers the existing End Stream execution path using the active SIP profile's default announcement fields.

Execution:

- requires an active SIP profile
- converts profile defaults into a `domain.Announcement`
- calls the same End Stream execution service used by the standalone UI
- preserves existing validation, destination resolution, end-stream publishing, and Live Now cleanup behavior
- records the most recent SIP end-stream result for `GET /api/v1/end-stream/status`

Execution does not:

- provide a generic command endpoint
- call Go Live or Force Go Live
- replace the standalone UI workflow

## Health

Health is intentionally simple:

- `ready`: settings and profiles are readable and no pending Live Now recovery sessions exist
- `degraded`: StreamSignal is running, but recovery state needs attention or cannot be checked
- `error`: core SIP dependencies, settings, or profiles are unavailable

Health does not attempt to predict whether a future publish will succeed.

## Deviations From StarsongDesignSystem Planning Docs

Older StarsongDesignSystem planning notes recommended action endpoints:

```text
GET  /api/v1/actions
POST /api/v1/action
```

The current SIP-v1.1 module contract omits generic command execution and action endpoints. This implementation follows the updated scope and focuses on explicit StreamSignal operations for discovery, health, status, profile discovery, profile activation, current profile lookup, announcement execution/confirmation/status, and end-stream execution/status.
