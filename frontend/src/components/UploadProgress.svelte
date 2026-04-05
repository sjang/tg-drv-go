<script lang="ts">
  import { uploads } from '../lib/stores';
  import { EventsOn } from '../../wailsjs/runtime/runtime';
  import type { UploadProgress } from '../lib/types';

  EventsOn('upload:progress', (data: UploadProgress) => {
    uploads.update(m => {
      const newMap = new Map(m);
      newMap.set(data.file_id, data);
      // Remove completed uploads after 3 seconds
      if (data.percent >= 100) {
        setTimeout(() => {
          uploads.update(m2 => {
            const map = new Map(m2);
            map.delete(data.file_id);
            return map;
          });
        }, 3000);
      }
      return newMap;
    });
  });

  function formatSize(bytes: number): string {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    let i = 0;
    let size = bytes;
    while (size >= 1024 && i < units.length - 1) {
      size /= 1024;
      i++;
    }
    return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
  }

  $: activeUploads = Array.from($uploads.values());
</script>

{#if activeUploads.length > 0}
  <div class="upload-panel">
    <div class="panel-header">
      <span>Uploads ({activeUploads.length})</span>
    </div>
    {#each activeUploads as upload}
      <div class="upload-item">
        <div class="upload-info">
          <span class="upload-name">{upload.file_name}</span>
          <span class="upload-size">{formatSize(upload.uploaded)} / {formatSize(upload.total)}</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" style="width: {upload.percent}%"
               class:complete={upload.percent >= 100}></div>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .upload-panel {
    position: fixed;
    bottom: 0;
    right: 0;
    width: 360px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border);
    border-left: 1px solid var(--border);
    border-top-left-radius: 12px;
    box-shadow: 0 -4px 16px var(--shadow);
    z-index: 50;
    max-height: 240px;
    overflow-y: auto;
  }

  .panel-header {
    padding: 10px 16px;
    font-weight: 600;
    font-size: 13px;
    border-bottom: 1px solid var(--border);
  }

  .upload-item {
    padding: 10px 16px;
    border-bottom: 1px solid var(--border);
  }

  .upload-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .upload-name {
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
  }

  .upload-size {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .progress-bar {
    height: 4px;
    background: var(--bg-tertiary);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--accent);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .progress-fill.complete {
    background: var(--success);
  }
</style>
