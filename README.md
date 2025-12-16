# auto-clip

Serverless Netlify Functions for creating short clips from YouTube URLs, plus a Vue 3 frontend (all in one project root). The function downloads the requested YouTube stream with [`ytdl-core`](https://github.com/fent/node-ytdl-core) and trims it with [`ffmpeg`](https://ffmpeg.org/) using the bundled [`ffmpeg-static`](https://www.npmjs.com/package/ffmpeg-static) binary.

## Quick start

- Prereq: Node 18+, npm, optional `netlify-cli` (`npm install -g netlify-cli`).
- Install deps once from the root: `npm install`
- Dev (Vite only): `npm run dev` → UI at `http://localhost:5173`
- Full stack (functions + UI via proxy): `netlify dev --dir . --functions netlify/functions`
- Build: `npm run build` (outputs `dist/`)
- Tests: `npm test` (runs `node --test`)

## Backend (Netlify Functions)

Network access to npm is required to install dependencies. The functions run on Node 18+.

```bash
npm install
```

### Local development

Use the Netlify CLI to run the functions and frontend together (both live in this root folder):

```bash
npm install -g netlify-cli  # if you don't already have it
npm install
netlify dev --dir . --functions netlify/functions
```

The functions are available at `http://localhost:8888/.netlify/functions/*`, while the redirect in `netlify.toml` exposes them at `/api/clip` and `/health`. The Vite dev server serves the UI at `http://localhost:5173`.

### Deployment

Deploy directly to Netlify—the provided `netlify.toml` builds the Vue app and publishes `dist`, while bundling the functions from `netlify/functions/`.

## API

### `POST /api/clip`
Create an MP4 clip from a YouTube video.

**Request body**

```json
{
  "url": "<YouTube URL>",
  "start": 30,
  "end": 45
}
```

- `url` – valid YouTube URL.
- `start` – clip start time in seconds (non-negative number).
- `end` – clip end time in seconds (must be greater than `start`).

**Response**

- On success, returns the clipped file as a download named `clip.mp4`.
- On failure, returns a JSON error payload with a relevant status code.

### `GET /health`
Basic health check endpoint that returns `{ "status": "ok" }`.

## Notes

- Temporary clip files are written to the function's temp directory and removed after they are sent to the client.
- The default setup uses the `ffmpeg-static` binary packaged with the function. To use a system-level ffmpeg instead, remove `ffmpeg-static` from `package.json` and ensure `ffmpeg` is available on the `PATH` during the build.
- Function invocations must stay within the Netlify function timeout and size limits.

## Frontend (Vue 3 + Vite + Tailwind)

The UI lives directly in the root (`index.html`, `src/`, and Vite/Tailwind configs).

### Setup and run

```bash
npm install
npm run dev     # or: netlify dev --dir . --functions netlify/functions
```

When served together with `netlify dev`, the UI calls the co-located functions at `/api/clip`. To target a different backend host, create a `.env.local` file in the project root with:

```
VITE_API_BASE=https://your-backend-host
```

During development you can open the app at [http://localhost:5173](http://localhost:5173).

### Deploy to Netlify

Netlify hosts both the frontend and the serverless clipping backend. Use the default build settings from `netlify.toml` (build command `npm run build`, publish directory `dist`, functions in `netlify/functions`).

If you want to call an external backend instead of the bundled functions, set `VITE_API_BASE` in the Netlify UI.
