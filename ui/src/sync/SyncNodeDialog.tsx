import React, {useEffect, useState} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    MenuItem,
    FormControlLabel,
    Switch,
    Box,
} from '@mui/material';
import {SyncNodePayload} from './SyncNodeStore';
import {ISyncNode} from '../types';

interface SyncNodeDialogProps {
    open: boolean;
    loading: boolean;
    mode: 'create' | 'edit';
    node?: ISyncNode | null;
    onClose: () => void;
    onSubmit: (payload: SyncNodePayload, nodeId?: number) => Promise<void>;
}

interface FormState {
    name: string;
    address: string;
    type: string;
    sshUser: string;
    sshPort: string;
    authType: string;
    credentialRef: string;
    tags: string;
    ignoreDefaults: boolean;
    ignorePatterns: string;
    ignoreFile: string;
    autoInstall: boolean;
}

const defaultState: FormState = {
    name: '',
    address: '',
    type: 'agent',
    sshUser: 'root',
    sshPort: '22',
    authType: 'key',
    credentialRef: '',
    tags: '',
    ignoreDefaults: true,
    ignorePatterns: '.git,runtime,tmp',
    ignoreFile: '',
    autoInstall: true,
};

const parseList = (value: string): string[] =>
    value
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item.length > 0);

const SyncNodeDialog: React.FC<SyncNodeDialogProps> = ({open, loading, mode, node, onClose, onSubmit}) => {
    const [form, setForm] = useState<FormState>(defaultState);

    useEffect(() => {
        if (node && mode === 'edit') {
            setForm({
                name: node.name,
                address: node.address,
                type: node.type || 'agent',
                sshUser: node.sshUser || 'root',
                sshPort: node.sshPort ? String(node.sshPort) : '22',
                authType: node.authType || 'key',
                credentialRef: node.credentialRef || '',
                tags: (node.tags || []).join(', '),
                ignoreDefaults: node.ignoreDefaults ?? true,
                ignorePatterns: (node.ignorePatterns || []).join(', '),
                ignoreFile: node.ignoreFile || '',
                autoInstall: false,
            });
        } else if (!open) {
            setForm(defaultState);
        } else {
            setForm(defaultState);
        }
    }, [node, mode, open]);

    const handleChange =
        (key: keyof FormState) =>
        (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            const value = event.target.value;
            setForm((prev) => ({...prev, [key]: value}));
        };

    const handleSwitchChange = (key: keyof FormState) => (_: React.ChangeEvent, checked: boolean) =>
        setForm((prev) => ({...prev, [key]: checked}));

    const handleSubmit = async () => {
        const payload: SyncNodePayload = {
            name: form.name.trim(),
            address: form.address.trim(),
            type: form.type,
            sshUser: form.sshUser.trim(),
            sshPort: Number(form.sshPort) || undefined,
            authType: form.authType.trim(),
            credentialRef: form.credentialRef.trim(),
            tags: parseList(form.tags),
            ignoreDefaults: form.ignoreDefaults,
            ignorePatterns: parseList(form.ignorePatterns),
            ignoreFile: form.ignoreFile.trim() || undefined,
            autoInstall: form.autoInstall,
        };

        await onSubmit(payload, node?.id);
        onClose();
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{mode === 'create' ? '新增节点' : '编辑节点'}</DialogTitle>
            <DialogContent dividers>
                <Box
                    display="grid"
                    gridTemplateColumns={{xs: '1fr', sm: 'repeat(2, 1fr)'}}
                    gap={2}
                    sx={{mt: 1}}>
                    <Box>
                        <TextField
                            label="节点名称"
                            value={form.name}
                            onChange={handleChange('name')}
                            fullWidth
                            required
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label="节点地址"
                            value={form.address}
                            onChange={handleChange('address')}
                            fullWidth
                            required
                            helperText="可填写 IP 或 DNS"
                        />
                    </Box>
                    <Box>
                        <TextField
                            select
                            label="类型"
                            value={form.type}
                            onChange={handleChange('type')}
                            fullWidth>
                            <MenuItem value="agent">Sync Agent</MenuItem>
                            <MenuItem value="ssh">SSH / rsync</MenuItem>
                        </TextField>
                    </Box>
                    <Box>
                        <TextField
                            label="认证方式"
                            value={form.authType}
                            onChange={handleChange('authType')}
                            fullWidth
                            placeholder="key/password"
                        />
                    </Box>
                    <Box>
                        <TextField
                            label="SSH 用户"
                            value={form.sshUser}
                            onChange={handleChange('sshUser')}
                            fullWidth
                        />
                    </Box>
                    <Box>
                        <TextField
                            label="SSH 端口"
                            value={form.sshPort}
                            onChange={handleChange('sshPort')}
                            type="number"
                            fullWidth
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label="凭证引用"
                            value={form.credentialRef}
                            onChange={handleChange('credentialRef')}
                            fullWidth
                            helperText="引用 server 端保存的密钥/凭证 ID"
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label="标签（逗号分隔）"
                            value={form.tags}
                            onChange={handleChange('tags')}
                            fullWidth
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label="忽略模式（逗号分隔）"
                            value={form.ignorePatterns}
                            onChange={handleChange('ignorePatterns')}
                            fullWidth
                            helperText="默认忽略 .git、runtime、tmp 等目录"
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label="忽略文件路径"
                            value={form.ignoreFile}
                            onChange={handleChange('ignoreFile')}
                            fullWidth
                            placeholder="可选，远程节点上的 ignore 文件"
                        />
                    </Box>
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={form.ignoreDefaults}
                                    onChange={handleSwitchChange('ignoreDefaults')}
                                />
                            }
                            label="启用默认忽略列表 (.git、runtime、tmp)"
                        />
                    </Box>
                    {mode === 'create' ? (
                        <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={form.autoInstall}
                                        onChange={handleSwitchChange('autoInstall')}
                                    />
                                }
                                label="创建后自动安装 Sync Agent"
                            />
                        </Box>
                    ) : null}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="secondary">
                    取消
                </Button>
                <Button onClick={handleSubmit} color="primary" variant="contained" disabled={loading}>
                    {loading ? '保存中...' : '保存'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default SyncNodeDialog;
