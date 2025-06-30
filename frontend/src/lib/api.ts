// src/lib/api.ts

export const PROXY_API_BASE_URL = ''; // Vite proxy will handle this
const directBase = import.meta.env.VITE_DIRECT_API_BASE_URL || '';
export const DIRECT_API_BASE_URL = directBase;
// --- 类型定义 ---

export interface FileMetadata {
    accessCode: string;
    filename: string;
    sizeBytes: number;
    originalSizeBytes: number;
    createdAt: string;
    isEncrypted: boolean;
    encryptionSalt: string;
    downloadOnce: boolean;
    expiresAt: string;
    scanStatus: 'pending' | 'clean' | 'infected' | 'error' | 'skipped';
    scanResult: string;
}

export interface PublicFileInfo {
    accessCode: string;
    filename: string;
    sizeBytes: number;
    expiresAt: string;
    isEncrypted: boolean;
}

export interface ShareDetails {
    id: string;
    accessCode: string;
    urlPath: string;
}

// --- API 请求函数 ---

export async function fetchFileMetadata(accessCode: string): Promise<FileMetadata> {
    const res = await fetch(`${PROXY_API_BASE_URL}/api/v1/files/meta/${accessCode}`);
    if (!res.ok) {
        const errorData = await res.json().catch(() => ({ message: '无法获取文件信息' }));
        throw new Error(errorData.message);
    }
    return res.json();
}

export async function fetchPublicFiles(): Promise<PublicFileInfo[]> {
    const res = await fetch(`${PROXY_API_BASE_URL}/api/v1/files/public`);
    if (!res.ok) {
        throw new Error("无法加载公共文件列表");
    }
    return (await res.json()) || [];
}

export async function submitReport(accessCode: string, reason: string): Promise<{ message: string }> {
    const res = await fetch(`${PROXY_API_BASE_URL}/api/v1/report`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ accessCode: accessCode.toUpperCase(), reason })
    });
    return res.json();
}