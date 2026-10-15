# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.8.x   | :white_check_mark: |

Only the latest release receives fixes. The client is distributed by the operator's software
deployment — there is no auto-update, so please update promptly.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: **christopher.abanilla@s2a-ot.digital**
(dedicated `security@` alias TBD — replace once one exists).

We will acknowledge receipt of your vulnerability report within 48 hours and send you regular updates about our progress.

**What to include:**
- Description of the vulnerability
- Steps to reproduce the issue
- Possible impact (and the platform: Windows / macOS / Linux)
- Suggested fix (if any)

Do not send real patient data, tokens or dictated text; redact server addresses if they are internal.

**What to expect:**
1. Initial response within 48 hours
2. Assessment within 5 business days
3. Fix timeline and disclosure coordination
4. Credit in security advisory (if desired)

Vulnerabilities in the **openDox server** (authentication, storage, API) belong to the server
repository's security policy. This policy covers the desktop client: its local data handling,
OS integration (clipboard, key injection, global hotkeys, USB-HID, WebView permissions), the native
SSO flow, the installers and the build/release tooling.

---

# Security Architecture

The client is a thin proxy between the user's microphone, the openDox server and the program the
user is working in. It deliberately keeps almost nothing: no audio, no transcripts, no passwords on
disk. For the complete data flow see [`docs/data-mapping.md`](../docs/data-mapping.md); for an
audit-style self-assessment see [`docs/audit-checklist-opendox-client.md`](../docs/audit-checklist-opendox-client.md).

## Authentication & Authorization

Authentication and roles are handled by the server; the client only presents the login.

- **Password login:** the form posts name and password to the server (`/api/v1/auth/login`). The
  password lives only in the component state until it is sent and is never stored.
- **Tokens:** the access and refresh token are kept in the WebView's `localStorage`
  (`access_token`, `refresh_token`); logout removes them. The Go side is stateless: bound methods
  receive the token as a parameter and attach it as a bearer header.
- **Native SSO (OIDC), RFC 8252 loopback flow** (`oidc.go`): a listener on `127.0.0.1` with a random
  port and a **per-attempt random callback path**, so no other local process can guess the URL and
  inject tokens; one attempt at a time; 5-minute timeout; the system browser (not the app) talks to
  the identity provider.

## Local data

| Item | Where | Protection |
|---|---|---|
| Audio, live text, transcript | memory only | not persisted |
| Access/refresh token, UI selections | WebView profile (`localStorage`) | user-profile permissions only — **not** encrypted by the client |
| `settings.json` (server address) | next to the executable or in the user config dir | no personal data |
| `client-policy-cache.json` | user cache dir, mode `0600` | no personal data |
| Clipboard | OS | Windows: excluded from history/cloud, cleared after 30 s if unchanged; macOS/Linux: no special handling |

## Clipboard & insert target

- Windows marks the text with `ExcludeClipboardContentFromMonitorProcessing`,
  `CanIncludeInClipboardHistory = 0`, `CanUploadToCloudClipboard = 0` and fails closed when marking
  fails (`internal/winclip`); the clipboard is cleared after `clipboard_clear_after_seconds` only if
  its sequence number is unchanged (`internal/clientpolicy/cleaner.go`).
- Text is inserted only after an explicit accept step. An optional allow-list of target programs
  (`insert_allowlist`, Windows) asks before inserting into any other program. The foreground window
  **title may contain patient names** and is therefore only displayed — never logged or stored.

## Network

The client talks only to the configured `server_url` (REST, WebSocket, the public
`/api/v1/settings/client-config/` policy endpoint) and, during SSO, to the loopback listener. No
telemetry, no update checks, no external resources. TLS is **not enforced by the client**, but production use requires `https://`
(the server demands it, and microphone capture does not work without it); with any other setup
audio, text and tokens would travel unencrypted. Certificates are validated
against the OS trust store; there is no option to skip validation.

## OS integration

- **Global hotkeys** only register shortcuts; there is no keystroke capture. On macOS the process
  needs Accessibility/Input-Monitoring grants (`accessibility_darwin.go`).
- **USB-HID** access is limited to known dictation-microphone vendor/product IDs
  (`dictation_mic_protocol.go`); only button states are read.
- **WebView permissions:** Linux auto-grants microphone/camera and denies all other WebKit
  permissions; macOS grants media capture for the app's own embedded UI (`webview_perms*.go`).
