import {action, observable, runInAction} from 'mobx';
import axios from 'axios';
import * as config from '../config';

export interface LogEntry {
    id: number;
    type: 'hook' | 'system' | 'user' | 'project';
    timestamp: string;
    level?: string;
    category?: string;
    message: string;
    details?: any;

    // Hook logs specific
    hookId?: string;
    hookName?: string;
    hookType?: string;
    method?: string;
    remoteAddr?: string;
    success?: boolean;
    output?: string;
    error?: string;
    duration?: number;
    userAgent?: string;
    queryParams?: Record<string, string[]>;
    headers?: Record<string, string[]>;
    body?: string;

    // User activity specific
    username?: string;
    action?: string;
    resource?: string;
    description?: string;
    ipAddress?: string;

    // Project activity specific
    projectName?: string;
    oldValue?: string;
    newValue?: string;
    commitHash?: string;

    // Common fields
    userId?: string;
}

export interface LogFilters {
    type?: string;
    level?: string;
    category?: string;
    startDate?: string;
    endDate?: string;
    search?: string;
    user?: string;
    project?: string;
    success?: boolean;
}

export interface LogResponse {
    logs: LogEntry[];
    total: number;
    page: number;
    pageSize: number;
    hasMore: boolean;
}

class LogStore {
    @observable
    logs: LogEntry[] = [];
    @observable
    loading = false;
    @observable
    filters: LogFilters = {};
    @observable
    page = 1;
    @observable
    pageSize = 20;
    @observable
    total = 0;
    @observable
    hasMore = true;
    @observable
    autoRefresh = false;
    @observable
    refreshInterval = 30; // seconds
    private refreshTimer?: NodeJS.Timeout;

    constructor() {}

    @action
    setFilters(filters: Partial<LogFilters>) {
        this.filters = {...this.filters, ...filters};
        this.resetPagination();
        this.loadLogs();
    }

    @action
    clearFilters() {
        this.filters = {};
        this.resetPagination();
        this.loadLogs();
    }

    @action
    setAutoRefresh(enabled: boolean, interval = 30) {
        this.autoRefresh = enabled;
        this.refreshInterval = interval;

        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = undefined;
        }

        if (enabled) {
            this.refreshTimer = setInterval(() => {
                this.refreshLogs();
            }, interval * 1000);
        }
    }

    @action
    resetPagination() {
        this.page = 1;
        this.logs = [];
        this.hasMore = true;
    }

    @action
    async loadLogs(append = false) {
        if (this.loading) return;

        runInAction(() => {
            this.loading = true;
        });

        try {
            const params = new URLSearchParams();
            params.append('page', this.page.toString());
            params.append('pageSize', this.pageSize.toString());

            if (this.filters.type) params.append('type', this.filters.type);
            if (this.filters.level) params.append('level', this.filters.level);
            if (this.filters.category) params.append('category', this.filters.category);
            if (this.filters.startDate) {
                // 确保时间格式正确
                const startDate = new Date(this.filters.startDate).toISOString();
                params.append('startDate', startDate);
            }
            if (this.filters.endDate) {
                // 确保时间格式正确
                const endDate = new Date(this.filters.endDate).toISOString();
                params.append('endDate', endDate);
            }
            if (this.filters.search) params.append('search', this.filters.search);
            if (this.filters.user) params.append('user', this.filters.user);
            if (this.filters.project) params.append('project', this.filters.project);
            if (this.filters.success !== undefined)
                params.append('success', this.filters.success.toString());

            const response = await axios.get<LogResponse>(
                `${config.get('url')}api/logs?${params.toString()}`
            );

            runInAction(() => {
                const logs = response.data.logs || [];
                if (append) {
                    this.logs.push(...logs);
                } else {
                    this.logs = logs;
                }
                this.total = response.data.total || 0;
                this.hasMore = response.data.hasMore !== false; // 修复hasMore逻辑
                this.loading = false;
            });
        } catch (error) {
            runInAction(() => {
                this.loading = false;
            });
            console.error('加载日志失败:', error);
            throw error;
        }
    }

    @action
    async loadMore() {
        if (!this.hasMore || this.loading) return;

        this.page += 1;
        await this.loadLogs(true);
    }

    async refreshLogs() {
        this.resetPagination();
        await this.loadLogs();
    }

    async clearLogs(days: number) {
        await axios.delete(`${config.get('url')}api/logs/cleanup?days=${days}`);
        await this.refreshLogs();
    }

    async exportLogs() {
        const params = new URLSearchParams();
        if (this.filters.type) params.append('type', this.filters.type);
        if (this.filters.level) params.append('level', this.filters.level);
        if (this.filters.category) params.append('category', this.filters.category);
        if (this.filters.startDate) {
            const startDate = new Date(this.filters.startDate).toISOString();
            params.append('startDate', startDate);
        }
        if (this.filters.endDate) {
            const endDate = new Date(this.filters.endDate).toISOString();
            params.append('endDate', endDate);
        }
        if (this.filters.search) params.append('search', this.filters.search);
        if (this.filters.user) params.append('user', this.filters.user);
        if (this.filters.project) params.append('project', this.filters.project);
        if (this.filters.success !== undefined)
            params.append('success', this.filters.success.toString());

        const response = await axios.get(
            `${config.get('url')}api/logs/export?${params.toString()}`,
            {
                responseType: 'blob',
            }
        );

        // Create download link
        const url = window.URL.createObjectURL(new Blob([response.data]));
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute('download', `logs_${new Date().toISOString().split('T')[0]}.csv`);
        document.body.appendChild(link);
        link.click();
        link.remove();
        window.URL.revokeObjectURL(url);
    }

    destroy() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
        }
    }
}

export default LogStore;
 