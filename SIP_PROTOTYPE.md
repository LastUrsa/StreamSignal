# StreamSignal SIP Prototype

This document captures the StreamSignal SIP-v1 prototype architecture and scope.

## Scope

The prototype implements only these SIP-v1 endpoints:

```text
GET  /api/v1/app
GET  /api/v1/health
GET  /api/v1/capabilities
GET  /api/v1/status
GET  /api/v1/profiles
POST /api/v1/profile
```

Action endpoints are intentionally not implemented in this phase.

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

SIP-v1 does not include authentication. This implementation relies on same-machine locality and avoids exposing command execution, destructive actions, or publish execution through SIP.

## Architecture

SIP code lives in `internal/sip`.

That package owns:

- HTTP routing and JSON responses
- SIP response models
- SIP health, capabilities, status, and profile translation
- in-memory active SIP profile state

The SIP layer uses existing StreamSignal services. It does not replace or redesign announcement profiles, settings, recovery, publishing, or frontend state.

## Profile Activation

`POST /api/v1/profile` activates an existing `AnnouncementProfile` by name.

Activation:

- matches existing announcement profiles case-insensitively by name
- marks the selected profile as used
- stores the active SIP profile in backend/SIP memory
- makes the active profile visible through `GET /api/v1/status`

Activation does not:

- publish announcements
- call Go Live
- call End Stream
- call Force Go Live
- edit destination configuration
- live-synchronize the React Home form

The Home screen's compose state is currently React-local state. For this prototype, SIP profile activation updates backend/SIP state and usage metadata only.

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

The current SIP-v1 module contract omits command execution and action endpoints. This implementation follows the updated scope and focuses on validating SIP-v1 discovery, health, status, capabilities, profile discovery, and profile activation.