- **Subprocesses:** `notify-send`/`osascript` with fixed arguments and the client's own binary as a
  HID scan helper (`dictation_mic_scan_helper.go`).

## Known limitations

These are documented rather than hidden — see "Bekannte Lücken" in
[`docs/data-mapping.md`](../docs/data-mapping.md):

- no client-side TLS enforcement or warning for `http://` server addresses (also affects the
  unauthenticated policy endpoint); in production the server requires `https://`;
- tokens in `localStorage` instead of the OS keychain;
- tokens appear in URLs (WebSocket, token refresh, SSO callback) and can end up in server/proxy
  access logs;
- no clipboard protection and no insert-target check on macOS/Linux;
- no Content-Security-Policy in the WebView;
- Windows builds are not code-signed (Smart App Control blocks them) and macOS builds are not
  notarized — sign before a customer rollout;
- no automatic updates.

---

# Deployment Security

For managed Windows environments see
[`build/windows/installer/DEPLOYMENT.md`](../build/windows/installer/DEPLOYMENT.md): disable the
clipboard history and cloud clipboard via GPO/Intune (the client's marking only works if the OS and
clipboard managers honor it), place `settings.json` next to the executable so only administrators
can change it, and manage the client policy centrally in the server (Admin → Clients).

## Security Checklist

### Pre-Deployment
- [ ] Server address uses `https://`
- [ ] Windows installers are code-signed; macOS builds notarized
- [ ] Clipboard history / cloud clipboard disabled by policy on shared workstations
- [ ] Client policy reviewed in the server (allow-list, clipboard clearing)
- [ ] Reverse proxy does not log query strings (tokens appear in WebSocket/refresh URLs)

### Regular Maintenance
- [ ] Update to the latest release (no auto-update)
- [ ] Review Dependabot pull requests and `python3 scripts/check_licenses.py`

---

# GDPR Compliance

> This section describes what is and isn't implemented today — verified against the code, not
> aspirational. None of this is legal advice.

## Technical Measures (Article 32)

| Measure | Implementation | Status |
|---------|---------------|--------|
| Data minimization | No local storage of audio/transcripts; only tokens and UI selections persist | ✅ |
| Encryption in transit | Depends on `server_url`; server requires `https://`, client does not enforce | ⚠️ |
| Encryption at rest | Not applicable to audio/text (not stored); tokens rely on OS profile permissions and disk encryption | ⚠️ |
| Access control | Server-side (JWT, optional LDAP/OIDC, roles); client holds only the user's own tokens | ✅ |
| Clipboard exposure | Windows hardening (history/cloud exclusion, auto-clear, optional target allow-list) | ✅ Windows / ❌ macOS, Linux |
| Human review | Explicit accept step before inserting text; "Kein Medizinprodukt" gate at first login | ✅ |

## Organizational Measures

Outside this repository's scope — operator responsibility: privacy policy, data processing
agreements (for the server and any external LLM/sanitizer endpoint), records of processing
activities (start with [`docs/data-mapping.md`](../docs/data-mapping.md)), works-council involvement,
device management (disk encryption, clipboard policies), breach notification procedure.

Data subject rights (access, erasure, portability, objection) concern data held on the server; the
client keeps no patient data to which they could apply.

---

# Incident Response

Report suspected vulnerabilities as described above. For a compromised workstation: log out of the
client (removes the tokens from the WebView profile), have the operator invalidate that user's
tokens on the server (e.g. by resetting the password) and delete the WebView profile folder listed in
[`docs/data-mapping.md`](../docs/data-mapping.md).

# Contact

**Security Contact:**
- Email: christopher.abanilla@s2a-ot.digital (dedicated `security@` alias TBD)

**Responsible Disclosure:**
We follow a 90-day disclosure policy. After fix confirmation, we will:
1. Issue security advisory
2. Credit reporter (if desired)
3. Update this document

---

# References

- [`docs/data-mapping.md`](../docs/data-mapping.md) — full data-flow mapping
- [`docs/audit-checklist-opendox-client.md`](../docs/audit-checklist-opendox-client.md) — self-assessment
- [`DISCLAIMER.md`](../DISCLAIMER.md) — "Kein Medizinprodukt" notice
- [RFC 8252 — OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [GDPR Guidelines](https://gdpr.eu/)
