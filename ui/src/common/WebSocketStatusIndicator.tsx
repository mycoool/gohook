import React from 'react';
import {observer} from 'mobx-react';
import {IconButton, Tooltip, Box, Typography} from '@mui/material';
import {
    Wifi as ConnectedIcon,
    WifiOff as DisconnectedIcon,
    Sync as ConnectingIcon,
    Warning as WarningIcon,
} from '@mui/icons-material';
import {WebSocketStore} from '../message/WebSocketStore';

interface WebSocketStatusIndicatorProps {
    wsStore: WebSocketStore;
}

@observer
export class WebSocketStatusIndicator extends React.Component<WebSocketStatusIndicatorProps> {
    private getStatusIcon = (status: string, isHealthy: boolean) => {
        switch (status) {
            case 'connected':
                return isHealthy ? <ConnectedIcon /> : <WarningIcon />;
            case 'connecting':
            case 'reconnecting':
                return (
                    <ConnectingIcon
                        sx={{
                            animation: 'spin 1s linear infinite',
                            '@keyframes spin': {
                                '0%': {
                                    transform: 'rotate(0deg)',
                                },
                                '100%': {
                                    transform: 'rotate(360deg)',
                                },
                            },
                        }}
                    />
                );
            case 'disconnected':
            default:
                return <DisconnectedIcon />;
        }
    };

    private getStatusColor = (status: string, isHealthy: boolean) => {
        switch (status) {
            case 'connected':
                return isHealthy ? '#4caf50' : '#ff9800'; // 绿色或橙色
            case 'connecting':
            case 'reconnecting':
                return '#2196f3'; // 蓝色
            case 'disconnected':
            default:
                return '#f44336'; // 红色
        }
    };

    private getStatusText = (status: string, reconnectAttempts: number) => {
        switch (status) {
            case 'connected':
                return 'WebSocket已连接';
            case 'connecting':
                return 'WebSocket连接中...';
            case 'reconnecting':
                return `重连中... (${reconnectAttempts}/10)`;
            case 'disconnected':
            default:
                return 'WebSocket已断开';
        }
    };

    private formatTime = (date: Date | null): string => {
        if (!date) return '无';
        return date.toLocaleTimeString('zh-CN');
    };

    public render() {
        const health = this.props.wsStore.getConnectionHealth();
        const {status, isHealthy, lastHeartbeat, error, reconnectAttempts} = health;

        const tooltipContent = (
            <Box>
                <Typography variant="body2">
                    <strong>状态:</strong> {this.getStatusText(status, reconnectAttempts)}
                </Typography>
                <Typography variant="body2">
                    <strong>健康状态:</strong> {isHealthy ? '良好' : '异常'}
                </Typography>
                <Typography variant="body2">
                    <strong>最后心跳:</strong> {this.formatTime(lastHeartbeat)}
                </Typography>
                {error && (
                    <Typography variant="body2" color="error">
                        <strong>错误:</strong> {error}
                    </Typography>
                )}
                <Typography variant="caption" color="textSecondary" sx={{display: 'block', mt: 1}}>
                    点击发送手动心跳
                </Typography>
            </Box>
        );

        return (
            <Tooltip title={tooltipContent} arrow>
                <IconButton
                    size="small"
                    onClick={() => this.props.wsStore.sendPing()}
                    sx={{
                        color: this.getStatusColor(status, isHealthy),
                        '&:hover': {
                            backgroundColor: 'rgba(0, 0, 0, 0.04)',
                        },
                    }}>
                    {this.getStatusIcon(status, isHealthy)}
                </IconButton>
            </Tooltip>
        );
    }
}
