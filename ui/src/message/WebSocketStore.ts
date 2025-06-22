import {SnackReporter} from '../snack/SnackManager';
import {CurrentUser} from '../CurrentUser';
import * as config from '../config';
import {AxiosError} from 'axios';
import {
    IMessage,
    IWebSocketMessage,
    IHookTriggeredMessage,
    IVersionSwitchMessage,
    IProjectManageMessage,
    IGitHookTriggeredMessage,
} from '../types';

export class WebSocketStore {
    private wsActive = false;
    private ws: WebSocket | null = null;
    private messageCallbacks: Array<(msg: IWebSocketMessage) => void> = [];
    private reconnectTimer: NodeJS.Timeout | null = null;

    public constructor(
        private readonly snack: SnackReporter,
        private readonly currentUser: CurrentUser
    ) {}

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    public listen = (_callback: (msg: IMessage) => void) => {
        this.connectWebSocket();
    };

    public onMessage = (callback: (msg: IWebSocketMessage) => void) => {
        this.messageCallbacks.push(callback);
        this.connectWebSocket();
    };

    public offMessage = (callback: (msg: IWebSocketMessage) => void) => {
        const index = this.messageCallbacks.indexOf(callback);
        if (index > -1) {
            this.messageCallbacks.splice(index, 1);
        }
    };

    private connectWebSocket = () => {
        if (!this.currentUser.token() || this.wsActive) {
            return;
        }
        this.wsActive = true;

        const wsUrl = config.get('url').replace('http', 'ws').replace('https', 'wss');
        const ws = new WebSocket(wsUrl + 'stream', ['Authorization', this.currentUser.token()]);

        ws.onerror = (e) => {
            this.wsActive = false;
            console.log('WebSocket connection errored', e);
            this.snack('WebSocket connection errored');
        };

        ws.onopen = () => {
            console.log('WebSocket connected');
            this.snack('WebSocket connected');
            if (this.reconnectTimer) {
                clearTimeout(this.reconnectTimer);
                this.reconnectTimer = null;
            }
        };

        ws.onmessage = (event) => {
            try {
                const message: IWebSocketMessage = JSON.parse(event.data);
                console.log('WebSocket message received:', message);

                this.showMessageNotification(message);

                this.messageCallbacks.forEach((callback) => {
                    try {
                        callback(message);
                    } catch (error) {
                        console.error('Error in WebSocket message callback:', error);
                    }
                });
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        ws.onclose = (event) => {
            this.wsActive = false;
            console.log('WebSocket connection closed', event);

            this.currentUser
                .tryAuthenticate()
                .then(() => {
                    this.snack('WebSocket连接已断开，30秒后重新连接');
                    this.reconnectTimer = setTimeout(() => this.connectWebSocket(), 30000);
                })
                .catch((error: AxiosError) => {
                    if (error?.response?.status === 401) {
                        this.snack('客户端token认证失败，请重新登录');
                    } else {
                        this.snack('WebSocket重连失败');
                    }
                });
        };

        this.ws = ws;
    };

    private showMessageNotification = (message: IWebSocketMessage) => {
        switch (message.type) {
            case 'connected':
                // 连接消息不显示通知
                break;
            case 'hook_triggered': {
                const hookMsg = message.data as IHookTriggeredMessage;
                if (hookMsg.success) {
                    this.snack(`Hook "${hookMsg.hookName}" 执行成功`);
                } else {
                    this.snack(
                        `Hook "${hookMsg.hookName}" 执行失败: ${hookMsg.error ?? '未知错误'}`
                    );
                }
                break;
            }
            case 'version_switched': {
                const versionMsg = message.data as IVersionSwitchMessage;
                if (versionMsg.success) {
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
                    this.snack(
                        `项目 "${versionMsg.projectName}" ${actionText}成功: ${versionMsg.target}`
                    );
                } else {
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
                    this.snack(
                        `项目 "${versionMsg.projectName}" ${actionText}失败: ${
                            versionMsg.error ?? '未知错误'
                        }`
                    );
                }
                break;
            }
            case 'project_managed': {
                const projectMsg = message.data as IProjectManageMessage;
                if (projectMsg.success) {
                    let actionText = '';
                    switch (projectMsg.action) {
                        case 'add':
                            actionText = '添加';
                            break;
                        case 'delete':
                            actionText = '删除';
                            break;
                        case 'edit':
                            actionText = '编辑';
                            break;
                        default:
                            actionText = projectMsg.action;
                    }
                    this.snack(`项目 "${projectMsg.projectName}" ${actionText}成功`);
                } else {
                    let actionText = '';
                    switch (projectMsg.action) {
                        case 'add':
                            actionText = '添加';
                            break;
                        case 'delete':
                            actionText = '删除';
                            break;
                        case 'edit':
                            actionText = '编辑';
                            break;
                        default:
                            actionText = projectMsg.action;
                    }
                    this.snack(
                        `项目 "${projectMsg.projectName}" ${actionText}失败: ${
                            projectMsg.error ?? '未知错误'
                        }`
                    );
                }
                break;
            }
            case 'githook_triggered': {
                const githookMsg = message.data as IGitHookTriggeredMessage;
                if (githookMsg.success) {
                    if (githookMsg.skipped) {
                        this.snack(`GitHook "${githookMsg.projectName}" 跳过处理: ${githookMsg.message || '无需操作'}`);
                    } else {
                        this.snack(`GitHook "${githookMsg.projectName}" 执行成功: ${githookMsg.message || ''}`);
                    }
                } else {
                    this.snack(
                        `GitHook "${githookMsg.projectName}" 执行失败: ${
                            githookMsg.error ?? '未知错误'
                        }`
                    );
                }
                break;
            }
            case 'pong':
                // 心跳响应不显示通知
                console.log('Heart beat pong received');
                break;
            default:
                console.log('Unknown WebSocket message type:', message.type);
        }
    };

    public sendPing = () => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(
                JSON.stringify({
                    type: 'ping',
                    timestamp: new Date().toISOString(),
                })
            );
        }
    };

    public close = () => {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.ws?.close(1000, 'WebSocketStore#close');
        this.wsActive = false;
    };
}
