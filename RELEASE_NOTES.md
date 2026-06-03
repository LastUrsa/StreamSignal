# StreamSignal Release Notes

## 0.2.1 Workflow Refinements

StreamSignal 0.2.1 is a patch release focused on tightening the Home publishing workflow and removing hidden default behavior.

### Highlights

- Added a Home destination filter for All, Production, and Test workflow scopes.
- Scoped Preview, Go Live, confirmed Go Live, and End Stream actions to the visible selected destinations in the active Home filter.
- Preserved Default Hashtags as an initial Home prefill while respecting user-cleared hashtags for preview and publish actions.
- Clarified documentation that Test and Production are destination workflow labels, not provider sandbox modes.

### Security And Reliability

- Runtime secrets remain stored in the OS credential store rather than directly in SQLite.
- Updated the pinned Go toolchain to `go1.26.4` after `govulncheck` flagged called standard-library vulnerabilities in `go1.26.3`.
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
