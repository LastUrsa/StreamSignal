# Security Posture Summary

This maintainer-facing summary records the current security posture for StreamSignal. It complements [SECURITY.md](./SECURITY.md), which is the public vulnerability reporting policy.

## Scope

The current review focus is:

- local secret storage
- SQLite persistence
- logs, diagnostics, and recovery data
- outbound network fetches for platform posting and previews
- dependency and vulnerability scanning

## Current Posture

- Runtime secrets are stored through the OS credential store: Windows Credential Manager on Windows and macOS Keychain on macOS.
- SQLite stores secret references, destination settings, local app data, and optional local image data; it should not store raw webhook URLs, access tokens, or app passwords.
- Legacy SQLite rows that contain raw secrets are migrated away from raw-secret storage when read through the app.
- Diagnostics and logs are designed to avoid raw credential output.
- Outbound image and preview fetches use public HTTP(S) URL validation, request timeouts, redirect/body limits where relevant, and bounded error-body reads.
- Link-preview fetching is best-effort and non-fatal; preview generation continues when metadata fetches fail.
- Go and frontend dependency checks are part of normal maintenance.

## Validated Areas

- Discord webhook URLs, Bluesky app passwords, and Mastodon access tokens are treated as secrets.
- Bluesky `Live Now` session tracking stores the credential reference needed for recovery, not the raw credential.
- Masked edit fields prevent stored secrets from being hydrated back into visible destination forms.
- Remote image URLs for Bluesky and Mastodon are rejected if they resolve to localhost, private-network, link-local, unspecified, or multicast addresses.
- Uploaded image data is size-limited before decode and again after decode.
- Preview metadata image URLs are validated before being sent to the frontend.

## Remaining Cautions

- Local image uploads are not credentials, but they may still be private content. Treat the local StreamSignal data directory as private application data.
- Future logging and diagnostics changes should continue treating secrets as excluded data.
- macOS Keychain behavior must be validated on macOS because the implementation depends on the platform `security` command.
- Signing, notarization, and installer behavior should be rechecked for each public release.

## Maintenance Checks

Run these during release prep and after dependency changes:

```bash
npm audit --prefix frontend
govulncheck ./...
```

Security-sensitive changes should also add or update targeted tests for:

- secret persistence and migration
- diagnostics redaction
- outbound URL validation
- publisher request payloads
- recovery workflows

## References

- OWASP Secrets Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- OWASP Logging Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
