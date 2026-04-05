<script lang="ts">
  import { files, currentFolder, viewMode, isLoading } from '../lib/stores';
  import { DeleteFile, RenameFile, GetStreamURL, GetThumbnailURL, GetFiles } from '../lib/api';
  import type { FileItem } from '../lib/types';

  let contextMenu: { x: number; y: number; file: FileItem } | null = null;
  let renameId: string | null = null;
  let renameValue = '';
  let mediaFile: { url: string; type: string; name: string } | null = null;

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

  function formatDate(dateStr: string): string {
    const d = new Date(dateStr);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function getFileIcon(mimeType: string): string {
    if (mimeType.startsWith('image/')) return '\u{1F5BC}';
    if (mimeType.startsWith('video/')) return '\u{1F3AC}';
    if (mimeType.startsWith('audio/')) return '\u{1F3B5}';
    if (mimeType.includes('pdf')) return '\u{1F4C4}';
    if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('tar')) return '\u{1F4E6}';
    return '\u{1F4CE}';
  }

  function isPlayable(mimeType: string): boolean {
    return mimeType.startsWith('video/') || mimeType.startsWith('audio/');
  }

  async function openFile(file: FileItem) {
    if (isPlayable(file.mime_type)) {
      const url = await GetStreamURL(file.id);
      mediaFile = { url, type: file.mime_type, name: file.name };
    } else if (file.mime_type.startsWith('image/')) {
      const url = await GetStreamURL(file.id);
      mediaFile = { url, type: file.mime_type, name: file.name };
    }
  }

  function showContext(e: MouseEvent, file: FileItem) {
    e.preventDefault();
    contextMenu = { x: e.clientX, y: e.clientY, file };
  }

  function hideContext() {
    contextMenu = null;
  }

  async function downloadFile(file: FileItem) {
    const url = await GetStreamURL(file.id);
    window.open(url.replace('/stream', '/download'), '_blank');
    hideContext();
  }

  async function deleteFile(file: FileItem) {
    hideContext();
    if (!confirm(`Delete "${file.name}"?`)) return;
    try {
      await DeleteFile(file.id);
      if ($currentFolder) {
        const result = await GetFiles($currentFolder.id);
        files.set(result || []);
      }
    } catch (e) {
      console.error('delete file:', e);
    }
  }

  function startRename(file: FileItem) {
    hideContext();
    renameId = file.id;
    renameValue = file.name;
  }

  async function finishRename() {
    if (!renameId || !renameValue.trim()) {
      renameId = null;
      return;
    }
    try {
      await RenameFile(renameId, renameValue.trim());
      renameId = null;
      if ($currentFolder) {
        const result = await GetFiles($currentFolder.id);
        files.set(result || []);
      }
    } catch (e) {
      console.error('rename file:', e);
    }
  }

  function closeMedia() {
    mediaFile = null;
  }
</script>

<svelte:window on:click={hideContext} />

