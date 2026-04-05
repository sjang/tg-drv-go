export interface AuthStatus {
  authenticated: boolean;
  phone_number?: string;
  first_name?: string;
  last_name?: string;
  username?: string;
}

export interface Folder {
  id: string;
  name: string;
  channel_id: number;
  access_hash: number;
  created_at: string;
  updated_at: string;
  file_count?: number;
  total_size?: number;
}

export interface FileItem {
  id: string;
  folder_id: string;
  name: string;
  size: number;
  mime_type: string;
  sha256_hash: string;
  message_id: number;
  has_thumbnail: boolean;
  upload_date: string;
  is_duplicate: boolean;
}

export interface UploadProgress {
  file_id: string;
  file_name: string;
  uploaded: number;
  total: number;
  percent: number;
}
