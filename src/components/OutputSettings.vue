<script setup>
import { reactive } from 'vue';

const presets = ['YouTube Shorts', 'TikTok', 'Instagram Reels', 'Landscape HD'];
const cropMethods = ['center', 'top', 'bottom', 'smart'];
const captionStyles = ['word_highlight', 'karaoke', 'subtitle'];
const languages = ['auto', 'en', 'id', 'jp'];

const settings = reactive({
  preset: presets[0],
  vertical: true,
  cropMethod: cropMethods[0],
  captions: true,
  captionStyle: captionStyles[0],
  fontSize: 56,
  highlight: '#F2C14E',
  language: languages[0],
  backgroundMusic: false,
  outputFolder: '/output/clips',
  thumbnailFolder: '/output/thumbnails',
});
</script>

<template>
  <aside class="flex flex-col gap-5 rounded-2xl border border-slate-800/70 bg-[#0f1724]/80 p-5 shadow-xl shadow-black/20">
    <header class="flex items-center justify-between">
      <div>
        <p class="text-xs uppercase tracking-[0.24em] text-slate-400">Output Settings</p>
        <h2 class="text-lg font-semibold text-slate-100">Publish-ready defaults</h2>
      </div>
      <span class="rounded-full border border-slate-700 bg-slate-900/70 px-3 py-1 text-xs text-slate-300">Preview</span>
    </header>

    <div class="rounded-xl border border-slate-800 bg-[#121c2a]/80 p-4">
      <label class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Platform Preset</label>
      <div class="mt-3 flex items-center gap-3">
        <select v-model="settings.preset" class="flex-1 rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-sm text-slate-100">
          <option v-for="preset in presets" :key="preset" :value="preset">{{ preset }}</option>
        </select>
        <span class="rounded-full bg-emerald-500/10 px-3 py-1 text-xs text-emerald-200">Active</span>
      </div>

      <div class="mt-4 flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 text-sm text-slate-200">
        <label class="flex items-center gap-2">
          <input v-model="settings.vertical" type="checkbox" class="h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400" />
          Convert to vertical (9:16)
        </label>
        <select v-model="settings.cropMethod" class="rounded-lg border border-slate-800 bg-[#0c141f] px-2 py-1 text-xs text-slate-200">
          <option v-for="method in cropMethods" :key="method" :value="method">{{ method }}</option>
        </select>
      </div>
    </div>

    <div class="rounded-xl border border-slate-800 bg-[#121c2a]/80 p-4">
      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Enhancements</p>
      <div class="mt-3 space-y-3 text-sm text-slate-200">
        <label class="flex items-center gap-2">
          <input type="checkbox" checked disabled class="h-4 w-4 rounded border-slate-700 bg-slate-900 text-emerald-400" />
          Auto scene detection
        </label>
        <label class="flex items-center gap-2">
          <input v-model="settings.captions" type="checkbox" class="h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400" />
          Word-by-word captions
        </label>
      </div>

      <div class="mt-3 grid gap-3 text-xs text-slate-300">
        <div class="flex items-center justify-between gap-3">
          <span>Caption style</span>
          <select v-model="settings.captionStyle" class="rounded-lg border border-slate-800 bg-[#0c141f] px-2 py-1 text-xs text-slate-200">
            <option v-for="style in captionStyles" :key="style" :value="style">{{ style }}</option>
          </select>
        </div>
        <div class="flex items-center justify-between gap-3">
          <span>Font size</span>
          <input v-model.number="settings.fontSize" type="range" min="32" max="80" class="flex-1 accent-emerald-400" />
          <span class="w-10 text-right">{{ settings.fontSize }}</span>
        </div>
        <div class="flex items-center justify-between gap-3">
          <span>Highlight</span>
          <input v-model="settings.highlight" type="color" class="h-8 w-10 rounded border border-slate-700 bg-transparent" />
        </div>
        <div class="flex items-center justify-between gap-3">
          <span>Language</span>
          <select v-model="settings.language" class="rounded-lg border border-slate-800 bg-[#0c141f] px-2 py-1 text-xs text-slate-200">
            <option v-for="lang in languages" :key="lang" :value="lang">{{ lang }}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="rounded-xl border border-slate-800 bg-[#121c2a]/80 p-4 text-sm text-slate-200">
      <label class="flex items-center gap-2">
        <input v-model="settings.backgroundMusic" type="checkbox" class="h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400" />
        Background music (coming soon)
      </label>
    </div>

    <div class="rounded-xl border border-slate-800 bg-[#121c2a]/80 p-4 text-xs text-slate-300">
      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Folders</p>
      <div class="mt-3 space-y-3">
        <div>
          <p class="text-slate-400">Clip output</p>
          <input v-model="settings.outputFolder" type="text" class="mt-1 w-full rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-xs text-slate-100" />
        </div>
        <div>
          <p class="text-slate-400">Thumbnails</p>
          <input v-model="settings.thumbnailFolder" type="text" class="mt-1 w-full rounded-lg border border-slate-800 bg-[#0c141f] px-3 py-2 text-xs text-slate-100" />
        </div>
      </div>
    </div>
  </aside>
</template>
