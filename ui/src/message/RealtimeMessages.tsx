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
    Refresh as RefreshIcon,
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
} from '../types';

// 空接口用于扩展
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface IProps {}

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
        if (message.type === 'connected' || message.type === 'pong') {
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
                    <SuccessIcon sx={{ color: '#4caf50', fontSize: '1.5rem' }} />
                ) : (
                    <ErrorIcon sx={{ color: '#f44336', fontSize: '1.5rem' }} />
                );
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                return versionMsg.success ? (
                    <SuccessIcon sx={{ color: '#4caf50', fontSize: '1.5rem' }} />
                ) : (
                    <ErrorIcon sx={{ color: '#f44336', fontSize: '1.5rem' }} />
                );
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                return projectMsg.success ? (
                    <SuccessIcon sx={{ color: '#4caf50', fontSize: '1.5rem' }} />
                ) : (
                    <ErrorIcon sx={{ color: '#f44336', fontSize: '1.5rem' }} />
                );
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                if (githookMsg.success) {
                    return githookMsg.skipped ? (
                        <InfoIcon sx={{ color: '#ff9800', fontSize: '1.5rem' }} />
                    ) : (
                        <SuccessIcon sx={{ color: '#4caf50', fontSize: '1.5rem' }} />
                    );
                } else {
                    return <ErrorIcon sx={{ color: '#f44336', fontSize: '1.5rem' }} />;
                }
            }
            default:
                return <InfoIcon sx={{ color: '#2196f3', fontSize: '1.5rem' }} />;
        }
    };

    private getMessageTitle = (message: IWebSocketMessage): string => {
        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                return `WebHook: ${hookMsg.hookName}`;
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                return `版本切换: ${versionMsg.projectName}`;
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                return `项目管理: ${projectMsg.projectName}`;
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                return `GitHook: ${githookMsg.projectName}`;
            }
            default:
                return message.type;
        }
    };

    private getMessageDescription = (message: IWebSocketMessage): string => {
        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                if (hookMsg.success) {
                    return `执行成功 (${hookMsg.method} from ${hookMsg.remoteAddr})${
                        hookMsg.output
                            ? '\n输出: ' +
                              hookMsg.output.substring(0, 100) +
                              (hookMsg.output.length > 100 ? '...' : '')
                            : ''
                    }`;
                } else {
                    return `执行失败: ${hookMsg.error ?? '未知错误'}`;
                }
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                let actionText = '';
                switch (githookMsg.action) {
                    case 'switch-branch':
                        actionText = '分支切换';
                        break;
                    case 'switch-tag':
                        actionText = '标签切换';
                        break;
                    case 'delete-tag':
                        actionText = '标签删除';
                        break;
                    case 'delete-branch':
                        actionText = '分支删除';
                        break;
                    case 'skip-branch-switch':
                        actionText = '分支检查';
                        break;
                    case 'skip-mode-mismatch':
                        actionText = '模式检查';
                        break;
                    default:
                        actionText = githookMsg.action;
                }

                if (githookMsg.success) {
                    if (githookMsg.skipped) {
                        return `${actionText}跳过: ${githookMsg.message || githookMsg.target}`;
                    } else {
                        return `${actionText}成功: ${githookMsg.message || githookMsg.target}`;
                    }
                } else {
                    return `${actionText}失败: ${githookMsg.error ?? '未知错误'}`;
                }
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                let actionText = '';
                switch (versionMsg.action) {
                    case 'switch-branch':
                        actionText = '分支切换';
                        break;
                    case 'switch-tag':
                        actionText = '标签切换';
                        break;
                    case 'delete-tag':
                        actionText = '标签删除';
                        break;
                    default:
                        actionText = versionMsg.action;
                }

                if (versionMsg.success) {
                    return `${actionText}成功: ${versionMsg.target}`;
                } else {
                    return `${actionText}失败: ${versionMsg.error ?? '未知错误'}`;
                }
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                if (projectMsg.success) {
                    //添加、删除、编辑
                    switch (projectMsg.action) {
                        case 'add':
                            return `项目添加成功: ${projectMsg.projectPath}`;
                        case 'delete':
                            return `项目删除成功: ${projectMsg.projectPath}`;
                        case 'edit':
                            return `项目编辑成功: ${projectMsg.projectPath}`;
                        default:
                            return `项目${projectMsg.action}成功: ${projectMsg.projectPath}`;
                    }
                } else {
                    switch (projectMsg.action) {
                        case 'add':
                            return `项目添加失败: ${projectMsg.error ?? '未知错误'}`;
                        case 'delete':
                            return `项目删除失败: ${projectMsg.error ?? '未知错误'}`;
                        case 'edit':
                            return `项目编辑失败: ${projectMsg.error ?? '未知错误'}`;
                        default:
                            return `项目${projectMsg.action}失败: ${
                                projectMsg.error ?? '未知错误'
                            }`;
                    }
                }
            }
            default:
                return JSON.stringify(message.data);
        }
    };

    private formatTime = (timestamp: string): string =>
        new Date(timestamp).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });

    private getStatusChip = (message: IWebSocketMessage) => {
        let success = true;
        let label = '';

        switch (message.type) {
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                success = hookMsg.success;
                label = success ? '成功' : '失败';
                break;
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                success = versionMsg.success;
                label = success ? '成功' : '失败';
                break;
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                success = projectMsg.success;
                label = success ? '成功' : '失败';
                break;
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                success = githookMsg.success;
                if (success && githookMsg.skipped) {
                    label = '跳过';
                } else {
                    label = success ? '成功' : '失败';
                }
                break;
            }
            default:
                label = '信息';
        }

        // 优化状态标签的样式
        let chipProps = {};
        if (label === '成功') {
            chipProps = {
                sx: {
                    backgroundColor: 'success.main',
                    color: 'success.contrastText',
                    fontWeight: 'bold',
                    '&.MuiChip-outlined': {
                        borderColor: 'success.main',
                        backgroundColor: 'success.light',
                        color: 'success.dark',
                    }
                },
                variant: 'filled' as const
            };
        } else if (label === '失败') {
            chipProps = {
                sx: {
                    backgroundColor: 'error.main',
                    color: 'error.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const
            };
        } else if (label === '跳过') {
            chipProps = {
                sx: {
                    backgroundColor: 'warning.main',
                    color: 'warning.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const
            };
        } else {
            chipProps = {
                sx: {
                    backgroundColor: 'info.main',
                    color: 'info.contrastText',
                    fontWeight: 'bold',
                },
                variant: 'filled' as const
            };
        }

        return (
            <Chip
                size="small"
                label={label}
                {...chipProps}
            />
        );
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
                            sx={{ borderBottom: 1, borderColor: 'divider' }}>
                            <Typography variant="h6">Messages ({this.messages.length})</Typography>
                            <Box>
                                <Tooltip title="清空消息">
                                    <IconButton size="small" onClick={this.clearMessages}>
                                        <DeleteIcon />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title="发送心跳">
                                    <IconButton
                                        size="small"
                                        onClick={() => this.props.wsStore.sendPing()}>
                                        <RefreshIcon />
                                    </IconButton>
                                </Tooltip>
                            </Box>
                        </Box>

                        {/* 消息列表 - 美化滚动条 */}
                        <Box sx={{
                            overflowY: 'auto',
                            maxHeight: 400,
                            // 美化滚动条
                            '&::-webkit-scrollbar': {
                                width: '8px',
                            },
                            '&::-webkit-scrollbar-track': {
                                backgroundColor: 'action.hover',
                                borderRadius: '4px',
                            },
                            '&::-webkit-scrollbar-thumb': {
                                backgroundColor: 'action.selected',
                                borderRadius: '4px',
                                '&:hover': {
                                    backgroundColor: 'action.focus',
                                },
                            },
                            // Firefox 滚动条样式
                            scrollbarWidth: 'thin',
                            scrollbarColor: 'action.selected action.hover',
                        }}>
                            {this.messages.length === 0 ? (
                                <Box p={2} textAlign="center">
                                    <Typography variant="body2" color="textSecondary">
                                        No messages
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
                                                    }
                                                }}>
                                                <ListItemIcon sx={{ minWidth: 40 }}>
                                                    {this.getMessageIcon(message)}
                                                </ListItemIcon>
                                                <ListItemText
                                                    primary={
                                                        <Box
                                                            display="flex"
                                                            justifyContent="space-between"
                                                            alignItems="center">
                                                            <Typography variant="subtitle2" sx={{ fontWeight: 'medium' }}>
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
                                                                {this.getMessageDescription(message)}
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

export default inject('wsStore')(RealtimeMessages);
