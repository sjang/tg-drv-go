<script lang="ts">
  import { currentFolder, files } from '../lib/stores';
  import { UploadFilePath, GetFiles } from '../lib/api';

  let isDragging = false;

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    if ($currentFolder) isDragging = true;
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    // Only hide if we're leaving the window
    if (e.relatedTarget === null) isDragging = false;
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;

    if (!$currentFolder || !e.dataTransfer?.files) return;

    const droppedFiles = e.dataTransfer.files;
    for (let i = 0; i < droppedFiles.length; i++) {
      const file = droppedFiles[i];
      // In Wails/WebKit on macOS, we can access the file path
      const path = (file as any).path;
      if (path) {
        try {
          await UploadFilePath($currentFolder.id, path);
        } catch (err) {
          console.error('upload error:', err);
        }
      }
    }

    // Refresh file list
    if ($currentFolder) {
      const result = await GetFiles($currentFolder.id);
      files.set(result || []);
    }
  }
</script>

<svelte:window
  on:dragenter={handleDragEnter}
  on:dragover={handleDragOver}
  on:dragleave={handleDragLeave}
  on:drop={handleDrop}
/>

{#if isDragging}
  <div class="drop-overlay">
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
    pointer-events: none;
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
