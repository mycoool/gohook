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
        await axios.post(
            `${config.get('url')}hook/${id}/trigger`,
            {},
            {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            }
        );
        this.snack('Hook triggered successfully');
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
}
