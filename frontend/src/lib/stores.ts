import { writable } from 'svelte/store';
import type { AuthStatus, Folder, FileItem, UploadProgress } from './types';

export const authStatus = writable<AuthStatus>({ authenticated: false });
export const folders = writable<Folder[]>([]);
export const currentFolder = writable<Folder | null>(null);
export const files = writable<FileItem[]>([]);
export const uploads = writable<Map<string, UploadProgress>>(new Map());
export const isLoading = writable(false);
export const errorMessage = writable<string | null>(null);
export const viewMode = writable<'grid' | 'list'>('grid');
export const isConfigured = writable(true);
