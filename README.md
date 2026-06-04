# StreamSignal

StreamSignal is a local-first desktop app for streamers and VTubers that creates one announcement and distributes it to multiple platforms.

It is intended for creators who want a repeatable Go Live and End Stream workflow without handing their platform credentials to a hosted service. StreamSignal stores app data locally, keeps runtime secrets in the OS credential store, and posts only to the destinations selected for the current session.

## What It Does

- publishes Go Live announcements to Discord, Bluesky, and Mastodon
- generates destination-specific previews before anything is sent
- supports optional End Stream messages per destination
- can set and clear Bluesky `Live Now` status, including recovery after an interrupted app session
- supports reusable Announcement Profiles for repeated stream types
- supports default stream URLs, hashtags, and optional announcement media for fresh sessions

## App Areas

- **Home**: session details, destination selection, preview, Go Live, End Stream, and recovery awareness.
- **Destinations**: Discord, Bluesky, and Mastodon setup, testing, and Destination Groups.
- **Profiles**: reusable announcement presets for repeated stream types.
- **Settings**: session defaults, Bluesky Live Now recovery, and legal notices.

Debug builds also include **Logs** for development and workflow validation. Logs are not shown in production releases.

## Quickstart

1. Open StreamSignal and go to **Destinations**.
2. Add a destination for each platform you want to publish to:
   - **Discord**: create a channel webhook in Discord, name the webhook app/bot that should appear as the messenger, copy its webhook URL, paste it into StreamSignal, and test the connection.
   - **Bluesky**: create a Bluesky app password, enter the account identifier and app password, then choose whether StreamSignal should change Bluesky Live Status during Go Live.
   - **Mastodon**: create a Mastodon application/access token, enter the instance URL and token, then test the connection.
3. On **Home**, enter the stream title, stream URL, optional category, message, and hashtags.
4. Optionally create Announcement Profiles in **Profiles** for repeated stream types. Profiles can prefill Home fields, media, and a Destination Group.
5. On Home, choose individual destinations or switch to a Destination Group for the current session. New destinations start in **All Destinations**.
6. Add optional media on Home when the selected destinations support it. The announcement image is shared by providers that support attached images; Bluesky link card thumbnails have their own field.
7. Click **Generate Preview** to inspect the Go Live and enabled End Stream messages before anything is sent. Generated previews start collapsed and can be expanded per destination.
8. Click **Go Live** to publish to the selected destinations. If duplicate protection warns you, confirm only when you intend to post again.
9. Click **End Stream** when the stream is over. End Stream only posts to selected destinations with **Post End Stream Message** enabled and clears tracked Bluesky Live Now status when needed.
10. Use **Settings > Live Now Recovery** if StreamSignal closed before it could clear a pending Bluesky Live Now status.

## Stack

- Go backend
- Wails desktop shell
- React + TypeScript frontend
- SQLite for application data
- Windows Credential Manager and macOS Keychain for secrets

## Development

Install dependencies before the first run:

```bash
npm install --prefix frontend
```

Run the app in development mode:

```bash
wails dev
```

Build the Windows desktop app from WSL:

```bash
./scripts/build-prod-windows.sh
```

Build and launch it for local review:

```bash
./scripts/build-prod-windows.sh --launch
```

Build a macOS app bundle on a Mac:

```bash
wails build -platform darwin/universal
```

Wails v2 does not cross-compile macOS app bundles from Windows. A DMG release artifact should be created on macOS after the `.app` bundle is built, signed, and tested.

Run the usual local checks:

```bash
npm test --prefix frontend
npm run build --prefix frontend
go test ./...
go vet ./...
```

Run provider-backed workflow E2E tests manually when you need to verify the real posting path outside GitHub Actions:

```bash
STREAMSIGNAL_E2E=1 go test -tags=e2e ./internal/e2e -v -count=1
```

The E2E tests create real posts through the StreamSignal execution workflow and then delete them. The GitHub `Provider Workflow E2E` action runs the same scenario on a schedule and before release artifact builds when the matching repo secrets are configured. Use test-only accounts with `E2E_BLUESKY_IDENTIFIER`, `E2E_BLUESKY_APP_PASSWORD`, `E2E_MASTODON_INSTANCE_URL`, and `E2E_MASTODON_ACCESS_TOKEN`. Mastodon tokens need `write:statuses`; add `write:media` only for future media-upload E2E coverage.

Run dependency and vulnerability checks:

```bash
npm audit --prefix frontend
govulncheck ./...
```

## Project Docs

- [SECURITY.md](./SECURITY.md): vulnerability reporting policy.
- [INTEGRATION_LIMITATIONS.md](./INTEGRATION_LIMITATIONS.md): practical platform constraints.
- [RELEASE_NOTES.md](./RELEASE_NOTES.md): user-facing release notes and artifact names.
- [BRAND_USAGE.md](./BRAND_USAGE.md): platform logo and legal-notice guardrails.
- [TEST_VALIDATION.md](./TEST_VALIDATION.md): maintainer testing posture and PR quality gate.
- [RELEASE_READINESS_CHECKLIST.md](./RELEASE_READINESS_CHECKLIST.md): maintainer release checklist.
- [MACOS_RELEASE_TEST_PLAN.md](./MACOS_RELEASE_TEST_PLAN.md): macOS packaging and validation checklist.
- [SECURITY_AUDIT.md](./SECURITY_AUDIT.md): maintainer security posture summary.
