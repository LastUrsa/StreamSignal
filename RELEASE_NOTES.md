# StreamSignal Release Notes

## 0.1.0 MVP Release Candidate

StreamSignal 0.1.0 is the first MVP release candidate for the local-first desktop app that helps streamers and VTubers write one stream announcement and send it to Discord, Bluesky, and Mastodon.

### Highlights

- Local Wails desktop app with Go backend and React + TypeScript frontend.
- Discord webhook posting with selected-destination targeting, optional image embeds, and destination-level End Stream messages.
- Bluesky posting with app-password authentication, rich stream links/cards, configurable `Live Now`, optional Live Status changes, End Stream clearing, and recovery cleanup.
- Mastodon posting with access-token authentication and optional additional image attachment.
- Network-free Preview that shows Go Live messages plus enabled End Stream messages before publishing.
- Explicit production and test destinations so test posts stay isolated from real accounts or channels.
- Duplicate-post protection with force-confirm behavior.
- Runtime secrets stored in the OS credential store instead of SQLite.
- Windows release build and NSIS installer validation are complete enough for release-candidate testing.

### Release Artifacts

Planned artifact names:

- `StreamSignal-0.1.0-windows-amd64-installer.exe`
- `StreamSignal-0.1.0-macos-universal.dmg`

The GitHub release build workflow uploads matching build artifacts and attaches them to GitHub Releases created from `v*` tags:

- `streamsignal-0.1.0-windows-amd64-installer`
- `streamsignal-0.1.0-macos-universal-dmg`

### Platform Notes

- Windows stores secrets in Windows Credential Manager.
- macOS stores secrets in macOS Keychain.
- Windows installer metadata is set to `DRDohr`, `StreamSignal`, and version `0.1.0`.
- The NSIS uninstaller removes the installed program directory. Per-user app config, WebView2 profile data, and Credential Manager entries are currently retained.
- Wails v2.12.0 does not cross-compile macOS app bundles from Windows. The macOS `.app` and DMG must be built and validated on macOS.

### Known Release-Candidate Follow-Up

- Validate the macOS `.app`, DMG, Keychain behavior, Gatekeeper behavior, and signing/notarization path on a Mac.
- Confirm Bluesky card thumbnail behavior with both direct image URLs and uploaded local images on the final release candidate.
- Confirm Bluesky and Mastodon additional image behavior on the final release candidate.
- Verify default Windows install path behavior on a clean Windows machine or VM.
- Decide whether the first public macOS DMG is unsigned tester-only, ad-hoc signed, or Developer ID signed and notarized.
