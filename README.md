# auto-clip

Serverless Netlify Functions for creating short clips from YouTube URLs, plus a Vue 3 frontend. The function downloads the requested YouTube stream with [`ytdl-core`](https://github.com/fent/node-ytdl-core) and trims it with [`ffmpeg`](https://ffmpeg.org/) using the bundled [`ffmpeg-static`](https://www.npmjs.com/package/ffmpeg-static) binary.

## Backend (Netlify Functions)

Network access to npm is required to install dependencies. The functions run on Node 18+.

```bash
npm install
```

### Local development

Use the Netlify CLI to run the functions and frontend together:

```bash
npm install -g netlify-cli  # if you don't already have it
netlify dev --dir frontend --functions netlify/functions
```

The functions are available at `http://localhost:8888/.netlify/functions/*`, while the redirect in `netlify.toml` exposes them at `/api/clip` and `/health`.

### Deployment

Deploy directly to Netlify—the provided `netlify.toml` installs root dependencies for the functions, builds the Vue app, and publishes `frontend/dist` while bundling the functions from `netlify/functions/`.

### Testing

Run the backend unit tests with the Node.js test runner:

```bash
npm test
```

If npm cannot download dependencies (for example, `ffmpeg-static` returning a `403 Forbidden` from the registry), resolve the
network or proxy issue first and rerun `npm install` so the test runner can load the required modules.

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
- On failure, returns a JSON error payload with a relevant status code. Errors include `error` and `detail` fields plus recent
  ffmpeg log lines (when available) to help you debug rendering issues.

### `GET /health`
Basic health check endpoint that returns `{ "status": "ok" }`.

## Notes

- Temporary clip files are written to the function's temp directory and removed after they are sent to the client.
- The default setup uses the `ffmpeg-static` binary packaged with the function. To use a system-level ffmpeg instead, remove `ffmpeg-static` from `package.json` and ensure `ffmpeg` is available on the `PATH` during the build.
- Function invocations must stay within the Netlify function timeout and size limits.

## Frontend (Vue 3 + Vite + Tailwind)

The `frontend/` directory contains a simple UI for submitting clip requests.

### Setup and run

```bash
cd frontend
npm install
npm run dev
```

When served together with `netlify dev`, the UI calls the co-located functions at `/api/clip`. To target a different backend host, create a `.env.local` file in `frontend/` with:

```
VITE_API_BASE=https://your-backend-host
```

During development you can open the app at [http://localhost:5173](http://localhost:5173).

### Deploy to Netlify

Netlify hosts both the frontend and the serverless clipping backend. Use the default build settings from `netlify.toml` (build command `npm install && npm install --prefix frontend && npm run build --prefix frontend`, publish directory `frontend/dist`, functions in `netlify/functions`, and `included_files` to ship the `ffmpeg-static` binary).

If you want to call an external backend instead of the bundled functions, set `VITE_API_BASE` in the Netlify UI.
