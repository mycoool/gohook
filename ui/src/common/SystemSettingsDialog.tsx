import React, {Component} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    Box,
    Typography,
    Alert,
    CircularProgress,
    InputAdornment,
    IconButton,
} from '@mui/material';
import {observer} from 'mobx-react';
import axios from 'axios';
import * as config from '../config';

interface SystemConfig {
    jwt_secret: string;
    jwt_expiry_duration: number;
    mode: string;
    panel_alias: string;
    language: string;
}

interface SystemSettingsDialogProps {
    open: boolean;
    onClose: () => void;
    token: string;
    t: (key: string, params?: Record<string, string | number>) => string;
    onConfigSaved?: () => void;
}

interface SystemSettingsDialogState {
    loading: boolean;
    saving: boolean;
    error: string | null;
    config: SystemConfig;
    originalConfig: SystemConfig;
}

@observer
class SystemSettingsDialog extends Component<SystemSettingsDialogProps, SystemSettingsDialogState> {
    constructor(props: SystemSettingsDialogProps) {
        super(props);
        this.state = {
            loading: false,
            saving: false,
            error: null,
            config: {
                jwt_secret: '',
                jwt_expiry_duration: 24,
                mode: 'dev',
                panel_alias: 'GoHook',
                language: 'zh',
            },
            originalConfig: {
                jwt_secret: '',
                jwt_expiry_duration: 24,
                mode: 'dev',
                panel_alias: 'GoHook',
                language: 'zh',
            },
        };
    }

    componentDidUpdate(prevProps: SystemSettingsDialogProps) {
        if (this.props.open && !prevProps.open) {
            this.loadConfig();
        }
    }

    componentDidMount() {
        if (this.props.open) {
            this.loadConfig();
        }
    }

    loadConfig = async () => {
        this.setState({loading: true, error: null});
        try {
            const response = await axios.get(config.get('url') + 'system/config', {
                headers: {'X-GoHook-Key': this.props.token},
            });
            const configData = response.data;

            // 设置显示配置：保持所有字段与后端一致，但JWT密钥显示为空
            const displayConfig = {
                jwt_secret: '', // 前端显示为空，表示不修改
                jwt_expiry_duration: configData.jwt_expiry_duration || 24,
                mode: configData.mode || 'dev',
                panel_alias: configData.panel_alias || 'GoHook',
                language: configData.language || 'zh',
            };

            this.setState({
                config: displayConfig,
                originalConfig: {...configData}, // 保存真实的原始配置
                loading: false,
            });
        } catch (error: any) {
            this.setState({
                error: error.response?.data?.error || '加载配置失败',
                loading: false,
            });
        }
    };

    saveConfig = async () => {
        this.setState({saving: true, error: null});
        try {
            // 准备发送的配置数据
            const configToSave = {...this.state.config};

            // 检查是否修改了 JWT 密钥
            const isJwtSecretChanged = configToSave.jwt_secret.trim() !== '';

            // 如果JWT密钥为空，则使用原始配置中的JWT密钥（表示不修改）
            if (!configToSave.jwt_secret.trim()) {
                configToSave.jwt_secret = this.state.originalConfig.jwt_secret;
            }

            await axios.put(config.get('url') + 'system/config', configToSave, {
                headers: {'X-GoHook-Key': this.props.token},
            });

            // 保存成功后,用保存的配置（除了jwt_secret）更新原始配置
            this.setState((prevState) => ({
                saving: false,
                originalConfig: {
                    ...prevState.originalConfig,
                    ...configToSave,
                    jwt_secret: prevState.originalConfig.jwt_secret, // 保持原始secret不变
                },
            }));

            // 更新浏览器标题
            const newPanelAlias = configToSave.panel_alias?.trim() || 'GoHook';
            document.title = newPanelAlias;

            this.props.onClose();
            if (this.props.onConfigSaved) {
                this.props.onConfigSaved();
            }

            // 如果修改了 JWT 密钥，立即清理本地状态并跳转到登录页面
            if (isJwtSecretChanged) {
                // 延迟一下，让用户看到保存成功的提示
                setTimeout(() => {
                    // 清理本地存储的 token
                    window.localStorage.removeItem('gohook-login-key');
                    // 刷新页面，强制跳转到登录页面
                    window.location.reload();
                }, 1000);
            }
        } catch (error: any) {
            this.setState({
                error: error.response?.data?.error || '保存配置失败',
                saving: false,
            });
        }
    };

