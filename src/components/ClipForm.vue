<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { outputSettings } from '../stores/outputSettings.js';

const apiBase = computed(() => {
  const base = import.meta.env.VITE_API_BASE?.trim();
  if (base) return base.endsWith('/') ? base.slice(0, -1) : base;

  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin;
  }

  return 'http://localhost:8888'; // fallback for non-browser contexts
});

const form = reactive({
  url: 'https://www.youtube.com/watch?v=YRcHWQFrvWg',
  start: 0,
  end: 30,
});

const isSubmitting = ref(false);
const status = reactive({
  type: 'idle',
  message: '',
});
const result = reactive({
  outputPath: '',
  thumbnailPath: '',
  thumbnailError: '',
});
const batchSummary = reactive({
  total: 0,
  success: 0,
  failed: 0,
});
const captionStatus = reactive({
  state: 'idle',
  message: '',
});
const thumbnailStatus = reactive({
  state: 'idle',
  message: '',
});
const sceneStatus = reactive({
  state: 'idle',
  message: '',
});
const uploadStatus = reactive({
  state: 'idle',
  message: '',
});
const showSegmentPreview = ref(false);
const previewSegment = ref(null);
const preview = reactive({
  title: '',
  durationLabel: '',
  durationSeconds: 0,
  width: 0,
  height: 0,
  fps: 0,
  sizeLabel: '',
  thumbnailUrl: '',
  quality: '',
});
const previewStatus = reactive({
  state: 'idle',
  error: '',
});
const previewStatusLabel = computed(() => (previewStatus.state === 'ready' ? 'ready' : 'not ready'));
const previewStatusClass = computed(() => {
  if (previewStatus.state === 'ready') {
    return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200';
  }
  if (previewStatus.state === 'loading') {
    return 'border-sky-500/40 bg-sky-500/10 text-sky-200';
  }
  if (previewStatus.state === 'error') {
    return 'border-rose-500/40 bg-rose-500/10 text-rose-200';
  }
  return 'border-slate-800 bg-slate-900/60 text-slate-400';
});
const endTouched = ref(false);
const segmentDuration = computed(() => {
  const diff = Number(form.end) - Number(form.start);
  return Number.isFinite(diff) ? Math.max(diff, 0) : 0;
});
const isValid = computed(() => form.url.trim().length > 0 && segmentDuration.value > 0);
const maxClipDuration = computed(() => {
  if (segmentDuration.value > 0) {
    return Math.ceil(segmentDuration.value);
  }
  if (preview.durationSeconds > 0) {
    return Math.ceil(preview.durationSeconds);
  }
  return 0;
});
const showAutoSplit = ref(false);
const showAddClip = ref(false);
const addClipError = ref('');
const autoSplitMethods = [
  { value: 'duration', label: 'By duration (every N seconds)' },
  { value: 'count', label: 'By count (split into N clips)' },
  { value: 'silence', label: 'By silence detection' },
  { value: 'scene', label: 'By scene detection' },
];
const autoSplit = reactive({
  method: autoSplitMethods[0].value,
  interval: 60,
  count: 10,
  min: 60,
  max: 180,
  sceneThreshold: 0.4,
  sceneMinDuration: 2,
  sceneMaxDuration: 20,
  sceneMaxSegments: 60,
});
const manualClip = reactive({
  start: 0,
  end: 0,
});
const segments = ref([]);
const autoSplitApplied = ref(false);
const segmentsMode = ref('single');
const selectedSegmentId = ref('');
const editingSegmentId = ref('');
const inlineEditId = ref('');
const inlineEdit = reactive({
  start: 0,
  end: 0,
});
const inlineEditError = ref('');
const estimatedClips = computed(() => {
  if (segmentDuration.value <= 0) return 0;
  if (autoSplit.method === 'duration') {
    const interval = Math.max(Number(autoSplit.interval) || 1, 1);
    return Math.ceil(segmentDuration.value / interval);
  }
  if (autoSplit.method === 'count') {
    return Math.max(Number(autoSplit.count) || 0, 0);
  }
  if (autoSplit.method === 'scene') {
    return 0;
  }
  const maxInterval = Math.max(Number(autoSplit.max) || 1, 1);
  return Math.ceil(segmentDuration.value / maxInterval);
});
const isBatchMode = computed(() => segments.value.length > 1 || segmentsMode.value !== 'single');
const hasSegments = computed(() => segments.value.length > 0);
const outputRootKey = '/output/';
const outputRootPrefix = 'output/';

const defaultThumbnailAt = 0.2;
let thumbnailTimer = null;
let thumbnailRequestId = 0;
let thumbnailStatusTimer = null;

const sceneSensitivityLabel = computed(() => {
  const value = Number(autoSplit.sceneThreshold);
  if (!Number.isFinite(value)) return 'Medium';
  if (value <= 0.3) return 'Low';
  if (value <= 0.6) return 'Medium';
  return 'High';
});
const sceneModeLabel = computed(() => (outputSettings.sceneDetectionMode === 'ai' ? 'Gemini' : 'standard'));

function formatTime(value) {
  const seconds = Number(value);
  if (!Number.isFinite(seconds) || seconds < 0) {
    return '0:00';
  }
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
  }
  return `${minutes}:${String(secs).padStart(2, '0')}`;
}

function resolveThumbnailAt(duration, override) {
  const safeDuration = Number.isFinite(duration) ? Math.max(duration, 0) : 0;
  if (Number.isFinite(override) && override >= 0 && override <= safeDuration) {
    return override;
  }
  if (safeDuration <= 0) {
    return defaultThumbnailAt;
  }
  const preferred = Math.max(safeDuration * 0.2, defaultThumbnailAt);
  return Math.min(preferred, Math.max(safeDuration - 0.05, 0));
}

function resolveMediaUrl(filePath) {
  if (!filePath) return '';
  const normalized = String(filePath).replace(/\\/g, '/');
  const outputIndex = normalized.lastIndexOf(outputRootKey);
  if (outputIndex !== -1) {
    const relative = normalized.slice(outputIndex + outputRootKey.length);
    return `${apiBase.value}/media/${relative}`;
  }
  if (normalized.startsWith(outputRootPrefix)) {
    return `${apiBase.value}/media/${normalized.slice(outputRootPrefix.length)}`;
  }
  return '';
}

function resetThumbnailStatusSoon() {
  if (thumbnailStatusTimer) {
    clearTimeout(thumbnailStatusTimer);
  }
  thumbnailStatusTimer = setTimeout(() => {
    if (thumbnailStatus.state === 'success') {
      thumbnailStatus.state = 'idle';
      thumbnailStatus.message = '';
    }
  }, 1600);
}

function buildUploadPayload(segment) {
  const platform = outputSettings.socialUploadPlatform;
  const title = segment.title || `Segment ${String(segment.index).padStart(2, '0')}`;
  return {
    platform,
    video_path: segment.outputPath,
    title,
    description: '',
    hashtags: Array.isArray(segment.hashtags) ? segment.hashtags : [],
    visibility: outputSettings.socialUploadVisibility,
  };
}

