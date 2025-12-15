<script setup>
import ClipForm from './components/ClipForm.vue';
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-50">
    <main class="mx-auto flex max-w-5xl flex-col gap-10 px-6 pb-16 pt-14">
      <header class="flex flex-col gap-4">
        <p class="text-sm font-semibold uppercase tracking-[0.25em] text-sky-400">Auto Clip</p>
        <h1 class="text-4xl font-bold leading-tight md:text-5xl">
          Smart clip generator for YouTube segments
        </h1>
        <p class="max-w-3xl text-lg text-slate-300">
          Paste a YouTube link, set your start and end times, and download a clean MP4 clip.
          The UI talks to the existing Express backend at <code>/api/clip</code>.
        </p>
      </header>

      <section class="grid gap-6 rounded-2xl border border-slate-800 bg-slate-900/60 p-6 shadow-xl shadow-slate-900/40 md:grid-cols-[2fr,1fr] md:gap-10">
        <ClipForm />

        <aside class="flex flex-col gap-4 rounded-xl border border-slate-800 bg-slate-950/60 p-4 text-sm text-slate-300">
          <h2 class="text-lg font-semibold text-slate-50">How it works</h2>
          <ol class="list-decimal space-y-3 pl-5">
            <li>Fill in the YouTube URL and your desired start/end times in seconds.</li>
            <li>Click <strong>Generate clip</strong> to call the Express backend.</li>
            <li>When ready, your browser downloads <code>clip.mp4</code>.</li>
          </ol>
          <div class="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
            <p class="font-semibold text-slate-100">Backend URL</p>
            <p class="mt-1 text-xs text-slate-400">
              Override with <code>VITE_API_BASE</code> in a <code>.env.local</code> if the backend is hosted elsewhere.
            </p>
          </div>
        </aside>
      </section>
    </main>
  </div>
</template>
