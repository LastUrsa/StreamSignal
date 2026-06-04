# Release Readiness Checklist

This maintainer checklist is for release candidates, packaging, and final pre-release validation. It is intentionally separate from the README so user setup instructions stay concise.

## Release Goal

Current release target: `0.3.1`.

Latest validation: GitHub Release Build run `26979277052` passed provider-backed E2E, Windows installer, and macOS DMG jobs.

Ship StreamSignal in a way that is:

- safe for real user credentials
- easy to install and update
- understandable for first-time users
- maintainable through normal CI, test, and dependency workflows

## Product Workflow

- [ ] verify Home, Destinations, Profiles, Settings, and debug-only Logs follow a clear workflow
- [ ] verify production builds hide debug-only surfaces
- [ ] verify Preview shows Go Live messages and enabled End Stream messages before publishing
- [ ] verify Go Live publishes only to the selected individual destinations or selected Destination Group
- [ ] verify End Stream posts only to selected destinations with End Stream enabled
- [ ] verify Destination Groups remain optional and do not block individual Home destination selection
- [ ] verify Announcement Profiles prefill Home fields, media, and optional Destination Groups
- [ ] verify Bluesky `Live Now` set, clear, opt-out, duration, and recovery behavior
- [ ] verify duplicate warning and force-confirm flows
- [ ] verify Destination Groups persist across app restarts and never imply provider sandbox modes
- [ ] verify no stale workflow state survives between real app sessions

## Platform Integrations

- [ ] Discord webhook posting succeeds
- [ ] Discord optional additional image behavior is understandable
- [ ] Bluesky app-password posting succeeds
- [ ] Bluesky hashtags, rich stream links, external cards, and thumbnails render correctly
- [ ] Bluesky additional image behavior is clear when no preview card is posted
- [ ] Mastodon access-token posting succeeds
- [ ] Mastodon additional image behavior works with both public image URLs and uploaded local images

## Packaging

- [ ] produce a clean Windows build with `wails build`
- [ ] produce a clean Windows NSIS installer with `wails build -platform windows/amd64 -nsis`
- [ ] produce a macOS app bundle on macOS with `wails build -platform darwin/universal`
- [ ] produce and inspect the macOS DMG
- [ ] verify installer metadata, icon, version, and default install path
- [ ] verify first launch after install succeeds without manual setup steps
- [ ] verify upgrade behavior from the previous release
- [ ] verify GitHub Release artifact names match [RELEASE_NOTES.md](./RELEASE_NOTES.md)

Wails v2 does not cross-compile macOS app bundles from Windows. Use a Mac runner or Mac tester for macOS app, DMG, Keychain, Gatekeeper, signing, and notarization checks.

## Secrets And Local Security

- [ ] verify Windows Credential Manager entries are created, reused, and updated correctly
- [ ] verify macOS Keychain entries are created, reused, and updated correctly
- [ ] verify secrets do not appear in SQLite, logs, diagnostics, or exported recovery data
- [ ] verify masked secret-edit UX works in release builds
- [ ] verify uninstall behavior expectations for retained config and credentials are documented
- [ ] run `npm audit --prefix frontend`
- [ ] run `govulncheck ./...`

## First-Run UX

- [ ] verify a new user can add a destination without internal project knowledge
- [ ] verify credential and URL errors are actionable
- [ ] verify templates, destination selection, preview, and publishing relate clearly
- [ ] verify recovery wording is understandable to a non-technical user

## CI And Release

- [ ] keep the GitHub quality gate green on the release branch
- [ ] confirm `Provider Workflow E2E` passes with the configured test-account secrets
- [ ] validate the release-build workflow before publishing artifacts
- [ ] confirm branch protection checks match the live job names
- [ ] confirm release notes describe user-visible changes
- [ ] confirm no open high-severity security concerns remain
