# Release Readiness Checklist

This checklist replaces the old implementation plan as the active next-phase document for StreamSignal.

Current reality:

- core workflows are implemented
- first-run setup and use has been manually validated end to end
- Discord has been manually validated for webhook auth, Go Live posting, selected-destination targeting, optional additional image embeds, and destination-level End Stream behavior
- Bluesky has been manually validated for app-password auth, Go Live posting, rich stream links/cards, configurable `Live Now`, optional Live Status changes, End Stream clearing, and recovery cleanup
- Mastodon has been manually validated for access-token auth, Go Live posting, and optional additional image attachment
- automated tests and CI gates are in place
- the app is in release-prep polish, not workflow discovery

## Release Goal

Ship StreamSignal in a way that is:

- safe for real user credentials
- easy to install and update
- understandable for first-time users
- maintainable through normal CI, test, and dependency workflows

## Product Readiness

- [x] verify all core workflows manually on a release candidate build
- [x] verify the current UI is understandable for first-run setup and use
- [x] verify the Home, Destinations, Settings, and Logs tabs follow a clear human workflow
- [x] confirm the production/release frontend build hides the debug-only Execution Results section
- [x] confirm the production/release frontend build hides the debug-only Logs tab
- [x] use the debug frontend build when workflow validation needs Execution Results and Logs
- [x] confirm Preview stays network-free
- [x] confirm Preview shows Go Live messages plus enabled destination-level End Stream messages in one pass
- [x] confirm Go Live publishes correctly to Discord, Bluesky, and Mastodon
- [x] confirm End Stream clears Bluesky `Live Now`
- [x] confirm destination-level End Stream posting can be enabled or disabled independently per destination
- [x] confirm Bluesky Live Status changes can be disabled while still posting to Bluesky
- [x] confirm Discord additional image behavior is understandable and does not imply Twitch inline playback
- [x] confirm optional end-stream posting works as expected
- [x] confirm test destinations publish only to their configured test accounts or channels
- [x] confirm duplicate warning and force-confirm flows behave correctly
- [x] confirm pending `Live Now` recovery works after restart
- [x] confirm no dummy, stale, or misleading workflow state survives between real app sessions
- [ ] confirm Bluesky card thumbnail behavior works with both a direct image URL and an uploaded local image on the final release candidate
- [ ] confirm Bluesky additional image behavior is clear when no preview card is posted on the final release candidate
- [ ] confirm Mastodon additional image behavior works with both a public image URL and an uploaded local image on the final release candidate

## Packaging And Installer

- [x] produce a clean Windows release build with `wails build`
- [x] produce a clean Windows NSIS installer with `wails build -platform windows/amd64 -nsis`
- [ ] produce a macOS app bundle on macOS with `wails build -platform darwin/universal`
  - Wails v2.12.0 reports that cross-compiling to Mac from Windows is not currently supported.
  - The macOS build needs a Mac runner or Mac tester with Xcode command line tools, Node, Go, and Wails installed.
- [ ] produce a macOS DMG from the tested `.app` bundle
  - Wails packages a `.app` bundle for macOS; DMG creation needs a macOS packaging step such as `hdiutil` after the app bundle exists.
  - Decide whether the first Mac artifact is an unsigned tester DMG, an ad-hoc signed DMG, or a Developer ID signed and notarized DMG.
- [ ] verify macOS app signing, notarization, and Gatekeeper behavior
- [x] validate installer metadata, icon, and version information
  - Installer and app executable metadata are set to `DRDohr`, `StreamSignal`, and `0.1.0`.
  - The app executable reports populated Windows Details fields through `FileVersionInfo` and direct Win32 version-resource lookup.
- [ ] verify install path behavior on a clean Windows machine or VM
  - Silent temp-directory install succeeded at `C:\Users\Don\AppData\Local\Temp\StreamSignalInstallTest`.
  - Default Program Files install path still needs a clean-machine or VM pass.
- [x] verify first launch after install succeeds without manual setup steps
- [ ] verify upgrade behavior from one build to the next
- [ ] decide how release artifacts will be attached to GitHub releases

## Secrets And Local Security

- [x] verify Windows Credential Manager entries are created, reused, and updated correctly
- [ ] verify macOS Keychain entries are created, reused, updated, and retained or removed as expected
  - This must be checked on macOS because the implementation shells out to the macOS `security` command.
- [x] verify secrets never appear in SQLite, logs, or diagnostics
- [x] verify masked secret-edit UX works in release builds
- [x] verify uninstall behavior expectations for retained or removed secrets are documented
  - The NSIS uninstaller removes the installed program directory.
  - The current installer runs with admin/all-users shell context, while StreamSignal writes per-user config to `%APPDATA%\StreamSignal`.
  - Uninstall currently retains per-user config, WebView2 profile data, and Windows Credential Manager entries.

## First-Run UX

- [x] document the minimum setup path for a new user
- [x] verify destination setup is understandable without internal project knowledge
- [x] verify test destination setup is understandable and clearly separated from production setup
- [x] verify error messages are actionable when credentials or URLs are invalid
- [x] verify recovery wording is understandable to a non-technical user
- [x] verify layout hierarchy feels like a functional desktop tool rather than a landing page or form dump
- [x] verify users can understand how templates, destination selection, preview, and live execution relate to each other

## Documentation

- [x] keep `README.md` focused on install, run, test, quickstart, and project-doc entry points
- [ ] keep `SECURITY.md` and `SECURITY_AUDIT.md` aligned with the current posture
- [x] keep `TEST_VALIDATION.md` aligned with real CI expectations
- [ ] add release notes guidance before the first public release
- [ ] decide whether a dedicated user setup guide should be added

## CI And Maintenance

- [x] keep the GitHub quality gate green on `main`
- [x] keep required branch protection checks aligned with the live job names
- [ ] keep Go toolchain and GitHub Action versions current
- [x] review `govulncheck` and `npm audit` output regularly
- [ ] re-run frontend coverage reporting when significant UI changes land

## Release Decision

Before calling the app release-ready, confirm:

- [ ] packaging is repeatable
- [x] installer behavior is understood
- [ ] macOS DMG behavior is understood
- [x] first-run setup is documented
- [x] usability and workflow validation are complete enough that the app can be trusted by a first-time real user
- [x] quality gate is green
- [x] no open high-severity security concerns remain
