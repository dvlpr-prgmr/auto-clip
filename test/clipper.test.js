const { test, describe } = require('node:test');
const assert = require('node:assert');
const { validatePayload } = require('../netlify/lib/clipper');

describe('validatePayload', () => {
  test('rejects missing payload', () => {
    assert.deepStrictEqual(validatePayload(), {
      error: 'Provide a JSON body with url, start, and end fields.',
    });
  });

  test('rejects invalid url', () => {
    assert.deepStrictEqual(
      validatePayload({ url: 'not-a-url', start: 0, end: 1 }),
      { error: 'Provide a valid YouTube URL.' }
    );
  });

  test('rejects non-numeric times', () => {
    assert.deepStrictEqual(
      validatePayload({ url: 'https://youtu.be/dQw4w9WgXcQ', start: 'a', end: 'b' }),
      { error: 'Start and end must be numbers representing seconds.' }
    );
  });

  test('rejects negative or reversed ranges', () => {
    assert.deepStrictEqual(
      validatePayload({ url: 'https://youtu.be/dQw4w9WgXcQ', start: -1, end: 1 }),
      {
        error:
          'End time must be greater than start time, and times must be non-negative.',
      }
    );

    assert.deepStrictEqual(
      validatePayload({ url: 'https://youtu.be/dQw4w9WgXcQ', start: 5, end: 4 }),
      {
        error:
          'End time must be greater than start time, and times must be non-negative.',
      }
    );
  });

  test('returns numeric values for valid payloads', () => {
    const result = validatePayload({
      url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
      start: '10',
      end: '15',
    });

    assert.strictEqual(result.error, undefined);
    assert.deepStrictEqual(result.value, {
      url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
      start: 10,
      end: 15,
    });
  });
});
