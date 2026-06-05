# StreamSignal SIP API Reference

This document is the implementation-aligned reference for StreamSignal's SIP-v1 prototype surface.

SIP-v1 is currently a local ecosystem protocol, not a public developer API. StreamSignal exposes SIP over localhost HTTP only, using JSON request and response bodies. Request and response model definitions are centralized in `internal/sip/models.go`.

Implemented endpoints:

```text
GET  /api/v1/app
GET  /api/v1/health
GET  /api/v1/capabilities
GET  /api/v1/status
GET  /api/v1/profiles
POST /api/v1/profile
```

Action endpoints are not implemented in this prototype.

## Security Posture

The SIP server is intentionally local and narrow:

- binds to `127.0.0.1` only
- rejects non-local `Host` headers
- does not emit CORS headers
- limits request bodies to 1 MiB
- requires `Content-Type: application/json` for `POST /api/v1/profile`
- sends `X-Content-Type-Options: nosniff`
- sends `Cache-Control: no-store`
- uses HTTP server read, write, idle, and header timeouts

SIP-v1 has no authentication layer. It is intended for same-machine module communication, and no destructive publish or command execution endpoints are exposed.

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
  "protocolVersion": "1.0"
}
```

## Notes

- `appId` is fixed as `streamsignal`.
- `version` comes from `wails.json` product metadata.
- `mode` is `standalone`. StreamSignal does not currently expose service-mode behavior.
- `protocolVersion` is fixed to SIP `1.0` for this prototype.

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
- `supportsAnnouncements` is `true` because the SIP surface represents StreamSignal's announcement workflow state and profiles. SIP does not expose publish execution.
- Command execution and action endpoints are intentionally out of scope for SIP-v1.

## GET /api/v1/status

## Purpose

Provides a concise user-facing summary of StreamSignal's SIP-visible state.

This endpoint is intended for lightweight polling by another local Starsong participant during prototype validation.

## Response

Idle example:

```json
{
  "state": "idle",
  "message": "Ready for announcement setup."
}
```

After SIP profile activation:

```json
{
  "state": "idle",
  "message": "Ready for announcement setup.",
  "activeProfile": "Music Stream",
  "activeProfileId": "music"
}
```

Warning example:

```json
{
  "state": "warning",
  "message": "Pending Live Now recovery sessions need attention.",
  "activeProfile": "Music Stream",
  "activeProfileId": "music"
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
- Activation does not publish announcements.
- Activation does not call Go Live, End Stream, or Force Go Live.
- Activation does not live-synchronize the React Home form in this prototype.

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
- `ErrorResponse`

Handlers and tests use these centralized models directly. No Swagger, OpenAPI generation, Swagger UI, code generation, API explorer, or API client tooling is part of this prototype.
