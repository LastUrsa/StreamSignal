# StreamSignal SIP API Reference

This document is the implementation-aligned reference for StreamSignal's SIP-v1.1 surface.

SIP-v1.1 is currently a local ecosystem protocol, not a public developer API. StreamSignal exposes SIP over localhost HTTP only, using JSON request and response bodies. Request and response model definitions are centralized in `internal/sip/models.go`.

Implemented endpoints:

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

Action endpoints are not implemented in this prototype.

## Security Posture

The SIP server is intentionally local and narrow:

- binds to `127.0.0.1` only
- rejects non-local `Host` headers
- does not emit CORS headers
- limits request bodies to 1 MiB
- requires `Content-Type: application/json` for `POST /api/v1/profile`
- requires `Content-Type: application/json` for `POST /api/v1/announce/confirm`
- accepts an empty body or `{}` for `POST /api/v1/announce` and `POST /api/v1/end-stream`
- sends `X-Content-Type-Options: nosniff`
- sends `Cache-Control: no-store`
- uses HTTP server read, write, idle, and header timeouts

SIP-v1.1 has no authentication layer. It is intended for same-machine module communication, and no destructive or generic command execution endpoints are exposed.

## GET /api/v1/app

## Purpose

Identifies StreamSignal as a SIP participant so another local Starsong application can discover the app, read its product version, and confirm the SIP protocol version.

## Response

```json
{
  "appId": "streamsignal",
  "name": "StreamSignal",
  "version": "0.3.1",
  "mode": "standalone",
  "protocolVersion": "1.1"
}
```

## Notes

- `appId` is fixed as `streamsignal`.
- `version` comes from `wails.json` product metadata.
- `mode` is `standalone` for normal desktop launches and `service` for `--service` launches.
- `protocolVersion` is fixed to SIP `1.1`.

## GET /api/v1/health

## Purpose

Reports whether StreamSignal is operational enough to participate in local SIP interactions.

Health is app-level operability. It does not predict whether a future publish attempt will succeed.

## Response

Ready example:

```json
{
  "status": "ready",
  "message": "StreamSignal is ready for local SIP participation."
}
```

Degraded example:

```json
{
  "status": "degraded",
  "message": "Pending Live Now recovery sessions need attention."
}
```

Error example:

```json
{
  "status": "error",
  "message": "Announcement profiles are unavailable."
}
```

## Notes

Implemented health states:

```text
ready
degraded
error
```

- `ready`: settings and announcement profiles are readable, and no pending Live Now recovery sessions exist.
- `degraded`: StreamSignal is running, but Live Now recovery state needs attention or cannot be checked.
- `error`: core SIP dependencies, settings, or announcement profiles are unavailable.

## GET /api/v1/capabilities

## Purpose

Advertises the StreamSignal SIP features that another local participant can rely on without app-specific assumptions.

## Response

```json
{
  "supportsProfiles": true,
  "supportsStatusReporting": true,
  "supportsAnnouncements": true
}
```

## Notes

- `supportsProfiles` is `true` because StreamSignal exposes existing `AnnouncementProfile` records through SIP.
- `supportsStatusReporting` is `true` because `/api/v1/status` is implemented.
- `supportsAnnouncements` is `true` because SIP can trigger the existing Go Live and End Stream announcement workflows for the active profile.
- Command execution and action endpoints are intentionally out of scope for SIP-v1.1.

## GET /api/v1/status

## Purpose

Provides a concise user-facing summary of StreamSignal's SIP-visible state.

This endpoint is intended for lightweight polling by another local Starsong participant during prototype validation.

## Response

Idle example:

```json
{
  "state": "idle",
  "message": "Ready for announcement setup.",
  "healthy": true
}
```

After SIP profile activation:

```json
{
  "state": "idle",
  "message": "Ready for announcement setup.",
  "activeProfile": "Music Stream",
  "activeProfileId": "music",
  "healthy": true,
  "destinationCount": 8
}
```

Warning example:

