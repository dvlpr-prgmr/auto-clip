# Changelog

### 2025-12-31 15:21 UTC+7

**Batch clip rendering**

- Backend: add `/api/clip/batch` for multi-segment renders with per-segment thumbnails.
- Backend: allow thumbnail generation at requested timestamps.
- Docs: mark Phase 1 backend tasks complete in `PLAN.md`.

### 2025-12-31 15:15 UTC+7

**Preview status details removed**

- Frontend: remove preview metadata line from the ready state.

### 2025-12-31 15:08 UTC+7

**Preview status badge styling**

- Frontend: adjust preview status badge sizing to match the reference layout.

### 2025-12-31 14:58 UTC+7

**Preview status badge wording**

- Frontend: show preview status only in the top-right badge with `ready/not ready` wording.

### 2025-12-31 14:46 UTC+7

**Add deployment guide**

- Docs: add `DEPLOY.md` with Netlify frontend and VPS backend steps.

### 2025-12-31 14:42 UTC+7

**Add development plan**

- Docs: add `PLAN.md` with phased roadmap and current progress checkboxes.

### 2025-12-31 14:20 UTC+7

**Segment delete icon**

- Frontend: replace the segment close icon with a trash icon.

### 2025-12-31 14:13 UTC+7

**Changelog ordering**

- Chore: reorder entries so newest changes appear first.

### 2025-12-31 14:11 UTC+7

**Consolidate logs**

- Chore: consolidate changes into `CHANGELOG.md` in the repo root.
- Docs: update `AGENT.md` to point logging to `CHANGELOG.md`.
- Chore: stop writing new entries to `logs/YYYY-MM-DD.md` and remove existing daily log files.

### 2025-12-31 14:08 UTC+7

**Agent log format guidance**

- Docs: update `AGENT.md` to require the new heading + summary + bullet logging style.

### 2025-12-31 14:02 UTC+7

**Action icons overflow**

- Frontend: let segment action icons overflow outside the card to avoid text wrapping.

### 2025-12-31 14:00 UTC+7

**Top-right icons restored**

- Frontend: move segment action icons back to the top-right corner while keeping the range/length line intact.

### 2025-12-31 13:59 UTC+7

**Range + length in one line**

- Frontend: show segment range and length in one line (for example, `0:00 - 4:43 (length 4:43)`).

### 2025-12-31 13:57 UTC+7

**Segment card layout tweak**

- Frontend: restructure segment cards into a three-column grid to keep length visible and align icons.

### 2025-12-31 13:54 UTC+7

**Avoid action overlap**

- Frontend: add right padding on segment cards so Length text does not overlap action icons.

### 2025-12-31 13:42 UTC+7

**Top-right action icons**

- Frontend: move edit/delete actions to top-right icon buttons (hover-only).

### 2025-12-31 13:41 UTC+7

**Hover-only segment actions**

- Frontend: show segment edit/delete controls only on hover/focus.

### 2025-12-31 12:00 UTC+7

**Segment selection + edit controls**

- Frontend: make segment cards selectable with click highlight.
- Frontend: add edit and delete controls for each segment.

### 2025-12-31 11:53 UTC+7

**Manual segment management**

- Frontend: enable manual segment management with Add Clip modal and Duplicate + shift action.

### 2025-12-31 11:50 UTC+7

**Autosplit estimate aligned**

- Frontend: align Auto-Split estimated clip count with the silence/max duration logic.

### 2025-12-31 11:46 UTC+7

**Auto-Split max duration clamped**

- Frontend: clamp Auto-Split max duration to the detected clip length and show the limit in the modal.

### 2025-12-31 11:14 UTC+7

**Clip filenames include YouTube ID**

- Backend: include the YouTube video ID in clip filenames when available (`clip-{id}-{unixnano}.mp4`).

### 2025-12-31 11:05 UTC+7

**Autosplit segments**

- Frontend: implement autosplit segment generation with per-segment thumbnails.
- Frontend: render the segments list in the queue UI.

### 2025-12-31 10:58 UTC+7

**Auto-fill end time**

- Frontend: auto-set clip end time to the detected video duration unless edited by the user.

### 2025-12-31 10:55 UTC+7

**Duration seconds displayed**

- Frontend: show duration seconds alongside the time label (for example, `4:40 (100s)`).

### 2025-12-31 10:54 UTC+7

**Remove duplicate summary cards**

- Frontend: remove duplicate summary cards under the preview panel.

### 2025-12-31 10:52 UTC+7

**Preview summary cards wired**

