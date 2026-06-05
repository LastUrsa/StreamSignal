# Security Policy

This file is the public vulnerability reporting policy for StreamSignal. It is standard practice for public repos, especially for software that stores credentials or posts to external services.

## Reporting a Vulnerability

If you believe you have found a security issue in StreamSignal, please do not open a public GitHub issue with exploit details.

Instead:

1. Contact the maintainer privately through the repository contact method or profile contact information.
2. Include the affected area, impact, reproduction steps, and environment details if they matter.
3. Do not include secrets, credentials, personal data, or public exploit details.

If private contact details are not yet available on the repository profile, open a minimal public issue asking for a secure reporting channel without including sensitive details.

## What To Include

Helpful reports usually include:

- affected feature or workflow
- expected behavior
- actual behavior
- impact assessment
- reproduction steps
- screenshots or logs, if safe to share

## Disclosure Approach

The goal is responsible disclosure:

- acknowledge the report
- assess severity and impact
- remediate or mitigate the issue
- publish higher-level notes after a fix when appropriate

## Current Security Notes

For the maintainer-facing security posture summary, see [SECURITY_AUDIT.md](./SECURITY_AUDIT.md).

## SIP Prototype Security Notes

The StreamSignal SIP prototype is a local module contract, not a public API.

Current SIP safeguards:

- binds only to `127.0.0.1`
- rejects non-local `Host` headers
- does not emit CORS headers
- requires JSON content type for profile activation
- limits request bodies to 1 MiB
- applies HTTP server timeouts
- exposes no command execution, publish execution, or destructive endpoints

SIP-v1 does not currently include authentication. Treat any future expansion beyond local read/status/profile behavior as requiring a fresh security review.

## Service Mode Security Notes

Service mode changes process startup behavior but does not add SIP endpoints or remote-control commands.

Current service-mode safeguards:

- `--service` starts the existing local backend and SIP server without showing the main window
- `--show` is a process launch argument only, not a SIP command
- duplicate process prevention uses Wails single-instance locking
- the single-instance lock is scoped by `STREAMSIGNAL_ENV` so dev/test builds cannot activate or block the production instance
- service mode continues to use the existing environment-separated app data and credential namespaces

Any future support for remotely activating the UI, executing commands, or controlling publish behavior over SIP should receive a fresh security review before implementation.
