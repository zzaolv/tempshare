// frontend/src/pages/SetupPage.tsx

import { useState } from 'react';
import type { FormEvent } from 'react'; // <-- 修正点 1
import { motion, AnimatePresence } from 'framer-motion';
import { Database, FolderKanban, Server, Save, Copy, Check } from 'lucide-react'; // <-- 修正点 2: 移除了 AlertTriangle
import type { InitPayload, DbConfig, StorageConfig } from '../lib/api';
import { submitInitConfig } from '../lib/api';

// 默认值
const initialDbConfig: DbConfig = { Type: 'sqlite', DSN: 'data/tempshare.db' };
const initialStorageConfig: StorageConfig = {
    Type: 'local',
    Local: { Path: 'data/tempshare-files' },
    S3: { Endpoint: '', Region: '', Bucket: '', AccessKeyID: '', SecretAccessKey: '', UsePathStyle: false },
    WebDAV: { URL: '', Username: '', Password: '' },
};

const SetupPage = () => {
    const [dbConfig, setDbConfig] = useState<DbConfig>(initialDbConfig);
    const [storageConfig, setStorageConfig] = useState<StorageConfig>(initialStorageConfig);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<{ field: string; message: string } | null>(null);
    const [result, setResult] = useState<{ envVars: string; composeExample: string } | null>(null);

    const [copiedEnv, setCopiedEnv] = useState(false);
    const [copiedCompose, setCopiedCompose] = useState(false);

    const handleCopy = (text: string, type: 'env' | 'compose') => {
        navigator.clipboard.writeText(text);
        if (type === 'env') {
            setCopiedEnv(true);
            setTimeout(() => setCopiedEnv(false), 2000);
        } else {
            setCopiedCompose(true);
            setTimeout(() => setCopiedCompose(false), 2000);
        }
    };

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        setError(null);
        setResult(null);

        const payload: InitPayload = { database: dbConfig, storage: storageConfig };
        try {
            const validationResult = await submitInitConfig(payload);
            setResult(validationResult);
        } catch (err: any) {
            setError({ field: err.field || 'general', message: err.message || '发生未知错误' });
        } finally {
            setIsLoading(false);
        }
    };

    const renderError = (fieldName: string) => {
        if (error && error.field === fieldName) {
            return <p className="text-red-600 text-sm mt-1">{error.message}</p>;
        }
        return null;
    };

    return (
        <div className="min-h-screen bg-slate-50 flex items-center justify-center p-4">
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
                className="w-full max-w-4xl"
            >
                {result ? (
                    <div className="bg-white p-8 rounded-xl shadow-lg">
                        <h1 className="text-3xl font-bold text-green-600 text-center">🎉 配置成功！</h1>
                        <p className="text-slate-600 text-center mt-2">
                            您的配置已通过验证。请使用以下信息更新您的 <code>.env</code> 文件或 <code>docker-compose.yml</code> 并重启服务。
                        </p>

                        <div className="mt-6">
                            <h2 className="text-xl font-semibold text-slate-800">环境变量 (.env)</h2>
                            <div className="relative mt-2">
                                <pre className="bg-slate-800 text-slate-200 p-4 rounded-lg overflow-x-auto text-sm">{result.envVars}</pre>
                                <button onClick={() => handleCopy(result.envVars, 'env')} className="absolute top-2 right-2 p-2 bg-slate-600 hover:bg-slate-500 rounded-md text-white">
                                    {copiedEnv ? <Check size={16} /> : <Copy size={16} />}
                                </button>
                            </div>
                        </div>

                        <div className="mt-6">
                            <h2 className="text-xl font-semibold text-slate-800">Docker Compose 示例 (docker-compose.yml)</h2>
                             <div className="relative mt-2">
                                <pre className="bg-slate-800 text-slate-200 p-4 rounded-lg overflow-x-auto text-sm">{result.composeExample}</pre>
                                <button onClick={() => handleCopy(result.composeExample, 'compose')} className="absolute top-2 right-2 p-2 bg-slate-600 hover:bg-slate-500 rounded-md text-white">
                                    {copiedCompose ? <Check size={16} /> : <Copy size={16} />}
                                </button>
                            </div>
                        </div>
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="bg-white p-8 rounded-xl shadow-lg space-y-8">
                        <div>
                            <h1 className="text-3xl font-bold text-slate-800 text-center">闪传驿站 - 首次设置</h1>
                            <p className="text-slate-600 text-center mt-2">请配置您的数据库和文件存储选项。</p>
                        </div>

                        {/* Database Configuration */}
                        <div className="space-y-4 p-6 border rounded-lg">
                            <h2 className="text-xl font-semibold text-slate-700 flex items-center gap-2"><Database className="text-brand-cyan" />数据库配置</h2>
                            <div>
                                <label className="font-medium">数据库类型</label>
                                <select value={dbConfig.Type} onChange={e => setDbConfig({ ...dbConfig, Type: e.target.value as DbConfig['Type'] })} className="w-full mt-1 p-2 border rounded-md focus:ring-brand-cyan focus:border-brand-cyan">
                                    <option value="sqlite">SQLite (推荐, 无需额外配置)</option>
                                    <option value="mysql">MySQL</option>
                                    <option value="postgres">PostgreSQL</option>
                                </select>
                            </div>
                            <AnimatePresence>
                                {dbConfig.Type !== 'sqlite' && (
                                    <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }}>
                                        <label className="font-medium">连接字符串 (DSN)</label>
                                        <input type="text" placeholder="user:password@tcp(hostname:3306)/dbname" value={dbConfig.DSN} onChange={e => setDbConfig({ ...dbConfig, DSN: e.target.value })} className="w-full mt-1 p-2 border rounded-md" />
                                        <p className="text-xs text-slate-500 mt-1">示例 (MySQL): <code>root:password@tcp(tempshare-db:3306)/tempshare?charset=utf8mb4&parseTime=True&loc=Local</code></p>
                                        <p className="text-xs text-slate-500">示例 (PostgreSQL): <code>postgres://user:password@tempshare-db:5432/tempshare?sslmode=disable</code></p>
                                    </motion.div>
                                )}
                            </AnimatePresence>
                            {renderError('database')}
                        </div>

                        {/* Storage Configuration */}
                        <div className="space-y-4 p-6 border rounded-lg">
                             <h2 className="text-xl font-semibold text-slate-700 flex items-center gap-2"><FolderKanban className="text-brand-yellow" />存储配置</h2>
                            <div>
                                <label className="font-medium">存储类型</label>
                                <select value={storageConfig.Type} onChange={e => setStorageConfig({ ...storageConfig, Type: e.target.value as StorageConfig['Type'] })} className="w-full mt-1 p-2 border rounded-md">
                                    <option value="local">本地文件系统 (推荐, 无需额外配置)</option>
                                    <option value="s3" disabled>对象存储 (S3) (即将支持)</option>
                                    <option value="webdav" disabled>WebDAV (即将支持)</option>
                                </select>
                            </div>
                             <AnimatePresence>
                                {storageConfig.Type === 'local' && (
                                    <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }}>
                                        <label className="font-medium">存储路径</label>
                                        <input type="text" value={storageConfig.Local.Path} onChange={e => setStorageConfig({ ...storageConfig, Local: { Path: e.target.value } })} className="w-full mt-1 p-2 border rounded-md" />
                                        <p className="text-xs text-slate-500 mt-1">这是相对于后端应用工作目录的路径。在 Docker 中，推荐使用容器内的路径，如 <code>data/tempshare-files</code>。</p>
                                    </motion.div>
                                )}
                            </AnimatePresence>
                            {renderError('storage')}
                        </div>

                        {error && error.field === 'general' && (
                             <div className="bg-red-100 border-l-4 border-red-500 text-red-700 p-4 rounded-md" role="alert">
                                <p className="font-bold">发生错误</p>
                                <p>{error.message}</p>
                            </div>
                        )}

                        <div className="flex justify-end">
                            <button type="submit" disabled={isLoading} className="inline-flex items-center gap-2 px-6 py-3 bg-brand-cyan text-white font-bold rounded-lg hover:brightness-110 disabled:bg-slate-400 transition-all">
                                {isLoading ? <Server className="animate-spin" /> : <Save />}
                                {isLoading ? '正在验证...' : '验证并生成配置'}
                            </button>
                        </div>
                    </form>
                )}
            </motion.div>
        </div>
    );
};

export default SetupPage;