```json
{
  "state": "warning",
  "message": "Pending Live Now recovery sessions need attention.",
  "activeProfile": "Music Stream",
  "activeProfileId": "music",
  "healthy": true
}
```

## Notes

Implemented status states:

```text
idle
warning
error
```

- `idle`: StreamSignal is ready for announcement setup.
- `warning`: health is degraded.
- `error`: health is error.
- `activeProfile` and `activeProfileId` are omitted until a profile has been activated through SIP.
- `healthy` is informational and mirrors whether StreamSignal is usable for local SIP work.
- `destinationCount` is informational and omitted if destinations cannot be read.
- SIP active profile state is backend/SIP state. It does not live-synchronize the React Home form in this prototype.

## GET /api/v1/profiles

## Purpose

Lists StreamSignal's existing announcement profiles so another local SIP participant can discover profile names for profile activation.

## Response

```json
{
  "profiles": [
    "Music Stream",
    "Gaming Stream"
  ]
}
```

## Notes

- Profiles come from StreamSignal's existing `AnnouncementProfile` system.
- The response contains profile names, matching Starsong Profiles v1.
- Empty profile names are not included.
- This endpoint does not create, edit, delete, import, export, or synchronize profiles.

## GET /api/v1/profile/current

## Purpose

Returns the currently active SIP announcement profile.

## Response

```json
{
  "id": "gaming-stream",
  "name": "Gaming Stream"
}
```

If no profile is active:

```json
{
  "id": "",
  "name": ""
}
```

## POST /api/v1/profile

## Purpose

Activates an existing StreamSignal announcement profile by name.

Activation validates that Profiles v1 can apply an application-owned profile without triggering publishing or higher-risk workflow execution.

## Request

```json
{
  "profile": "Music Stream"
}
```

## Response

```json
{
  "success": true,
  "profile": "Music Stream",
  "profileId": "music"
}
```

## Notes

- Profile names are matched case-insensitively.
- Activation marks the selected announcement profile as used.
- Activation stores the active SIP profile in backend/SIP memory.
- The active SIP profile can be verified with `GET /api/v1/status`.
- Activation does not publish announcements by itself.
- Activation does not call Go Live, End Stream, or Force Go Live.
- Activation does not live-synchronize the React Home form in this prototype.

## POST /api/v1/announce

## Purpose

Triggers a Go Live announcement using the currently active SIP profile. This uses StreamSignal's existing execution service, including template rendering, destination resolution, validation, duplicate protection, publish behavior, and Live Now handling.

## Request

```json
{}
```

The body may also be empty.

## Response

Success:

```json
{
  "success": true
}
```

Failure:

```json
{
  "success": false,
  "error": "No active profile"
}
```

Duplicate guard:

```json
{
  "success": false,
  "requiresConfirmation": true,
  "confirmationId": "a32a66ed-bbf2-48cb-b5ea-70e4f6bde947",
  "error": "A similar announcement was recently posted within the last 10 minutes. Continue?"
}
```

## Notes

- The active profile's default fields become the announcement input.
- Execution uses the same `GoLive` path as the standalone UI.
- Duplicate guard responses include a single-use `confirmationId` for `POST /api/v1/announce/confirm`.
- Validation errors are returned as `success: false`.

## POST /api/v1/announce/confirm

## Purpose

Confirms a pending duplicate guard from `POST /api/v1/announce`.

This endpoint is intentionally scoped to duplicate-guard confirmation. It uses the exact announcement snapshot that produced the duplicate warning and calls StreamSignal's existing `ForceGoLive` path, matching the standalone UI's continue-anyway behavior.

## Request

```json
{
  "confirmationId": "a32a66ed-bbf2-48cb-b5ea-70e4f6bde947"
}
```

## Response

Success:

```json
{
  "success": true
}
```

Invalid, expired, reused, or missing confirmation IDs return `InvalidRequest`.

## Notes

- Confirmations are stored in SIP memory.
- Confirmations are single-use.
- Confirmations expire after five minutes.
- Starting a new announcement attempt replaces the pending duplicate confirmation.
- This endpoint does not provide a general force or command mechanism.

