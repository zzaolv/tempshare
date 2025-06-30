// src/lib/api.ts

export const PROXY_API_BASE_URL = ''; // Vite proxy will handle this
export const DIRECT_API_BASE_URL = 'https://localhost:8080';

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

export interface InitStatus {
    needsInit: boolean;
}

export interface DbConfig {
    Type: 'sqlite' | 'mysql' | 'postgres';
    DSN: string;
}

export interface StorageConfig {
    Type: 'local' | 's3' | 'webdav';
    Local: { Path: string };
    S3: { Endpoint: string; Region: string; Bucket: string; AccessKeyID: string; SecretAccessKey: string; UsePathStyle: boolean };
    WebDAV: { URL: string; Username: string; Password: string };
}

export interface InitPayload {
    database: DbConfig;
    storage: StorageConfig;
}

export interface InitValidationResult {
    message: string;
    envVars: string;
    composeExample: string;
}

/**
 * 检查后端是否需要初始化
 */
export async function checkInitStatus(): Promise<InitStatus> {
    try {
        const res = await fetch(`${PROXY_API_BASE_URL}/api/v1/init/status`);
        // 如果后端处于完整模式但代理失败，可能会抛出网络错误，我们假设它已初始化
        if (!res.ok) {
            // 在初始化模式下，这个端点总是可用的。如果失败，说明可能处于完整模式但代理有问题，或者服务没起来。
            // 为了避免卡在加载界面，我们假定它不需要初始化。
            return { needsInit: false };
        }
        return res.json();
    } catch (error) {
        console.error("无法连接到后端API，假定已配置。", error);
        return { needsInit: false };
    }
}

/**
 * 提交配置进行验证
 */
export async function submitInitConfig(payload: InitPayload): Promise<InitValidationResult> {
    const res = await fetch(`${PROXY_API_BASE_URL}/api/v1/init/validate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });

    const data = await res.json();
    if (!res.ok) {
        // 将后端的错误结构转化为一个可抛出的 Error 对象
        throw Object.assign(new Error(data.message || '验证失败'), { field: data.field });
    }
    return data;
}