async function uploadSegment(segment, { silent = false } = {}) {
  if (!segment || !segment.outputPath) return;
  if (!outputSettings.socialUploadPlatform) return;
  if (segment.uploadState === 'uploading') return;

  segment.uploadState = 'uploading';
  segment.uploadMessage = 'Uploading...';
  segment.uploadId = '';
  if (!silent) {
    uploadStatus.state = 'loading';
    uploadStatus.message = `Uploading ${segment.title || `Segment ${String(segment.index).padStart(2, '0')}`}...`;
  }

  try {
    const payload = buildUploadPayload(segment);
    const response = await fetch(`${apiBase.value}/api/social/upload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      let detail = 'Upload failed.';
      try {
        detail = await response.text();
      } catch (e) {
        // ignore
      }
      throw new Error(detail || 'Upload failed.');
    }

    const resultPayload = await response.json();
    segment.uploadState = 'success';
    segment.uploadId = resultPayload.request_id || '';
    segment.uploadMessage = resultPayload.request_id ? `Uploaded (${resultPayload.request_id})` : 'Uploaded';

    if (!silent) {
      uploadStatus.state = 'success';
      uploadStatus.message = 'Upload completed.';
    }
  } catch (error) {
    segment.uploadState = 'error';
    segment.uploadMessage = error.message || 'Upload failed.';
    if (!silent) {
      uploadStatus.state = 'error';
      uploadStatus.message = error.message || 'Upload failed.';
    }
  }
}

async function autoUploadRenderedSegments() {
  if (!outputSettings.socialUploadEnabled) return;
  if (!segments.value.length) return;

  uploadStatus.state = 'loading';
  uploadStatus.message = `Auto uploading to ${outputSettings.socialUploadPlatform}...`;

  let uploadedCount = 0;
  let failedCount = 0;
  for (const segment of segments.value) {
    if (!segment.outputPath || segment.renderError) {
      continue;
    }
    if (segment.uploadState === 'success') {
      continue;
    }
    await uploadSegment(segment, { silent: true });
    if (segment.uploadState === 'success') {
      uploadedCount += 1;
    } else if (segment.uploadState === 'error') {
      failedCount += 1;
    }
  }

  if (uploadedCount === 0 && failedCount === 0) {
    uploadStatus.state = 'idle';
    uploadStatus.message = '';
    return;
  }

  uploadStatus.state = failedCount > 0 ? 'error' : 'success';
  uploadStatus.message = `Auto upload done: ${uploadedCount} uploaded, ${failedCount} failed.`;
}

async function refreshSegmentThumbnails() {
  const payloadUrl = form.url.trim();
  if (!payloadUrl || !segments.value.length) return;

  thumbnailStatus.state = 'loading';
  thumbnailStatus.message = 'Updating thumbnails...';
  const requestId = ++thumbnailRequestId;

  try {
    const payload = {
      url: payloadUrl,
      segments: segments.value.map((segment) => buildSegmentPayload(segment)),
      output_settings: buildOutputSettingsPayload(),
    };
    const response = await fetch(`${apiBase.value}/api/thumbnail/batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      let detail = 'Failed to update thumbnails.';
      try {
        const errorPayload = await response.json();
        detail = errorPayload.error || errorPayload.detail || detail;
      } catch (e) {
        // If parsing fails, fall back to generic message.
      }
      throw new Error(detail);
    }

    const resultPayload = await response.json();
    if (requestId !== thumbnailRequestId) {
      return;
    }

    const results = Array.isArray(resultPayload.results) ? resultPayload.results : [];
    const resultMap = new Map(results.map((entry) => [entry.index, entry]));

    segments.value = reindexSegments(
      segments.value.map((segment) => {
        const entry = resultMap.get(segment.index);
        if (!entry) return segment;
        const resolvedThumbUrl = resolveMediaUrl(entry.thumbnail_path || '');
        return {
          ...segment,
          thumbnailPath: entry.thumbnail_path || '',
          thumbnailError: entry.thumbnail_error || '',
          thumbnailAt: Number.isFinite(entry.thumbnail_at) ? entry.thumbnail_at : segment.thumbnailAt,
          thumbnailUrl: resolvedThumbUrl || segment.thumbnailUrl || '',
        };
      }),
    );

    thumbnailStatus.state = 'success';
    thumbnailStatus.message = 'Thumbnails updated.';
    resetThumbnailStatusSoon();
  } catch (error) {
    if (requestId !== thumbnailRequestId) {
      return;
    }
    thumbnailStatus.state = 'error';
    thumbnailStatus.message = error.message || 'Failed to update thumbnails.';
  }
}

function scheduleThumbnailRefresh() {
  if (thumbnailTimer) {
    clearTimeout(thumbnailTimer);
  }
  thumbnailTimer = setTimeout(() => {
    refreshSegmentThumbnails();
  }, 500);
}

function closeAutoSplit() {
  showAutoSplit.value = false;
}

function openAddClip() {
  const range = getRange();
  const last = segments.value.length ? segments.value[segments.value.length - 1] : null;
  let start = last ? last.end : range.start;
  let end = last ? last.end + Math.max(last.duration, 1) : range.end;
  if (range.end > range.start && end > range.end) {
    end = range.end;
  }
  manualClip.start = Number(start.toFixed(1));
  manualClip.end = Number(Math.max(end, start + 1).toFixed(1));
  addClipError.value = '';
  editingSegmentId.value = '';
  showAddClip.value = true;
}

function closeAddClip() {
  showAddClip.value = false;
  addClipError.value = '';
  editingSegmentId.value = '';
}

function openEditSegment(segment) {
  if (!segment) return;
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  manualClip.start = Number(segment.start.toFixed(1));
  manualClip.end = Number(segment.end.toFixed(1));
  addClipError.value = '';
  editingSegmentId.value = segment.id;
  selectedSegmentId.value = segment.id;
  showAddClip.value = true;
}

function selectSegment(segment) {
  if (!segment) return;
  selectedSegmentId.value = segment.id;
}

function startInlineEdit(segment) {
  if (!segment) return;
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  inlineEdit.start = Number(segment.start.toFixed(1));
  inlineEdit.end = Number(segment.end.toFixed(1));
  inlineEditError.value = '';
  inlineEditId.value = segment.id;
  selectedSegmentId.value = segment.id;
}

function cancelInlineEdit() {
  inlineEditId.value = '';
  inlineEditError.value = '';
}

