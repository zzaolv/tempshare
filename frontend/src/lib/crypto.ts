// src/lib/crypto.ts

// --- 常量定义 (无变化) ---
const SALT_LENGTH = 16;
const IV_LENGTH = 12; // AES-GCM 推荐长度
const PBKDF2_ITERATIONS = 100000;
const PLAINTEXT_CHUNK_SIZE = 64 * 1024; // 64KB, 明文分块大小
const LENGTH_PREFIX_SIZE = 4; // 4字节 (Uint32) 用于存储每个加密块的长度

// --- 密钥派生函数 (无变化) ---
async function deriveKeyFromPassword(password: string, salt: Uint8Array): Promise<CryptoKey> {
    const passwordEncoder = new TextEncoder();
    const passwordBuffer = passwordEncoder.encode(password);
    
    const baseKey = await crypto.subtle.importKey(
        'raw', passwordBuffer, { name: 'PBKDF2' }, false, ['deriveKey']
    );

    return await crypto.subtle.deriveKey(
        { name: 'PBKDF2', salt, iterations: PBKDF2_ITERATIONS, hash: 'SHA-256' },
        baseKey,
        { name: 'AES-GCM', length: 256 },
        true,
        ['encrypt', 'decrypt']
    );
}

// ✨ 核心修改点: 新增创建验证哈希的函数 ✨
async function createVerificationHash(password: string, salt: Uint8Array): Promise<string> {
    const encoder = new TextEncoder();
    const passwordBuffer = encoder.encode(password);
    
    // 将密码和盐连接在一起进行哈希，增加破解难度
    const combinedBuffer = new Uint8Array(passwordBuffer.length + salt.length);
    combinedBuffer.set(passwordBuffer);
    combinedBuffer.set(salt, passwordBuffer.length);
    
    const hashBuffer = await crypto.subtle.digest('SHA-256', combinedBuffer);
    
    // 将 ArrayBuffer 转换为十六进制字符串
    return Array.from(new Uint8Array(hashBuffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}


// --- 盐生成 (无变化) ---
export function generateSalt(): Uint8Array {
    return crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
}

// --- 文件分块流 (无变化) ---
export function createChunkingStream(chunkSize = PLAINTEXT_CHUNK_SIZE): TransformStream<Uint8Array, Uint8Array> {
    let buffer = new Uint8Array();

    return new TransformStream({
        transform(chunk, controller) {
            const newBuffer = new Uint8Array(buffer.length + chunk.length);
            newBuffer.set(buffer);
            newBuffer.set(chunk, buffer.length);
            buffer = newBuffer;

            while (buffer.length >= chunkSize) {
                const chunkToEnqueue = buffer.slice(0, chunkSize);
                buffer = buffer.slice(chunkSize);
                controller.enqueue(chunkToEnqueue);
            }
        },
        flush(controller) {
            if (buffer.length > 0) {
                controller.enqueue(buffer);
            }
        }
    });
}

// --- 加密流 (无变化) ---
export function createEncryptionStream(key: CryptoKey): TransformStream<Uint8Array, Uint8Array> {
    return new TransformStream({
        async transform(chunk, controller) {
            const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH));
            const encryptedData = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, chunk);
            
            const combined = new Uint8Array(iv.length + encryptedData.byteLength);
            combined.set(iv);
            combined.set(new Uint8Array(encryptedData), iv.length);

            const lengthPrefix = new Uint8Array(LENGTH_PREFIX_SIZE);
            const view = new DataView(lengthPrefix.buffer);
            view.setUint32(0, combined.byteLength, false);

            controller.enqueue(lengthPrefix);
            controller.enqueue(combined);
        }
    });
}

// --- 解密流 (无变化) ---
export function createDecryptionStream(key: CryptoKey): TransformStream<Uint8Array, Uint8Array> {
    let internalBuffer = new Uint8Array(0);

    return new TransformStream({
        async transform(chunk, controller) {
            const newBuffer = new Uint8Array(internalBuffer.length + chunk.length);
            newBuffer.set(internalBuffer);
            newBuffer.set(chunk, internalBuffer.length);
            internalBuffer = newBuffer;

            while (true) {
                if (internalBuffer.length < LENGTH_PREFIX_SIZE) {
                    break;
                }
                
                const lengthView = new DataView(internalBuffer.buffer, internalBuffer.byteOffset, LENGTH_PREFIX_SIZE);
                const frameLength = lengthView.getUint32(0, false);

                const totalFrameLength = LENGTH_PREFIX_SIZE + frameLength;
                if (internalBuffer.length < totalFrameLength) {
                    break;
                }

                const frame = internalBuffer.slice(LENGTH_PREFIX_SIZE, totalFrameLength);
                internalBuffer = internalBuffer.slice(totalFrameLength);
                
                try {
                    const iv = frame.slice(0, IV_LENGTH);
                    const data = frame.slice(IV_LENGTH);
                    const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, data);
                    controller.enqueue(new Uint8Array(decrypted));
                } catch (err) {
                    controller.error(new Error("Decryption failed. Incorrect password or corrupt data."));
                    return;
                }
            }
        },
        flush(controller) {
            if (internalBuffer.length > 0) {
                controller.error(new Error("Incomplete data stream at the end. The file might be corrupted."));
            }
        }
    });
}


// --- Base64 转换函数 (无变化) ---
export function bufferToBase64(buffer: ArrayBuffer): string {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

export function base64ToBuffer(base64: string): ArrayBuffer {
    const binaryString = window.atob(base64);
    const len = binaryString.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes.buffer;
}

// --- 导出模块 ---
export const E2EE = {
    generateSalt,
    deriveKeyFromPassword,
    createVerificationHash, // ✨ 导出新函数
    bufferToBase64,
    base64ToBuffer,
    createChunkingStream,
    createEncryptionStream,
    createDecryptionStream
};