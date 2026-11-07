# macOS Release Test Plan

This maintainer checklist covers macOS packaging, Keychain behavior, Gatekeeper behavior, and DMG validation. StreamSignal has macOS Keychain-backed secret storage, but macOS app and DMG packaging must be built and validated on macOS. Use the native macOS build environment for this checklist.

## Build Inputs

Required on the Mac build machine:

- Go
- Node.js and npm
- Wails CLI
- Xcode command line tools

Recommended first build:

```bash
npm install --prefix frontend
wails build -platform darwin/universal
```

Expected output:

- `build/bin/StreamSignal.app`

## DMG Packaging

Wails creates the `.app` bundle. DMG creation should happen after the `.app` bundle is built and tested.

Initial tester DMG option:

```bash
hdiutil create -volname StreamSignal -srcfolder build/bin/StreamSignal.app -ov -format UDZO build/bin/StreamSignal-0.5.0-macos-universal.dmg
```

Before public release, decide whether the DMG should be:

- unsigned tester-only
- ad-hoc signed for limited distribution
- Developer ID signed and notarized

## Tester Checklist

- Launch `StreamSignal.app` directly from the build output.
- Launch from the DMG after dragging the app to Applications.
- Confirm Gatekeeper behavior and record any warning text exactly.
- Add Discord, Bluesky, and Mastodon destinations.
- Confirm credentials save without remaining in SQLite.
- Quit and reopen the app.
- Confirm saved destinations still work after restart.
- Confirm macOS Keychain has `StreamSignal` generic password entries.
- Run Preview, Go Live, End Stream, duplicate confirmation, and Live Now Recovery checks.
- Delete the app from Applications and document what remains in app config and Keychain.

## Notes To Report Back

- macOS version and CPU type: Intel, Apple Silicon, or both.
- Whether the universal build launches natively on Apple Silicon.
- Any Keychain prompts or permission wording.
- Any Gatekeeper, quarantine, signing, or notarization warnings.
- Exact DMG creation command used.
