import axios from 'axios';
import {action, observable} from 'mobx';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';

export interface IAppConfig {
    mode: string; // "dev" | "test" | "prod"
    panel_alias: string; // 面板别名，用于浏览器标题
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
            const response = await axios.get<IAppConfig>(`${config.get('url')}app/config`);
            this.appConfig = response.data;

            // 设置浏览器标题
            const panelAlias = response.data.panel_alias?.trim() || 'GoHook';
            document.title = panelAlias;
            config.set('panelAlias', panelAlias);
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '获取应用配置失败';
            console.warn('获取应用配置失败:', errorMessage);

            // 如果获取失败，使用默认标题
            const fallbackAlias = config.get('panelAlias') || 'GoHook';
            document.title = fallbackAlias;
        } finally {
            this.loading = false;
        }
    }

    public getEnvironmentMode(): string {
        return this.appConfig?.mode || 'unknown';
    }

    public getPanelAlias(): string {
        return this.appConfig?.panel_alias || 'GoHook';
    }
}
