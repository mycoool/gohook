import {action, IObservableArray, observable} from 'mobx';
import axios, {AxiosResponse} from 'axios';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {IMessage, IPagedMessages} from '../types';

interface MessagesState {
    messages: IObservableArray<IMessage>;
    hasMore: boolean;
    nextSince: number;
    loaded: boolean;
}

export class MessagesStore {
    @observable
    private state: MessagesState;

    private loading = false;

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {
        this.state = this.emptyState();
    }

    public loaded = () => this.state.loaded;

    public canLoadMore = () => this.state.hasMore;

    @action
    public loadMore = async () => {
        if (!this.state.hasMore || this.loading) {
            return Promise.resolve();
        }
        this.loading = true;

        const pagedResult = await this.fetchMessages(this.state.nextSince).then(
            (resp) => resp.data
        );

        this.state.messages.replace([...this.state.messages, ...pagedResult.messages]);
        this.state.nextSince = pagedResult.paging.since ?? 0;
        this.state.hasMore = 'next' in pagedResult.paging;
        this.state.loaded = true;
        this.loading = false;
        return Promise.resolve();
    };

    @action
    public publishSingleMessage = (message: IMessage) => {
        this.state.messages.unshift(message);
    };

    @action
    public removeByApp = async () => {
        await axios.delete(config.get('url') + 'message', {
            headers: {'X-GoHook-Key': this.tokenProvider()},
        });
        this.snack('已删除所有消息');
        this.clearAll();
        await this.loadMore();
    };

    @action
    public removeSingle = async (message: IMessage) => {
        await axios.delete(config.get('url') + 'message/' + message.id, {
            headers: {'X-GoHook-Key': this.tokenProvider()},
        });
        this.removeFromList(this.state.messages, message);
        this.snack('消息已删除');
    };

    @action
    public clearAll = () => {
        this.state = this.emptyState();
    };

    @action
    public refreshByApp = async () => {
        this.clearAll();
        this.loadMore();
    };

    public exists = () => this.state.loaded;

    private removeFromList(messages: IMessage[], messageToDelete: IMessage): false | number {
        if (messages) {
            const index = messages.findIndex((message) => message.id === messageToDelete.id);
            if (index !== -1) {
                messages.splice(index, 1);
                return index;
            }
        }
        return false;
    }

    private fetchMessages = (since: number): Promise<AxiosResponse<IPagedMessages>> =>
        axios.get(config.get('url') + 'message?since=' + since, {
            headers: {'X-GoHook-Key': this.tokenProvider()},
        });

    public get = (): Array<IMessage & {image: string | null}> =>
        this.state.messages.map((message: IMessage) => ({
            ...message,
            image: message.image ?? null, // 使用空值合并操作符
        })) as Array<IMessage & {image: string | null}>;

    private emptyState = (): MessagesState => ({
        messages: observable.array(),
        hasMore: true,
        nextSince: 0,
        loaded: false,
    });
}
