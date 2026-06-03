# Build Assets

This directory contains Wails build assets and platform packaging metadata for StreamSignal.

## Structure

- `appicon.png`: source app icon used by Wails.
- `bin/`: local build output.
- `darwin/`: macOS bundle metadata.
- `windows/`: Windows icon, manifest, version metadata, and NSIS installer files.

## macOS

- `darwin/Info.plist`: production app bundle metadata.
- `darwin/Info.dev.plist`: development app bundle metadata.

Build macOS artifacts on macOS:

```bash
wails build -platform darwin/universal
```

## Windows

- `windows/icon.ico`: Windows app icon.
- `windows/info.json`: app and installer version metadata.
- `windows/wails.exe.manifest`: Windows app manifest.
- `windows/installer/`: NSIS installer files.

Build a Windows installer:

```bash
wails build -platform windows/amd64 -nsis
```
