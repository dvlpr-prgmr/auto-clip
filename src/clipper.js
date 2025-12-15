const ytdl = require('ytdl-core');
const ffmpegPath = require('ffmpeg-static');
const ffmpeg = require('fluent-ffmpeg');
const fs = require('fs/promises');
const os = require('os');
const path = require('path');
const { randomUUID } = require('crypto');

ffmpeg.setFfmpegPath(ffmpegPath);

const clipOutputDir = path.join(os.tmpdir(), 'auto-clip');

async function ensureOutputDir() {
  await fs.mkdir(clipOutputDir, { recursive: true });
}

function validatePayload(payload) {
  if (!payload || typeof payload !== 'object') {
    return { error: 'Provide a JSON body with url, start, and end fields.' };
  }

  const { url, start, end } = payload;

  if (!url || !ytdl.validateURL(url)) {
    return { error: 'Provide a valid YouTube URL.' };
  }

  const numericStart = Number(start);
  const numericEnd = Number(end);

  if (!Number.isFinite(numericStart) || !Number.isFinite(numericEnd)) {
    return { error: 'Start and end must be numbers representing seconds.' };
  }

  if (numericStart < 0 || numericEnd <= numericStart) {
    return { error: 'End time must be greater than start time, and times must be non-negative.' };
  }

  return { value: { url, start: numericStart, end: numericEnd } };
}

async function renderClip(url, start, end) {
  await ensureOutputDir();
  const outputPath = path.join(clipOutputDir, `${randomUUID()}.mp4`);
  const duration = end - start;
  const ffmpegLog = [];
  const videoStream = ytdl(url, {
    filter: 'audioandvideo',
    quality: 'highestvideo',
  });

  try {
    await new Promise((resolve, reject) => {
      const handleError = (error) => {
        videoStream.destroy();
        const wrappedError = error instanceof Error ? error : new Error(String(error));
        wrappedError.ffmpegLog = ffmpegLog.slice(-8);
        reject(wrappedError);
      };

      videoStream.once('error', handleError);

      ffmpeg(videoStream)
        .setStartTime(start)
        .setDuration(duration)
        .outputOptions('-movflags', 'frag_keyframe+empty_moov')
        .format('mp4')
        .on('stderr', (line) => {
          const trimmed = (line || '').toString().trim();
          if (!trimmed) return;

          ffmpegLog.push(trimmed);
          if (ffmpegLog.length > 20) ffmpegLog.shift();
        })
        .save(outputPath)
        .once('error', handleError)
        .once('end', resolve);
    });

    return outputPath;
  } catch (error) {
    await fs.unlink(outputPath).catch(() => {});
    throw error;
  }
}

module.exports = {
  validatePayload,
  renderClip,
};
