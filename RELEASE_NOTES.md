# StreamSignal Release Notes

User-facing release notes and planned artifact names. Keep this focused on changes that matter to users, testers, and release packaging.

## Unreleased

- Adds the additive SIP observability contract, safe operation correlation and diagnostics.
- Adds capability-advertised SIP operations and diagnostics, correlated safe operation logs, per-destination publication/follow-up results, credential readiness, opt-in asynchronous publishing, and durable idempotency/restart recovery. Existing synchronous clients remain supported. Publishing operations use a five-minute server write deadline; asynchronous work has a separate 4.5-minute execution budget. The operation journal is `sip-operations.json` beside the selected app database (including the isolated dev store), retains up to 256 operations/14 days, and never contains request bodies, rendered content, credentials or receipt URLs. A failed journal write prevents new publishing; interrupted operations are unknown and never automatically replayed. Idempotency compares the route and exact request body bytes. A key is only protected while its operation remains in retained history.
- Freezes the active profile/default fields when accepting publishing work and rejects zero-enabled-target announcements instead of reporting an empty success.
- Adds regression tests and updates patched build/test dependencies.


## 0.5.1 SIP One-Shot Announcement Field Overrides

StreamSignal 0.5.1 is a patch release focused on letting LivePanel send temporary SIP announcement field values for a single Go Live announcement without changing StreamSignal's stored announcement fields or saved profiles.

### Highlights

- Added optional one-shot `fields` overrides to `POST /api/v1/announce`.
- Kept empty announce requests working for the existing active-profile workflow.
- Preserved one-shot field overrides through duplicate-guard confirmation, so LivePanel does not need to resend draft values when confirming.
- Updated SIP API reference docs and Postman collections for the expanded announce request body.

### Security And Reliability

- SIP remains localhost-only, strict-decodes JSON, rejects unknown request fields, and keeps the existing request size limit.
- One-shot overrides validate field IDs against existing StreamSignal announcement field definitions and apply only to the in-memory announcement snapshot.
- One-shot overrides do not update the `announcement_fields` table or announcement profiles.
- Local release prep quality gate and security scan were run for this release.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.5.1-windows-amd64-installer.exe`
- `StreamSignal-0.5.1-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.5.1-windows-amd64-installer`
- `streamsignal-0.5.1-macos-universal-dmg`

## 0.5.0 SIP Announcement Fields

StreamSignal 0.5.0 is a feature release focused on expanding SIP v1 so LivePanel and other local Starsong tools can read and use the announcement fields that StreamSignal stores for the active profile workflow.

### Highlights

- Added SIP announcement-field capability reporting through `/api/v1/app`.
- Added SIP announcement-field models, handlers, service coverage, SQLite persistence, and Wails service wiring.
- Added template rendering support for the expanded announcement field set.
- Updated SIP API reference docs and Postman collections for the announcement-field surface.
- Added a Newman SIP smoke collection and script, plus CI workflow coverage for the local SIP smoke path.

### Security And Reliability

- SIP remains localhost-only, rejects non-local Host headers, avoids CORS, and exposes no generic command/action endpoints.
- Announcement-field responses remain scoped to local SIP integration data and do not expose raw destination credentials, webhook URLs, access tokens, or app passwords.
- Added targeted tests for SIP announcement fields, SQLite migrations, and template rendering.
- Local release prep quality gate and security scan were run for this release.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.5.0-windows-amd64-installer.exe`
- `StreamSignal-0.5.0-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.5.0-windows-amd64-installer`
- `streamsignal-0.5.0-macos-universal-dmg`

## 0.4.0 LivePanel And SIP Control

StreamSignal 0.4.0 is a feature release focused on LivePanel integration through StreamSignal's localhost-only SIP API, service-mode launch support, and release validation around that new control surface.

### Highlights

