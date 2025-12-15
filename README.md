# auto-clip

A lightweight Node.js backend for creating short clips from YouTube URLs. The service downloads the provided video stream with [`ytdl-core`](https://github.com/fent/node-ytdl-core) and trims it with [`ffmpeg`](https://ffmpeg.org/) (using [`ffmpeg-static`](https://www.npmjs.com/package/ffmpeg-static) so no system install is required).

## Setup

Network access to npm is required to install dependencies.

```bash
npm install
```

## Run the service

```bash
npm run start
```

The server listens on port `3000` by default (override with the `PORT` environment variable).

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

- Temporary clip files are written to your system temp directory and removed after they are sent to the client.
- Ensure the host running the service has sufficient disk space for the temporary clip and outbound connectivity to YouTube.