function applyInlineEdit(segment) {
  if (!segment || inlineEditId.value !== segment.id) return;
  const start = Number(inlineEdit.start);
  const end = Number(inlineEdit.end);
  if (!Number.isFinite(start) || !Number.isFinite(end)) {
    inlineEditError.value = 'Start/end must be valid numbers.';
    return;
  }
  if (start < 0) {
    inlineEditError.value = 'Start time must be >= 0.';
    return;
  }
  if (end <= start) {
    inlineEditError.value = 'End time must be greater than start time.';
    return;
  }

  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  segments.value = reindexSegments(
    segments.value.map((item) => {
      if (item.id !== segment.id) return item;
      const duration = Math.max(end - start, 0);
      return {
        ...item,
        start,
        end,
        duration,
        thumbnailAt: resolveThumbnailAt(duration, item.thumbnailAt),
        thumbnailUrl: preview.thumbnailUrl || item.thumbnailUrl || '',
        outputPath: '',
        thumbnailPath: '',
        thumbnailError: '',
        renderDuration: '',
        renderError: '',
      };
    }),
  );

  inlineEditId.value = '';
  inlineEditError.value = '';
  scheduleThumbnailRefresh();
}

function moveSegment(segment, offset) {
  if (!segment || !Number.isFinite(offset)) return;
  const index = segments.value.findIndex((item) => item.id === segment.id);
  if (index === -1) return;
  const target = index + offset;
  if (target < 0 || target >= segments.value.length) return;
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  const next = [...segments.value];
  const [moved] = next.splice(index, 1);
  next.splice(target, 0, moved);
  segments.value = reindexSegments(next);
  selectedSegmentId.value = segment.id;
}

function getRange() {
  const start = Math.max(Number(form.start) || 0, 0);
  const end = Math.max(Number(form.end) || 0, 0);
  return end > start ? { start, end } : { start, end: start };
}

function createSegment(start, end, index, thumbnailAt, title, hashtags) {
  const duration = Math.max(end - start, 0);
  return {
    id: `${index}-${start}-${end}`,
    index,
    start,
    end,
    duration,
    title: title || '',
    hashtags: Array.isArray(hashtags) ? [...hashtags] : [],
    uploadState: 'idle',
    uploadMessage: '',
    uploadId: '',
    thumbnailUrl: preview.thumbnailUrl || '',
    thumbnailAt: resolveThumbnailAt(duration, thumbnailAt),
    outputPath: '',
    thumbnailPath: '',
    thumbnailError: '',
    renderDuration: '',
    renderError: '',
  };
}

function reindexSegments(items) {
  return items.map((segment, idx) => ({
    ...segment,
    index: idx + 1,
    duration: Math.max(segment.end - segment.start, 0),
    thumbnailUrl: segment.thumbnailUrl || preview.thumbnailUrl || '',
    thumbnailAt: resolveThumbnailAt(Math.max(segment.end - segment.start, 0), segment.thumbnailAt),
    uploadState: segment.uploadState || 'idle',
    uploadMessage: segment.uploadMessage || '',
    uploadId: segment.uploadId || '',
  }));
}

function buildSegments(rangeStart, rangeEnd) {
  if (rangeEnd <= rangeStart) {
    return [];
  }

  if (autoSplit.method === 'count') {
    const count = Math.max(Number(autoSplit.count) || 1, 1);
    const length = (rangeEnd - rangeStart) / count;
    const items = [];
    for (let i = 0; i < count; i += 1) {
      const start = rangeStart + length * i;
      const end = i === count - 1 ? rangeEnd : rangeStart + length * (i + 1);
      items.push(createSegment(start, end, i + 1));
    }
    return items;
  }

  const intervalSource = autoSplit.method === 'silence' ? autoSplit.max : autoSplit.interval;
  const interval = Math.max(Number(intervalSource) || 1, 1);
  const segmentsList = [];
  let cursor = rangeStart;
  let index = 1;
  while (cursor < rangeEnd && index < 500) {
    const end = Math.min(cursor + interval, rangeEnd);
    segmentsList.push(createSegment(cursor, end, index));
    cursor = end;
    index += 1;
  }
  return segmentsList;
}

function refreshSegments() {
  const range = getRange();
  if (segmentsMode.value === 'auto') {
    segments.value = buildSegments(range.start, range.end);
  } else if (segmentsMode.value === 'manual') {
    segments.value = reindexSegments(segments.value);
  } else {
    segments.value = range.end > range.start ? [createSegment(range.start, range.end, 1)] : [];
  }

  if (selectedSegmentId.value && !segments.value.some((segment) => segment.id === selectedSegmentId.value)) {
    selectedSegmentId.value = '';
  }
}