- Added a SIP API for LivePanel to inspect app identity, read the current active profile, trigger Go Live, confirm duplicate-guarded announcements, trigger End Stream, and read last-run operation status.
- Added `--service` startup so StreamSignal can run backend services and SIP without showing the main window.
- Added `--show` startup so LivePanel or shortcuts can restore an already-running StreamSignal window.
- Kept SIP scoped to the active profile workflow instead of exposing generic command execution or raw destination configuration.
- Added SIP API docs and a Postman collection for local integration testing.
- Added drawer/status metadata for LivePanel control surfaces.
- Aligned the StreamSignal UI with the Starsong design direction.

### Security And Reliability

- SIP remains localhost-only, rejects non-local Host headers, avoids CORS, and exposes no raw webhook URLs, credential keys, account identifiers, tokens, local secret references, raw image URLs, or image data.
- Duplicate-guard confirmation through SIP is single-use and scoped to the pending active-profile announcement.
- Service mode uses the same environment-separated app data, credential namespaces, and single-instance lock behavior as the desktop app.
- Added Windows service-mode smoke validation for hidden startup, SIP service reporting, `--show` activation, standalone startup, and duplicate-process suppression.
- Fixed the Windows service-mode smoke gate so standalone launches pass no empty argument list and nested smoke failures stop the build script.
- Allowed Home fields referenced by destination templates to be left blank for a specific announcement. Preview now warns about the empty template field without blocking publish when the rendered message remains valid.
- Aligned Windows installer metadata with the Starsong Installer Standard v1 publisher identity, default install root, and upgrade-in-place behavior.
- Updated Wails application metadata and platform identifiers so installer, executable, SIP, and About surfaces report the 0.4.0 release identity consistently.
- `go test`, `go vet`, `npm test`, `npm run build`, `npm audit`, `govulncheck`, and the Windows service-mode smoke test passed locally during release prep.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.4.0-windows-amd64-installer.exe`
- `StreamSignal-0.4.0-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.4.0-windows-amd64-installer`
- `streamsignal-0.4.0-macos-universal-dmg`

## 0.3.1 Profile And Destination Workflow Polish

StreamSignal 0.3.1 is a patch release focused on profile editing polish, destination workflow cleanup, safer credential handling, and release-prep validation.

### Highlights

- Added profile usage tracking so recently used profiles sort more helpfully.
- Added separate Home actions to save the current announcement as a new profile or update the selected profile.
- Added Stream Title to announcement profiles so a profile can prefill the full Home announcement context.
- Refined Destinations with quieter group membership display, destination-only and group-only sub-tabs, compact template token controls, and contextual platform setup modals.
- Removed the extra Go Live confirmation step; duplicate-protection confirmation remains when needed.

### Security And Reliability

- Added runtime environment namespacing so non-production review builds can use separate app data and credential stores from a streamer's personal production setup.
- Hardened grouped-destination publishing so secure credentials are hydrated before publish attempts.
- Added validation for unresolved stored credential references so missing OS credential or webhook entries ask the user to re-enter and save credentials.
- Improved publish-result handling so a successful provider post is not hidden by later duplicate-tracking or Live Now recovery bookkeeping failures.
- Added local release-build support for environment-specific review launches and NSIS installer validation.
- `npm audit` and `govulncheck` passed locally during release prep.
- Provider-backed E2E, Windows installer, and macOS DMG release-build jobs passed.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.3.1-windows-amd64-installer.exe`
- `StreamSignal-0.3.1-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.3.1-windows-amd64-installer`
- `streamsignal-0.3.1-macos-universal-dmg`

Release prep validation uploaded both artifacts successfully from the `release-prep-v0.3.1` branch.

## 0.3.0 Announcement Profiles

StreamSignal 0.3.0 is a feature release focused on reusable announcement presets and cleaner media handling on Home.

### Highlights

