<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime';
  import { currentFolder, uploadQueue, refreshFiles } from '../lib/stores';
  import { UploadFilePath } from '../lib/api';

  let isDragging = false;
  let uploading = false;
  let dragCount = 0;

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCount++;
    if ($currentFolder) isDragging = true;
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    dragCount--;
    if (dragCount <= 0) {
      isDragging = false;
      dragCount = 0;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    dragCount = 0;
    // File paths are handled by Wails native file-drop event below
  }

  // Wails native file drop handler
  async function onFileDrop(_x: number, _y: number, paths: string[]) {
    if (!$currentFolder || uploading || !paths || paths.length === 0) return;
    uploading = true;
    const total = paths.length;
    uploadQueue.set({ current: 1, total });
    try {
      for (let i = 0; i < paths.length; i++) {
        uploadQueue.set({ current: i + 1, total });
        try {
          await UploadFilePath($currentFolder.id, paths[i]);
        } catch (err) {
          console.error('upload error:', paths[i], err);
        }
      }
      await refreshFiles();
    } finally {
      uploading = false;
      uploadQueue.set({ current: 0, total: 0 });
    }
  }

  onMount(() => {
    OnFileDrop((x: number, y: number, paths: string[]) => {
      onFileDrop(x, y, paths);
    }, true);
  });

  onDestroy(() => {
    OnFileDropOff();
  });
</script>

<svelte:window
  on:dragenter={handleDragEnter}
  on:dragover={handleDragOver}
  on:dragleave={handleDragLeave}
  on:drop={handleDrop}
/>

{#if isDragging && $currentFolder}
  <div class="drop-overlay" style="--wails-drop-target: drop">
    <div class="drop-content">
      <span class="drop-icon">{'\u{1F4E5}'}</span>
      <p>Drop files to upload</p>
      <p class="drop-hint">to "{$currentFolder?.name}"</p>
    </div>
  </div>
{/if}


<style>
  .drop-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(74, 158, 255, 0.15);
    border: 3px dashed var(--accent);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 300;
  }

  .drop-content {
    text-align: center;
    color: var(--accent);
  }

  .drop-icon {
    font-size: 64px;
    display: block;
    margin-bottom: 16px;
  }

  .drop-content p {
    font-size: 20px;
    font-weight: 600;
  }

  .drop-hint {
    font-size: 14px !important;
    font-weight: 400 !important;
    opacity: 0.8;
    margin-top: 4px;
  }

</style>
