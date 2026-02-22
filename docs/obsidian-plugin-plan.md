# Obsidian WHOOP Plugin — Project Plan

## Overview

A native Obsidian plugin that fetches WHOOP fitness data and writes daily/weekly
health notes directly into the vault — no terminal, no Go binary, no env vars.
Separate repo from `obsidian-whoop-garden` (different language, toolchain,
artifact, and lifecycle).

Suggested repo name: `obsidian-whoop-plugin`

---

## Tech Stack

| Concern | Choice |
|---|---|
| Language | TypeScript |
| Bundler | esbuild (standard for Obsidian plugins) |
| HTTP | `requestUrl()` from `obsidian` module |
| Storage | `Plugin.loadData()` / `saveData()` (built-in) |
| Templates | Template literals (no external lib needed) |

---

## Repo Structure

```
obsidian-whoop-plugin/
├── manifest.json          # id, name, version, minAppVersion
├── package.json
├── tsconfig.json
├── esbuild.config.mjs
├── styles.css
└── src/
    ├── main.ts            # Plugin entry point
    ├── auth.ts            # OAuth2 flow + token lifecycle
    ├── client.ts          # WHOOP API HTTP wrapper
    ├── fetch.ts           # Endpoint calls + pagination
    ├── models.ts          # TypeScript interfaces for WHOOP v2 JSON
    ├── render.ts          # Note generation (template strings)
    ├── settings.ts        # PluginSettingTab UI
    └── templates/
        ├── daily.ts       # Daily note template
        └── weekly.ts      # Weekly note template
```

---

## OAuth2 Strategy (Key Decision)

The plugin cannot bind a TCP port to receive OAuth callbacks. Options ranked by UX:

### Option A — `obsidian://` URI handler (preferred)
- Register `registerObsidianProtocolHandler('whoop-callback', ...)` in the plugin
- Set `redirect_uri=obsidian://whoop-callback` in the WHOOP developer console
- Electron registers `obsidian://` with the OS; browser redirects hand control back
- **Blocker:** WHOOP must accept non-`http://` redirect URIs — verify this first

### Option B — Manual code paste (fallback)
- Plugin opens WHOOP auth URL in browser
- User copies `?code=...` from the browser URL bar after redirect
- Pastes into a modal text field in the plugin
- Plugin completes the token exchange
- Ugly but zero infrastructure; works regardless of what WHOOP allows

### Option C — Companion sidecar (avoid)
- Tiny local server binary just for OAuth
- Platform-specific, fragile, bad install UX

**Recommendation:** Implement Option B first (unblocks everything), add Option A
once the WHOOP developer console redirect URI allowlist is confirmed.

---

## Features (MVP)

- [ ] Settings tab: Client ID, Client Secret, output folder path
- [ ] "Authorize with WHOOP" flow (OAuth2, Option B initially)
- [ ] Command: Generate today's daily note
- [ ] Command: Generate daily note for a specific date
- [ ] Command: Generate current week's weekly note
- [ ] Command: Fetch last N days (batch backfill)
- [ ] Ribbon icon for one-click daily generation
- [ ] Notes written via `vault.create()` / `vault.modify()` — no file system paths

## Features (post-MVP)

- [ ] Auto-generate daily note on Obsidian startup
- [ ] Status bar showing today's recovery score
- [ ] Option A OAuth via `obsidian://` URI
- [ ] Persona / 30-day summary command
- [ ] Submit to Obsidian community plugin registry

---

## What Carries Over from obsidian-whoop-garden

| Asset | How it ports |
|---|---|
| WHOOP v2 API endpoint paths | Copied directly into `fetch.ts` |
| Pagination pattern (nextToken) | Direct translation to async/await |
| `models.go` structs | Rewritten as TypeScript interfaces in `models.ts` |
| Daily/weekly template content | Converted from `text/template` to template literals |
| `recoveryColor`, `strainCategory`, `sportName` helpers | Plain TS functions |
| HRV linear regression in `RenderPersonaSection` | Direct math port |

---

## Distribution

Obsidian plugins are distributed as a GitHub release containing:
- `main.js` (esbuild output)
- `manifest.json`
- `styles.css`

To list in the community registry, submit a PR to
`obsidianmd/obsidian-releases` once the plugin is stable.

---

## First Steps When Starting

1. Confirm WHOOP developer console redirect URI allowlist (http only vs custom schemes)
2. `npx create-obsidian-plugin` or clone the official sample plugin as scaffold
3. Port `models.ts` from `models.go` — mechanical but establishes the data shapes
4. Implement `client.ts` + `fetch.ts` with stub token
5. Implement daily note render → wire to `vault.create()`
6. Add settings tab + OAuth Option B flow
7. End-to-end test against real WHOOP account
