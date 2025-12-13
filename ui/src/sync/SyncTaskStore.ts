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
    includeLogs?: boolean;
};

export class SyncTaskStore {
    @observable
    public loading = false;

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
    }

    @action
    public async loadTasks(query: TaskQuery): Promise<void> {
        this.loading = true;
        try {
            const params: Record<string, string> = {};
            if (query.projectName) params.projectName = query.projectName;
            if (query.nodeId) params.nodeId = String(query.nodeId);
            if (query.status) params.status = query.status;
            if (query.limit) params.limit = String(query.limit);
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
            });
        }
    }
}
