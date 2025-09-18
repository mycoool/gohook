import React, {useState, useEffect} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    FormControl,
    FormLabel,
    RadioGroup,
    FormControlLabel,
    Radio,
    TextField,
    Switch,
    Typography,
    Box,
    Chip,
    Divider,
    IconButton,
    InputAdornment,
    Snackbar,
} from '@mui/material';

import {styled} from '@mui/material/styles';
import {FileCopy, Refresh} from '@mui/icons-material';
import {IVersion} from '../types';
import {useTranslation} from '../i18n/useTranslation';

const StyledDialogContent = styled(DialogContent)(({theme}) => ({
    minWidth: '500px',
    paddingTop: '16px',
}));

const StyledSection = styled(Box)(({theme}) => ({
    marginBottom: '24px',
}));

const StyledStatusChip = styled(Chip)(({theme}) => ({
    marginLeft: '8px',
}));

const StyledBranchInput = styled(TextField)(({theme}) => ({
    marginTop: '8px',
    width: '100%',
}));

const StyledDescription = styled(Typography)(({theme}) => ({
    color: theme.palette.text.secondary,
    fontSize: '0.875rem',
    marginTop: '8px',
}));

const StyledWebhookUrl = styled(Box)(({theme}) => ({
    backgroundColor: theme.palette.grey[100],
    color: theme.palette.common.black,
    padding: '8px',
    borderRadius: theme.shape.borderRadius,
    fontFamily: 'monospace',
    fontSize: '0.875rem',
    wordBreak: 'break-all',
    marginTop: '8px',
    border: `1px solid ${theme.palette.grey[300]}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    ...theme.applyStyles('dark', {
        backgroundColor: theme.palette.grey[800],
        color: theme.palette.common.white,
        border: `1px solid ${theme.palette.grey[600]}`,
    }),
}));

const StyledWebhookUrlText = styled(Box)(({theme}) => ({
    flex: 1,
    marginRight: '8px',
}));

const StyledPasswordField = styled(TextField)({
    '& .MuiInputBase-input': {
        fontFamily: 'monospace',
    },
});

interface GitHookDialogProps {
    open: boolean;
    project: IVersion | null;
    onClose: () => void;
    onSave: (projectName: string, config: GitHookConfig) => Promise<void>;
}

export interface GitHookConfig {
    enhook: boolean;
    hookmode: 'branch' | 'tag';
    hookbranch?: string;
    hooksecret?: string;
}

const GitHookDialog: React.FC<GitHookDialogProps> = ({open, project, onClose, onSave}) => {
    const {t} = useTranslation();
    const [enhook, setEnhook] = useState(false);
    const [hookmode, setHookmode] = useState<'branch' | 'tag'>('branch');
    const [hookbranch, setHookbranch] = useState('*');
    const [hooksecret, setHooksecret] = useState('');
    const [saving, setSaving] = useState(false);
    const [snackbarOpen, setSnackbarOpen] = useState(false);
    const [snackbarMessage, setSnackbarMessage] = useState('');

    // 生成随机密码
    const generatePassword = () => {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let result = '';
        for (let i = 0; i < 16; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    };

    // 复制到剪贴板
    const copyToClipboard = async (text: string, successMessage: string) => {
        try {
            await navigator.clipboard.writeText(text);
            setSnackbarMessage(successMessage);
            setSnackbarOpen(true);
        } catch (error) {
            console.error('Failed to copy to clipboard:', error);
            setSnackbarMessage(t('githook.copyFailed'));
            setSnackbarOpen(true);
        }
    };

    // 处理启用GitHook的变化
    const handleEnhookChange = (enabled: boolean) => {
        setEnhook(enabled);
        // 如果启用GitHook且当前没有密码，自动生成密码
        if (enabled && !hooksecret) {
            setHooksecret(generatePassword());
        }
    };

    useEffect(() => {
        if (project) {
            setEnhook(project.enhook || false);
            setHookmode(project.hookmode || 'branch');
            setHookbranch(project.hookbranch || '*');
            setHooksecret(project.hooksecret || '');
        }
    }, [project]);

    const handleSave = async () => {
        if (!project) return;

        setSaving(true);
        try {
            await onSave(project.name, {
                enhook,
                hookmode,
                hookbranch: hookmode === 'branch' ? hookbranch : undefined,
                hooksecret,
            });
            onClose();
        } catch (error) {
            console.error('Failed to save githook config:', error);
        } finally {
            setSaving(false);
        }
    };

    const getWebhookUrl = () => {
        if (!project) return '';
        return `${window.location.origin}/githook/${project.name}`;
    };

    if (!project) return null;

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {t('githook.title')} - {project.name}
                <StyledStatusChip
                    label={enhook ? t('githook.enabled') : t('githook.disabled')}
                    size="small"
                    color={enhook ? 'primary' : 'default'}
                />
            </DialogTitle>
            <StyledDialogContent>
                <StyledSection>
                    <FormControlLabel
                        control={
                            <Switch
                                checked={enhook}
                                onChange={(e) => handleEnhookChange(e.target.checked)}
                                color="primary"
                            />
                        }
                        label={t('githook.enable')}
                    />
                    <StyledDescription>{t('githook.enableDescription')}</StyledDescription>
                </StyledSection>

                {enhook && (
                    <>
                        <StyledSection>
                            <FormControl component="fieldset">
                                <FormLabel component="legend">{t('githook.runMode')}</FormLabel>
                                <RadioGroup
                                    value={hookmode}
                                    onChange={(e) =>
                                        setHookmode(e.target.value as 'branch' | 'tag')
                                    }>
                                    <FormControlLabel
                                        value="branch"
                                        control={<Radio />}
                                        label={t('githook.branchMode')}
                                    />
                                    <FormControlLabel
                                        value="tag"
                                        control={<Radio />}
                                        label={t('githook.tagMode')}
                                    />
                                </RadioGroup>
                            </FormControl>
                            <StyledDescription>
                                {hookmode === 'branch'
                                    ? t('githook.branchModeDescription')
                                    : t('githook.tagModeDescription')}
                            </StyledDescription>
                        </StyledSection>

                        {hookmode === 'branch' && (
                            <StyledSection>
                                <FormControl component="fieldset" fullWidth>
                                    <FormLabel component="legend">
                                        {t('githook.branchSettings')}
                                    </FormLabel>
                                    <RadioGroup
                                        value={hookbranch === '*' ? 'all' : 'specific'}
                                        onChange={(e) => {
                                            if (e.target.value === 'all') {
                                                setHookbranch('*');
                                            } else {
                                                setHookbranch(project.currentBranch || 'main');
                                            }
                                        }}>
                                        <FormControlLabel
                                            value="all"
                                            control={<Radio />}
                                            label={t('githook.anyBranch')}
                                        />
                                        <FormControlLabel
                                            value="specific"
                                            control={<Radio />}
                                            label={t('githook.specificBranch')}
                                        />
                                    </RadioGroup>

                                    {hookbranch !== '*' && (
                                        <StyledBranchInput
                                            label={t('githook.branchName')}
                                            value={hookbranch}
                                            onChange={(e) => setHookbranch(e.target.value)}
                                            placeholder={t('githook.branchNamePlaceholder')}
                                            variant="outlined"
                                            size="small"
                                        />
                                    )}
                                </FormControl>
                                <StyledDescription>
                                    {hookbranch === '*'
                                        ? t('githook.anyBranchDescription')
                                        : t('githook.specificBranchDescription', {
                                              branch: hookbranch,
                                          })}
                                </StyledDescription>
                            </StyledSection>
                        )}

                        <StyledSection>
                            <Typography variant="subtitle2" gutterBottom>
                                {t('githook.webhookPassword')}
                            </Typography>
                            <StyledPasswordField
                                fullWidth
                                label={t('githook.webhookPassword')}
                                value={hooksecret}
                                onChange={(e) => setHooksecret(e.target.value)}
                                placeholder={t('githook.webhookPasswordPlaceholder')}
                                variant="outlined"
                                size="small"
                                type="text"
                                InputProps={{
                                    endAdornment: (
                                        <InputAdornment position="end">
                                            <IconButton
                                                onClick={() =>
                                                    copyToClipboard(
                                                        hooksecret,
                                                        t('githook.passwordCopied')
                                                    )
                                                }
                                                disabled={!hooksecret}
                                                size="small"
                                                title={t('githook.copyPassword')}>
                                                <FileCopy fontSize="small" />
                                            </IconButton>
                                            <IconButton
                                                onClick={() => setHooksecret(generatePassword())}
                                                size="small"
                                                title={t('githook.generatePassword')}>
                                                <Refresh fontSize="small" />
                                            </IconButton>
                                        </InputAdornment>
                                    ),
                                }}
                            />
                            <StyledDescription>
                                {t('githook.webhookPasswordDescription')}
                            </StyledDescription>
                        </StyledSection>

                        <StyledSection>
                            <Typography variant="subtitle2" gutterBottom>
                                {t('githook.webhookUrl')}
                            </Typography>
                            <StyledWebhookUrl>
                                <StyledWebhookUrlText>{getWebhookUrl()}</StyledWebhookUrlText>
                                <IconButton
                                    onClick={() =>
                                        copyToClipboard(getWebhookUrl(), t('githook.urlCopied'))
                                    }
                                    size="small"
                                    title={t('githook.copyUrl')}>
                                    <FileCopy fontSize="small" />
                                </IconButton>
                            </StyledWebhookUrl>
                            <StyledDescription>
                                {t('githook.webhookUrlDescription')}
                            </StyledDescription>
                        </StyledSection>
                    </>
                )}
            </StyledDialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={saving} variant="contained" color="secondary">
                    {t('common.cancel')}
                </Button>
                <Button onClick={handleSave} color="primary" variant="contained" disabled={saving}>
                    {saving ? t('githook.saving') : t('common.save')}
                </Button>
            </DialogActions>
            <Snackbar
                open={snackbarOpen}
                autoHideDuration={3000}
                onClose={() => setSnackbarOpen(false)}
                message={snackbarMessage}
                anchorOrigin={{vertical: 'bottom', horizontal: 'center'}}
            />
        </Dialog>
    );
};

export default GitHookDialog;
