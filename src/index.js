const express = require('express');
const ytdl = require('ytdl-core');
const ffmpegPath = require('ffmpeg-static');
const ffmpeg = require('fluent-ffmpeg');
const morgan = require('morgan');
const fs = require('fs');
const fsPromises = require('fs/promises');
const os = require('os');
const path = require('path');
const { randomUUID } = require('crypto');

ffmpeg.setFfmpegPath(ffmpegPath);

const app = express();
const PORT = process.env.PORT || 3000;
const clipOutputDir = path.join(os.tmpdir(), 'auto-clip');

app.use(express.json());
app.use(morgan('dev'));

async function ensureOutputDir() {
  await fsPromises.mkdir(clipOutputDir, { recursive: true });
}

function validateTimes(start, end) {
  if (!Number.isFinite(start) || !Number.isFinite(end)) {
    return 'Start and end must be numbers representing seconds.';
  }

  if (start < 0 || end <= start) {
    return 'End time must be greater than start time, and times must be non-negative.';
  }

  return null;
}

app.get('/health', (_req, res) => {
  res.json({ status: 'ok' });
});

app.post('/api/clip', async (req, res) => {
  const { url, start, end } = req.body || {};

  if (!url || !ytdl.validateURL(url)) {
    return res.status(400).json({ error: 'Provide a valid YouTube URL.' });
  }

  const validationError = validateTimes(start, end);
  if (validationError) {
    return res.status(400).json({ error: validationError });
  }

  const duration = end - start;
  await ensureOutputDir();

  const outputPath = path.join(clipOutputDir, `${randomUUID()}.mp4`);
  const videoStream = ytdl(url, {
    filter: 'audioandvideo',
    quality: 'highestvideo',
  });

  const cleanUp = () => fsPromises.unlink(outputPath).catch(() => {});

  videoStream.on('error', (error) => {
    cleanUp();
    if (!res.headersSent) {
      res.status(502).json({ error: 'Unable to download the YouTube stream.', detail: error.message });
    }
  });

  ffmpeg(videoStream)
    .setStartTime(start)
    .setDuration(duration)
    .outputOptions('-movflags', 'frag_keyframe+empty_moov')
    .format('mp4')
    .save(outputPath)
    .on('error', (error) => {
      cleanUp();
      if (!res.headersSent) {
        res.status(500).json({ error: 'Failed to render the clip.', detail: error.message });
      }
    })
    .on('end', () => {
      res.download(outputPath, 'clip.mp4', (error) => {
        cleanUp();
        if (error && !res.headersSent) {
          res.status(500).json({ error: 'Clip generated but failed to send to client.' });
        }
      });
    });
});

app.use((err, _req, res, _next) => {
  // Fallback error handler
  // eslint-disable-next-line no-console
  console.error(err);
  if (!res.headersSent) {
    res.status(500).json({ error: 'Unexpected server error.' });
  }
});

app.listen(PORT, () => {
  // eslint-disable-next-line no-console
  console.log(`Auto-clip backend listening on port ${PORT}`);
});