- Added Announcement Profiles so repeated stream types can prefill Home fields, hashtags, media, and an optional Destination Group.
- Moved announcement media to Home, with one shared announcement image for providers that support attached images and a separate Bluesky link card thumbnail.
- Kept destination-level media as a fallback for existing saved destination configuration.
- Added a dedicated Profiles workspace and Home **Announcement Preset** selector so profiles read as prefill presets, not a separate posting mode.

### Announcement Profiles

Announcement Profiles are prefill presets for repeated stream types. Selecting one on Home fills the announcement fields and optional media defaults; if the preset has a Destination Group, Home switches to that group for review before publishing.

### Security And Reliability

- Runtime secrets remain stored in the OS credential store rather than directly in SQLite.
- Announcement Profile Wails bindings now have app-level coverage for save, duplicate, and delete behavior.
- Provider-backed workflow E2E still runs separately because it creates real test-account posts and deletes them afterward.
- `npm audit` and `govulncheck` passed locally during release prep.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.3.0-windows-amd64-installer.exe`
- `StreamSignal-0.3.0-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.3.0-windows-amd64-installer`
- `streamsignal-0.3.0-macos-universal-dmg`

## 0.2.1 Workflow Refinements

StreamSignal 0.2.1 is a patch release focused on tightening the Home publishing workflow and removing hidden default behavior.

### Highlights

- Added optional Destination Groups while preserving individual destination selection on Home.
- Preserved Default Hashtags as an initial Home prefill while respecting user-cleared hashtags for preview and publish actions.
- Removed Test and Production workflow labels from the publishing model; destinations and optional groups now handle organization.

### Security And Reliability

- Runtime secrets remain stored in the OS credential store rather than directly in SQLite.
- Updated the pinned Go toolchain to `go1.26.4` after `govulncheck` flagged called standard-library vulnerabilities in `go1.26.3`.
- Provider-backed workflow E2E now runs as a separate GitHub Action and gates release artifact builds.
- `npm audit` and `govulncheck` passed locally during release prep.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.2.1-windows-amd64-installer.exe`
- `StreamSignal-0.2.1-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.2.1-windows-amd64-installer`
- `streamsignal-0.2.1-macos-universal-dmg`

## 0.2.0 UX Refresh

StreamSignal 0.2.0 is a feature release focused on making the app feel like a desktop creator control panel: faster to scan, clearer to operate, and better suited to repeated stream workflows.

### Highlights

- Replaced the top-navigation feel with a permanent desktop sidebar for Home, Destinations, Settings, and debug-only Logs.
- Rebuilt Home as the primary command center for session setup, destination selection, preview, Go Live, End Stream, and recovery awareness.
- Redesigned Destinations as an integration-management workspace with quick actions, platform branding, guided setup, and an in-place editor.
- Added bundled Discord, Bluesky, and Mastodon logo assets with legal notice and brand-usage guidance.
- Added collapsible generated previews so multi-destination previews start compact and can be expanded per destination.
- Added best-effort approximate Discord and Mastodon link previews from Open Graph/Twitter metadata, guarded by public-network URL validation, short timeouts, body limits, and non-fatal failure behavior.
- Preserved Default Stream URL prefill for fresh Home sessions while respecting a user-cleared or edited Home Stream URL.
- Kept production builds focused by hiding debug-only workflow surfaces such as Logs.
- Updated tests, security docs, release docs, and dependency posture for the new UI and preview behavior.

### Security And Reliability

- Runtime secrets remain stored in the OS credential store rather than directly in SQLite.
- Link-preview fetches reject localhost, private-network, link-local, unspecified, and multicast targets through the existing netguard path.
- Link-preview image URLs discovered from page metadata are also validated before being sent to the frontend.
- `golang.org/x/net` was upgraded to `v0.38.0` after `govulncheck` flagged reachable GO-2025-3595 through HTML preview parsing.
- `npm audit` and `govulncheck` passed locally during release prep.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.2.0-windows-amd64-installer.exe`
- `StreamSignal-0.2.0-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.2.0-windows-amd64-installer`
- `streamsignal-0.2.0-macos-universal-dmg`
