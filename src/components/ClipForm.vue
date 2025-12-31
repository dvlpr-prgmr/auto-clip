<script setup>
import { computed, reactive, ref, watch } from 'vue';

const apiBase = computed(() => {
  const base = import.meta.env.VITE_API_BASE?.trim();
  if (base) return base.endsWith('/') ? base.slice(0, -1) : base;

  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin;
  }

  return 'http://localhost:8888'; // fallback for non-browser contexts
});

const form = reactive({
  url: 'https://www.youtube.com/watch?v=2-Z6H2bgrnM',
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
];
const autoSplit = reactive({
  method: autoSplitMethods[0].value,
  interval: 60,
  count: 10,
  min: 60,
  max: 180,
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
const estimatedClips = computed(() => {
  if (segmentDuration.value <= 0) return 0;
  if (autoSplit.method === 'duration') {
    const interval = Math.max(Number(autoSplit.interval) || 1, 1);
    return Math.ceil(segmentDuration.value / interval);
  }
  if (autoSplit.method === 'count') {
    return Math.max(Number(autoSplit.count) || 0, 0);
  }
  const maxInterval = Math.max(Number(autoSplit.max) || 1, 1);
  return Math.ceil(segmentDuration.value / maxInterval);
});

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

function getRange() {
  const start = Math.max(Number(form.start) || 0, 0);
  const end = Math.max(Number(form.end) || 0, 0);
  return end > start ? { start, end } : { start, end: start };
}

function createSegment(start, end, index) {
  return {
    id: `${index}-${start}-${end}`,
    index,
    start,
    end,
    duration: Math.max(end - start, 0),
    thumbnailUrl: preview.thumbnailUrl || '',
  };
}

function reindexSegments(items) {
  return items.map((segment, idx) => ({
    ...segment,
    index: idx + 1,
    duration: Math.max(segment.end - segment.start, 0),
    thumbnailUrl: preview.thumbnailUrl || segment.thumbnailUrl || '',
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

function applyAutoSplit() {
  segmentsMode.value = 'auto';
  autoSplitApplied.value = true;
  refreshSegments();
  showAutoSplit.value = false;
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
              thumbnailUrl: preview.thumbnailUrl || segment.thumbnailUrl || '',
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
}

function deleteSegment(segment) {
  if (!segment) return;
  segmentsMode.value = 'manual';
  autoSplitApplied.value = false;
  segments.value = reindexSegments(segments.value.filter((item) => item.id !== segment.id));
  if (selectedSegmentId.value === segment.id) {
    selectedSegmentId.value = '';
  }
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

refreshSegments();

async function handleSubmit() {
  status.type = 'idle';
  status.message = '';
  result.outputPath = '';
  result.thumbnailPath = '';
  result.thumbnailError = '';
  isSubmitting.value = true;

  try {
    const payload = {
      url: form.url.trim(),
      start: Number(form.start),
      end: Number(form.end),
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
      throw new Error(detail);
    }

    const resultPayload = await response.json();
    result.outputPath = resultPayload.output_path || '';
    result.thumbnailPath = resultPayload.thumbnail_path || '';
    result.thumbnailError = resultPayload.thumbnail_error || '';

    status.type = 'success';
    status.message = 'Clip saved on the server.';
  } catch (error) {
    status.type = 'error';
    status.message = error.message || 'Unexpected error generating clip.';
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
          </div>
          <span class="rounded-full border border-slate-800 bg-slate-900/60 px-3 py-1 text-xs text-slate-400">Manual queue</span>
        </div>

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
                  <p class="text-sm font-semibold text-slate-100">Segment {{ String(segment.index).padStart(2, '0') }}</p>
                  <p class="text-xs text-slate-400">
                    {{ formatTime(segment.start) }} - {{ formatTime(segment.end) }} (length {{ formatTime(segment.duration) }})
                  </p>
                </div>
              </div>
              <div class="absolute -right-3 -top-3 flex items-center gap-2 opacity-0 transition group-hover:opacity-100 group-focus-within:opacity-100">
                <button
                  type="button"
                  class="grid h-7 w-7 place-items-center rounded-full border border-slate-700 bg-slate-900/80 text-slate-300 transition hover:border-emerald-400 hover:text-emerald-200"
                  @click.stop="openEditSegment(segment)"
                  aria-label="Edit segment"
                >
                  ✎
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
        <button
          type="submit"
          :disabled="isSubmitting || !isValid"
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
          {{ isSubmitting ? 'Rendering...' : 'Generate clip' }}
        </button>
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
              <label v-for="method in autoSplitMethods" :key="method.value" class="flex items-start gap-3">
                <input
                  v-model="autoSplit.method"
                  type="radio"
                  :value="method.value"
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
            <p class="mt-3 text-xs text-slate-400">
              This will create approximately <span class="font-semibold text-slate-200">{{ estimatedClips }}</span> clips.
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
