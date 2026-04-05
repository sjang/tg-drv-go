<script lang="ts">
  import { files, currentFolder, viewMode, isLoading, refreshFiles } from '../lib/stores';
  import { DeleteFile, RenameFile, DownloadFileTo, DownloadFilesTo, OpenPlayer } from '../lib/api';
  import type { FileItem } from '../lib/types';

  let contextMenu: { x: number; y: number; file: FileItem } | null = null;
  let renameId: string | null = null;
  let renameValue = '';
  let deleteConfirm: FileItem | null = null;
  let selectedIds: Set<string> = new Set();
  let deleteMultiConfirm = false;
  let lastClickedId: string | null = null;

  $: selectedCount = selectedIds.size;

  function clearSelection() {
    selectedIds = new Set();
  }

  // Clear selection when folder changes (including leaving a folder)
  let prevFolderId: string | null = null;
  $: {
    const id = $currentFolder?.id ?? null;
    if (id !== prevFolderId) {
      prevFolderId = id;
      selectedIds = new Set();
      lastClickedId = null;
      contextMenu = null;
      deleteConfirm = null;
      deleteMultiConfirm = false;
    }
  }

  function toggleSelect(e: MouseEvent, file: FileItem) {
    e.stopPropagation();
    const newSet = new Set(selectedIds);

    if (e.shiftKey && lastClickedId) {
      // Range select
      const ids = $files.map(f => f.id);
      const from = ids.indexOf(lastClickedId);
      const to = ids.indexOf(file.id);
      const [start, end] = from < to ? [from, to] : [to, from];
      for (let i = start; i <= end; i++) {
        newSet.add(ids[i]);
      }
    } else {
      if (newSet.has(file.id)) {
        newSet.delete(file.id);
      } else {
        newSet.add(file.id);
      }
    }

    lastClickedId = file.id;
    selectedIds = newSet;
  }

  function selectAll() {
    selectedIds = new Set($files.map(f => f.id));
  }

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
    if (selectedCount > 0) return; // don't open while selecting
    if (isPlayable(file.mime_type) || file.mime_type.startsWith('image/')) {
      await OpenPlayer(file.id);
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
    hideContext();
    try {
      await DownloadFileTo(file.id);
    } catch (e) {
      if (!String(e).includes('no save location')) {
        console.error('download:', e);
      }
    }
  }

  async function downloadSelected() {
    if (selectedCount === 0) return;
    const ids = [...selectedIds];
    if (ids.length === 1) {
      const file = $files.find(f => f.id === ids[0]);
      if (file) await downloadFile(file);
    } else {
      try {
        await DownloadFilesTo(ids);
      } catch (e) {
        if (!String(e).includes('no folder selected')) {
          console.error('download:', e);
        }
      }
    }
  }

  function deleteFile(file: FileItem) {
    hideContext();
    if (selectedCount > 1) {
      deleteMultiConfirm = true;
    } else {
      deleteConfirm = file;
    }
  }

  function requestDeleteSelected() {
    if (selectedCount === 0) return;
    if (selectedCount === 1) {
      const id = [...selectedIds][0];
      deleteConfirm = $files.find(f => f.id === id) || null;
    } else {
      deleteMultiConfirm = true;
    }
  }

  async function confirmDelete() {
    if (!deleteConfirm) return;
    const fileId = deleteConfirm.id;
    deleteConfirm = null;
    try {
      await DeleteFile(fileId);
      selectedIds.delete(fileId);
      selectedIds = selectedIds;
      await refreshFiles();
    } catch (e) {
      console.error('delete file:', e);
    }
  }

  async function confirmDeleteMulti() {
    deleteMultiConfirm = false;
    const ids = [...selectedIds];
    try {
      for (const id of ids) {
        await DeleteFile(id);
      }
      clearSelection();
      await refreshFiles();
    } catch (e) {
      console.error('delete files:', e);
    }
  }

  function cancelDelete() {
    deleteConfirm = null;
    deleteMultiConfirm = false;
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
      await refreshFiles();
    } catch (e) {
      console.error('rename file:', e);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && selectedCount > 0) {
      clearSelection();
    }
    if ((e.metaKey || e.ctrlKey) && e.key === 'a' && $currentFolder && $files.length > 0) {
      e.preventDefault();
      selectAll();
    }
    if (e.key === 'Delete' || e.key === 'Backspace') {
      if (selectedCount > 0 && !renameId) {
        e.preventDefault();
        requestDeleteSelected();
      }
    }
  }
</script>

<svelte:window on:click={hideContext} on:keydown={handleKeydown} />

