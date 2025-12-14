import axios from 'axios';
import {action, observable, runInAction} from 'mobx';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {ISyncTask} from '../types';

export type TaskQuery = {
    projectName?: string;
    nodeId?: number;
    status?: string;
    limit?: number;
    beforeId?: number;
    includeLogs?: boolean;
};

export class SyncTaskStore {
    @observable
    public loading = false;

    @observable
    public refreshing = false;

    @observable
    public tasks: ISyncTask[] = [];

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {}

    private get headers() {
        return {'X-GoHook-Key': this.tokenProvider()};
    }

    @action
    public clear() {
        this.tasks = [];
        this.loading = false;
        this.refreshing = false;
    }

    @action
    public async loadTasks(query: TaskQuery, opts?: {silent?: boolean}): Promise<void> {
        const silent = !!opts?.silent;
        if (silent) {
            this.refreshing = true;
        } else {
            this.loading = true;
        }
        try {
            const params: Record<string, string> = {};
            if (query.projectName) params.projectName = query.projectName;
            if (query.nodeId) params.nodeId = String(query.nodeId);
            if (query.status) params.status = query.status;
            if (query.limit) params.limit = String(query.limit);
            if (query.beforeId) params.beforeId = String(query.beforeId);
            if (query.includeLogs) params.includeLogs = 'true';

            const response = await axios.get<ISyncTask[]>(`${config.get('url')}api/sync/tasks`, {
                headers: this.headers,
                params,
            });
            runInAction(() => {
                this.tasks = response.data || [];
            });
        } catch (error: unknown) {
            this.snack('加载任务失败');
            throw error;
        } finally {
            runInAction(() => {
                this.loading = false;
                this.refreshing = false;
            });
        }
    }

    @action
    public async loadTask(id: number, opts?: {includeLogs?: boolean}): Promise<ISyncTask> {
        const taskID = Number(id);
        if (!Number.isFinite(taskID) || taskID <= 0) {
            throw new Error('invalid task id');
        }
        const includeLogs = opts?.includeLogs !== false;
        const response = await axios.get<ISyncTask>(
            `${config.get('url')}api/sync/tasks/${taskID}`,
            {
                headers: this.headers,
                params: includeLogs ? {includeLogs: 'true'} : {includeLogs: 'false'},
            }
        );
        const item = response.data as ISyncTask;
        runInAction(() => {
            const idx = this.tasks.findIndex((t) => Number(t.id) === taskID);
            if (idx >= 0) {
                this.tasks[idx] = {...this.tasks[idx], ...item};
            } else {
                this.tasks = [item, ...this.tasks];
            }
        });
        return item;
    }

    public async clearTasks(query: TaskQuery, opts?: {includeActive?: boolean}): Promise<number> {
        const params: Record<string, string> = {};
        if (query.projectName) params.projectName = query.projectName;
        if (query.nodeId) params.nodeId = String(query.nodeId);
        if (query.status) params.status = query.status;
        if (opts?.includeActive) params.includeActive = 'true';
        const resp = await axios.delete<{deleted: number}>(`${config.get('url')}api/sync/tasks`, {
            headers: this.headers,
            params,
        });
        return Number(resp.data?.deleted || 0);
    }
}
