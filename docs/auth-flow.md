# OAuth 2.0 Auth Flow

whoop-garden uses the WHOOP OAuth 2.0 Authorization Code flow to obtain
access and refresh tokens.

## Flow Overview

```
User                  CLI                    WHOOP API         Browser
 |                     |                         |                |
 |  whoop-garden auth  |                         |                |
 |-------------------->|                         |                |
 |                     | build auth URL          |                |
 |                     |-------------------------------- open ------->|
 |                     |                         |                |
 |                     | start HTTP server :3000 |                |
 |                     |                         |     user logs in |
 |                     |                         |<---------------|
 |                     |<-- GET /callback?code=X |                |
 |                     |                         |                |
 |                     | POST /oauth/oauth2/token|                |
 |                     |------------------------>|                |
 |                     |<-- access_token,        |                |
 |                     |    refresh_token        |                |
 |                     |                         |                |
 |                     | save tokens.json (0600) |                |
 |                     |                         |                |
```

## Required Environment Variables

| Variable | Description |
|---|---|
| `WHOOP_CLIENT_ID` | OAuth app client ID from developer.whoop.com |
| `WHOOP_CLIENT_SECRET` | OAuth app client secret |
| `WHOOP_REDIRECT_URI` | Must match `http://localhost:3000/callback` exactly |

## Scopes Requested

| Scope | Data |
|---|---|
| `offline` | Allows refresh tokens |
| `read:profile` | User profile (name, email) |
| `read:body_measurement` | Height, weight, max heart rate |
| `read:cycles` | Physiological cycles (day strain) |
| `read:recovery` | Recovery score, HRV, RHR, SpO2, skin temp |
| `read:sleep` | Sleep stages, performance, consistency |
| `read:workout` | Workout activity, heart rate zones |

## Token Storage

Tokens are written to `tokens.json` in the current working directory with
`0600` permissions (owner read/write only). The file is excluded from git.

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "scope": "offline read:profile ...",
  "expires_at": "2026-02-20T20:00:00Z"
}
```

`expires_at` is computed at save time as `now + expires_in` and persisted so
subsequent runs can check expiry without calling the API.

## Automatic Token Refresh

Every command calls `auth.RefreshIfNeeded()` before making API calls:

1. Load `tokens.json`
2. Check if `expires_at` is within 5 minutes of now
3. If expiring soon: POST to token endpoint with `grant_type=refresh_token`
4. Save new tokens back to `tokens.json`
5. Return the valid access token

You should only need to run `whoop-garden auth` once. The refresh token is
long-lived; tokens auto-renew as long as you run a command at least once per
refresh token lifetime.

## Troubleshooting

**"no code in callback"** — The WHOOP redirect included an error parameter.
Check the browser URL bar for an `error=` query parameter and its description.

**"token endpoint returned 401"** — Client ID or secret is wrong, or the
authorization code expired. Authorization codes are single-use and
short-lived — complete the flow in one browser session without navigating away.

**Browser doesn't open** — The CLI falls back to printing the full auth URL.
Copy and paste it into a browser to continue.

**Port 3000 already in use** — The callback server cannot start. Stop whatever
is using port 3000 and retry. Common culprits: local dev servers, other OAuth
tools.

**tokens.json not found** — Run `go run . auth` first. The file must exist in
the working directory where you run commands.
