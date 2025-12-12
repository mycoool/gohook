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
    IHookManageMessage,
} from '../types';
import {observable} from 'mobx'; // 导入observable

export class WebSocketStore {
    private wsActive = false;
    private ws: WebSocket | null = null;
    private messageCallbacks: Array<(msg: IWebSocketMessage) => void> = [];
    private reconnectTimer: NodeJS.Timeout | null = null;
    private heartbeatTimer: NodeJS.Timeout | null = null; // 添加心跳定时器
    private firstHeartbeatTimer: NodeJS.Timeout | null = null; // 首次心跳定时器
    private initialHeartbeatDelay = 10000; // 首次心跳延迟10秒
    private heartbeatInterval = 30000; // 30秒心跳间隔
    private reconnectAttempts = 0; // 重连尝试次数
    private maxReconnectAttempts = 10; // 最大重连次数
    private baseReconnectDelay = 5000; // 基础重连延迟5秒

    // 可观察的连接状态
    @observable
    public connectionStatus: 'disconnected' | 'connecting' | 'connected' | 'reconnecting' =
        'disconnected';

    @observable
    public lastHeartbeatTime: Date | null = null;

    @observable
    public connectionError: string | null = null;

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
        this.connectionStatus = this.reconnectAttempts > 0 ? 'reconnecting' : 'connecting';
        this.connectionError = null;

        const wsUrl = config.get('url').replace('http', 'ws').replace('https', 'wss');
        const ws = new WebSocket(wsUrl + 'stream', ['Authorization', this.currentUser.token()]);

        ws.onerror = (e) => {
            this.wsActive = false;
            this.connectionStatus = 'disconnected';
            this.connectionError = 'WebSocket connection error';
            console.log('WebSocket connection errored', e);
            this.snack('WebSocket connection errored');
            this.stopHeartbeat(); // 停止心跳
        };

        ws.onopen = () => {
            console.log('WebSocket connected');
            this.connectionStatus = 'connected';
            this.connectionError = null;
            this.lastHeartbeatTime = new Date(); // 连接建立时初始化心跳时间
            this.snack('WebSocket connected');
            this.resetReconnectState(); // 连接成功时重置重连状态
            this.startHeartbeat(); // 开始心跳
        };

        ws.onmessage = (event) => {
            try {
                const message: IWebSocketMessage = JSON.parse(event.data);
                console.log('WebSocket message received:', message);

                // 更新心跳时间
                if (message.type === 'pong') {
                    this.lastHeartbeatTime = new Date();
                }

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
                this.connectionError = 'Failed to parse WebSocket message';
            }
        };

        ws.onclose = (event) => {
            this.wsActive = false;
            this.connectionStatus = 'disconnected';
            this.stopHeartbeat(); // 停止心跳
            console.log('WebSocket connection closed', event);

            // 根据关闭代码设置错误信息
            if (event.code !== 1000) {
                this.connectionError = `Connection closed unexpectedly (code: ${event.code})`;
                this.attemptReconnect();
            } else {
                this.connectionError = null; // 正常关闭
            }
        };

