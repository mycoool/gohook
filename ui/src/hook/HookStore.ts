import axios from 'axios';
import * as config from '../config';
import {action, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IHook} from '../types';

export class HookStore {
    @observable
    protected items: IHook[] = [];

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {}

    protected requestItems = (): Promise<IHook[]> =>
        axios
            .get<IHook[]>(`${config.get('url')}hook`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            })
            .then((response) => response.data);

    protected requestDelete = (id: string): Promise<void> =>
        axios
            .delete(`${config.get('url')}hook/${id}`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            })
            .then(() => this.snack('Hook deleted'));

    @action
    public remove = async (id: string): Promise<void> => {
        await this.requestDelete(id);
        await this.refresh();
    };

    @action
    public refresh = async (): Promise<void> => {
        this.items = await this.requestItems().then((items) => items || []);
    };

    @action
    public reloadConfig = async (): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}hook/reload-config`,
                {},
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || 'Hooks配置加载成功');
            await this.refresh(); // 加载后刷新数据
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('加载Hook失败: ' + errorMessage);
        }
    };

    @action
    public triggerHook = async (id: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}hook/${id}/trigger`,
                {},
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || 'Hook triggered successfully');
        } catch (error: unknown) {
            // 处理错误情况
            if (error && typeof error === 'object' && 'response' in error) {
                const axiosError = error as {
                    response?: {
                        status?: number;
                        data?: {
                            error?: string;
                            message?: string;
                            output?: string;
                            hook?: string;
                        };
                    };
                };

                const responseData = axiosError.response?.data;
                if (responseData) {
                    // 构建详细的错误消息
                    let errorMessage = responseData.message || 'Hook execution failed';
                    if (responseData.hook) {
                        errorMessage += ` (${responseData.hook})`;
                    }
                    if (responseData.error) {
                        errorMessage += `\n错误详情: ${responseData.error}`;
                    }
                    if (responseData.output) {
                        errorMessage += `\n输出: ${responseData.output}`;
                    }

                    this.snack(errorMessage);
                    return; // 不重新抛出错误，避免未捕获的异常
                }
            }

            // 兜底错误处理
            const errorMessage = error instanceof Error ? error.message : 'Hook执行失败：未知错误';
            this.snack(errorMessage);
        }
    };

    @action
    public getHookDetails = async (id: string): Promise<IHook> => {
        const response = await axios.get<IHook>(`${config.get('url')}hook/${id}`, {
            headers: {'X-GoHook-Key': this.tokenProvider()},
        });
        return response.data;
    };

    public getName = (id: string): string => {
        const hook = this.getByIDOrUndefined(id);
        return hook !== undefined ? hook.name : 'unknown';
    };

    public getByIDOrUndefined = (id: string): IHook | undefined =>
        this.items.find((hook) => hook.id === id);

    public getByID = (id: string): IHook => {
        const hook = this.getByIDOrUndefined(id);
        if (hook === undefined) {
            throw new Error(`Hook with id ${id} not found`);
        }
        return hook;
    };

    public getItems = (): IHook[] => this.items;

    @action
    public clear = (): void => {
        this.items = [];
    };

    @action
    public saveHook = async (hook: IHook): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}hook/${hook.id}/config`,
                hook,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || 'Hook配置保存成功');
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '保存Hook配置失败';
            this.snack('保存Hook配置失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    // 脚本文件管理方法

    public getScript = async (
        hookId: string
    ): Promise<{content: string; exists: boolean; path: string}> => {
        try {
            const response = await axios.get<{content: string; exists: boolean; path: string}>(
                `${config.get('url')}hook/${hookId}/script`,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            return response.data;
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '获取脚本文件失败';
            this.snack('获取脚本文件失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    public saveScript = async (hookId: string, content: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}hook/${hookId}/script`,
                {
                    content: content,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '脚本文件保存成功');
        } catch (error: unknown) {
            if (error && typeof error === 'object' && 'response' in error) {
                const axiosError = error as {
                    response?: {data?: {error?: string; details?: string[]}};
                };
                const errorData = axiosError.response?.data;

                if (errorData?.details && Array.isArray(errorData.details)) {
                    // 格式验证失败，显示详细错误信息
                    const details = errorData.details.join('\n');
                    this.snack(`脚本文件格式验证失败:\n${details}`);
                    throw new Error(`格式验证失败:\n${details}`);
                } else {
                    const errorMessage = errorData?.error ?? '保存脚本文件失败';
                    this.snack('保存脚本文件失败: ' + errorMessage);
                    throw new Error(errorMessage);
                }
            } else {
                const errorMessage = error instanceof Error ? error.message : '未知错误';
                this.snack('保存脚本文件失败: ' + errorMessage);
                throw new Error(errorMessage);
            }
        }
    };

    public deleteScript = async (hookId: string): Promise<void> => {
        try {
            const response = await axios.delete(
                `${config.get('url')}hook/${hookId}/script`,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '脚本文件删除成功');
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '删除脚本文件失败';
            this.snack('删除脚本文件失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };
}
