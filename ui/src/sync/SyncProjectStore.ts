import axios from 'axios';
import {action, observable, runInAction} from 'mobx';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {IProjectSyncConfig, ISyncProjectSummary} from '../types';

export class SyncProjectStore {
    @observable
    public loading = false;

    @observable
    public saving = false;

    @observable
    public projects: ISyncProjectSummary[] = [];

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {}

    private get headers() {
        return {'X-GoHook-Key': this.tokenProvider()};
    }

    @action
    public clear() {
        this.projects = [];
        this.loading = false;
        this.saving = false;
    }

    @action
    public async refreshProjects(): Promise<void> {
        this.loading = true;
        try {
            const response = await axios.get<ISyncProjectSummary[]>(
                `${config.get('url')}api/sync/projects`,
                {
                    headers: this.headers,
                }
            );
            runInAction(() => {
                this.projects = response.data || [];
            });
        } catch (error: unknown) {
            this.handleError(error, '加载同步项目失败');
            throw error;
        } finally {
            runInAction(() => {
                this.loading = false;
            });
        }
    }

    @action
    public async updateSyncConfig(projectName: string, sync: IProjectSyncConfig): Promise<void> {
        this.saving = true;
        try {
            await axios.put(
                `${config.get('url')}api/sync/projects/${encodeURIComponent(projectName)}/config`,
                {sync},
                {headers: this.headers}
            );
            await this.refreshProjects();
            this.snack('同步配置已保存');
        } catch (error: unknown) {
            this.handleError(error, '保存同步配置失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    }

    @action
    public async runProjectSync(projectName: string): Promise<void> {
        this.saving = true;
        try {
            await axios.post(
                `${config.get('url')}api/sync/projects/${encodeURIComponent(projectName)}/run`,
                {},
                {headers: this.headers}
            );
            await this.refreshProjects();
            this.snack('已触发同步');
        } catch (error: unknown) {
            this.handleError(error, '触发同步失败');
            throw error;
        } finally {
            runInAction(() => {
                this.saving = false;
            });
        }
    }

    private handleError(error: unknown, fallback: string) {
        const detail =
            (error as {response?: {data?: {error?: string; message?: string}}})?.response?.data
                ?.error ||
            (error as {response?: {data?: {message?: string}}})?.response?.data?.message ||
            (error instanceof Error ? error.message : '');
        this.snack(detail ? `${fallback}: ${detail}` : fallback);
    }
}