        this.ws = ws;
    };

    // 尝试重连的方法
    private attemptReconnect = () => {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            this.snack(`WebSocket重连失败，已达到最大重连次数(${this.maxReconnectAttempts})`);
            return;
        }

        this.currentUser
            .tryAuthenticate()
            .then(() => {
                this.reconnectAttempts++;
                // 指数退避：延迟时间随重连次数增加
                const delay = Math.min(
                    this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
                    60000 // 最大延迟60秒
                );

                this.snack(
                    `WebSocket连接已断开，${Math.round(delay / 1000)}秒后进行第${
                        this.reconnectAttempts
                    }次重连`
                );
                this.reconnectTimer = setTimeout(() => {
                    this.connectWebSocket();
                }, delay);
            })
            .catch((error: AxiosError) => {
                if (error?.response?.status === 401) {
                    this.snack('客户端token认证失败，请重新登录');
                    this.reconnectAttempts = 0; // 重置重连次数
                } else {
                    this.snack('WebSocket重连失败');
                    // 认证失败时也尝试重连，但延迟更长
                    setTimeout(() => this.attemptReconnect(), 30000);
                }
            });
    };

    // 重置重连状态（连接成功时调用）
    private resetReconnectState = () => {
        this.reconnectAttempts = 0;
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
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
            case 'hook_managed': {
                const hookManageMsg = message.data as IHookManageMessage;
                if (hookManageMsg.success) {
                    let actionText = '';
                    switch (hookManageMsg.action) {
                        case 'create':
                            actionText = '创建';
                            break;
                        case 'update_basic':
                            actionText = '基本信息更新';
                            break;
                        case 'update_parameters':
                            actionText = '参数配置更新';
                            break;
                        case 'update_triggers':
                            actionText = '触发规则更新';
                            break;
                        case 'update_response':
                            actionText = '响应配置更新';
                            break;
                        case 'update_script':
                            actionText = '脚本更新';
                            break;
                        case 'delete':
                            actionText = '删除';
                            break;
                        default:
                            actionText = hookManageMsg.action;
                    }
                    this.snack(`Hook "${hookManageMsg.hookName}" ${actionText}成功`);
                } else {
                    let actionText = '';
                    switch (hookManageMsg.action) {
                        case 'create':
                            actionText = '创建';
                            break;
                        case 'update_basic':
                            actionText = '基本信息更新';
                            break;
                        case 'update_parameters':
                            actionText = '参数配置更新';
                            break;
                        case 'update_triggers':
                            actionText = '触发规则更新';
                            break;
                        case 'update_response':
                            actionText = '响应配置更新';
                            break;
                        case 'update_script':
                            actionText = '脚本更新';
                            break;
                        case 'delete':
                            actionText = '删除';
                            break;
                        default:
                            actionText = hookManageMsg.action;
                    }
                    this.snack(
                        `Hook "${hookManageMsg.hookName}" ${actionText}失败: ${
                            hookManageMsg.error ?? '未知错误'
                        }`
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
                        this.snack(
                            `GitHook "${githookMsg.projectName}" 跳过处理: ${
                                githookMsg.message || '无需操作'
                            }`
                        );
                    } else {
                        this.snack(
                            `GitHook "${githookMsg.projectName}" 执行成功: ${
                                githookMsg.message || ''
                            }`
                        );
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

    // 开始心跳
    private startHeartbeat = () => {
        this.stopHeartbeat(); // 确保没有重复的定时器
        // 首次心跳提前到连接成功后10秒发送，后续保持30秒间隔
        this.firstHeartbeatTimer = setTimeout(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.sendPing();
                this.heartbeatTimer = setInterval(() => {
                    this.sendPing();
                }, this.heartbeatInterval);
                console.log('WebSocket heartbeat started');
            } else {
                console.warn('Cannot start heartbeat: WebSocket is not open');
            }
        }, this.initialHeartbeatDelay);
        console.log('WebSocket first heartbeat scheduled');
    };

    // 停止心跳
    private stopHeartbeat = () => {
        if (this.firstHeartbeatTimer) {
            clearTimeout(this.firstHeartbeatTimer);
            this.firstHeartbeatTimer = null;
        }
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
            console.log('WebSocket heartbeat stopped');
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
            console.log('WebSocket ping sent');
        } else {
            console.warn('Cannot send ping: WebSocket is not open');
            this.connectionError = 'WebSocket is not connected';
        }
    };

    // 获取连接健康状态
    public getConnectionHealth = () => {
        const now = new Date();
        let isHealthy = false;

        if (this.connectionStatus === 'connected') {
            if (this.lastHeartbeatTime) {
                // 如果有心跳记录，检查是否在合理时间内
                const timeSinceLastHeartbeat = now.getTime() - this.lastHeartbeatTime.getTime();
                isHealthy = timeSinceLastHeartbeat < this.heartbeatInterval * 3; // 放宽到3倍心跳间隔
            } else {
                // 刚连接但还没有心跳记录，给一定宽限期
                isHealthy = true; // 刚连接时认为是健康的
            }
        }

        return {
            status: this.connectionStatus,
            isHealthy,
            lastHeartbeat: this.lastHeartbeatTime,
            error: this.connectionError,
            reconnectAttempts: this.reconnectAttempts,
        };
    };

    public close = () => {
        this.stopHeartbeat(); // 停止心跳
        this.resetReconnectState(); // 重置重连状态
        this.connectionStatus = 'disconnected';
        this.connectionError = null;
        this.ws?.close(1000, 'WebSocketStore#close'); // 使用正常关闭代码
        this.wsActive = false;
    };
}
