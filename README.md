# StreamSignal

StreamSignal is a local-first desktop app for streamers and VTubers that creates one announcement and distributes it to multiple platforms.

Current product status:

- Wails desktop shell, Go backend, and React + TypeScript frontend are all wired together
- First-run setup and use has been validated end to end with real Discord, Bluesky, and Mastodon destinations
- Preview, Go Live, End Stream, duplicate confirmation, and explicit production/test destinations are implemented and manually exercised
- Discord, Bluesky, and Mastodon publishers are connected
- Discord webhook posting has been manually validated, including selected-destination targeting, optional additional image embeds, and destination-level End Stream enablement
- Bluesky posting has been validated with app-password auth, rich stream links, external cards, configurable `Live Now` duration, an opt-out for Live Status changes, End Stream clearing, and manual recovery if the app closes mid-stream
- Mastodon posting has been validated with manual access-token auth and optional additional image attachment
- Preview is network-free and shows both Go Live messages and enabled destination-level End Stream messages when available
- runtime secrets are stored through the OS credential store rather than directly in SQLite
- remote image URL fetches for Bluesky and Mastodon are guarded against local/private-network targets
- automated backend and frontend tests are part of the normal workflow

The app is past core workflow validation and is now in release-prep polish: documentation, packaging, installer behavior, and final release-candidate checks.

## Quickstart

1. Open StreamSignal and go to **Destinations**.
2. Add a destination for each platform you want to publish to:
   - **Discord**: create a channel webhook in Discord, name the webhook app/bot that should appear as the messenger, copy its webhook URL, paste it into StreamSignal, and test the connection.
   - **Bluesky**: create a Bluesky app password, enter the account identifier and app password, then choose whether StreamSignal should change Bluesky Live Status during Go Live.
   - **Mastodon**: create a Mastodon application/access token, enter the instance URL and token, then test the connection.
3. On **Home**, enter the stream title, stream URL, optional category, message, and hashtags.
4. Use **Send To** to choose which destinations are included in the current session.
5. Click **Generate Preview** to inspect the exact Go Live and enabled End Stream messages before anything is sent.
6. Click **Go Live** to publish to the selected destinations. If duplicate protection warns you, confirm only when you intend to post again.
7. Click **End Stream** when the stream is over. End Stream only posts to selected destinations with **Post End Stream Message** enabled and clears tracked Bluesky Live Now status when needed.
8. Use **Settings > Live Now Recovery** if StreamSignal closed before it could clear a pending Bluesky Live Now status.

## Stack

- Go backend
- Wails desktop shell
- React + TypeScript frontend
- SQLite for application data
- Windows Credential Manager and macOS Keychain for secrets

## Development

Run the app in development mode:

```bash
wails dev
```

Build the desktop app:

```bash
wails build
```

Build a macOS app bundle on a Mac:

```bash
wails build -platform darwin/universal
```

Wails v2 does not cross-compile macOS app bundles from Windows. A DMG release artifact should be created on macOS after the `.app` bundle is built, signed, and tested.

Run the automated test suites:

```bash
cd frontend && npm run build
go test ./...
npm test --prefix frontend
```

Run the frontend dependency audit:

```bash
npm audit --prefix frontend
```

## Project Docs

- [RELEASE_NOTES.md](./RELEASE_NOTES.md): release notes draft and artifact naming
- [SECURITY.md](./SECURITY.md): vulnerability reporting policy
- [SECURITY_AUDIT.md](./SECURITY_AUDIT.md): public-facing security posture summary
- [INTEGRATION_LIMITATIONS.md](./INTEGRATION_LIMITATIONS.md): current platform integration constraints and boundaries
- [TEST_VALIDATION.md](./TEST_VALIDATION.md): testing strategy and PR quality-gate guidance
- [RELEASE_READINESS_CHECKLIST.md](./RELEASE_READINESS_CHECKLIST.md): release and packaging checklist for the next phase
- [MACOS_RELEASE_TEST_PLAN.md](./MACOS_RELEASE_TEST_PLAN.md): macOS app, DMG, Keychain, and Gatekeeper validation plan
- [.github/workflows/quality-gate.yml](./.github/workflows/quality-gate.yml): GitHub PR checks
