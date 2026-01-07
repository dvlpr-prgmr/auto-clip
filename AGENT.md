# Agent Notes for auto-clip

## Document Sync Rules (Required)

### Documents that must be kept in sync

| Document | When to update | Contents |
|------|---------|------|
| `CHANGELOG.md` | After code changes | Update log (with timestamp), progress status |
| `AGENT.md` | When rules change | Project rules and workflow notes |

### Timestamp format

All update logs must include a timestamp:
```
### 2025-12-25 20:27 UTC+8
```

### Context docs

At the start of a new session, read:
1. `AGENT.md` - project rules
2. `PLAN.md` - development plan and progress

## Gambaran Umum
- Monorepo: frontend Vue 3 + Vite + Tailwind di root; backend Go HTTP server di `backend/`.
- Tujuan: membuat klip MP4 dari URL YouTube (POST `/api/clip`) dan health check (GET `/health`).

## Frontend (Vite/Vue)
- Entry: `index.html`, `src/main.js`, `src/App.vue`, `src/components/ClipForm.vue`.
- Konfigurasi: `vite.config.js` (port 5173), `tailwind.config.js`, `postcss.config.js`.
- Form mem-post ke `${VITE_API_BASE || window.location.origin}/api/clip`, lalu mendownload blob sebagai `clip.mp4`.
- Jalankan: `npm install` lalu `npm run dev` (atau `npm run build` untuk produksi). Tests frontend tidak ada.

## Backend (Go)
- Modul: `auto-clip-backend` (`go 1.25.4`).
- Entry: `backend/main.go` (router sederhana dengan CORS + logging).
- Handler: `backend/api/clip.go` menggunakan `github.com/kkdai/youtube/v2` untuk dapat stream URL, lalu `ffmpeg` untuk trim (`-ss`, `-t`, `libx264`, `aac`, `frag_keyframe+empty_moov`) dan streaming langsung ke response `video/mp4` download.
- Logging: `backend/logger/logger.go` menulis audit log harian markdown ke `backend/logs/`.
- Jalankan: dari folder `backend/`, set `PORT` (default 8888), pastikan `ffmpeg` ada di PATH. Log disimpan di `backend/logs/YYYY-MM-DD.md`.

## Lingkungan & Env
- `.env.local` contoh: `VITE_API_BASE`, `YOUTUBE_COOKIE`, `HTTP_PROXY`. Backend Go dapat dikonfigurasi untuk menggunakan proxy/cookie jika library `github.com/kkdai/youtube/v2` mendukungnya.
- Backend Go saat ini belum membaca header khusus untuk cookie YouTube secara otomatis.

## Testing
- Node: `npm test` menjalankan `node --test` terhadap sisa artefak lama; sesuaikan/bersihkan jika tidak relevan dengan stack Go + Vite.
- Go: belum ada test. Tambahkan bila perlu.

## Hal yang Perlu Diwaspadai
- Dependensi eksternal `ffmpeg` wajib tersedia di runtime backend Go.
- CORS di backend Go diizinkan untuk semua origin; sesuaikan untuk produksi.