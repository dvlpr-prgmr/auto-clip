# Changelog

### 2026-01-08 10:05 UTC+7

**Backend auto-deploy workflow**

- CI: add GitHub Actions workflow to SSH deploy backend on `main`.
- Ops: add `scripts/deploy-backend.sh` for pull/build/restart via systemd.

### 2026-01-07 16:55 UTC+7

**Social upload UI triggers**

- Frontend: add per-segment upload button and status messages.
- Frontend: add social upload settings (auto upload, platform, visibility).
- Frontend: auto-upload rendered segments when enabled.

### 2026-01-07 16:42 UTC+7

**Scene detection logging**

- Backend: log scene detection requests with mode, range, and thresholds.

### 2026-01-07 16:38 UTC+7

**Social upload config tweaks**

- Config: add `YOUTUBE_CATEGORY_ID` placeholder for YouTube uploads.
- Backend: mark social upload success as completed rather than queued.

### 2026-01-07 16:37 UTC+7

**YouTube upload implementation**

- Backend: implement OAuth refresh + resumable upload for YouTube Shorts.
- Backend: build metadata (title/description/hashtags/tags) and return uploaded video ID.

### 2026-01-07 16:28 UTC+7

**Social upload env placeholders**

- Config: add social upload credential placeholders to `.env.local`.

### 2026-01-05 08:08 UTC+7

**Social upload API scaffold**

- Backend: add `/api/social/upload` endpoint with platform routing and dry-run support.
- Backend: add upload placeholders for YouTube Shorts, TikTok, and Instagram Reels.

### 2026-01-05 08:01 UTC+7

**Remove Netlify references**

- Frontend: drop Netlify-specific entrypoints, scripts, and keywords from `package.json`.

### 2026-01-04 15:50 UTC+7

**AI segment hashtags**

- Backend: add AI hashtags to scene segments and pass them through batch clip requests.
- Backend: update Gemini prompts to generate multiple hashtags per clip.
- Frontend: show hashtags on segment cards and include them in batch payloads.

### 2026-01-04 15:45 UTC+7

**Segment title clamping**

- Frontend: clamp AI segment titles to a single line using a local utility class.

### 2026-01-04 15:36 UTC+7

**AI segment titles**

- Backend: include AI-generated titles in scene segments and pass titles through batch clip requests.
- Backend: update Gemini prompts to output Indonesian clickbait-neutral titles.
- Frontend: display segment titles and include them in batch payloads.

### 2026-01-02 08:11 UTC+7

**AI multimodal audio sampling**

- Backend: include a short audio snippet alongside keyframes in Gemini multimodal analysis.
- Backend: add audio sample struct to autoscene models.

### 2026-01-02 08:06 UTC+7

**AI auto-split multimodal fallback**

- Backend: switch to Gemini multimodal (keyframe sampling) when transcripts are unavailable.
- Backend: add frame sampling helpers and multimodal analysis method in `autoscene`.

### 2026-01-01 20:32 UTC+7

**Autoscene client build fixes**

- Backend: align `autoscene` Gemini client with genai v1.40.0 field names and types.
- Backend: add `google.golang.org/genai` module dependency.

### 2026-01-01 20:24 UTC+7

**Autoscene Gemini client**

- Backend: route AI auto-split through `api/autoscene` Gemini client and parse MM:SS clip timecodes.
- Backend: add AI prompt options (max clips, duration hints, language) in autoscene client.

### 2026-01-01 19:35 UTC+7

**Fix YouTube caption parsing**

- Backend: handle `<timedtext>` root element in XML captions (format 3).
- Backend: parse nested `<s>` tags within paragraphs to correctly extract text.

### 2026-01-01 16:02 UTC+7

**Gemini auto-split AI**

- Backend: call Gemini API to propose viral segments from YouTube captions, with JSON parsing and constraints.
- Backend: add transcript prompt builder and helpers for AI scene detection.
- Frontend: send caption language with scene detection requests.

### 2026-01-01 15:12 UTC+7

**AI scene detection hook**

- Backend: allow `/api/scenes` to run AI scene detection via external command (`SCENE_AI_PATH`, `SCENE_AI_ARGS`).
- Backend: normalize AI segments and reuse the existing min/max/thumbnail logic.
- Frontend: add scene detection mode selector (standard vs AI) and pass mode to the scenes request.

### 2026-01-01 15:02 UTC+7

**Segment preview modal**

- Frontend: add per-segment play button and modal video preview for rendered clips.

### 2026-01-01 14:54 UTC+7

**Scene detection controls**

- Frontend: add sensitivity slider and min/max duration + max segments controls for scene detection.
- Backend: honor max duration and max segments when building scene-based segments.

### 2026-01-01 14:46 UTC+7

**Scene detection**

- Frontend: enable auto scene detection toggle and add scene detect controls.
- Backend: add `/api/scenes` to detect scene cuts and return segments with thumbnail timestamps.
- Docs: mark auto scene detection as complete in `PLAN.md`.

