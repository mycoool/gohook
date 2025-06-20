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
} from '@material-ui/core';

import {makeStyles} from '@material-ui/core/styles';
import {FileCopy, Refresh} from '@material-ui/icons';
import {IVersion} from '../types';
import {useTranslation} from '../i18n/useTranslation';

const useStyles = makeStyles((theme) => ({
    content: {
        minWidth: '500px',
        paddingTop: theme.spacing(2),
    },
    section: {
        marginBottom: theme.spacing(3),
    },
    statusChip: {
        marginLeft: theme.spacing(1),
    },
    branchInput: {
        marginTop: theme.spacing(1),
        width: '100%',
    },
    description: {
        color: theme.palette.text.secondary,
        fontSize: '0.875rem',
        marginTop: theme.spacing(1),
    },
    webhookUrl: {
        backgroundColor:
            theme.palette.type === 'dark' ? theme.palette.grey[800] : theme.palette.grey[100],
        color:
            theme.palette.type === 'dark' ? theme.palette.common.white : theme.palette.common.black,
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '0.875rem',
        wordBreak: 'break-all',
        marginTop: theme.spacing(1),
        border: `1px solid ${
            theme.palette.type === 'dark' ? theme.palette.grey[600] : theme.palette.grey[300]
        }`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
    },
    webhookUrlText: {
        flex: 1,
        marginRight: theme.spacing(1),
    },
    passwordField: {
        '& .MuiInputBase-input': {
            fontFamily: 'monospace',
        },
    },
}));

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
    const classes = useStyles();
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
                <Chip
                    label={enhook ? t('githook.enabled') : t('githook.disabled')}
                    size="small"
                    color={enhook ? 'primary' : 'default'}
                    className={classes.statusChip}
                />
            </DialogTitle>
            <DialogContent className={classes.content}>
                <Box className={classes.section}>
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
                    <Typography className={classes.description}>
                        {t('githook.enableDescription')}
                    </Typography>
                </Box>

                {enhook && (
                    <>
                        <Box className={classes.section}>
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
                            <Typography className={classes.description}>
                                {hookmode === 'branch'
                                    ? t('githook.branchModeDescription')
                                    : t('githook.tagModeDescription')}
                            </Typography>
                        </Box>

                        {hookmode === 'branch' && (
                            <Box className={classes.section}>
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
                                        <TextField
                                            className={classes.branchInput}
                                            label={t('githook.branchName')}
                                            value={hookbranch}
                                            onChange={(e) => setHookbranch(e.target.value)}
                                            placeholder={t('githook.branchNamePlaceholder')}
                                            variant="outlined"
                                            size="small"
                                        />
                                    )}
                                </FormControl>
                                <Typography className={classes.description}>
                                    {hookbranch === '*'
                                        ? t('githook.anyBranchDescription')
                                        : t('githook.specificBranchDescription', {
                                              branch: hookbranch,
                                          })}
                                </Typography>
                            </Box>
                        )}

                        <Box className={classes.section}>
                            <Typography variant="subtitle2" gutterBottom>
                                {t('githook.webhookPassword')}
                            </Typography>
                            <TextField
                                fullWidth
                                label={t('githook.webhookPassword')}
                                value={hooksecret}
                                onChange={(e) => setHooksecret(e.target.value)}
                                placeholder={t('githook.webhookPasswordPlaceholder')}
                                variant="outlined"
                                size="small"
                                type="text"
                                className={classes.passwordField}
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
                            <Typography className={classes.description}>
                                {t('githook.webhookPasswordDescription')}
                            </Typography>
                        </Box>

                        <Box className={classes.section}>
                            <Typography variant="subtitle2" gutterBottom>
                                {t('githook.webhookUrl')}
                            </Typography>
                            <Box className={classes.webhookUrl}>
                                <span className={classes.webhookUrlText}>{getWebhookUrl()}</span>
                                <IconButton
                                    onClick={() =>
                                        copyToClipboard(getWebhookUrl(), t('githook.urlCopied'))
                                    }
                                    size="small"
                                    title={t('githook.copyUrl')}>
                                    <FileCopy fontSize="small" />
                                </IconButton>
                            </Box>
                            <Typography className={classes.description}>
                                {t('githook.webhookUrlDescription')}
                            </Typography>
                        </Box>
                    </>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={saving}>
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
