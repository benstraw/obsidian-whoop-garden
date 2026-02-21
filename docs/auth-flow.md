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
 |                     | save tokens.json        |                |
 |                     |                         |                |
```

## Environment Variables Required

| Variable | Description |
|---|---|
| `WHOOP_CLIENT_ID` | OAuth app client ID from WHOOP developer portal |
| `WHOOP_CLIENT_SECRET` | OAuth app client secret |
| `WHOOP_REDIRECT_URI` | Must be `http://localhost:3000/callback` |

## Scopes Requested

- `offline` — allows refresh tokens
- `read:profile` — user profile data
- `read:body_measurement` — height, weight, max HR
- `read:cycles` — physiological cycles (day strain)
- `read:recovery` — recovery scores, HRV, RHR
- `read:sleep` — sleep stages and scoring
- `read:workout` — workout activity data

## Token Storage

Tokens are stored in `tokens.json` in the working directory with `0600`
permissions (owner read/write only). The file is excluded from git via
`.gitignore`.

Structure:
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

## Token Refresh

On each command, `RefreshIfNeeded()` is called:
1. Load `tokens.json`
2. Check if `expires_at` is within 5 minutes
3. If expiring soon, POST to token endpoint with `grant_type=refresh_token`
4. Save new tokens to `tokens.json`

## Testing the Auth Flow

1. Ensure your `.env` is configured with valid credentials
2. Run: `go run . auth`
3. Browser opens to WHOOP authorization page
4. Log in and approve scopes
5. Browser redirects to `http://localhost:3000/callback`
6. CLI captures the code and exchanges for tokens
7. `tokens.json` is created in current directory
8. Run `go run . daily` to verify API access works

## Troubleshooting

**"no code in callback"** — The WHOOP redirect may have included an error
parameter. Check the URL shown in the browser.

**"token endpoint returned 401"** — Client ID or secret may be wrong, or
the authorization code has expired (codes expire quickly — complete auth
in one session).

**Browser doesn't open** — The CLI falls back to printing the URL. Copy
and paste it into a browser manually.

**Port 3000 already in use** — Stop whatever is using port 3000 before
running `auth`.