    handleConfigChange = (field: keyof SystemConfig, value: string | number) => {
        this.setState((prevState) => ({
            config: {
                ...prevState.config,
                [field]: value,
            },
        }));
    };

    hasChanges = () => {
        const {config, originalConfig} = this.state;
        // 如果jwt_secret有输入值，则认为有更改
        if (config.jwt_secret.trim() !== '') {
            return true;
        }
        // 比较其他字段是否有更改
        return (
            config.jwt_expiry_duration !== originalConfig.jwt_expiry_duration ||
            config.mode !== originalConfig.mode ||
            config.panel_alias !== originalConfig.panel_alias ||
            config.language !== originalConfig.language
        );
    };

    handleClose = () => {
        // 直接关闭弹窗并重置配置，无需确认弹窗
        this.props.onClose();
    };

    render() {
        const {open, t} = this.props;
        const {loading, saving, error, config} = this.state;

        return (
            <Dialog open={open} onClose={this.handleClose} maxWidth="sm" fullWidth>
                <DialogTitle>{t('settings.systemSettings')}</DialogTitle>
                <DialogContent>
                    {loading ? (
                        <Box display="flex" justifyContent="center" p={3}>
                            <CircularProgress />
                        </Box>
                    ) : (
                        <Box sx={{pt: 2}}>
                            {error && (
                                <Alert severity="error" sx={{mb: 2}}>
                                    {error}
                                </Alert>
                            )}

                            <TextField
                                fullWidth
                                label={t('settings.jwtSecret')}
                                value={config.jwt_secret}
                                onChange={(e) =>
                                    this.handleConfigChange('jwt_secret', e.target.value)
                                }
                                margin="normal"
                                helperText={t('settings.jwtSecretHelp')}
                                type="text"
                                placeholder={t('settings.jwtSecretPlaceholder')}
                                autoComplete="off"
                                name="jwt-config-secret"
                            />

                            <TextField
                                fullWidth
                                label={t('settings.jwtExpiryDuration')}
                                value={config.jwt_expiry_duration}
                                onChange={(e) =>
                                    this.handleConfigChange(
                                        'jwt_expiry_duration',
                                        parseInt(e.target.value) || 1440
                                    )
                                }
                                margin="normal"
                                type="number"
                                helperText={t('settings.jwtExpiryDurationHelp')}
                                inputProps={{min: 1, max: 525600}} // 1分钟到1年
                            />

                            <TextField
                                fullWidth
                                label={t('settings.panelAlias')}
                                value={config.panel_alias}
                                onChange={(e) =>
                                    this.handleConfigChange('panel_alias', e.target.value)
                                }
                                margin="normal"
                                helperText={t('settings.panelAliasHelp')}
                                placeholder={t('settings.panelAliasPlaceholder')}
                                autoComplete="off"
                                inputProps={{maxLength: 100}}
                            />

                            <FormControl fullWidth margin="normal">
                                <InputLabel>{t('settings.mode')}</InputLabel>
                                <Select
                                    value={config.mode}
                                    onChange={(e) =>
                                        this.handleConfigChange('mode', e.target.value)
                                    }
                                    label={t('settings.mode')}>
                                    <MenuItem value="dev">{t('settings.modeDev')}</MenuItem>
                                    <MenuItem value="test">{t('settings.modeTest')}</MenuItem>
                                    <MenuItem value="prod">{t('settings.modeProd')}</MenuItem>
                                </Select>
                            </FormControl>

                            <FormControl fullWidth margin="normal">
                                <InputLabel>{t('settings.language')}</InputLabel>
                                <Select
                                    value={config.language}
                                    onChange={(e) =>
                                        this.handleConfigChange('language', e.target.value)
                                    }
                                    label={t('settings.language')}>
                                    <MenuItem value="zh">{t('settings.languageChinese')}</MenuItem>
                                    <MenuItem value="en">{t('settings.languageEnglish')}</MenuItem>
                                </Select>
                            </FormControl>
                        </Box>
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.handleClose} disabled={saving}>
                        {t('common.cancel')}
                    </Button>
                    <Button
                        onClick={this.saveConfig}
                        variant="contained"
                        disabled={loading || saving || !this.hasChanges()}
                        startIcon={saving ? <CircularProgress size={16} /> : null}>
                        {saving ? t('common.saving') : t('common.save')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
}

export default SystemSettingsDialog;
