import React, { useState, useEffect } from 'react';
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
    Snackbar
} from '@material-ui/core';

import { makeStyles } from '@material-ui/core/styles';
import { FileCopy, Refresh } from '@material-ui/icons';
import { IVersion } from '../types';

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
        backgroundColor: theme.palette.type === 'dark' ? theme.palette.grey[800] : theme.palette.grey[100],
        color: theme.palette.type === 'dark' ? theme.palette.common.white : theme.palette.common.black,
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '0.875rem',
        wordBreak: 'break-all',
        marginTop: theme.spacing(1),
        border: `1px solid ${theme.palette.type === 'dark' ? theme.palette.grey[600] : theme.palette.grey[300]}`,
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

const GitHookDialog: React.FC<GitHookDialogProps> = ({
    open,
    project,
    onClose,
    onSave
}) => {
    const classes = useStyles();
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
            setSnackbarMessage('复制失败');
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
                GitHook 配置 - {project.name}
                <Chip
                    label={enhook ? '已启用' : '已禁用'}
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
                        label="启用 GitHook"
                    />
                    <Typography className={classes.description}>
                        启用后，Git仓库的webhook将自动触发代码拉取和部署
                    </Typography>
                </Box>

                {enhook && (
                    <>
                        <Divider />
                        <Box className={classes.section}>
                            <FormControl component="fieldset">
                                <FormLabel component="legend">运行模式</FormLabel>
                                <RadioGroup
                                    value={hookmode}
                                    onChange={(e) => setHookmode(e.target.value as 'branch' | 'tag')}
                                >
                                    <FormControlLabel
                                        value="branch"
                                        control={<Radio />}
                                        label="分支模式"
                                    />
                                    <FormControlLabel
                                        value="tag"
                                        control={<Radio />}
                                        label="标签模式"
                                    />
                                </RadioGroup>
                            </FormControl>
                            <Typography className={classes.description}>
                                {hookmode === 'branch' 
                                    ? '当指定分支有新提交时触发部署'
                                    : '当创建新标签时触发部署'
                                }
                            </Typography>
                        </Box>

                        {hookmode === 'branch' && (
                            <Box className={classes.section}>
                                <FormControl component="fieldset" fullWidth>
                                    <FormLabel component="legend">分支设置</FormLabel>
                                    <RadioGroup
                                        value={hookbranch === '*' ? 'all' : 'specific'}
                                        onChange={(e) => {
                                            if (e.target.value === 'all') {
                                                setHookbranch('*');
                                            } else {
                                                setHookbranch(project.currentBranch || 'main');
                                            }
                                        }}
                                    >
                                        <FormControlLabel
                                            value="all"
                                            control={<Radio />}
                                            label="任意分支（webhook触发哪个分支就切换到哪个分支）"
                                        />
                                        <FormControlLabel
                                            value="specific"
                                            control={<Radio />}
                                            label="指定分支"
                                        />
                                    </RadioGroup>
                                    
                                    {hookbranch !== '*' && (
                                        <TextField
                                            className={classes.branchInput}
                                            label="分支名称"
                                            value={hookbranch}
                                            onChange={(e) => setHookbranch(e.target.value)}
                                            placeholder="例如: main, develop, master"
                                            variant="outlined"
                                            size="small"
                                        />
                                    )}
                                </FormControl>
                                <Typography className={classes.description}>
                                    {hookbranch === '*' 
                                        ? '任意分支模式：webhook触发时会自动切换到对应分支并拉取最新代码'
                                        : `指定分支模式：只有 ${hookbranch} 分支的webhook才会触发部署`
                                    }
                                </Typography>
                            </Box>
                        )}

                        <Divider />
                        <Box className={classes.section}>
                            <Typography variant="subtitle2" gutterBottom>
                                Webhook 密码
                            </Typography>
                            <TextField
                                fullWidth
                                label="Webhook 密码"
                                value={hooksecret}
                                onChange={(e) => setHooksecret(e.target.value)}
                                placeholder="设置webhook验证密码（可选）"
                                variant="outlined"
                                size="small"
                                type="password"
                                className={classes.passwordField}
                                InputProps={{
                                    endAdornment: (
                                        <InputAdornment position="end">
                                            <IconButton
                                                onClick={() => copyToClipboard(hooksecret, '密码已复制到剪贴板')}
                                                disabled={!hooksecret}
                                                size="small"
                                                title="复制密码"
                                            >
                                                <FileCopy fontSize="small" />
                                            </IconButton>
                                            <IconButton
                                                onClick={() => setHooksecret(generatePassword())}
                                                size="small"
                                                title="生成随机密码"
                                            >
                                                <Refresh fontSize="small" />
                                            </IconButton>
                                        </InputAdornment>
                                    )
                                }}
                            />
                            <Typography className={classes.description}>
                                设置后，webhook请求需要包含此密码才能触发部署。可使用右侧按钮生成随机密码或复制当前密码。
                            </Typography>
                        </Box>

                        <Divider />
                        <Box className={classes.section}>
                            <Typography variant="subtitle2" gutterBottom>
                                Webhook URL
                            </Typography>
                            <Box className={classes.webhookUrl}>
                                <span className={classes.webhookUrlText}>{getWebhookUrl()}</span>
                                <IconButton
                                    onClick={() => copyToClipboard(getWebhookUrl(), 'Webhook URL已复制到剪贴板')}
                                    size="small"
                                >
                                    <FileCopy fontSize="small" />
                                </IconButton>
                            </Box>
                            <Typography className={classes.description}>
                                请将此URL配置到您的Git仓库的webhook设置中
                            </Typography>
                        </Box>
                    </>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={saving}>
                    取消
                </Button>
                <Button 
                    onClick={handleSave} 
                    color="primary" 
                    variant="contained"
                    disabled={saving}
                >
                    {saving ? '保存中...' : '保存'}
                </Button>
            </DialogActions>
            <Snackbar
                open={snackbarOpen}
                autoHideDuration={3000}
                onClose={() => setSnackbarOpen(false)}
                message={snackbarMessage}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
            />
        </Dialog>
    );
};

export default GitHookDialog; 