## GET /api/v1/announce/status

## Purpose

Returns the most recent SIP announcement operation result.

## Response

```json
{
  "lastRun": "2026-06-05T12:00:00Z",
  "success": true
}
```

Failure example:

```json
{
  "lastRun": "2026-06-05T12:00:00Z",
  "success": false,
  "error": "Discord destination unavailable"
}
```

Duplicate guard example:

```json
{
  "lastRun": "2026-06-05T12:00:00Z",
  "success": false,
  "requiresConfirmation": true,
  "confirmationId": "a32a66ed-bbf2-48cb-b5ea-70e4f6bde947",
  "error": "A similar announcement was recently posted within the last 10 minutes. Continue?"
}
```

## POST /api/v1/end-stream

## Purpose

Triggers the existing End Stream workflow using the currently active SIP profile. This uses StreamSignal's existing execution service, including destination resolution, end-stream template validation, end-stream publish behavior, and Bluesky Live Now cleanup.

## Request

```json
{}
```

The body may also be empty.

## Response

Success:

```json
{
  "success": true
}
```

Failure:

```json
{
  "success": false,
  "error": "No active profile"
}
```

## Notes

- The active profile's default fields become the end-stream announcement input.
- Execution uses the same `EndStream` path as the standalone UI.
- Validation and publish failures are returned as `success: false`.

## GET /api/v1/end-stream/status

## Purpose

Returns the most recent SIP end-stream operation result.

## Response

```json
{
  "lastRun": "2026-06-05T12:00:00Z",
  "success": true
}
```

Failure example:

```json
{
  "lastRun": "2026-06-05T12:00:00Z",
  "success": false,
  "error": "No active profile"
}
```

## Standard Error Format

SIP errors use this JSON shape:

```json
{
  "success": false,
  "error": "ProfileNotFound"
}
```

## Supported Errors

```text
InvalidRequest
ProfileNotFound
Forbidden
UnexpectedFailure
```

`InvalidRequest` is returned when:

- an endpoint is called with the wrong HTTP method
- `POST /api/v1/profile` receives invalid JSON
- `POST /api/v1/profile` receives an empty `profile` value
- `POST /api/v1/profile` receives unknown JSON fields
- `POST /api/v1/announce` receives a body without `Content-Type: application/json`
- `POST /api/v1/announce` receives malformed JSON
- `POST /api/v1/announce` receives unknown JSON fields
- `POST /api/v1/announce` receives trailing JSON values
- `POST /api/v1/announce/confirm` receives invalid JSON
- `POST /api/v1/announce/confirm` receives an empty, expired, reused, or unknown `confirmationId`
- `POST /api/v1/announce/confirm` receives unknown JSON fields
- `POST /api/v1/announce/confirm` receives trailing JSON values
- `POST /api/v1/end-stream` receives a body without `Content-Type: application/json`
- `POST /api/v1/end-stream` receives malformed JSON
- `POST /api/v1/end-stream` receives unknown JSON fields
- `POST /api/v1/end-stream` receives trailing JSON values

`ProfileNotFound` is returned when:

- `POST /api/v1/profile` names a profile that does not exist

`Forbidden` is returned when:

- a request uses a non-local `Host` header

`UnexpectedFailure` is returned when:

- an unexpected internal error prevents an endpoint from returning its normal response

## Contract Model Types

The shared SIP model layer defines these contract types in `internal/sip/models.go`:

- `AppResponse`
- `HealthResponse`
- `CapabilitiesResponse`
- `StatusResponse`
- `ProfilesResponse`
- `ActivateProfileRequest`
- `ProfileActivationResponse`
- `CurrentProfileResponse`
- `AnnounceResponse`
- `AnnounceStatusResponse`
- `AnnounceConfirmRequest`
- `EndStreamResponse`
- `EndStreamStatusResponse`
- `ErrorResponse`

Handlers and tests use these centralized models directly. No Swagger, OpenAPI generation, Swagger UI, code generation, API explorer, or API client tooling is part of this prototype.
