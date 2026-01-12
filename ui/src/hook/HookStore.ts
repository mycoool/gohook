import axios from 'axios';
import * as config from '../config';
import {action, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IHook} from '../types';
import translate from '../i18n/translator';

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
            .then(() => this.snack(translate('hook.snack.deleteSuccess')));

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
            this.snack(response.data.message || translate('hook.snack.reloadConfigSuccess'));
            await this.refresh(); // 加载后刷新数据
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.unknownError');
            this.snack(translate('hook.snack.loadFailedWithError', {error: errorMessage}));
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
            this.snack(response.data.message || translate('hook.snack.triggerSuccess'));
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
                    let errorMessage =
                        responseData.message || translate('hook.snack.triggerFailed');
                    if (responseData.hook) {
                        errorMessage += ` (${responseData.hook})`;
                    }
                    if (responseData.error) {
                        errorMessage += `\n${translate('hook.snack.errorDetails', {
                            error: responseData.error,
                        })}`;
                    }
                    if (responseData.output) {
                        errorMessage += `\n${translate('hook.snack.commandOutput', {
                            output: responseData.output,
                        })}`;
                    }

                    this.snack(errorMessage);
                    return; // 不重新抛出错误，避免未捕获的异常
                }
            }

            // 兜底错误处理
            const errorMessage =
                error instanceof Error ? error.message : translate('hook.snack.triggerFailed');
            this.snack(errorMessage);
        }
    };

    @action
    public getHookDetails = async (id: string): Promise<IHook> => {
        try {
            const response = await axios.get<IHook>(`${config.get('url')}hook/${id}`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            return response.data;
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.detailsFailed');
            this.snack(translate('hook.snack.detailsFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
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
    public createHook = async (hookData: {
        id: string;
        'execute-command': string;
        'command-working-directory': string;
        'response-message': string;
    }): Promise<void> => {
        try {
            const response = await axios.post(`${config.get('url')}hook`, hookData, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            this.snack(response.data.message || translate('hook.snack.createSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.createFailed');
            this.snack(translate('hook.snack.createFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
    };

    @action
    public updateHookBasic = async (
        hookId: string,
        basicData: {
            'execute-command': string;
            'command-working-directory': string;
            'response-message': string;
        }
    ): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}hook/${hookId}/basic`,
                basicData,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.updateBasicSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.updateBasicFailed');
            this.snack(translate('hook.snack.updateBasicFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
    };

    @action
    public updateHookParameters = async (
        hookId: string,
        parametersData: {
            'pass-arguments-to-command': {source: string; name: string}[];
            'pass-environment-to-command': {name: string; source: string}[];
            'parse-parameters-as-json': string[];
        }
    ): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}hook/${hookId}/parameters`,
                parametersData,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.updateParametersSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.updateParametersFailed');
            this.snack(
                translate('hook.snack.updateParametersFailedWithError', {error: errorMessage})
            );
            throw new Error(errorMessage);
        }
    };

    @action
    public updateHookTriggers = async (
        hookId: string,
        triggersData: {
            'trigger-rule': any;
            'trigger-rule-mismatch-http-response-code': number;
        }
    ): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}hook/${hookId}/triggers`,
                triggersData,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.updateTriggersSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.updateTriggersFailed');
            this.snack(
                translate('hook.snack.updateTriggersFailedWithError', {error: errorMessage})
            );
            throw new Error(errorMessage);
        }
    };

    @action
    public updateHookResponse = async (
        hookId: string,
        responseData: {
            'http-methods': string[];
            'response-headers': {[key: string]: string};
            'include-command-output-in-response': boolean;
            'include-command-output-in-response-on-error': boolean;
        }
    ): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}hook/${hookId}/response`,
                responseData,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.updateResponseSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.updateResponseFailed');
            this.snack(
                translate('hook.snack.updateResponseFailedWithError', {error: errorMessage})
            );
            throw new Error(errorMessage);
        }
    };

    // 脚本文件管理方法

    public getScript = async (
        hookId: string
    ): Promise<{
        content: string;
        exists: boolean;
        path: string;
        isExecutable?: boolean;
        editable?: boolean;
        message?: string;
        suggestion?: string;
    }> => {
        try {
            const response = await axios.get<{
                content: string;
                exists: boolean;
                path: string;
                isExecutable?: boolean;
                editable?: boolean;
                message?: string;
                suggestion?: string;
            }>(`${config.get('url')}hook/${hookId}/script`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            return response.data;
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.getScriptFailed');
            this.snack(translate('hook.snack.getScriptFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
    };

    public saveScript = async (hookId: string, content: string, path?: string): Promise<void> => {
        try {
            const requestData: any = {content};
            if (path) {
                requestData.path = path;
            }

            const response = await axios.post(
                `${config.get('url')}hook/${hookId}/script`,
                requestData,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.saveScriptSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.saveScriptFailed');
            this.snack(translate('hook.snack.saveScriptFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
    };

    public updateExecuteCommand = async (hookId: string, executeCommand: string): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}hook/${hookId}/execute-command`,
                {
                    'execute-command': executeCommand,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || translate('hook.snack.updateCommandSuccess'));
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      translate('hook.snack.updateCommandFailed');
            this.snack(translate('hook.snack.updateCommandFailedWithError', {error: errorMessage}));
            throw new Error(errorMessage);
        }
    };
}
