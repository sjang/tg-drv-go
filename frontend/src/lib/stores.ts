import { writable, get } from 'svelte/store';
import type { AuthStatus, Folder, FileItem, UploadProgress } from './types';

export const authStatus = writable<AuthStatus>({ authenticated: false });
export const folders = writable<Folder[]>([]);
export const currentFolder = writable<Folder | null>(null);
export const files = writable<FileItem[]>([]);
export const uploads = writable<Map<string, UploadProgress>>(new Map());
export const isLoading = writable(false);
export const errorMessage = writable<string | null>(null);
export const viewMode = writable<'grid' | 'list'>('list');
export const isConfigured = writable(true);
export const uploadQueue = writable<{ current: number; total: number }>({ current: 0, total: 0 });

// Refresh files and folder stats after upload/delete
export async function refreshFiles() {
  const folder = get(currentFolder);
  if (!folder) return;
  const { GetFiles, GetFolders } = await import('./api');
  const [fileResult, folderResult] = await Promise.all([
    GetFiles(folder.id),
    GetFolders(),
  ]);
  files.set(fileResult || []);
  folders.set(folderResult || []);
}
