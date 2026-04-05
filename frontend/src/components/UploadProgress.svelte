<script lang="ts">
  import { uploads, uploadQueue } from '../lib/stores';
  import { EventsOn } from '../../wailsjs/runtime/runtime';
  import type { UploadProgress } from '../lib/types';

  interface DownloadProgress {
    file_id: string;
    file_name: string;
    downloaded: number;
    total: number;
    percent: number;
  }

  let downloads = new Map<string, DownloadProgress>();

  // Upload queue events
  EventsOn('upload:queue', (data: { current: number; total: number }) => {
    uploadQueue.set(data);
  });

  EventsOn('upload:progress', (data: UploadProgress) => {
    uploads.update(m => {
      const newMap = new Map(m);
      newMap.set(data.file_id, data);
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

  let downloadQueue = { current: 0, total: 0 };

  EventsOn('download:queue', (data: { current: number; total: number }) => {
    downloadQueue = data;
  });

  // Download progress events
  EventsOn('download:progress', (data: DownloadProgress) => {
    downloads.set(data.file_id, data);
    downloads = downloads; // trigger reactivity
    if (data.percent >= 100) {
      setTimeout(() => {
        downloads.delete(data.file_id);
        downloads = downloads;
      }, 3000);
    }
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
  $: activeDownloads = Array.from(downloads.values());
  $: hasActivity = activeUploads.length > 0 || activeDownloads.length > 0;
</script>

{#if hasActivity}
  <div class="transfer-panel">
    {#if activeUploads.length > 0}
      <div class="panel-header">
        <span>
          {#if $uploadQueue.total > 0}<span class="spinner">&#x21BB;</span>{/if}
          Uploads {#if $uploadQueue.total > 0}({$uploadQueue.current}/{$uploadQueue.total}){:else}({activeUploads.length}){/if}
        </span>
      </div>
      {#each activeUploads as upload}
        <div class="transfer-item">
          <div class="transfer-info">
            <span class="transfer-name">{upload.file_name}</span>
            <span class="transfer-size">{formatSize(upload.uploaded)} / {formatSize(upload.total)}</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill upload-fill" style="width: {upload.percent}%"
                 class:complete={upload.percent >= 100}></div>
          </div>
        </div>
      {/each}
    {/if}

    {#if activeDownloads.length > 0}
      <div class="panel-header download-header">
        <span>
          <span class="spinner">&#x21BB;</span>
          Downloads {#if downloadQueue.total > 0}({downloadQueue.current}/{downloadQueue.total}){:else}({activeDownloads.length}){/if}
        </span>
      </div>
      {#each activeDownloads as dl}
        <div class="transfer-item">
          <div class="transfer-info">
            <span class="transfer-name">{dl.file_name}</span>
            <span class="transfer-size">{formatSize(dl.downloaded)} / {formatSize(dl.total)}</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill download-fill" style="width: {dl.percent}%"
                 class:complete={dl.percent >= 100}></div>
          </div>
        </div>
      {/each}
    {/if}
  </div>
{/if}

<style>
  .transfer-panel {
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
    max-height: 300px;
    overflow-y: auto;
  }

  .panel-header {
    padding: 10px 16px;
    font-weight: 600;
    font-size: 13px;
    border-bottom: 1px solid var(--border);
  }

  .download-header {
    border-top: 1px solid var(--border);
  }

  .spinner {
    display: inline-block;
    font-size: 13px;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .transfer-item {
    padding: 10px 16px;
    border-bottom: 1px solid var(--border);
  }

  .transfer-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .transfer-name {
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
  }

  .transfer-size {
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
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .upload-fill {
    background: var(--accent);
  }

  .download-fill {
    background: #10b981;
  }

  .progress-fill.complete {
    background: var(--success);
  }
</style>
