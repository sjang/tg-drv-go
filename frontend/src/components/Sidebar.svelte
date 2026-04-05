<script lang="ts">
  import { folders, currentFolder, files, isLoading } from '../lib/stores';
  import { GetFolders, CreateFolder, DeleteFolder, RenameFolder, GetFiles, SyncFolders } from '../lib/api';
  import type { Folder } from '../lib/types';

  let newFolderName = '';
  let showNewFolder = false;
  let editingId: string | null = null;
  let editName = '';

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

  async function deleteFolder(id: string) {
    if (!confirm('Delete this folder and all its files?')) return;
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
    try {
      await SyncFolders();
      await loadFolders();
    } catch (e) {
      console.error('sync folders:', e);
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
      <button class="icon-btn" on:click={syncFolders} title="Sync from Telegram">&#x21BB;</button>
      <button class="icon-btn" on:click={() => showNewFolder = !showNewFolder} title="New Folder">+</button>
    </div>
  </div>

  {#if showNewFolder}
    <div class="new-folder">
      <input
        type="text"
        bind:value={newFolderName}
        placeholder="Folder name"
        on:keydown={(e) => e.key === 'Enter' && createFolder()}
        autofocus
      />
      <button on:click={createFolder}>Create</button>
    </div>
  {/if}

  <div class="folder-list">
    {#each $folders as folder}
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
          <button class="delete-btn" on:click|stopPropagation={() => deleteFolder(folder.id)}>&times;</button>
        {/if}
      </div>
    {/each}

    {#if $folders.length === 0}
      <div class="empty-state">No folders yet</div>
    {/if}
  </div>
</div>

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

  .new-folder {
    display: flex;
    gap: 8px;
    padding: 12px;
    border-bottom: 1px solid var(--border);
  }

  .new-folder input {
    flex: 1;
    padding: 8px 12px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
  }

  .new-folder button {
    padding: 8px 12px;
    background: var(--accent);
    color: white;
    border-radius: 6px;
    font-weight: 500;
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

  .delete-btn {
    width: 24px;
    height: 24px;
    background: transparent;
    color: var(--text-secondary);
    border-radius: 4px;
    font-size: 16px;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .folder-item:hover .delete-btn {
    opacity: 1;
  }

  .delete-btn:hover {
    background: var(--danger);
    color: white;
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
</style>