<div class="file-area">
  {#if selectedCount > 0}
    <div class="selection-bar">
      <span>{selectedCount} selected</span>
      <div class="selection-actions">
        <button class="sel-btn download" on:click={downloadSelected}>Download</button>
        <button class="sel-btn danger" on:click={requestDeleteSelected}>Delete</button>
        <button class="sel-btn" on:click={clearSelection}>Cancel</button>
      </div>
    </div>
  {/if}

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
          <span class="col-check"></span>
          <span class="col-name">Name</span>
          <span class="col-size">Size</span>
          <span class="col-type">Type</span>
          <span class="col-date">Date</span>
        </div>
      {/if}

      {#each $files as file}
        <div
          class="file-item"
          class:selected={selectedIds.has(file.id)}
          on:click={() => openFile(file)}
          on:contextmenu={(e) => showContext(e, file)}
          on:dblclick={() => startRename(file)}
        >
          {#if $viewMode === 'grid'}
            <input type="checkbox" class="file-check" checked={selectedIds.has(file.id)} on:click={(e) => toggleSelect(e, file)} />
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
            <span class="col-check">
              <input type="checkbox" class="file-check" checked={selectedIds.has(file.id)} on:click={(e) => toggleSelect(e, file)} />
            </span>
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
    {#if selectedCount <= 1}
      {#if isPlayable(contextMenu.file.mime_type)}
        <button on:click={() => openFile(contextMenu.file)}>Play</button>
      {/if}
      <button on:click={() => downloadFile(contextMenu.file)}>Download</button>
      <button on:click={() => startRename(contextMenu.file)}>Rename</button>
    {:else}
      <button on:click={downloadSelected}>Download ({selectedCount})</button>
    {/if}
    <button class="danger" on:click={() => deleteFile(contextMenu.file)}>
      Delete{#if selectedCount > 1} ({selectedCount}){/if}
    </button>
  </div>
{/if}

<!-- Single Delete Confirmation -->
{#if deleteConfirm}
  <div class="modal-overlay" on:click={cancelDelete}>
    <div class="confirm-dialog" on:click|stopPropagation>
      <p>Delete "<strong>{deleteConfirm.name}</strong>"?</p>
      <div class="confirm-actions">
        <button class="cancel-btn" on:click={cancelDelete}>Cancel</button>
        <button class="delete-btn" on:click={confirmDelete}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<!-- Multi Delete Confirmation -->
{#if deleteMultiConfirm}
  <div class="modal-overlay" on:click={cancelDelete}>
    <div class="confirm-dialog" on:click|stopPropagation>
      <p>Delete <strong>{selectedCount} files</strong>?</p>
      <div class="confirm-actions">
        <button class="cancel-btn" on:click={cancelDelete}>Cancel</button>
        <button class="delete-btn" on:click={confirmDeleteMulti}>Delete All</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .file-area {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
  }

  .selection-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    margin-bottom: 12px;
    background: var(--accent);
    color: white;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 600;
  }

  .selection-actions {
    display: flex;
    gap: 8px;
  }

  .sel-btn {
    padding: 4px 14px;
    background: rgba(255,255,255,0.2);
    color: white;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
  }

  .sel-btn:hover {
    background: rgba(255,255,255,0.3);
  }

  .sel-btn.download {
    background: #10b981;
  }

  .sel-btn.download:hover {
    opacity: 0.9;
  }

  .sel-btn.danger {
    background: var(--danger);
  }

  .sel-btn.danger:hover {
    opacity: 0.9;
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
    position: relative;
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

  .grid-view .file-check {
    position: absolute;
    top: 8px;
    left: 8px;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .grid-view .file-item:hover .file-check,
  .grid-view .file-item.selected .file-check {
    opacity: 1;
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
    grid-template-columns: 30px 1fr 100px 120px 160px;
    padding: 8px 12px;
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid var(--border);
  }

  .list-view .file-item {
    display: grid;
    grid-template-columns: 30px 1fr 100px 120px 160px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
    cursor: pointer;
    align-items: center;
  }

  .list-view .file-item:hover {
    background: var(--bg-hover);
  }

  .file-item.selected {
    background: rgba(74, 158, 255, 0.12);
  }

  .col-check {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .file-check {
    width: 15px;
    height: 15px;
    cursor: pointer;
    accent-color: var(--accent);
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

  .confirm-dialog {
    background: var(--bg-secondary);
    border-radius: 12px;
    padding: 24px;
    min-width: 320px;
    box-shadow: 0 8px 32px var(--shadow);
  }

  .confirm-dialog p {
    margin-bottom: 20px;
    font-size: 15px;
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .cancel-btn {
    padding: 8px 20px;
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-radius: 8px;
  }

  .cancel-btn:hover {
    background: var(--bg-hover);
  }

  .confirm-actions .delete-btn {
    padding: 8px 20px;
    background: var(--danger);
    color: white;
    border-radius: 8px;
    font-weight: 600;
  }

  .confirm-actions .delete-btn:hover {
    opacity: 0.9;
  }
</style>
