import React, {useEffect, useState} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    MenuItem,
    Box,
    Typography,
    InputAdornment,
    IconButton,
    Tooltip,
} from '@mui/material';
import {SyncNodePayload} from './SyncNodeStore';
import {ISyncNode} from '../types';
import useTranslation from '../i18n/useTranslation';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import RefreshIcon from '@mui/icons-material/Refresh';

interface SyncNodeDialogProps {
    open: boolean;
    loading: boolean;
    mode: 'create' | 'edit';
    node?: ISyncNode | null;
    onClose: () => void;
    onSubmit: (payload: SyncNodePayload, nodeId?: number) => Promise<ISyncNode | undefined>;
    onRotateToken: (id: number) => Promise<ISyncNode>;
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
}

const defaultState: FormState = {
    name: '',
    address: '',
    type: 'agent',
    sshUser: '',
    sshPort: '',
    authType: 'key',
    credentialRef: '',
    tags: '',
};

const parseList = (value: string): string[] =>
    value
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item.length > 0);

const SyncNodeDialog: React.FC<SyncNodeDialogProps> = ({
    open,
    loading,
    mode,
    node,
    onClose,
    onSubmit,
    onRotateToken,
}) => {
    const {t} = useTranslation();
    const [form, setForm] = useState<FormState>(defaultState);
    const [createdNode, setCreatedNode] = useState<ISyncNode | null>(null);
    const [token, setToken] = useState<string | null>(null);
    const [tokenVisible, setTokenVisible] = useState(false);
    const [tokenCopied, setTokenCopied] = useState(false);
    const [copyError, setCopyError] = useState(false);

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
            });
            setCreatedNode(null);
            setToken(node.agentToken || null);
            setTokenVisible(false);
            setTokenCopied(false);
            setCopyError(false);
        } else if (!open) {
            setForm(defaultState);
            setCreatedNode(null);
            setToken(null);
            setTokenVisible(false);
            setTokenCopied(false);
            setCopyError(false);
        } else {
            setForm(defaultState);
            setCreatedNode(null);
            setToken(null);
            setTokenVisible(false);
            setTokenCopied(false);
            setCopyError(false);
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

    const showSSHFields = form.type === 'ssh';

    const handleSubmit = async () => {
        const payload: SyncNodePayload = {
            name: form.name.trim(),
            type: form.type,
            address: showSSHFields ? form.address.trim() : undefined,
            sshUser: showSSHFields ? form.sshUser.trim() : undefined,
            sshPort: showSSHFields ? Number(form.sshPort) || undefined : undefined,
            authType: showSSHFields ? form.authType.trim() : undefined,
            credentialRef: showSSHFields ? form.credentialRef.trim() : undefined,
            tags: parseList(form.tags),
        };

        const result = await onSubmit(payload, node?.id);
        if (mode === 'create' && result) {
            setCreatedNode(result);
            setToken(result.agentToken || null);
            return;
        }
        if (mode === 'edit' && result) {
            setToken(result.agentToken || null);
        }
        onClose();
    };

    const currentNode = createdNode || node || null;
    const showTokenSection = !!token && !!currentNode;

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {mode === 'create' ? t('syncNodes.createTitle') : t('syncNodes.editTitle')}
            </DialogTitle>
            <DialogContent dividers>
                <Box
                    display="grid"
                    gridTemplateColumns={{xs: '1fr', sm: 'repeat(2, 1fr)'}}
                    gap={2}
                    sx={{mt: 1}}>
                    <Box>
                        <TextField
                            label={t('syncNodes.name')}
                            value={form.name}
                            onChange={handleChange('name')}
                            fullWidth
                            required
                        />
                    </Box>
                    {showSSHFields ? (
                        <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                            <TextField
                                label={t('syncNodes.address')}
                                value={form.address}
                                onChange={handleChange('address')}
                                fullWidth
                                required
                                helperText={t('syncNodes.addressHelp')}
                            />
                        </Box>
                    ) : null}
                    <Box>
                        <TextField
                            select
                            label={t('syncNodes.type')}
                            value={form.type}
                            onChange={handleChange('type')}
                            fullWidth>
                            <MenuItem value="agent">{t('syncNodes.typeAgent')}</MenuItem>
                            <MenuItem value="ssh">{t('syncNodes.typeSSH')}</MenuItem>
                        </TextField>
                    </Box>
                    {showSSHFields ? (
                        <Box>
                            <TextField
                                label={t('syncNodes.authType')}
                                value={form.authType}
                                onChange={handleChange('authType')}
                                fullWidth
                                placeholder="key/password"
                            />
                        </Box>
                    ) : null}
                    {showSSHFields ? (
                        <>
                            <Box>
                                <TextField
                                    label={t('syncNodes.sshUser')}
                                    value={form.sshUser}
                                    onChange={handleChange('sshUser')}
                                    fullWidth
                                />
                            </Box>
                            <Box>
                                <TextField
                                    label={t('syncNodes.sshPort')}
                                    value={form.sshPort}
                                    onChange={handleChange('sshPort')}
                                    type="number"
                                    fullWidth
                                />
                            </Box>
                        </>
                    ) : null}
                    {showSSHFields ? (
                        <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                            <TextField
                                label={t('syncNodes.credentialRef')}
                                value={form.credentialRef}
                                onChange={handleChange('credentialRef')}
                                fullWidth
                                helperText={t('syncNodes.credentialRefHelp')}
                            />
                        </Box>
                    ) : null}
                    <Box sx={{gridColumn: {xs: 'span 1', sm: 'span 2'}}}>
                        <TextField
                            label={t('syncNodes.tags')}
                            value={form.tags}
                            onChange={handleChange('tags')}
                            fullWidth
                        />
                    </Box>
                </Box>

                {showTokenSection ? (
                    <Box sx={{mt: 2}}>
                        <TextField
                            label={t('syncNodes.tokenLabel')}
                            value={token || ''}
                            type={tokenVisible ? 'text' : 'password'}
                            InputProps={{
                                readOnly: true,
                                endAdornment: (
                                    <InputAdornment position="end">
                                        <Tooltip
                                            title={
                                                tokenCopied
                                                    ? t('syncNodes.copied')
                                                    : copyError
                                                    ? t('syncNodes.copyFailed')
                                                    : t('syncNodes.copyToken')
                                            }>
                                            <IconButton
                                                edge="end"
                                                onClick={async () => {
                                                    if (!token) return;
                                                    try {
                                                        await navigator.clipboard.writeText(token);
                                                        setCopyError(false);
                                                        setTokenCopied(true);
                                                        window.setTimeout(
                                                            () => setTokenCopied(false),
                                                            1500
                                                        );
                                                    } catch (err) {
                                                        console.warn('Failed to copy token', err);
                                                        setCopyError(true);
                                                        window.setTimeout(
                                                            () => setCopyError(false),
                                                            1500
                                                        );
                                                    }
                                                }}>
                                                {tokenCopied ? (
                                                    <CheckCircleOutlineIcon fontSize="small" />
                                                ) : (
                                                    <ContentCopyIcon fontSize="small" />
                                                )}
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip
                                            title={
                                                tokenVisible
                                                    ? t('syncNodes.hideToken')
                                                    : t('syncNodes.showToken')
                                            }>
                                            <IconButton
                                                edge="end"
                                                onClick={() => setTokenVisible((v) => !v)}>
                                                {tokenVisible ? (
                                                    <VisibilityOffIcon fontSize="small" />
                                                ) : (
                                                    <VisibilityIcon fontSize="small" />
                                                )}
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t('syncNodes.refreshToken')}>
                                            <IconButton
                                                edge="end"
                                                disabled={loading}
                                                onClick={async () => {
                                                    if (!currentNode) return;
                                                    const updated = await onRotateToken(currentNode.id);
                                                    if (mode === 'create') {
                                                        setCreatedNode(updated);
                                                    }
                                                    setToken(updated.agentToken || null);
                                                    setTokenVisible(true);
                                                }}>
                                                <RefreshIcon fontSize="small" />
                                            </IconButton>
                                        </Tooltip>
                                    </InputAdornment>
                                ),
                            }}
                            fullWidth
                        />
                        {mode === 'create' ? (
                            <Typography variant="caption" color="textSecondary">
                                {t('syncNodes.tokenCreateHelp')}
                            </Typography>
                        ) : null}
                    </Box>
                ) : null}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="secondary">
                    {t('common.cancel')}
                </Button>
                <Button onClick={handleSubmit} color="primary" variant="contained" disabled={loading}>
                    {loading ? t('common.saving') : t('common.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default SyncNodeDialog;
