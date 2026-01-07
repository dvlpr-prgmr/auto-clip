# Deployment Guide

## Frontend (Netlify)
1) Connect the repo to Netlify.
2) Build command: `npm run build` (or `yarn build`).
3) Publish directory: `dist`.
4) Set environment variable:
   - `VITE_API_BASE=https://api.yourdomain.com`
5) Optional: add SPA redirect if you add client-side routing.

## Backend (VPS)
### Prerequisites
- `ffmpeg` installed and on PATH.
- Go toolchain installed.

### Build + Run
1) Build the server:
   - `cd backend && go build -o server`
2) Create folders for output:
   - `mkdir -p /var/auto-clip/output/clips /var/auto-clip/output/thumbnails`
3) Create an environment file (example: `/etc/auto-clip.env`):
   - `PORT=8888`
   - `CLIP_OUTPUT_DIR=/var/auto-clip/output/clips`
   - `THUMBNAIL_OUTPUT_DIR=/var/auto-clip/output/thumbnails`
   - `YOUTUBE_COOKIE_FILE=/var/auto-clip/youtube.cookie.txt` (optional)
   - `YOUTUBE_PROXY=` (optional)
4) Run the server from `backend/` or point to the built binary.

### Reverse Proxy
Use Caddy or Nginx to proxy HTTPS traffic to `127.0.0.1:8888`.

### Notes
- Ensure the VPS firewall allows inbound 80/443.
- Update `VITE_API_BASE` to the public backend URL.
