<script lang="ts">
  import { folders, currentFolder, files, isLoading } from '../lib/stores';
  import { GetFolders, CreateFolder, DeleteFolder, RenameFolder, ToggleFolderHidden, GetFiles, SyncFolders } from '../lib/api';
  import type { Folder } from '../lib/types';

  let newFolderName = '';
  let showNewFolder = false;
  let editingId: string | null = null;
  let editName = '';
  let deletingFolderId: string | null = null;
  let syncing = false;
  let showHidden = false;

  $: visibleFolders = $folders.filter(f => !f.hidden);
  $: hiddenFolders = $folders.filter(f => f.hidden);

  async function toggleHidden(id: string, hidden: boolean) {
    try {
      await ToggleFolderHidden(id, hidden);
      await loadFolders();
      if (hidden && $currentFolder?.id === id) {
        currentFolder.set(null);
        files.set([]);
      }
    } catch (e) {
      console.error('toggle hidden:', e);
    }
  }

  export async function loadFolders() {
    try {
      const result = await GetFolders();
      folders.set(result || []);
    } catch (e) {
      console.error('load folders:', e);
    }
  }

  async function selectFolder(folder: Folder) {
    currentFolder.set(folder);
    isLoading.set(true);
    try {
      const result = await GetFiles(folder.id);
      files.set(result || []);
    } catch (e) {
      console.error('load files:', e);
    } finally {
      isLoading.set(false);
    }
  }

  async function createFolder() {
    if (!newFolderName.trim()) return;
    try {
      await CreateFolder(newFolderName.trim());
      newFolderName = '';
      showNewFolder = false;
      await loadFolders();
    } catch (e) {
      console.error('create folder:', e);
    }
  }

  function deleteFolder(id: string) {
    deletingFolderId = id;
  }

  async function confirmDeleteFolder() {
    if (!deletingFolderId) return;
    const id = deletingFolderId;
    deletingFolderId = null;
    try {
      await DeleteFolder(id);
      if ($currentFolder?.id === id) {
        currentFolder.set(null);
        files.set([]);
      }
      await loadFolders();
    } catch (e) {
      console.error('delete folder:', e);
    }
  }

  function cancelDeleteFolder() {
    deletingFolderId = null;
  }

  async function startRename(folder: Folder) {
    editingId = folder.id;
    editName = folder.name;
  }

  async function finishRename() {
    if (!editingId || !editName.trim()) {
      editingId = null;
      return;
    }
    try {
      await RenameFolder(editingId, editName.trim());
      editingId = null;
      await loadFolders();
    } catch (e) {
      console.error('rename folder:', e);
    }
  }

  async function syncFolders() {
    if (syncing) return;
    syncing = true;
    try {
      await SyncFolders();
      await loadFolders();
    } catch (e) {
      console.error('sync folders:', e);
    } finally {
      syncing = false;
    }
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

  loadFolders();
</script>

<div class="sidebar">
  <div class="sidebar-header">
    <h2>Folders</h2>
    <div class="header-actions">
      <button class="icon-btn" class:spinning={syncing} on:click={syncFolders} title="Sync from Telegram" disabled={syncing}>
        <span class="sync-icon">&#x21BB;</span>
      </button>
      <button class="icon-btn" on:click={() => showNewFolder = !showNewFolder} title="New Folder">+</button>
    </div>
  </div>

  {#if showNewFolder}
    <div class="new-folder">
      <input
        type="text"
        bind:value={newFolderName}
        placeholder="Folder name"
        on:keydown={(e) => {
          if (e.key === 'Enter') createFolder();
          if (e.key === 'Escape') { newFolderName = ''; showNewFolder = false; }
        }}
        autofocus
      />
      <div class="new-folder-btns">
        <button on:click={createFolder}>Create</button>
        <button class="new-folder-cancel" on:click={() => { newFolderName = ''; showNewFolder = false; }}>&times;</button>
      </div>
    </div>
  {/if}

  <div class="folder-list">
    {#each visibleFolders as folder}
      <div
        class="folder-item"
        class:active={$currentFolder?.id === folder.id}
        on:click={() => selectFolder(folder)}
        on:dblclick={() => startRename(folder)}
      >
        {#if editingId === folder.id}
          <input
            type="text"
            class="rename-input"
            bind:value={editName}
            on:blur={finishRename}
            on:keydown={(e) => e.key === 'Enter' && finishRename()}
            autofocus
          />
        {:else}
          <div class="folder-info">
            <span class="folder-icon">&#x1F4C1;</span>
            <div class="folder-details">
              <span class="folder-name">{folder.name}</span>
              <span class="folder-meta">
                {folder.file_count || 0} files &middot; {formatSize(folder.total_size || 0)}
              </span>
            </div>
          </div>
          <div class="folder-actions">
            <button class="hide-btn" on:click|stopPropagation={() => toggleHidden(folder.id, true)} title="Hide">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
            </button>
            <button class="delete-btn" on:click|stopPropagation={() => deleteFolder(folder.id)}>&times;</button>
          </div>
        {/if}
      </div>
    {/each}

    {#if visibleFolders.length === 0 && hiddenFolders.length === 0}
      <div class="empty-state">No folders yet</div>
    {/if}

    {#if hiddenFolders.length > 0}
      <button class="hidden-toggle" on:click={() => showHidden = !showHidden}>
        {showHidden ? '\u25BC' : '\u25B6'} Hidden ({hiddenFolders.length})
      </button>
      {#if showHidden}
        {#each hiddenFolders as folder}
          <div
            class="folder-item hidden-folder"
            class:active={$currentFolder?.id === folder.id}
            on:click={() => selectFolder(folder)}
          >
            <div class="folder-info">
              <span class="folder-icon">&#x1F4C1;</span>
              <div class="folder-details">
                <span class="folder-name">{folder.name}</span>
                <span class="folder-meta">
                  {folder.file_count || 0} files &middot; {formatSize(folder.total_size || 0)}
                </span>
              </div>
            </div>
            <div class="folder-actions">
              <button class="show-btn" on:click|stopPropagation={() => toggleHidden(folder.id, false)} title="Show">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
            </div>
          </div>
        {/each}
      {/if}
    {/if}
  </div>
</div>

{#if deletingFolderId}
  {@const target = $folders.find(f => f.id === deletingFolderId)}
  <div class="modal-overlay" on:click={cancelDeleteFolder}>
    <div class="confirm-dialog" on:click|stopPropagation>
      <div class="warn-icon">&#x26A0;</div>
      <p class="warn-title">Are you sure?</p>
      <p class="warn-detail">
        Folder "<strong>{target?.name || ''}</strong>" will be permanently deleted.
        This will also delete the Telegram channel and all {target?.file_count || 0} files in it.
        This action cannot be undone.
      </p>
      <div class="confirm-actions">
        <button class="cancel-btn" on:click={cancelDeleteFolder}>Cancel</button>
        <button class="del-confirm-btn" on:click={confirmDeleteFolder}>Yes, Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .sidebar {
    width: 260px;
    min-width: 260px;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 16px 12px;
    border-bottom: 1px solid var(--border);
  }

  .sidebar-header h2 {
    font-size: 16px;
    font-weight: 600;
  }

  .header-actions {
    display: flex;
    gap: 4px;
  }

  .icon-btn {
    width: 32px;
    height: 32px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-btn:hover {
    background: var(--bg-hover);
  }

  .icon-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .sync-icon {
    display: inline-block;
  }

  .spinning .sync-icon {
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .new-folder {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    border-bottom: 1px solid var(--border);
  }

  .new-folder input {
    width: 100%;
    padding: 8px 12px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    box-sizing: border-box;
  }

  .new-folder-btns {
    display: flex;
    gap: 6px;
  }

  .new-folder-btns button {
    flex: 1;
    padding: 8px 12px;
    background: var(--accent);
    color: white;
    border-radius: 6px;
    font-weight: 500;
  }

  .new-folder-cancel {
    flex: none !important;
    width: 36px;
    background: var(--bg-tertiary) !important;
    color: var(--text-secondary) !important;
    font-size: 16px;
  }

  .new-folder-cancel:hover {
    background: var(--bg-hover) !important;
    color: var(--text-primary) !important;
  }

  .folder-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }

  .folder-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border-radius: 8px;
    cursor: pointer;
    margin-bottom: 2px;
  }

  .folder-item:hover {
    background: var(--bg-hover);
  }

  .folder-item.active {
    background: var(--accent);
  }

  .folder-info {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 0;
  }

  .folder-icon {
    font-size: 20px;
    flex-shrink: 0;
  }

  .folder-details {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .folder-name {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .folder-meta {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .folder-item.active .folder-meta {
    color: rgba(255, 255, 255, 0.7);
  }

  .folder-actions {
    display: flex;
    gap: 2px;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .folder-item:hover .folder-actions {
    opacity: 1;
  }

  .delete-btn, .hide-btn, .show-btn {
    width: 24px;
    height: 24px;
    background: transparent;
    color: var(--text-secondary);
    border-radius: 4px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .delete-btn:hover {
    background: var(--danger);
    color: white;
  }

  .hide-btn:hover {
    background: var(--bg-tertiary);
  }

  .show-btn {
    opacity: 0;
    transition: opacity 0.2s;
  }

  .folder-item:hover .show-btn {
    opacity: 1;
  }

  .show-btn:hover {
    background: var(--accent);
    color: white;
  }

  .hidden-toggle {
    display: block;
    width: 100%;
    padding: 8px 12px;
    margin-top: 8px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    text-align: left;
    border-radius: 6px;
  }

  .hidden-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .hidden-folder {
    opacity: 0.6;
  }

  .rename-input {
    width: 100%;
    padding: 6px 10px;
    background: var(--bg-tertiary);
    border: 1px solid var(--accent);
    border-radius: 4px;
    color: var(--text-primary);
  }

  .empty-state {
    color: var(--text-secondary);
    text-align: center;
    padding: 32px 16px;
    font-size: 13px;
  }

  .modal-overlay {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
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

  .warn-icon {
    font-size: 36px;
    text-align: center;
    margin-bottom: 12px;
  }

  .warn-title {
    font-size: 17px;
    font-weight: 600;
    text-align: center;
    margin-bottom: 12px;
  }

  .warn-detail {
    font-size: 14px;
    color: var(--text-secondary);
    line-height: 1.5;
    margin-bottom: 20px;
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

  .del-confirm-btn {
    padding: 8px 20px;
    background: var(--danger);
    color: white;
    border-radius: 8px;
    font-weight: 600;
  }

  .del-confirm-btn:hover {
    opacity: 0.9;
  }
</style>
