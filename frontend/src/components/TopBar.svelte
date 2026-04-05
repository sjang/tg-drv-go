<script lang="ts">
  import { currentFolder, viewMode, files, isLoading } from '../lib/stores';
  import { UploadFile, GetFiles, RebuildIndex } from '../lib/api';

  async function uploadFile() {
    if (!$currentFolder) return;
    try {
      await UploadFile($currentFolder.id);
      const result = await GetFiles($currentFolder.id);
      files.set(result || []);
    } catch (e) {
      console.error('upload:', e);
    }
  }

  async function rebuildIndex() {
    if (!$currentFolder) return;
    if (!confirm('Rebuild index from Telegram? This will rescan all messages.')) return;
    isLoading.set(true);
    try {
      const count = await RebuildIndex($currentFolder.id);
      const result = await GetFiles($currentFolder.id);
      files.set(result || []);
      alert(`Index rebuilt: ${count} files found`);
    } catch (e) {
      console.error('rebuild:', e);
    } finally {
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
    {:else}
      <h2>TG Drive</h2>
    {/if}
  </div>

  <div class="actions">
    {#if $currentFolder}
      <button class="action-btn" on:click={rebuildIndex} title="Rebuild Index">
        &#x1F504;
      </button>
      <button class="action-btn" on:click={toggleView} title="Toggle View">
        {$viewMode === 'grid' ? '\u{2630}' : '\u{2637}'}
      </button>
      <button class="upload-btn" on:click={uploadFile}>
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
