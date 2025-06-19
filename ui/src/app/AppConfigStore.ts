import axios from 'axios';
import {action, observable} from 'mobx';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';

export interface IAppConfig {
    port: number;
    mode: string; // "dev" | "test" | "prod"
}

export class AppConfigStore {
    @observable
    public appConfig: IAppConfig | null = null;

    @observable
    public loading = false;

    public constructor(
        private readonly tokenProvider: () => string,
        private readonly snack: SnackReporter
    ) {}

    @action
    public async fetchAppConfig(): Promise<void> {
        if (this.loading) return;
        
        this.loading = true;
        try {
            const response = await axios.get<IAppConfig>(`${config.get('url')}app/config`, {
                headers: {'X-GoHook-Key': this.tokenProvider()}
            });
            this.appConfig = response.data;
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message :
                (error as { response?: { data?: { error?: string } } })?.response?.data?.error ??
                '获取应用配置失败';
            console.warn('获取应用配置失败:', errorMessage);
            // 不显示错误消息，静默失败
        } finally {
            this.loading = false;
        }
    }

    public getEnvironmentMode(): string {
        return this.appConfig?.mode || 'unknown';
    }
} 