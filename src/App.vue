<script setup>
import ClipForm from './components/ClipForm.vue';
import OutputSettings from './components/OutputSettings.vue';
</script>

<template>
  <div class="min-h-screen bg-[#0b1016] text-slate-50">
    <div class="relative overflow-hidden">
      <div class="pointer-events-none absolute -top-40 left-[-10rem] h-[28rem] w-[28rem] rounded-full bg-[#1dd7a1]/20 blur-[120px]"></div>
      <div class="pointer-events-none absolute bottom-[-14rem] right-[-6rem] h-[26rem] w-[26rem] rounded-full bg-[#4aa3ff]/20 blur-[120px]"></div>
      <div class="pointer-events-none absolute inset-x-0 top-0 h-24 bg-gradient-to-b from-black/50 to-transparent"></div>

      <main class="relative mx-auto flex max-w-6xl flex-col gap-10 px-6 pb-16 pt-12">
        <header class="flex flex-col gap-5 animate-fade-rise">
          <div class="flex flex-wrap items-center gap-3">
            <span class="inline-flex items-center rounded-full border border-emerald-500/40 bg-emerald-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.32em] text-emerald-200">
              Auto Clip Studio
            </span>
            <span class="inline-flex items-center rounded-full border border-slate-800 bg-slate-900/70 px-3 py-1 text-xs text-slate-300">
              Server-first exports
            </span>
          </div>
          <h1 class="text-3xl font-semibold leading-tight text-slate-100 md:text-5xl">
            Build precise YouTube clips with a pro editor workflow
          </h1>
          <p class="max-w-3xl text-base text-slate-300 md:text-lg">
            Load a YouTube URL, mark the in and out points, and render a clean MP4 to your server output
            folder. The UI calls the backend API at <code class="font-['Fira_Code']">/api/clip</code>.
          </p>
        </header>

        <section class="grid gap-6 lg:grid-cols-[1.55fr,1fr] animate-fade-rise-delay">
          <ClipForm />
          <OutputSettings />
        </section>
      </main>
    </div>
  </div>
</template>
