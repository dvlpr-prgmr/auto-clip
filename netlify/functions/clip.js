const fs = require('fs/promises');
const { validatePayload, renderClip } = require('../../src/clipper');

function jsonResponse(statusCode, payload) {
  return {
    statusCode,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  };
}

exports.handler = async (event) => {
  if (event.httpMethod !== 'POST') {
    return jsonResponse(405, { error: 'Method not allowed. Use POST.' });
  }

  let payload;
  try {
    payload = JSON.parse(event.body || '{}');
  } catch (error) {
    return jsonResponse(400, { error: 'Request body must be valid JSON.' });
  }

  const validation = validatePayload(payload);
  if (validation.error) {
    return jsonResponse(400, { error: validation.error });
  }

  const { url, start, end } = validation.value;
  let outputPath;

  try {
    outputPath = await renderClip(url, start, end);
    const buffer = await fs.readFile(outputPath);

    return {
      statusCode: 200,
      headers: {
        'Content-Type': 'video/mp4',
        'Content-Disposition': 'attachment; filename="clip.mp4"',
        'Cache-Control': 'no-store',
      },
      isBase64Encoded: true,
      body: buffer.toString('base64'),
    };
  } catch (error) {
    console.error('Failed to render clip', error);
    return jsonResponse(500, {
      error: 'Failed to render the clip.',
      detail: error.message,
      logs: Array.isArray(error.ffmpegLog) && error.ffmpegLog.length ? error.ffmpegLog : undefined,
    });
  } finally {
    if (outputPath) {
      await fs.unlink(outputPath).catch(() => {});
    }
  }
};
