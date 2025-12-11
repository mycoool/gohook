import axios from 'axios';
import {action, computed, observable, runInAction} from 'mobx';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {ISyncNode} from '../types';

export interface SyncNodePayload {
    name: string;
    address: string;
    type: string;
    sshUser?: string;
    sshPort?: number;
    authType?: string;
    credentialRef?: string;
    tags?: string[];
    metadata?: Record<string, unknown>;
    ignoreDefaults?: boolean;
    ignorePatterns?: string[];
    ignoreFile?: string;
    autoInstall?: boolean;
}

export interface SyncNodeUpdatePayload extends SyncNodePayload {
    autoInstall?: boolean;
}

export class SyncNodeStore {
    @observable
    private nodes: ISyncNode[] = [];

    @observable
    public loading = false;

    @observable
    public saving = false;

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {}

    private get headers() {
        return {'X-GoHook-Key': this.tokenProvider()};
    }

    @computed
    public get all(): ISyncNode[] {
        return this.nodes;
    }

    @action
    public clear() {
        this.nodes = [];
        this.loading = false;
        this.saving = false;
    }

    @action
    public refreshNodes = async () => {
        this.loading = true;
        try {
            const response = await axios.get<ISyncNode[]>(`${config.get('url')}api/sync/nodes`, {
                headers: this.headers,
            });
            runInAction(() => {
                this.nodes = response.data || [];
            });
        } catch (error: unknown) {
            this.handleError(error, '加载节点列表失败');
            throw error;
        } finally {
            runInAction(() => {
                this.loading = false;
            });
        }
    };

    @action
    public createNode = async (payload: SyncNodePayload) => {
        this.saving = true;
        try {
            await axios.post(`${config.get('url')}api/sync/nodes`, payload, {
                headers: this.headers,
            });
            this.snack('节点创建成功');
            await this.refreshNodes();
        } catch (error: unknown) {
            this.handleError(error, '节点创建失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    };

    @action
    public updateNode = async (id: number, payload: SyncNodeUpdatePayload) => {
        this.saving = true;
        try {
            await axios.put(`${config.get('url')}api/sync/nodes/${id}`, payload, {
                headers: this.headers,
            });
            this.snack('节点更新成功');
            await this.refreshNodes();
        } catch (error: unknown) {
            this.handleError(error, '节点更新失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    };

    @action
    public deleteNode = async (id: number) => {
        this.saving = true;
        try {
            await axios.delete(`${config.get('url')}api/sync/nodes/${id}`, {
                headers: this.headers,
            });
            this.snack('节点已删除');
            await this.refreshNodes();
        } catch (error: unknown) {
            this.handleError(error, '删除节点失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    };

    @action
    public triggerInstall = async (id: number, payload?: {sshUser?: string; sshPort?: number}) => {
        this.saving = true;
        try {
            await axios.post(`${config.get('url')}api/sync/nodes/${id}/install`, payload ?? {}, {
                headers: this.headers,
            });
            this.snack('已启动安装任务');
            await this.refreshNodes();
        } catch (error: unknown) {
            this.handleError(error, '启动安装失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    };

    private handleError(error: unknown, fallback: string) {
        const detail =
            (error as {response?: {data?: {error?: string; message?: string}}})?.response?.data
                ?.error ||
            (error as {response?: {data?: {message?: string}}})?.response?.data?.message ||
            (error instanceof Error ? error.message : '');
        this.snack(detail ? `${fallback}: ${detail}` : fallback);
    }
}