- Frontend: hook Duration/Resolution/Frame rate cards to preview metadata with loading fallbacks.

### 2025-12-31 10:49 UTC+7

**YouTube preview auto-detect**

- Backend: add `/api/preview` to fetch video metadata.
- Frontend: auto-fetch preview data when a URL is filled.

### 2025-12-31 10:43 UTC+7

**ClipForm preview + auto-split modal**

- Frontend: add source preview details and multi-segment queue styling.
- Frontend: introduce the Auto-Split settings modal.

### 2025-12-31 10:39 UTC+7

**UI overhaul**

- Frontend: redesign the clip form into an editor-style layout.
- Frontend: add the output settings panel.
- Frontend: update fonts to Space Grotesk + Fira Code and add fade-rise animations.

### 2025-12-31 10:14 UTC+7

**Ignore server binaries**

- Chore: ignore `server` binaries anywhere in the repo by adding `**/server` to `.gitignore`.

### 2025-12-31 10:08 UTC+7

**ClipForm payload rename**

- Frontend: rename response payload variable to avoid duplicate declaration.

### 2025-12-31 10:03 UTC+7

**Clip response saved to disk**

- Backend: stop streaming clip to the client; render to file and return JSON with output/thumbnail paths.
- Frontend: update copy to reflect server-side saving.

### 2025-12-29 16:50 UTC+7

**Postman proxy check**

- Postman: add `Proxy Check` request for `/proxy-check`.

### 2025-12-29 16:40 UTC+7

**Proxy check endpoint**

- Backend: add `/proxy-check` endpoint and refactor proxy/user-agent helpers.

### 2025-12-29 16:25 UTC+7

**Sanitized ffmpeg args**

- Backend: log sanitized ffmpeg arguments (redacts cookies, proxy credentials, stream URL tokens).

### 2025-12-29 16:15 UTC+7

**ffmpeg proxy support**

- Backend: add `-http_proxy` to ffmpeg args when `YOUTUBE_PROXY` is set.

### 2025-12-29 16:05 UTC+7

**Proxy settings logging**

- Backend: log proxy configuration per request with credential redaction.

### 2025-12-29 15:50 UTC+7

**Cookie file conversion**

- Backend: convert Netscape cookie file into a single-line Cookie header string; strip comment lines and `#HttpOnly_`.
- File: update `backend/youtube.cookie.txt` (values redacted).

### 2025-12-29 15:40 UTC+7

**Error caller location**

- Backend: include caller file + line for error-level logs.

### 2025-12-29 15:30 UTC+7

**Cookie file path fix**

- Backend: fix `YOUTUBE_COOKIE_FILE` path to `youtube.cookie.txt` (relative to backend).

### 2025-12-29 15:20 UTC+7

**Proxy fallback**

- Backend: allow fallback to `HTTP_PROXY` when proxying is enabled; `buildFFmpegEnv` keeps system proxy settings when allowed.
- Config: update `.env.local` with proxy defaults.

### 2025-12-29 15:00 UTC+7

**Auto-load env files**

- Backend: auto-load environment variables from `.env.local`/`.env` (root or backend).

### 2025-12-29 14:16 UTC+7

**Proxy defaults**

- Backend: disable proxies by default for YouTube fetch/ffmpeg; allow override via `YOUTUBE_PROXY` or `YOUTUBE_DISABLE_PROXY=false`.

### 2025-12-29 14:14 UTC+7

**YouTube headers and cookie file**

- Backend: inject configurable User-Agent and optional `YOUTUBE_COOKIE` into YouTube fetches and ffmpeg headers.
- Backend: allow loading YouTube cookie from `YOUTUBE_COOKIE_FILE`; update `.env.local` to point at the cookie file.

### 2025-12-29 09:55 UTC+7

**Thumbnail generation**

- Backend: generate a thumbnail JPEG from the saved clip using `THUMBNAIL_OUTPUT_DIR`.
- Config: add `THUMBNAIL_OUTPUT_DIR` to `.env.local`.

### 2025-12-29 09:43 UTC+7

**Add app favicon**

- Frontend: add `favicon.svg` referenced by `index.html`.

### 2025-12-29 09:42 UTC+7

**Server-side clip saving**

- Backend: save successful clip output to `output/clips` (`CLIP_OUTPUT_DIR`) while still streaming to the client.

### 2025-12-29 09:38 UTC+7

**YouTube cookie parsing**

- Backend: add Netscape cookie parser to convert cookie files into a Cookie header value when loading `YOUTUBE_COOKIE_FILE`.