<div class="file-area">
  {#if $isLoading}
    <div class="loading">Loading...</div>
  {:else if !$currentFolder}
    <div class="empty-state">
      <span class="empty-icon">{'\u{1F4C2}'}</span>
      <p>Select a folder from the sidebar</p>
    </div>
  {:else if $files.length === 0}
    <div class="empty-state">
      <span class="empty-icon">{'\u{1F4E4}'}</span>
      <p>This folder is empty</p>
      <p class="hint">Drag & drop files or use the upload button</p>
    </div>
  {:else}
    <div class="file-container" class:grid-view={$viewMode === 'grid'} class:list-view={$viewMode === 'list'}>
      {#if $viewMode === 'list'}
        <div class="list-header">
          <span class="col-name">Name</span>
          <span class="col-size">Size</span>
          <span class="col-type">Type</span>
          <span class="col-date">Date</span>
        </div>
      {/if}

      {#each $files as file}
        <div
          class="file-item"
          on:click={() => openFile(file)}
          on:contextmenu={(e) => showContext(e, file)}
          on:dblclick={() => startRename(file)}
        >
          {#if $viewMode === 'grid'}
            <div class="file-icon">{getFileIcon(file.mime_type)}</div>
            {#if renameId === file.id}
              <input
                class="rename-input"
                bind:value={renameValue}
                on:blur={finishRename}
                on:keydown={(e) => e.key === 'Enter' && finishRename()}
                autofocus
              />
            {:else}
              <span class="file-name" title={file.name}>{file.name}</span>
            {/if}
            <span class="file-size">{formatSize(file.size)}</span>
          {:else}
            <span class="col-name">
              <span class="file-icon-small">{getFileIcon(file.mime_type)}</span>
              {#if renameId === file.id}
                <input
                  class="rename-input"
                  bind:value={renameValue}
                  on:blur={finishRename}
                  on:keydown={(e) => e.key === 'Enter' && finishRename()}
                  autofocus
                />
              {:else}
                <span title={file.name}>{file.name}</span>
              {/if}
            </span>
            <span class="col-size">{formatSize(file.size)}</span>
            <span class="col-type">{file.mime_type.split('/')[1] || file.mime_type}</span>
            <span class="col-date">{formatDate(file.upload_date)}</span>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Context Menu -->
{#if contextMenu}
  <div class="context-menu" style="left: {contextMenu.x}px; top: {contextMenu.y}px">
    {#if isPlayable(contextMenu.file.mime_type)}
      <button on:click={() => openFile(contextMenu.file)}>Play</button>
    {/if}
    <button on:click={() => downloadFile(contextMenu.file)}>Download</button>
    <button on:click={() => startRename(contextMenu.file)}>Rename</button>
    <button class="danger" on:click={() => deleteFile(contextMenu.file)}>Delete</button>
  </div>
{/if}

<!-- Media Player Modal -->
{#if mediaFile}
  <div class="modal-overlay" on:click={closeMedia}>
    <div class="modal-content" on:click|stopPropagation>
      <div class="modal-header">
        <span>{mediaFile.name}</span>
        <button class="close-btn" on:click={closeMedia}>&times;</button>
      </div>
      {#if mediaFile.type.startsWith('video/')}
        <video controls autoplay src={mediaFile.url} style="max-width: 100%; max-height: 80vh;">
          <track kind="captions" />
        </video>
      {:else if mediaFile.type.startsWith('audio/')}
        <audio controls autoplay src={mediaFile.url}></audio>
      {:else if mediaFile.type.startsWith('image/')}
        <img src={mediaFile.url} alt={mediaFile.name} style="max-width: 100%; max-height: 80vh;" />
      {/if}
    </div>
  </div>
{/if}

<style>
  .file-area {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    gap: 8px;
  }

  .empty-icon {
    font-size: 48px;
    margin-bottom: 8px;
  }

  .hint {
    font-size: 12px;
    opacity: 0.7;
  }

  .grid-view {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
  }

  .grid-view .file-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 16px 12px;
    background: var(--bg-secondary);
    border-radius: 10px;
    cursor: pointer;
    gap: 8px;
    transition: background 0.2s;
  }

  .grid-view .file-item:hover {
    background: var(--bg-hover);
  }

  .file-icon {
    font-size: 36px;
  }

  .file-name {
    font-size: 13px;
    text-align: center;
    word-break: break-all;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .file-size {
    font-size: 11px;
    color: var(--text-secondary);
  }

  /* List View */
  .list-view {
    display: flex;
    flex-direction: column;
  }

  .list-header {
    display: grid;
    grid-template-columns: 1fr 100px 120px 160px;
    padding: 8px 12px;
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid var(--border);
  }

  .list-view .file-item {
    display: grid;
    grid-template-columns: 1fr 100px 120px 160px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
    cursor: pointer;
    align-items: center;
  }

  .list-view .file-item:hover {
    background: var(--bg-hover);
  }

  .col-name {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .col-size, .col-type, .col-date {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .file-icon-small {
    font-size: 18px;
    flex-shrink: 0;
  }

  .rename-input {
    width: 100%;
    padding: 4px 8px;
    background: var(--bg-tertiary);
    border: 1px solid var(--accent);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 13px;
  }

  /* Context Menu */
  .context-menu {
    position: fixed;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 4px;
    box-shadow: 0 4px 16px var(--shadow);
    z-index: 100;
    min-width: 140px;
  }

  .context-menu button {
    display: block;
    width: 100%;
    padding: 8px 14px;
    background: transparent;
    color: var(--text-primary);
    text-align: left;
    border-radius: 4px;
  }

  .context-menu button:hover {
    background: var(--bg-hover);
  }

  .context-menu button.danger:hover {
    background: var(--danger);
    color: white;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
  }

  .modal-content {
    background: var(--bg-secondary);
    border-radius: 12px;
    overflow: hidden;
    max-width: 90vw;
    max-height: 90vh;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
  }

  .close-btn {
    width: 32px;
    height: 32px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 20px;
  }

  .close-btn:hover {
    background: var(--danger);
  }

  video, audio {
    display: block;
  }
</style>
