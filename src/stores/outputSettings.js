import { reactive } from 'vue';

export const outputSettingPresets = ['YouTube Shorts', 'TikTok', 'Instagram Reels', 'Landscape HD'];
export const outputSettingCropMethods = ['center', 'top', 'bottom', 'smart'];
export const outputSettingCaptionStyles = ['word_highlight', 'karaoke', 'subtitle'];
export const outputSettingLanguages = ['auto', 'en', 'id', 'jp'];
export const outputSettingCaptionSources = [
  { value: 'youtube', label: 'YouTube captions' },
  { value: 'upload', label: 'Upload subtitle file' },
  { value: 'speech', label: 'Speech-to-text' },
];
export const outputSettingAspectRatios = [
  { value: '9:16', label: '9:16 (vertical)' },
  { value: '4:5', label: '4:5 (portrait)' },
  { value: '1:1', label: '1:1 (square)' },
];
export const outputSettingAspectModes = [
  { value: 'crop', label: 'Crop to ratio' },
  { value: 'fit', label: 'Fit with black bars' },
];
export const outputSettingSceneModes = [
  { value: 'classic', label: 'Standard (ffmpeg)' },
  { value: 'ai', label: 'AI (Gemini)' },
];

export const outputSettings = reactive({
  preset: outputSettingPresets[0],
  vertical: true,
  aspectRatio: outputSettingAspectRatios[0].value,
  aspectMode: outputSettingAspectModes[0].value,
  cropMethod: outputSettingCropMethods[0],
  captions: false,
  captionStyle: outputSettingCaptionStyles[0],
  captionSource: outputSettingCaptionSources[0].value,
  captionLanguage: outputSettingLanguages[0],
  captionFormat: 'srt',
  captionText: '',
  captionFileName: '',
  autoSceneDetection: false,
  sceneDetectionMode: outputSettingSceneModes[0].value,
  fontSize: 56,
  highlight: '#F2C14E',
  language: outputSettingLanguages[0],
  backgroundMusic: false,
  outputFolder: '/output/clips',
  thumbnailFolder: '/output/thumbnails',
});
