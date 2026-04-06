<script lang="ts">
  import { currentFolder, viewMode, files, isLoading, refreshFiles, errorMessage } from '../lib/stores';
  import { UploadFiles, RebuildIndex } from '../lib/api';

  let rebuilding = false;
  let rebuildResult: string | null = null;

  async function uploadFiles() {
    if (!$currentFolder) return;
    try {
      const count = await UploadFiles($currentFolder.id);
      await refreshFiles();
      if (count === 0) {
        // All uploads may have failed silently on the Go side
        errorMessage.set('Upload completed but no files were added. Check if files already exist.');
        setTimeout(() => errorMessage.set(null), 5000);
      }
    } catch (e) {
      // "no files selected" is not a real error
      if (String(e).includes('no files selected')) return;
      console.error('upload:', e);
      errorMessage.set('Upload failed: ' + String(e));
      setTimeout(() => errorMessage.set(null), 5000);
    }
  }

  async function rebuildIndex() {
    if (!$currentFolder || rebuilding) return;
    rebuilding = true;
    rebuildResult = null;
    isLoading.set(true);
    try {
      const count = await RebuildIndex($currentFolder.id);
      await refreshFiles();
      rebuildResult = `${count} files indexed`;
      setTimeout(() => { rebuildResult = null; }, 3000);
    } catch (e) {
      console.error('rebuild:', e);
      rebuildResult = 'Rebuild failed';
      setTimeout(() => { rebuildResult = null; }, 3000);
    } finally {
      rebuilding = false;
      isLoading.set(false);
    }
  }

  function toggleView() {
    viewMode.update(v => v === 'grid' ? 'list' : 'grid');
  }
</script>

<div class="topbar">
  <div class="folder-title">
    {#if $currentFolder}
      <h2>{$currentFolder.name}</h2>
      <span class="file-count">{$files.length} files</span>
      {#if rebuildResult}
        <span class="rebuild-result">{rebuildResult}</span>
      {/if}
    {:else}
      <h2>TG Drive</h2>
    {/if}
  </div>

  <div class="actions">
    {#if $currentFolder}
      <button class="action-btn" class:spinning={rebuilding} on:click={rebuildIndex} title="Rebuild Index" disabled={rebuilding}>
        <span class="reload-icon">&#x21BB;</span>
      </button>
      <button class="action-btn" on:click={toggleView} title="Toggle View">
        {$viewMode === 'grid' ? '\u{2630}' : '\u{2637}'}
      </button>
      <button class="upload-btn" on:click={uploadFiles}>
        Upload
      </button>
    {/if}
  </div>
</div>

<style>
  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-primary);
    -webkit-app-region: drag;
  }

  .folder-title {
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .folder-title h2 {
    font-size: 18px;
    font-weight: 600;
  }

  .file-count {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .rebuild-result {
    font-size: 12px;
    color: var(--accent);
    animation: fadeIn 0.3s ease;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
    -webkit-app-region: no-drag;
  }

  .action-btn {
    width: 36px;
    height: 36px;
    background: var(--bg-secondary);
    border-radius: 8px;
    color: var(--text-primary);
    font-size: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .action-btn:hover {
    background: var(--bg-hover);
  }

  .action-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .reload-icon {
    display: inline-block;
    font-size: 18px;
  }

  .spinning .reload-icon {
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .upload-btn {
    padding: 8px 20px;
    background: var(--accent);
    color: white;
    border-radius: 8px;
    font-weight: 600;
    transition: background 0.2s;
  }

  .upload-btn:hover {
    background: var(--accent-hover);
  }
</style>
