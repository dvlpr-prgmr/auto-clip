<script setup>
import { computed, reactive, ref } from 'vue';

const apiBase = computed(() => {
  const base = import.meta.env.VITE_API_BASE?.trim();
  if (base) return base.endsWith('/') ? base.slice(0, -1) : base;

  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin;
  }

  return 'http://localhost:8888'; // fallback for non-browser contexts
});

const form = reactive({
  url: '',
  start: 0,
  end: 30,
});

const isSubmitting = ref(false);
const status = reactive({
  type: 'idle',
  message: '',
  detail: '',
  logs: [],
});

async function handleSubmit() {
  status.type = 'idle';
  status.message = '';
  status.detail = '';
  status.logs = [];
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
      let errorMessage = 'Failed to generate clip.';
      let detail = '';
      let logs = [];
      try {
        const errorPayload = await response.json();
        errorMessage = errorPayload.error || errorMessage;
        detail = errorPayload.detail || detail;
        logs = Array.isArray(errorPayload.logs) ? errorPayload.logs : [];
      } catch (e) {
        // If parsing fails, fall back to generic message.
      }

      status.type = 'error';
      status.message = errorMessage;
      status.detail = detail;
      status.logs = logs;
      return;
    }

    const blob = await response.blob();
    const downloadUrl = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = 'clip.mp4';
    link.click();
    setTimeout(() => URL.revokeObjectURL(downloadUrl), 5000);

    status.type = 'success';
    status.message = 'Clip ready! The download should begin automatically.';
    status.detail = '';
    status.logs = [];
  } catch (error) {
    status.type = 'error';
    status.message = error.message || 'Unexpected error generating clip.';
    status.detail = status.detail || '';
    status.logs = Array.isArray(status.logs) ? status.logs : [];
  } finally {
    isSubmitting.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <form @submit.prevent="handleSubmit" class="flex flex-col gap-4">
      <div class="flex flex-col gap-2">
        <label class="text-sm font-semibold text-slate-200">YouTube URL</label>
        <input
          v-model="form.url"
          type="url"
          required
          placeholder="https://www.youtube.com/watch?v=..."
          class="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-sky-500 focus:ring-sky-500"
        />
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="flex flex-col gap-2">
          <label class="text-sm font-semibold text-slate-200">Start time (seconds)</label>
          <input
            v-model.number="form.start"
            type="number"
            min="0"
            step="0.1"
            class="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-sky-500 focus:ring-sky-500"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label class="text-sm font-semibold text-slate-200">End time (seconds)</label>
          <input
            v-model.number="form.end"
            type="number"
            min="0"
            step="0.1"
            class="w-full rounded-lg border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-100 outline-none ring-1 ring-transparent transition focus:border-sky-500 focus:ring-sky-500"
          />
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3 text-xs text-slate-400">
        <span>Start must be &ge; 0.</span>
        <span>End must be greater than start.</span>
        <span>Backend: <code>{{ apiBase }}</code></span>
      </div>

      <div class="flex items-center gap-3">
        <button
          type="submit"
          :disabled="isSubmitting"
          class="inline-flex items-center justify-center rounded-lg bg-sky-500 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-sky-400 disabled:cursor-not-allowed disabled:opacity-70"
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
          {{ isSubmitting ? 'Generating...' : 'Generate clip' }}
        </button>
        <p class="text-xs text-slate-400">Response is downloaded as <code>clip.mp4</code>.</p>
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

    <div v-if="status.type === 'error' && (status.detail || status.logs.length)" class="rounded-lg border border-rose-500/40 bg-rose-950/30 p-3 text-xs text-rose-50">
      <p v-if="status.detail" class="mb-2 font-semibold">Detail: {{ status.detail }}</p>
      <div v-if="status.logs.length" class="space-y-1">
        <p class="font-semibold">Recent ffmpeg logs:</p>
        <ul class="list-disc space-y-1 pl-4">
          <li v-for="line in status.logs" :key="line" class="font-mono text-[11px]">{{ line }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>
