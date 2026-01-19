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
import useTranslation from '../i18n/useTranslation';

interface WebSocketStatusIndicatorProps {
    wsStore: WebSocketStore;
    t: (key: string, params?: Record<string, string | number>) => string;
    language: string;
}
interface WebSocketStatusIndicatorPublicProps {
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
        const {t} = this.props;
        switch (status) {
            case 'connected':
                return t('websocket.status.connected');
            case 'connecting':
                return t('websocket.status.connecting');
            case 'reconnecting':
                return t('websocket.status.reconnecting', {current: reconnectAttempts, total: 10});
            case 'disconnected':
            default:
                return t('websocket.status.disconnected');
        }
    };

    private formatTime = (date: Date | null): string => {
        const {t, language} = this.props;
        if (!date) return t('websocket.none');
        const locale = language.startsWith('zh') ? 'zh-CN' : 'en-US';
        return date.toLocaleTimeString(locale);
    };

    public render() {
        const {t} = this.props;
        const health = this.props.wsStore.getConnectionHealth();
        const {status, isHealthy, lastHeartbeat, error, reconnectAttempts} = health;

        const tooltipContent = (
            <Box
                sx={{
                    maxWidth: 280,
                    whiteSpace: 'normal',
                    overflowWrap: 'anywhere',
                    wordBreak: 'break-word',
                }}>
                <Typography variant="body2">
                    <strong>{t('websocket.labels.status')}:</strong>{' '}
                    {this.getStatusText(status, reconnectAttempts)}
                </Typography>
                <Typography variant="body2">
                    <strong>{t('websocket.labels.health')}:</strong>{' '}
                    {isHealthy ? t('websocket.health.healthy') : t('websocket.health.unhealthy')}
                </Typography>
                <Typography variant="body2">
                    <strong>{t('websocket.labels.lastHeartbeat')}:</strong>{' '}
                    {this.formatTime(lastHeartbeat)}
                </Typography>
                {error && (
                    <Typography variant="body2" color="error">
                        <strong>{t('websocket.labels.error')}:</strong> {error}
                    </Typography>
                )}
                <Typography variant="caption" color="textSecondary" sx={{display: 'block', mt: 1}}>
                    {t('websocket.manualPing')}
                </Typography>
            </Box>
        );

        return (
            <Tooltip
                title={tooltipContent}
                arrow
                placement="top-end"
                PopperProps={{
                    modifiers: [
                        {
                            name: 'preventOverflow',
                            options: {boundary: 'viewport', padding: 8},
                        },
                        {
                            name: 'flip',
                            options: {fallbackPlacements: ['top', 'bottom', 'left']},
                        },
                    ],
                }}>
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

const WebSocketStatusIndicatorWithTranslation: React.FC<WebSocketStatusIndicatorPublicProps> = (
    props
) => {
    const {t, i18n} = useTranslation();
    return <WebSocketStatusIndicator {...props} t={t} language={i18n.language} />;
};

export default WebSocketStatusIndicatorWithTranslation;