### 2026-01-01 14:34 UTC+7

**Subtitle safe margins**

- Backend: increase ASS subtitle margins and apply force_style margins for SRT to prevent clipping.

### 2026-01-01 14:30 UTC+7

**Default captions off**

- Frontend: disable captions by default in output settings.

### 2025-12-31 19:23 UTC+7

**Caption failure status**

- Frontend: surface caption-related errors as a separate status message.

### 2025-12-31 19:21 UTC+7

**Caption styling**

- Frontend: send font size + highlight color to backend caption renderer.
- Backend: apply font size and highlight color to ASS captions (karaoke/word_highlight).

### 2025-12-31 19:15 UTC+7

**Speech-to-text captions**

- Frontend: add caption source + language + subtitle upload controls for speech-to-text selection.
- Backend: integrate whisper.cpp transcription for per-segment subtitles.
- Docs: mark speech-to-text caption source as complete in `PLAN.md`.

### 2025-12-31 19:00 UTC+7

**Caption processing**

- Frontend: add caption source + language + subtitle upload controls and include caption content in payload.
- Backend: fetch YouTube captions or use uploaded subtitles, trim per segment, and burn subtitles into clips.
- Backend: generate karaoke/word_highlight as ASS with simple per-word timing.

### 2025-12-31 18:21 UTC+7

**Default URL update**

- Frontend: update the default YouTube URL in the clip form.

### 2025-12-31 18:16 UTC+7

**Caption settings wiring**

- Frontend: include captions toggle + caption style in output settings payload.
- Backend: accept caption settings in clip and batch requests.
- Docs: mark caption wiring complete in `PLAN.md`.

### 2025-12-31 18:08 UTC+7

**Aspect ratio fit mode**

- Frontend: add layout mode selector for aspect ratio framing.
- Backend: support fit mode with black bars (pad to 9:16 canvas).
- Docs: mark aspect ratio fit mode done in `PLAN.md`.

### 2025-12-31 18:02 UTC+7

**Aspect ratio options**

- Frontend: add aspect ratio selector shared across panels and include it in output settings payload.
- Backend: support 1:1 and 4:5 crop targets in vertical render filter.
- Docs: mark aspect ratio wiring complete in `PLAN.md`.

### 2025-12-31 17:57 UTC+7

**Vertical output crop**

- Backend: apply 9:16 scale + crop when output settings request vertical render.
- Backend: apply the same filter for batch thumbnails.
- Frontend: send output settings in thumbnail refresh payloads.

### 2025-12-31 17:46 UTC+7

**Output settings payload wiring**

- Frontend: share output settings state between panels and include preset + vertical/crop in clip payloads.
- Backend: accept `output_settings` in clip and batch requests (no render changes yet).
- Docs: mark preset + vertical/crop wiring complete in `PLAN.md`.

### 2025-12-31 17:44 UTC+7

**Plan cleanup**

- Docs: remove redundant Output Settings parent tasks from Phase 2.

### 2025-12-31 17:37 UTC+7

**Segment inline edits + reorder**

- Frontend: add inline start/end editing controls on segment cards.
- Frontend: add move up/down controls to reorder segments.
- Docs: mark segment reordering + inline edits as complete in `PLAN.md`.

### 2025-12-31 17:29 UTC+7

**Hide thumbnail path text**

- Frontend: remove per-segment thumbnail path line from the queue cards.

### 2025-12-31 17:17 UTC+7

**Plan: output settings wiring**

- Docs: break down Output Settings implementation tasks in Phase 2.

### 2025-12-31 17:13 UTC+7

**Plan updates**

- Docs: expand Phase 2 with output settings persistence.
- Docs: add Phase 5 automation + Telegram bot workflow.

### 2025-12-31 17:09 UTC+7

**Segment guardrails**

- Frontend: disable Generate clips when no segments exist and show helper text.
- Frontend: add Clear all segments button beside Manual queue.

### 2025-12-31 16:47 UTC+7

**Live thumbnail refresh**

- Backend: add `/api/thumbnail/batch` for per-segment thumbnail extraction without full renders.
- Frontend: refresh segment thumbnails after add/duplicate/auto-split actions with status feedback.

### 2025-12-31 16:40 UTC+7

**Media root alignment**

- Backend: resolve `/media/*` root from `THUMBNAIL_OUTPUT_DIR`/`CLIP_OUTPUT_DIR` so files under `../output` are served.

### 2025-12-31 15:56 UTC+7

**Serve rendered thumbnails**

- Backend: expose `/media/*` to serve files under `output/`.
- Frontend: map `thumbnail_path` to `/media/` so rendered segment thumbnails replace the preview image.

### 2025-12-31 15:38 UTC+7

**Batch render UI wiring**

- Frontend: send segment queues to `/api/clip/batch` with per-segment thumbnail timestamps.
- Frontend: show per-segment thumbnail timestamp + render status details in the queue.

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