async function detectScenes() {
  const range = getRange();
  const payloadUrl = form.url.trim();
  if (!outputSettings.autoSceneDetection) {
    sceneStatus.state = 'error';
    sceneStatus.message = 'Enable auto scene detection in Output Settings.';
    return;
  }
  if (!payloadUrl || range.end <= range.start) {
    sceneStatus.state = 'error';
    sceneStatus.message = 'Set a valid URL and range first.';
    return;
  }

  sceneStatus.state = 'loading';
  sceneStatus.message = `Detecting scenes (${sceneModeLabel.value})...`;

  try {
    const payload = {
      url: payloadUrl,
      start: range.start,
      end: range.end,
      threshold: Number(autoSplit.sceneThreshold) || 0.4,
      min_duration: Number(autoSplit.sceneMinDuration) || 0,
      max_duration: Number(autoSplit.sceneMaxDuration) || 0,
      max_segments: Number(autoSplit.sceneMaxSegments) || 0,
      mode: outputSettings.sceneDetectionMode,
      language: outputSettings.captionLanguage,
    };
    const response = await fetch(`${apiBase.value}/api/scenes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      let detail = 'Failed to detect scenes.';
      try {
        const errorPayload = await response.json();
        detail = errorPayload.error || errorPayload.detail || detail;
      } catch (e) {
        // Ignore JSON parse errors.
      }
      throw new Error(detail);
    }

    const resultPayload = await response.json();
    const segmentsPayload = Array.isArray(resultPayload.segments) ? resultPayload.segments : [];
    if (!segmentsPayload.length) {
      sceneStatus.state = 'error';
      sceneStatus.message = 'No scenes detected.';
      return;
    }

    segmentsMode.value = 'manual';
    autoSplitApplied.value = true;
    segments.value = reindexSegments(
      segmentsPayload.map((segment, index) =>
        createSegment(segment.start, segment.end, index + 1, segment.thumbnail_at, segment.title, segment.hashtags),
      ),
    );

    sceneStatus.state = 'success';
    sceneStatus.message = `Detected ${segmentsPayload.length} scenes.`;
    scheduleThumbnailRefresh();
  } catch (error) {
    sceneStatus.state = 'error';
    sceneStatus.message = error.message || 'Scene detection failed.';
  }
}

async function applyAutoSplit() {
  if (autoSplit.method === 'scene') {
    await detectScenes();
    if (sceneStatus.state === 'success') {
      showAutoSplit.value = false;
    }
    return;
  }

  segmentsMode.value = 'auto';
  autoSplitApplied.value = true;
  refreshSegments();
  showAutoSplit.value = false;
  scheduleThumbnailRefresh();
}

function addManualClip() {
  const start = Number(manualClip.start);
  const end = Number(manualClip.end);
  if (!Number.isFinite(start) || !Number.isFinite(end)) {
    addClipError.value = 'Start/end must be valid numbers.';
    return;
  }
  if (end <= start) {
    addClipError.value = 'End time must be greater than start time.';
    return;
  }

  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  if (editingSegmentId.value) {
    segments.value = reindexSegments(
      segments.value.map((segment) =>
        segment.id === editingSegmentId.value
          ? {
              ...segment,
              start,
              end,
              duration: Math.max(end - start, 0),
              thumbnailUrl: segment.thumbnailUrl || preview.thumbnailUrl || '',
            }
          : segment,
      ),
    );
    selectedSegmentId.value = editingSegmentId.value;
  } else {
    const nextIndex = segments.value.length + 1;
    const next = createSegment(start, end, nextIndex);
    segments.value = reindexSegments([...segments.value, next]);
    selectedSegmentId.value = next.id;
  }
  editingSegmentId.value = '';
  showAddClip.value = false;
  scheduleThumbnailRefresh();
}

function duplicateSegmentShift() {
  const range = getRange();
  const last = segments.value.length ? segments.value[segments.value.length - 1] : null;
  const base = last || createSegment(range.start, range.end, 1);
  const duration = Math.max(base.duration, 1);
  const start = base.end;
  let end = start + duration;
  if (range.end > range.start && end > range.end) {
    end = range.end;
  }
  if (end <= start) {
    addClipError.value = 'Cannot duplicate beyond the selected range.';
    return;
  }

  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  const next = createSegment(start, end, segments.value.length + 1);
  segments.value = reindexSegments([...segments.value, next]);
  selectedSegmentId.value = next.id;
  scheduleThumbnailRefresh();
}

function deleteSegment(segment) {
  if (!segment) return;
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  segments.value = reindexSegments(segments.value.filter((item) => item.id !== segment.id));
  if (selectedSegmentId.value === segment.id) {
    selectedSegmentId.value = '';
  }
  if (inlineEditId.value === segment.id) {
    cancelInlineEdit();
  }
  scheduleThumbnailRefresh();
}

function clearAllSegments() {
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  segments.value = [];
  selectedSegmentId.value = '';
  editingSegmentId.value = '';
  cancelInlineEdit();
}

const segmentPreviewUrl = computed(() => {
  if (!previewSegment.value) return '';
  return resolveMediaUrl(previewSegment.value.outputPath || '');
});

function openSegmentPreview(segment) {
  if (!segment || !segment.outputPath) return;
  previewSegment.value = segment;
  showSegmentPreview.value = true;
}

function closeSegmentPreview() {
  showSegmentPreview.value = false;
  previewSegment.value = null;
}

let previewTimer = null;
let previewRequestId = 0;
let suppressEndTouch = false;

function resetPreview() {
  preview.title = '';
  preview.durationLabel = '';
  preview.durationSeconds = 0;
  preview.width = 0;
  preview.height = 0;
  preview.fps = 0;
  preview.sizeLabel = '';
  preview.thumbnailUrl = '';
  preview.quality = '';
}

async function fetchPreview(url) {
  previewStatus.state = 'loading';
  previewStatus.error = '';
  const requestId = ++previewRequestId;

  try {
    const response = await fetch(`${apiBase.value}/api/preview?url=${encodeURIComponent(url)}`);
    if (!response.ok) {
      throw new Error('Failed to fetch preview');
    }
    const payload = await response.json();
    if (requestId !== previewRequestId) {
      return;
    }

    preview.title = payload.title || '';
    preview.durationLabel = payload.duration_label || '';
    preview.durationSeconds = payload.duration_seconds || 0;
    preview.width = payload.width || 0;
    preview.height = payload.height || 0;
    preview.fps = payload.fps || 0;
    preview.sizeLabel = payload.size_label || '';
    preview.thumbnailUrl = payload.thumbnail_url || '';
    preview.quality = payload.preview_quality || '';
    previewStatus.state = 'ready';

    if (preview.durationSeconds > 0 && !endTouched.value) {
      suppressEndTouch = true;
      form.end = preview.durationSeconds;
      setTimeout(() => {
        suppressEndTouch = false;
      }, 0);
    }
  } catch (error) {
    if (requestId !== previewRequestId) {
      return;
    }
    previewStatus.state = 'error';
    previewStatus.error = error.message || 'Preview unavailable';
    resetPreview();
  }
}

watch(
  () => form.end,
  (value, prev) => {
    if (suppressEndTouch) return;
    if (value !== prev) {
      endTouched.value = true;
    }
  },
);

watch(
  () => maxClipDuration.value,
  (value) => {
    if (!value || value <= 0) return;
    const limit = Math.max(value, 1);
    if (!autoSplit.max || autoSplit.max > limit) {
      autoSplit.max = limit;
    }
    if (autoSplit.min > limit) {
      autoSplit.min = limit;
    }
  },
  { immediate: true },
);

watch(
  [() => form.start, () => form.end],
  () => {
    refreshSegments();
  },
);

watch(
  () => preview.thumbnailUrl,
  () => {
    refreshSegments();
  },
);

watch(
  () => form.url,
  (value) => {
    const nextUrl = value.trim();
    if (previewTimer) {
      clearTimeout(previewTimer);
    }
    if (!nextUrl) {
      previewStatus.state = 'idle';
      previewStatus.error = '';
      resetPreview();
      return;
    }

    previewTimer = setTimeout(() => {
      fetchPreview(nextUrl);
    }, 600);
  },
  { immediate: true },
);

watch(
  () => outputSettings.autoSceneDetection,
  (enabled) => {
    if (!enabled && autoSplit.method === 'scene') {
      autoSplit.method = 'duration';
    }
  },
);

refreshSegments();

function clearSegmentRenderData() {
  segments.value = segments.value.map((segment) => ({
    ...segment,
    outputPath: '',
    thumbnailPath: '',
    thumbnailError: '',
    renderDuration: '',
    renderError: '',
    uploadState: 'idle',
    uploadMessage: '',
    uploadId: '',
  }));
}

function buildSegmentPayload(segment) {
  const start = Number(segment.start);
  const end = Number(segment.end);
  const duration = Number.isFinite(end - start) ? Math.max(end - start, 0) : 0;
  return {
    start,
    end,
    title: segment.title || '',
    hashtags: Array.isArray(segment.hashtags) ? segment.hashtags : [],
    thumbnail_at: resolveThumbnailAt(duration, segment.thumbnailAt),
  };
}

function buildOutputSettingsPayload() {
  return {
    preset: outputSettings.preset,
    vertical: outputSettings.vertical,
    aspect_ratio: outputSettings.aspectRatio,
    aspect_mode: outputSettings.aspectMode,
    crop_method: outputSettings.cropMethod,
    captions: outputSettings.captions,
    caption_style: outputSettings.captionStyle,
    caption_source: outputSettings.captionSource,
    caption_language: outputSettings.captionLanguage,
    caption_format: outputSettings.captionFormat,
    caption_text: outputSettings.captionText,
    font_size: outputSettings.fontSize,
    highlight_color: outputSettings.highlight,
  };
}

async function handleSubmit() {
  status.type = 'idle';
  status.message = '';
  result.outputPath = '';
  result.thumbnailPath = '';
  result.thumbnailError = '';
  batchSummary.total = 0;
  batchSummary.success = 0;
  batchSummary.failed = 0;
  captionStatus.state = 'idle';
  captionStatus.message = '';
  uploadStatus.state = 'idle';
  uploadStatus.message = '';
  clearSegmentRenderData();
  isSubmitting.value = true;

  try {
    const payloadUrl = form.url.trim();
    const shouldBatch = isBatchMode.value;

    if (shouldBatch) {
      const payload = {
        url: payloadUrl,
        segments: segments.value.map((segment) => buildSegmentPayload(segment)),
        output_settings: buildOutputSettingsPayload(),
      };

      const response = await fetch(`${apiBase.value}/api/clip/batch`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        let detail = 'Failed to generate clips.';
        try {
          const errorPayload = await response.json();
          detail = errorPayload.error || errorPayload.detail || detail;
        } catch (e) {
          // If parsing fails, fall back to generic message.
        }
        if (outputSettings.captions) {
          captionStatus.state = 'error';
          captionStatus.message = detail;
        }
        throw new Error(detail);
      }

      const resultPayload = await response.json();
      const results = Array.isArray(resultPayload.results) ? resultPayload.results : [];
      const resultMap = new Map(results.map((entry) => [entry.index, entry]));

      batchSummary.total = Number(resultPayload.total) || results.length;
      batchSummary.success = Number(resultPayload.success) || 0;
      batchSummary.failed = Number(resultPayload.failed) || 0;

      segments.value = reindexSegments(
        segments.value.map((segment) => {
          const entry = resultMap.get(segment.index);
          if (!entry) return segment;
          const resolvedThumbUrl = resolveMediaUrl(entry.thumbnail_path || '');
          return {
            ...segment,
            outputPath: entry.output_path || '',
            thumbnailPath: entry.thumbnail_path || '',
            thumbnailError: entry.thumbnail_error || '',
            renderDuration: entry.render_duration || '',
            renderError: entry.error || '',
            thumbnailAt: Number.isFinite(entry.thumbnail_at) ? entry.thumbnail_at : segment.thumbnailAt,
            thumbnailUrl: resolvedThumbUrl || segment.thumbnailUrl || '',
          };
        }),
      );

      status.type = batchSummary.failed > 0 ? 'error' : 'success';
      status.message = `Batch done: ${batchSummary.success} succeeded, ${batchSummary.failed} failed.`;
      await autoUploadRenderedSegments();
    } else {
      const payload = {
        url: payloadUrl,
        start: Number(form.start),
        end: Number(form.end),
        output_settings: buildOutputSettingsPayload(),
      };

      const response = await fetch(`${apiBase.value}/api/clip`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        let detail = 'Failed to generate clip.';
        try {
          const errorPayload = await response.json();
          detail = errorPayload.error || errorPayload.detail || detail;
        } catch (e) {
          // If parsing fails, fall back to generic message.
        }
        if (outputSettings.captions) {
          captionStatus.state = 'error';
          captionStatus.message = detail;
        }
        throw new Error(detail);
      }

      const resultPayload = await response.json();
      result.outputPath = resultPayload.output_path || '';
      result.thumbnailPath = resultPayload.thumbnail_path || '';
      result.thumbnailError = resultPayload.thumbnail_error || '';

      status.type = 'success';
      status.message = 'Clip saved on the server.';
      if (outputSettings.socialUploadEnabled && result.outputPath) {
        const title = preview.title || 'Segment 01';
        await uploadSegment({
          index: 1,
          title,
          outputPath: result.outputPath,
          hashtags: [],
        }, { silent: false });
      }
    }
  } catch (error) {
    status.type = 'error';
    status.message = error.message || 'Unexpected error generating clip.';
    if (outputSettings.captions && captionStatus.state !== 'error') {
      captionStatus.state = 'error';
      captionStatus.message = error.message || 'Caption processing failed.';
    }
  } finally {
    isSubmitting.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <form @submit.prevent="handleSubmit" class="flex flex-col gap-6">
      <section class="rounded-2xl border border-slate-800/70 bg-[#0f1724]/80 p-5 shadow-xl shadow-black/20">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Source Video</p>
            <h2 class="text-lg font-semibold text-slate-100">Add your input</h2>
          </div>
          <div class="flex items-center gap-2 rounded-full border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-300">
            <span class="h-2 w-2 rounded-full bg-emerald-400"></span>
            API: <code class="font-['Fira_Code']">{{ apiBase }}</code>
          </div>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-3">
          <button type="button" class="rounded-lg border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-xs font-semibold text-emerald-100">
            YouTube URL
          </button>
          <button type="button" disabled class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-500">
            Local file (soon)
          </button>
        </div>

        <div class="mt-4 flex flex-col gap-2">
          <label class="text-sm font-semibold text-slate-200">YouTube URL</label>
          <input
            v-model="form.url"
            type="url"
            required
            placeholder="https://www.youtube.com/watch?v=..."
            class="w-full rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-emerald-400 focus:ring-emerald-400"
          />
        </div>

        <div class="mt-4 rounded-xl border border-slate-800 bg-[#0c141f] p-4">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <p class="text-base font-semibold leading-snug text-slate-100 md:text-lg">
                {{ preview.title || 'Waiting for URL...' }}
              </p>
              <p class="text-xs text-slate-400 md:text-sm">
                <template v-if="previewStatus.state === 'loading'">
                  Detecting video metadata...
                </template>
                <template v-else-if="previewStatus.state === 'error'">
                  {{ previewStatus.error }}
                </template>
                <template v-else-if="previewStatus.state === 'idle'">
                  Paste a YouTube URL to auto-detect duration and resolution.
                </template>
              </p>
            </div>
            <span
              class="self-start rounded-full border px-5 py-2 text-[11px] font-semibold uppercase tracking-[0.32em]"
              :class="previewStatusClass"
            >
              {{ previewStatusLabel }}
            </span>
          </div>
          <div class="mt-3 flex flex-wrap gap-2">
            <span class="rounded-lg bg-emerald-500/10 px-3 py-1 text-xs text-emerald-200">
              {{ preview.durationLabel || '0:00' }}{{ preview.durationSeconds ? ` (${preview.durationSeconds}s)` : '' }}
            </span>
            <span class="rounded-lg bg-sky-500/10 px-3 py-1 text-xs text-sky-200">
              {{ preview.width && preview.height ? `${preview.width}x${preview.height}` : '—' }}
            </span>
            <span class="rounded-lg bg-amber-500/10 px-3 py-1 text-xs text-amber-200">
              {{ preview.fps ? `${preview.fps} fps` : '—' }}
            </span>
            <span class="rounded-lg bg-slate-800/70 px-3 py-1 text-xs text-slate-200">
              {{ preview.quality ? preview.quality : 'MP4 + AAC' }}
            </span>
            <span class="rounded-lg bg-slate-800/70 px-3 py-1 text-xs text-slate-200">
              {{ preview.sizeLabel || 'Size —' }}
            </span>
          </div>
          <div v-if="preview.thumbnailUrl" class="mt-4 overflow-hidden rounded-lg border border-slate-800">
            <img :src="preview.thumbnailUrl" alt="Preview thumbnail" class="h-40 w-full object-cover" />
          </div>
        </div>

      </section>

      <section class="rounded-2xl border border-slate-800/70 bg-[#0f1724]/80 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Clip Range</p>
            <h3 class="text-lg font-semibold text-slate-100">Set in and out points</h3>
          </div>
          <div class="text-xs text-slate-400">
            Est. duration <span class="font-semibold text-slate-200">{{ formatTime(segmentDuration) }}</span>
          </div>
        </div>

        <div class="mt-4 grid gap-4 md:grid-cols-2">
          <div class="flex flex-col gap-2">
            <label class="text-sm font-semibold text-slate-200">Start time (seconds)</label>
            <input
              v-model.number="form.start"
              type="number"
              min="0"
              step="0.1"
              class="w-full rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-emerald-400 focus:ring-emerald-400"
            />
          </div>
          <div class="flex flex-col gap-2">
            <label class="text-sm font-semibold text-slate-200">End time (seconds)</label>
            <input
              v-model.number="form.end"
              type="number"
              min="0"
              step="0.1"
              class="w-full rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-emerald-400 focus:ring-emerald-400"
            />
          </div>
        </div>

        <div class="mt-4 rounded-xl border border-slate-800 bg-[#0c141f] p-3">
          <div class="flex items-center justify-between text-[11px] text-slate-400">
            <span>0:00</span>
            <span>{{ formatTime(form.start) }}</span>
            <span>{{ formatTime(form.end) }}</span>
          </div>
          <div class="mt-2 h-2 w-full overflow-hidden rounded-full bg-slate-800/80">
            <div class="h-full w-full bg-[linear-gradient(90deg,_#1dd7a1,_#4aa3ff,_#f2c14e)]"></div>
          </div>
          <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400">
            <span>Segment length: <span class="font-semibold text-slate-200">{{ segmentDuration.toFixed(1) }}s</span></span>
            <span>Start must be >= 0 and end must be greater than start.</span>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-800/70 bg-[#0f1724]/80 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Clip Segments</p>
            <h3 class="text-lg font-semibold text-slate-100">Queue</h3>
          </div>
          <span class="text-xs text-slate-400">{{ segments.length }} segment{{ segments.length === 1 ? '' : 's' }}</span>
        </div>

        <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-slate-400">
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 transition hover:border-emerald-400/60 hover:text-emerald-200"
              @click="openAddClip"
            >
              Add clip
            </button>
            <button
              type="button"
              class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 transition hover:border-emerald-400/60 hover:text-emerald-200"
              @click="duplicateSegmentShift"
            >
              Duplicate + shift
            </button>
            <button
              type="button"
              class="rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-200 transition hover:border-emerald-400 hover:text-emerald-100"
              @click="showAutoSplit = true"
            >
              Auto-split
            </button>
            <button
              type="button"
              class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 transition hover:border-emerald-400/60 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!outputSettings.autoSceneDetection || !isValid"
              @click="detectScenes"
            >
              Scene detect
            </button>
          </div>
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-200 transition hover:border-rose-400/60 hover:text-rose-200 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!hasSegments"
              @click="clearAllSegments"
            >
              Clear all segments
            </button>
            <span class="rounded-full border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-400">Manual queue</span>
          </div>
        </div>
        <p
          v-if="thumbnailStatus.message"
          class="mt-2 text-[11px]"
          :class="thumbnailStatus.state === 'error'
            ? 'text-rose-300'
            : thumbnailStatus.state === 'loading'
              ? 'text-sky-200'
              : 'text-emerald-200'"
        >
          {{ thumbnailStatus.message }}
        </p>
        <p
          v-if="sceneStatus.message"
          class="mt-2 text-[11px]"
          :class="sceneStatus.state === 'error'
            ? 'text-rose-300'
            : sceneStatus.state === 'loading'
              ? 'text-sky-200'
              : 'text-emerald-200'"
        >
          {{ sceneStatus.message }}
        </p>

        <div class="mt-4 space-y-3">
          <div v-if="segments.length" class="grid gap-3 md:grid-cols-2">
            <div
              v-for="segment in segments"
              :key="segment.id"
              class="group relative flex items-center gap-4 overflow-visible rounded-xl border border-slate-800 bg-slate-950/40 p-4 transition hover:border-emerald-500/30 hover:bg-emerald-500/5"
              :class="segment.id === selectedSegmentId ? 'border-emerald-500/50 bg-emerald-500/10' : ''"
              @click="selectSegment(segment)"
            >
              <div class="flex items-center gap-3">
                <div class="h-12 w-20 overflow-hidden rounded-lg border border-slate-800 bg-slate-900/70">
                  <img
                    v-if="segment.thumbnailUrl"
                    :src="segment.thumbnailUrl"
                    alt="Segment thumbnail"
                    class="h-full w-full object-cover"
                  />
                  <div v-else class="flex h-full w-full items-center justify-center text-[10px] text-slate-500">
                    No thumb
                  </div>
                </div>
                <div>
                  <p class="text-sm font-semibold text-slate-100 line-clamp-1">
                    {{ segment.title || `Segment ${String(segment.index).padStart(2, '0')}` }}
                  </p>
                  <p v-if="segment.title" class="text-[11px] text-slate-400">
                    Segment {{ String(segment.index).padStart(2, '0') }}
                  </p>
                  <p v-if="segment.hashtags && segment.hashtags.length" class="text-[11px] text-slate-400 line-clamp-1">
                    {{ segment.hashtags.join(' ') }}
                  </p>
                  <p v-if="segment.uploadMessage" class="text-[10px] text-slate-400">
                    Upload: {{ segment.uploadMessage }}
                  </p>
                  <p class="text-xs text-slate-400">
                    {{ formatTime(segment.start) }} - {{ formatTime(segment.end) }} (length {{ formatTime(segment.duration) }})
                  </p>
                  <p class="text-[11px] text-slate-500">
                    Thumb @ {{ formatTime(segment.thumbnailAt || 0) }}
                    <template v-if="segment.renderError"> · Render failed</template>
                    <template v-else-if="segment.outputPath"> · Rendered</template>
                    <template v-else> · Pending</template>
                  </p>
                  <p v-if="segment.renderError" class="text-[10px] text-rose-300 break-all">
                    Render error: {{ segment.renderError }}
                  </p>
                  <p v-if="segment.thumbnailError" class="text-[10px] text-amber-300 break-all">
                    Thumb error: {{ segment.thumbnailError }}
                  </p>
                  <div v-if="inlineEditId === segment.id" class="mt-2 flex flex-wrap items-center gap-2 text-[11px] text-slate-300">
                    <label class="flex items-center gap-2 rounded-md border border-slate-800 bg-slate-950/40 px-2 py-1">
                      Start
                      <input
                        v-model.number="inlineEdit.start"
                        type="number"
                        min="0"
                        step="0.1"
                        class="w-20 rounded border border-slate-700 bg-slate-950 px-2 py-0.5 text-[11px] text-slate-100"
                      />
                    </label>
                    <label class="flex items-center gap-2 rounded-md border border-slate-800 bg-slate-950/40 px-2 py-1">
                      End
                      <input
                        v-model.number="inlineEdit.end"
                        type="number"
                        min="0"
                        step="0.1"
                        class="w-20 rounded border border-slate-700 bg-slate-950 px-2 py-0.5 text-[11px] text-slate-100"
                      />
                    </label>
                    <button
                      type="button"
                      class="rounded-md border border-emerald-500/50 bg-emerald-500/10 px-2 py-1 text-[11px] text-emerald-200 transition hover:border-emerald-400 hover:text-emerald-100"
                      @click.stop="applyInlineEdit(segment)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="rounded-md border border-slate-800 bg-slate-900/60 px-2 py-1 text-[11px] text-slate-300 transition hover:border-slate-600 hover:text-slate-200"
                      @click.stop="cancelInlineEdit"
                    >
                      Cancel
                    </button>
                  </div>
                  <p v-if="inlineEditId === segment.id && inlineEditError" class="text-[10px] text-rose-300">
                    {{ inlineEditError }}
                  </p>
                </div>
              </div>
              <div class="absolute -right-3 -top-3 flex items-center gap-2 opacity-0 transition group-hover:opacity-100 group-focus-within:opacity-100">
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="!segment.outputPath"
                  @click.stop="openSegmentPreview(segment)"
                  aria-label="Preview segment"
                >
                  ▶
                </button>
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="!segment.outputPath || segment.uploadState === 'uploading'"
                  @click.stop="uploadSegment(segment)"
                  aria-label="Upload segment"
                >
                  ⤴
                </button>
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200"
                  @click.stop="startInlineEdit(segment)"
                  aria-label="Edit segment"
                >
                  ✎
                </button>
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="segment.index === 1"
                  @click.stop="moveSegment(segment, -1)"
                  aria-label="Move segment up"
                >
                  ▲
                </button>
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="segment.index === segments.length"
                  @click.stop="moveSegment(segment, 1)"
                  aria-label="Move segment down"
                >
                  ▼
                </button>
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-rose-400 hover:text-rose-200"
                  @click.stop="deleteSegment(segment)"
                  aria-label="Delete segment"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4">
                    <path
                      fill="currentColor"
                      d="M9 3h6l1 2h4v2H4V5h4l1-2zm1 6h2v9h-2V9zm4 0h2v9h-2V9zM7 9h2v9H7V9z"
                    />
                  </svg>
                </button>
              </div>
            </div>
          </div>
          <div v-else class="rounded-xl border border-slate-800 bg-slate-950/40 p-4 text-xs text-slate-400">
            No segments yet. Add a valid start/end or run auto-split.
          </div>
        </div>
      </section>

      <div class="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-slate-800/70 bg-[#0f1724]/80 p-4">
        <div class="text-xs text-slate-400">
          <p class="text-sm font-semibold text-slate-100">Ready to render</p>
          <p>Clip is saved on the server output folder.</p>
        </div>
        <div class="flex flex-col items-end gap-1 text-xs text-slate-400">
          <button
            type="submit"
            :disabled="isSubmitting || !isValid || !hasSegments"
            class="inline-flex items-center justify-center rounded-xl bg-emerald-400 px-5 py-2 text-sm font-semibold text-slate-950 transition hover:bg-emerald-300 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <svg
              v-if="isSubmitting"
              class="mr-2 h-4 w-4 animate-spin text-slate-900"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ isSubmitting ? (isBatchMode ? 'Rendering clips...' : 'Rendering...') : (isBatchMode ? 'Generate clips' : 'Generate clip') }}
          </button>
          <span v-if="!hasSegments" class="text-[11px] text-slate-500">Tambahkan minimal satu segmen untuk render.</span>
        </div>
      </div>
    </form>

    <div v-if="status.message" class="flex items-center gap-3 rounded-lg border p-3 text-sm" :class="{
      'border-sky-500/60 bg-sky-500/10 text-sky-100': status.type === 'success',
      'border-rose-500/60 bg-rose-500/10 text-rose-100': status.type === 'error',
      'border-slate-800 bg-slate-900 text-slate-200': status.type === 'idle',
    }">
      <span v-if="status.type === 'success'" aria-hidden="true">✅</span>
      <span v-else-if="status.type === 'error'" aria-hidden="true">⚠️</span>
      <span v-else aria-hidden="true">ℹ️</span>
      <span>{{ status.message }}</span>
    </div>

    <div v-if="uploadStatus.message" class="flex items-center gap-3 rounded-lg border p-3 text-xs" :class="{
      'border-emerald-500/60 bg-emerald-500/10 text-emerald-100': uploadStatus.state === 'success',
      'border-rose-500/60 bg-rose-500/10 text-rose-100': uploadStatus.state === 'error',
      'border-sky-500/60 bg-sky-500/10 text-sky-100': uploadStatus.state === 'loading',
      'border-slate-800 bg-slate-900 text-slate-200': uploadStatus.state === 'idle',
    }">
      <span aria-hidden="true">🚀</span>
      <span>{{ uploadStatus.message }}</span>
    </div>

    <div
      v-if="captionStatus.message"
      class="flex items-center gap-3 rounded-lg border p-3 text-xs"
      :class="captionStatus.state === 'error'
        ? 'border-amber-500/60 bg-amber-500/10 text-amber-100'
        : 'border-slate-800 bg-slate-900 text-slate-200'"
    >
      <span aria-hidden="true">💬</span>
      <span>Captions: {{ captionStatus.message }}</span>
    </div>

    <div v-if="result.outputPath" class="rounded-lg border border-slate-800 bg-slate-950/60 p-3 text-xs text-slate-300">
      <p class="text-slate-200">
        Clip saved at <code>{{ result.outputPath }}</code>
      </p>
      <p v-if="result.thumbnailPath" class="mt-1">
        Thumbnail saved at <code>{{ result.thumbnailPath }}</code>
      </p>
      <p v-else-if="result.thumbnailError" class="mt-1 text-amber-300">
        Thumbnail failed: {{ result.thumbnailError }}
      </p>
    </div>

    <div
      v-if="showSegmentPreview"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4 py-6 backdrop-blur"
      @click.self="closeSegmentPreview"
    >
      <div class="w-full max-w-3xl rounded-2xl border border-slate-800 bg-[#0f1724] p-6 shadow-2xl shadow-black/40 animate-fade-rise">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Segment Preview</p>
            <h3 class="text-lg font-semibold text-slate-100">
              {{ previewSegment ? `Segment ${String(previewSegment.index).padStart(2, '0')}` : 'Segment' }}
            </h3>
          </div>
          <button type="button" class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-300" @click="closeSegmentPreview">
            Close
          </button>
        </div>

        <div class="mt-4">
          <video
            v-if="segmentPreviewUrl"
            :src="segmentPreviewUrl"
            controls
            playsinline
            class="w-full rounded-xl border border-slate-800 bg-slate-950"
          ></video>
          <p v-else class="text-xs text-slate-400">Preview not available for this segment yet.</p>
        </div>
      </div>
    </div>

    <div
      v-if="showAddClip"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4 py-6 backdrop-blur"
      @click.self="closeAddClip"
    >
      <div class="w-full max-w-md rounded-2xl border border-slate-800 bg-[#0f1724] p-6 shadow-2xl shadow-black/40 animate-fade-rise">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Manual Clip</p>
            <h3 class="text-lg font-semibold text-slate-100">
              {{ editingSegmentId ? 'Edit segment' : 'Add a segment' }}
            </h3>
          </div>
          <button type="button" class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-300" @click="closeAddClip">
            Close
          </button>
        </div>

        <div class="mt-4 grid gap-3 text-xs text-slate-300">
          <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
            Start (sec)
            <input
              v-model.number="manualClip.start"
              type="number"
              min="0"
              step="0.1"
              class="w-24 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
            />
          </label>
          <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
            End (sec)
            <input
              v-model.number="manualClip.end"
              type="number"
              min="0"
              step="0.1"
              class="w-24 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
            />
          </label>
          <p v-if="addClipError" class="text-xs text-rose-300">{{ addClipError }}</p>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
          <button type="button" class="rounded-lg border border-slate-800 bg-slate-900/60 px-4 py-2 text-xs text-slate-300" @click="closeAddClip">
            Cancel
          </button>
          <button type="button" class="rounded-lg bg-emerald-400 px-4 py-2 text-xs font-semibold text-slate-950" @click="addManualClip">
            {{ editingSegmentId ? 'Save changes' : 'Add clip' }}
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="showAutoSplit"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4 py-6 backdrop-blur"
      @click.self="closeAutoSplit"
    >
      <div class="w-full max-w-lg rounded-2xl border border-slate-800 bg-[#0f1724] p-6 shadow-2xl shadow-black/40 animate-fade-rise">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Auto-Split Settings</p>
            <h3 class="text-lg font-semibold text-slate-100">Generate clip batches</h3>
          </div>
          <button type="button" class="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-300" @click="closeAutoSplit">
            Close
          </button>
        </div>

        <div class="mt-4 space-y-4 text-sm text-slate-200">
          <div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Split method</p>
            <div class="mt-3 space-y-3">
              <label
                v-for="method in autoSplitMethods"
                :key="method.value"
                class="flex items-start gap-3"
                :class="method.value === 'scene' && !outputSettings.autoSceneDetection ? 'opacity-60' : ''"
              >
                <input
                  v-model="autoSplit.method"
                  type="radio"
                  :value="method.value"
                  :disabled="method.value === 'scene' && !outputSettings.autoSceneDetection"
                  class="mt-1 h-4 w-4 border-slate-600 text-emerald-400"
                />
                <span>{{ method.label }}</span>
              </label>
            </div>

            <div class="mt-4 grid gap-3 md:grid-cols-2 text-xs text-slate-300">
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Interval (sec)
                <input
                  v-model.number="autoSplit.interval"
                  type="number"
                  min="10"
                  step="1"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                  :disabled="autoSplit.method !== 'duration'"
                />
              </label>
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Clip count
                <input
                  v-model.number="autoSplit.count"
                  type="number"
                  min="2"
                  step="1"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                  :disabled="autoSplit.method !== 'count'"
                />
              </label>
            </div>

            <div v-if="autoSplit.method === 'scene'" class="mt-4 grid gap-3 md:grid-cols-2 text-xs text-slate-300">
              <label class="flex flex-col gap-2 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                <span class="flex items-center justify-between">
                  Sensitivity
                  <span class="text-slate-400">{{ sceneSensitivityLabel }}</span>
                </span>
                <input
                  v-model.number="autoSplit.sceneThreshold"
                  type="range"
                  min="0.1"
                  max="0.9"
                  step="0.05"
                  class="w-full accent-emerald-400"
                />
              </label>
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Min duration
                <input
                  v-model.number="autoSplit.sceneMinDuration"
                  type="number"
                  min="0"
                  step="0.5"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                />
              </label>
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Max duration
                <input
                  v-model.number="autoSplit.sceneMaxDuration"
                  type="number"
                  min="2"
                  step="0.5"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                />
              </label>
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Max segments
                <input
                  v-model.number="autoSplit.sceneMaxSegments"
                  type="number"
                  min="5"
                  step="5"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                />
              </label>
            </div>
          </div>

          <div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Clip duration limits</p>
            <div class="mt-3 grid gap-3 md:grid-cols-2 text-xs text-slate-300">
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                Min (sec)
                <input
                  v-model.number="autoSplit.min"
                  type="number"
                  min="10"
                  :max="maxClipDuration || undefined"
                  step="1"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                />
              </label>
              <label class="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2">
                <span>
                  Max (sec)
                  <span class="text-slate-500">/ {{ maxClipDuration || '—' }}</span>
                </span>
                <input
                  v-model.number="autoSplit.max"
                  type="number"
                  :min="maxClipDuration && maxClipDuration < 20 ? maxClipDuration : 20"
                  :max="maxClipDuration || undefined"
                  step="1"
                  class="w-20 rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100"
                />
              </label>
            </div>
            <p v-if="autoSplit.method !== 'scene'" class="mt-3 text-xs text-slate-400">
              This will create approximately <span class="font-semibold text-slate-200">{{ estimatedClips }}</span> clips.
            </p>
            <p v-else class="mt-3 text-xs text-slate-400">
              Scene detection will split on visual cuts in the selected range.
            </p>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
          <button type="button" class="rounded-lg border border-slate-800 bg-slate-900/60 px-4 py-2 text-xs text-slate-300" @click="closeAutoSplit">
            Cancel
          </button>
          <button type="button" class="rounded-lg bg-emerald-400 px-4 py-2 text-xs font-semibold text-slate-950" @click="applyAutoSplit">
            Generate clips
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
