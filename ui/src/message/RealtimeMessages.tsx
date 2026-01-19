import React, {Component} from 'react';
import {
    Typography,
    Paper,
    List,
    ListItem,
    ListItemText,
    ListItemIcon,
    Chip,
    Box,
    IconButton,
    Tooltip,
    Fab,
} from '@mui/material';
import {
    CheckCircle as SuccessIcon,
    Error as ErrorIcon,
    Info as InfoIcon,
    Delete as DeleteIcon,
    Visibility as VisibilityIcon,
    VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';
import {observer} from 'mobx-react';
import {observable, action} from 'mobx';
import {inject, Stores} from '../inject';
import {
    IWebSocketMessage,
    IHookTriggeredMessage,
    IVersionSwitchMessage,
    IProjectManageMessage,
    IGitHookTriggeredMessage,
    IHookManageMessage,
} from '../types';
import WebSocketStatusIndicator from '../common/WebSocketStatusIndicator';
import useTranslation from '../i18n/useTranslation';

// 空接口用于扩展
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface IProps {
    t: (key: string, params?: Record<string, string | number>) => string;
    language: string;
}

@observer
class RealtimeMessages extends Component<IProps & Stores<'wsStore'>> {
    @observable
    private messages: IWebSocketMessage[] = [];

    @observable
    private isVisible = true;

    @observable
    private maxMessages = 100;

    public componentDidMount() {
        // 订阅WebSocket消息
        this.props.wsStore.onMessage(this.handleWebSocketMessage);
    }

    public componentWillUnmount() {
        // 取消订阅
        this.props.wsStore.offMessage(this.handleWebSocketMessage);
    }

    @action
    private handleWebSocketMessage = (message: IWebSocketMessage) => {
        // 过滤掉连接和心跳消息
        if (
            message.type === 'connected' ||
            message.type === 'pong' ||
            message.type === 'sync_node_event' ||
            message.type === 'sync_task_event' ||
            message.type === 'sync_project_event' ||
            message.type === 'sync_node_status'
        ) {
            return;
        }

        // 添加消息到列表开头
        this.messages.unshift(message);

        // 限制消息数量
        if (this.messages.length > this.maxMessages) {
            this.messages = this.messages.slice(0, this.maxMessages);
        }
    };

    @action
    private clearMessages = () => {
        this.messages = [];
    };

    @action
    private toggleVisibility = () => {
        this.isVisible = !this.isVisible;
    };

    private getMessageIcon = (message: IWebSocketMessage) => {
        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                return hookMsg.success ? (
                    <SuccessIcon sx={{color: '#4caf50', fontSize: '1.5rem'}} />
                ) : (
                    <ErrorIcon sx={{color: '#f44336', fontSize: '1.5rem'}} />
                );
            }
            case 'hook_managed': {
                const hookManageMsg = message.data as IHookManageMessage;
                return hookManageMsg.success ? (
                    <SuccessIcon sx={{color: '#4caf50', fontSize: '1.5rem'}} />
                ) : (
                    <ErrorIcon sx={{color: '#f44336', fontSize: '1.5rem'}} />
                );
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                return versionMsg.success ? (
                    <SuccessIcon sx={{color: '#4caf50', fontSize: '1.5rem'}} />
                ) : (
                    <ErrorIcon sx={{color: '#f44336', fontSize: '1.5rem'}} />
                );
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                return projectMsg.success ? (
                    <SuccessIcon sx={{color: '#4caf50', fontSize: '1.5rem'}} />
                ) : (
                    <ErrorIcon sx={{color: '#f44336', fontSize: '1.5rem'}} />
                );
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                if (githookMsg.success) {
                    return githookMsg.skipped ? (
                        <InfoIcon sx={{color: '#ff9800', fontSize: '1.5rem'}} />
                    ) : (
                        <SuccessIcon sx={{color: '#4caf50', fontSize: '1.5rem'}} />
                    );
                } else {
                    return <ErrorIcon sx={{color: '#f44336', fontSize: '1.5rem'}} />;
                }
            }
            default:
                return <InfoIcon sx={{color: '#2196f3', fontSize: '1.5rem'}} />;
        }
    };

    private getMessageTitle = (message: IWebSocketMessage): string => {
        const {t} = this.props;
        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                return t('realtimeMessages.title.hookTriggered', {name: hookMsg.hookName});
            }
            case 'hook_managed': {
                const hookManageMsg = message.data as IHookManageMessage;
                return t('realtimeMessages.title.hookManaged', {name: hookManageMsg.hookName});
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                return t('realtimeMessages.title.versionSwitched', {name: versionMsg.projectName});
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                return t('realtimeMessages.title.projectManaged', {name: projectMsg.projectName});
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                return t('realtimeMessages.title.githookTriggered', {name: githookMsg.projectName});
            }
            default:
                return message.type;
        }
    };

    private getMessageDescription = (message: IWebSocketMessage): string => {
        const {t} = this.props;
        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                if (hookMsg.success) {
                    const output =
                        hookMsg.output?.substring(0, 100) +
                        (hookMsg.output && hookMsg.output.length > 100 ? '...' : '');
                    const outputSuffix = hookMsg.output
                        ? t('realtimeMessages.description.hookTriggeredOutput', {output})
                        : '';
                    return t('realtimeMessages.description.hookTriggeredSuccess', {
                        method: hookMsg.method,
                        remote: hookMsg.remoteAddr,
                        output: outputSuffix,
                    });
                } else {
                    return t('realtimeMessages.description.hookTriggeredFailed', {
                        error: hookMsg.error ?? t('realtimeMessages.unknownError'),
                    });
                }
            }
            case 'hook_managed': {
                const hookManageMsg = message.data as IHookManageMessage;
                const actionText = this.getHookManageActionLabel(hookManageMsg.action);

                if (hookManageMsg.success) {
                    return t('realtimeMessages.description.hookManagedSuccess', {action: actionText});
                } else {
                    return t('realtimeMessages.description.hookManagedFailed', {
                        action: actionText,
                        error: hookManageMsg.error ?? t('realtimeMessages.unknownError'),
                    });
                }
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                const actionText = this.getGitHookActionLabel(githookMsg.action);

                if (githookMsg.success) {
                    if (githookMsg.skipped) {
                        return t('realtimeMessages.description.githookSkipped', {
                            action: actionText,
                            message: githookMsg.message || githookMsg.target,
                        });
                    } else {
                        return t('realtimeMessages.description.githookSuccess', {
                            action: actionText,
                            message: githookMsg.message || githookMsg.target,
                        });
                    }
                } else {
                    return t('realtimeMessages.description.githookFailed', {
                        action: actionText,
                        error: githookMsg.error ?? t('realtimeMessages.unknownError'),
                    });
                }
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                const actionText = this.getVersionActionLabel(versionMsg.action);

                if (versionMsg.success) {
                    return t('realtimeMessages.description.versionSuccess', {
                        action: actionText,
                        target: versionMsg.target,
                    });
                } else {
                    return t('realtimeMessages.description.versionFailed', {
                        action: actionText,
                        error: versionMsg.error ?? t('realtimeMessages.unknownError'),
                    });
                }
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                if (projectMsg.success) {
                    //添加、删除、编辑
                    switch (projectMsg.action) {
                        case 'add':
                            return t('realtimeMessages.description.projectAddSuccess', {
                                path: projectMsg.projectPath,
                            });
                        case 'delete':
                            return t('realtimeMessages.description.projectDeleteSuccess', {
                                path: projectMsg.projectPath,
                            });
                        case 'edit':
                            return t('realtimeMessages.description.projectEditSuccess', {
                                path: projectMsg.projectPath,
                            });
                        default:
                            return t('realtimeMessages.description.projectGenericSuccess', {
                                action: projectMsg.action,
                                path: projectMsg.projectPath,
                            });
                    }
                } else {
                    switch (projectMsg.action) {
                        case 'add':
                            return t('realtimeMessages.description.projectAddFailed', {
                                error: projectMsg.error ?? t('realtimeMessages.unknownError'),
                            });
                        case 'delete':
                            return t('realtimeMessages.description.projectDeleteFailed', {
                                error: projectMsg.error ?? t('realtimeMessages.unknownError'),
                            });
                        case 'edit':
                            return t('realtimeMessages.description.projectEditFailed', {
                                error: projectMsg.error ?? t('realtimeMessages.unknownError'),
                            });
                        default:
                            return t('realtimeMessages.description.projectGenericFailed', {
                                action: projectMsg.action,
                                error: projectMsg.error ?? t('realtimeMessages.unknownError'),
                            });
                    }
                }
            }
            default:
                return JSON.stringify(message.data);
        }
    };

    private formatTime = (timestamp: string): string => {
        const {language} = this.props;
        const locale = language.startsWith('zh') ? 'zh-CN' : 'en-US';
        return new Date(timestamp).toLocaleString(locale, {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    };

    private getStatusChip = (message: IWebSocketMessage) => {
        let success = true;
        let status: 'success' | 'failed' | 'skipped' | 'info' = 'info';

        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                success = hookMsg.success;
                status = success ? 'success' : 'failed';
                break;
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                success = versionMsg.success;
                status = success ? 'success' : 'failed';
                break;
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                success = projectMsg.success;
                status = success ? 'success' : 'failed';
                break;
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                success = githookMsg.success;
                if (success && githookMsg.skipped) {
                    status = 'skipped';
                } else {
                    status = success ? 'success' : 'failed';
                }
                break;
            }
            default:
                status = 'info';
        }

        // 优化状态标签的样式
        let chipProps = {};
        if (status === 'success') {
            chipProps = {
                sx: {
                    backgroundColor: 'success.main',
                    color: 'success.contrastText',
                    fontWeight: 'bold',
                    '&.MuiChip-outlined': {
                        borderColor: 'success.main',
                        backgroundColor: 'success.light',
                        color: 'success.dark',
                    },
                },
                variant: 'filled' as const,
            };
        } else if (status === 'failed') {
            chipProps = {
                sx: {
                    backgroundColor: 'error.main',
                    color: 'error.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const,
            };
        } else if (status === 'skipped') {
            chipProps = {
                sx: {
                    backgroundColor: 'warning.main',
                    color: 'warning.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const,
            };
        } else {
            chipProps = {
                sx: {
                    backgroundColor: 'info.main',
                    color: 'info.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const,
            };
        }

        return (
            <Chip size="small" label={this.props.t(`realtimeMessages.status.${status}`)} {...chipProps} />
        );
    };

    private getHookManageActionLabel = (action: string) => {
        const {t} = this.props;
        switch (action) {
            case 'create':
                return t('realtimeMessages.actions.hook.create');
            case 'update_basic':
                return t('realtimeMessages.actions.hook.updateBasic');
            case 'update_parameters':
                return t('realtimeMessages.actions.hook.updateParameters');
            case 'update_triggers':
                return t('realtimeMessages.actions.hook.updateTriggers');
            case 'update_response':
                return t('realtimeMessages.actions.hook.updateResponse');
            case 'update_script':
                return t('realtimeMessages.actions.hook.updateScript');
            case 'delete':
                return t('realtimeMessages.actions.hook.delete');
            default:
                return action;
        }
    };

    private getGitHookActionLabel = (action: string) => {
        const {t} = this.props;
        switch (action) {
            case 'switch-branch':
                return t('realtimeMessages.actions.githook.switchBranch');
            case 'switch-tag':
                return t('realtimeMessages.actions.githook.switchTag');
            case 'delete-tag':
                return t('realtimeMessages.actions.githook.deleteTag');
            case 'delete-branch':
                return t('realtimeMessages.actions.githook.deleteBranch');
            case 'skip-branch-switch':
                return t('realtimeMessages.actions.githook.skipBranchSwitch');
            case 'skip-mode-mismatch':
                return t('realtimeMessages.actions.githook.skipModeMismatch');
            default:
                return action;
        }
    };

    private getVersionActionLabel = (action: string) => {
        const {t} = this.props;
        switch (action) {
            case 'switch-branch':
                return t('realtimeMessages.actions.version.switchBranch');
            case 'switch-tag':
                return t('realtimeMessages.actions.version.switchTag');
            case 'delete-tag':
                return t('realtimeMessages.actions.version.deleteTag');
            default:
                return action;
        }
    };

    public render() {
        return (
            <Box position="fixed" bottom={16} right={16} zIndex={1000}>
                {/* 切换显示/隐藏的按钮 */}
                <Fab
                    color="primary"
                    size="small"
                    onClick={this.toggleVisibility}
                    style={{marginBottom: 8}}
                    data-realtime-messages="button">
                    {this.isVisible ? <VisibilityOffIcon /> : <VisibilityIcon />}
                </Fab>

                {this.isVisible && (
                    <Paper
                        elevation={8}
                        sx={{
                            width: 400,
                            maxHeight: 500,
                            overflow: 'hidden',
                            display: 'flex',
                            flexDirection: 'column',
                            bgcolor: 'background.paper',
                            border: 1,
                            borderColor: 'divider',
                        }}
                        data-realtime-messages="expanded">
                        {/* 头部 */}
                        <Box
                            p={2}
                            display="flex"
                            justifyContent="space-between"
                            alignItems="center"
                            sx={{borderBottom: 1, borderColor: 'divider'}}>
                            <Typography variant="h6">
                                {this.props.t('realtimeMessages.header.title', {
                                    count: this.messages.length,
                                })}
                            </Typography>
                            <Box>
                                <Tooltip title={this.props.t('realtimeMessages.clear')}>
                                    <IconButton size="small" onClick={this.clearMessages}>
                                        <DeleteIcon />
                                    </IconButton>
                                </Tooltip>
                                <WebSocketStatusIndicator wsStore={this.props.wsStore} />
                            </Box>
                        </Box>

                        {/* 消息列表 - 使用环境文件编辑器样式的滚动条 */}
                        <Box
                            sx={{
                                overflowY: 'auto',
                                maxHeight: 400,
                                // 使用环境文件编辑器的精美滚动条样式
                                '&::-webkit-scrollbar': {
                                    width: '8px',
                                    height: '8px',
                                },
                                '&::-webkit-scrollbar-track': {
                                    backgroundColor: (theme) =>
                                        theme.palette.mode === 'dark' ? '#2d2d2d' : '#f1f3f4',
                                    borderRadius: '4px',
                                },
                                '&::-webkit-scrollbar-thumb': {
                                    backgroundColor: (theme) =>
                                        theme.palette.mode === 'dark' ? '#30363d' : '#c1c8cd',
                                    borderRadius: '4px',
                                    '&:hover': {
                                        backgroundColor: (theme) =>
                                            theme.palette.mode === 'dark' ? '#484f58' : '#a8b3ba',
                                    },
                                },
                                // Firefox 滚动条样式
                                scrollbarWidth: 'thin',
                                scrollbarColor: (theme) =>
                                    theme.palette.mode === 'dark'
                                        ? '#30363d #2d2d2d'
                                        : '#c1c8cd #f1f3f4',
                            }}>
                            {this.messages.length === 0 ? (
                                <Box p={2} textAlign="center">
                                    <Typography variant="body2" color="textSecondary">
                                        {this.props.t('realtimeMessages.empty')}
                                    </Typography>
                                </Box>
                            ) : (
                                <List dense>
                                    {this.messages.map((message, index) => {
                                        // 生成稳定的key
                                        const messageKey = `${message.type}-${message.timestamp}-${index}`;
                                        return (
                                            <ListItem
                                                key={messageKey}
                                                divider
                                                sx={{
                                                    '&:hover': {
                                                        backgroundColor: 'action.hover',
                                                    },
                                                }}>
                                                <ListItemIcon sx={{minWidth: 40}}>
                                                    {this.getMessageIcon(message)}
                                                </ListItemIcon>
                                                <ListItemText
                                                    primary={
                                                        <Box
                                                            display="flex"
                                                            justifyContent="space-between"
                                                            alignItems="center">
                                                            <Typography
                                                                variant="subtitle2"
                                                                sx={{fontWeight: 'medium'}}>
                                                                {this.getMessageTitle(message)}
                                                            </Typography>
                                                            {this.getStatusChip(message)}
                                                        </Box>
                                                    }
                                                    secondary={
                                                        <Box>
                                                            <Typography
                                                                variant="body2"
                                                                sx={{
                                                                    marginBottom: 0.5,
                                                                    wordBreak: 'break-word',
                                                                    color: 'text.primary',
                                                                }}>
                                                                {this.getMessageDescription(
                                                                    message
                                                                )}
                                                            </Typography>
                                                            <Typography
                                                                variant="caption"
                                                                color="textSecondary">
                                                                {this.formatTime(message.timestamp)}
                                                            </Typography>
                                                        </Box>
                                                    }
                                                />
                                            </ListItem>
                                        );
                                    })}
                                </List>
                            )}
                        </Box>
                    </Paper>
                )}
            </Box>
        );
    }
}

const RealtimeMessagesWithTranslation = inject('wsStore')((props: Stores<'wsStore'>) => {
    const {t, i18n} = useTranslation();
    return <RealtimeMessages {...props} t={t} language={i18n.language} />;
});

export default RealtimeMessagesWithTranslation;
