import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import React, {Component} from 'react';
import {RouteComponentProps} from 'react-router';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
import Message from './Message';
import {observer} from 'mobx-react';
import {inject, Stores} from '../inject';
import {observable} from 'mobx';
import ReactInfinite from 'react-infinite';
import {IMessage} from '../types';
import ConfirmDialog from '../common/ConfirmDialog';
import LoadingSpinner from '../common/LoadingSpinner';
import useTranslation from '../i18n/useTranslation';

type IProps = RouteComponentProps<{id: string}>;

interface IState {
    appId: number;
}

@observer
class Messages extends Component<IProps & Stores<'messagesStore'>, IState> {
    @observable
    private heights: Record<string, number> = {};
    @observable
    private deleteAll = false;

    private static appId(props: IProps) {
        if (props === undefined) {
            return -1;
        }
        const {match} = props;
        return match.params.id !== undefined ? parseInt(match.params.id, 10) : -1;
    }

    public state = {appId: -1};

    private isLoadingMore = false;

    public componentDidMount() {
        window.onscroll = () => {
            if (
                window.innerHeight + window.pageYOffset >=
                document.body.offsetHeight - window.innerHeight * 2
            ) {
                this.checkIfLoadMore();
            }
        };
        this.updateAll();
    }

    public componentDidUpdate(prevProps: IProps & Stores<'messagesStore'>) {
        if (prevProps.match.params.id !== this.props.match.params.id) {
            this.updateAllWithProps(this.props);
        }
    }

    public render() {
        const {appId} = this.state;
        const {messagesStore} = this.props;
        const messages = messagesStore.get();
        const hasMore = messagesStore.canLoadMore();
        const name = appId === -1 ? 'All Messages' : 'Messages';
        const hasMessages = messages.length !== 0;

        return (
            <MessagesContainer
                appId={appId}
                messages={messages}
                hasMore={hasMore}
                name={name}
                hasMessages={hasMessages}
                loaded={messagesStore.loaded()}
                deleteAll={this.deleteAll}
                heights={this.heights}
                onRefresh={() => messagesStore.refreshByApp()}
                onDeleteAll={() => (this.deleteAll = true)}
                onCloseDeleteAll={() => (this.deleteAll = false)}
                onConfirmDeleteAll={() => messagesStore.removeByApp()}
                onLoadMore={() => this.checkIfLoadMore()}
                renderMessage={this.renderMessage}
            />
        );
    }

    private updateAllWithProps = (props: IProps & Stores<'messagesStore'>) => {
        const appId = Messages.appId(props);
        this.setState({appId});
        if (!props.messagesStore.exists()) {
            props.messagesStore.loadMore();
        }
    };

    private updateAll = () => this.updateAllWithProps(this.props);

    private deleteMessage = (message: IMessage) => () =>
        this.props.messagesStore.removeSingle(message);

    private renderMessage = (message: IMessage) => (
        <Message
            key={message.id}
            height={(height: number) => {
                if (!this.heights[message.id]) {
                    this.heights[message.id] = height;
                }
            }}
            fDelete={this.deleteMessage(message)}
            title={message.title}
            date={message.date}
            content={message.message}
            image={message.image}
            extras={message.extras}
            priority={message.priority}
        />
    );

    private checkIfLoadMore() {
        if (!this.isLoadingMore && this.props.messagesStore.canLoadMore()) {
            this.isLoadingMore = true;
            this.props.messagesStore.loadMore().then(() => (this.isLoadingMore = false));
        }
    }

    private label = (text: string) => (
        <Grid size={12}>
            <Typography variant="caption" component="div" gutterBottom align="center">
                {text}
            </Typography>
        </Grid>
    );
}

// Separate container component to use Hook
const MessagesContainer: React.FC<{
    appId: number;
    messages: IMessage[];
    hasMore: boolean;
    name: string;
    hasMessages: boolean;
    loaded: boolean;
    deleteAll: boolean;
    heights: Record<string, number>;
    onRefresh: () => void;
    onDeleteAll: () => void;
    onCloseDeleteAll: () => void;
    onConfirmDeleteAll: () => void;
    onLoadMore: () => void;
    renderMessage: (message: IMessage) => React.ReactElement;
}> = ({
    appId,
    messages,
    hasMore,
    name,
    hasMessages,
    loaded,
    deleteAll,
    heights,
    onRefresh,
    onDeleteAll,
    onCloseDeleteAll,
    onConfirmDeleteAll,
    renderMessage,
}) => {
    const {t} = useTranslation();

    const label = (text: string) => (
        <Grid size={12}>
            <Typography variant="caption" component="div" gutterBottom align="center">
                {text}
            </Typography>
        </Grid>
    );

    return (
        <DefaultPage
            title={name}
            rightControl={
                <div>
                    <Button
                        id="refresh-all"
                        variant="contained"
                        color="primary"
                        onClick={onRefresh}
                        style={{marginRight: 5}}>
                        {t('common.refresh')}
                    </Button>
                    <Button
                        id="delete-all"
                        variant="contained"
                        disabled={!hasMessages}
                        color="primary"
                        onClick={onDeleteAll}>
                        {t('message.clearMessages')}
                    </Button>
                </div>
            }>
            {!loaded ? (
                <LoadingSpinner />
            ) : hasMessages ? (
                <div style={{width: '100%'}} id="messages">
                    {React.createElement(
                        ReactInfinite as any,
                        {
                            key: appId,
                            useWindowAsScrollContainer: true,
                            preloadBatchSize: window.innerHeight * 3,
                            elementHeight: messages.map((m) => heights[m.id] || 1),
                        },
                        messages.map(renderMessage)
                    )}

                    {hasMore ? <LoadingSpinner /> : label(t('message.reachedEnd'))}
                </div>
            ) : (
                label(t('message.noMessages'))
            )}

            {deleteAll && (
                <ConfirmDialog
                    title={t('message.clearMessages')}
                    text={t('message.confirmClearAll')}
                    fClose={onCloseDeleteAll}
                    fOnSubmit={onConfirmDeleteAll}
                />
            )}
        </DefaultPage>
    );
};

export default inject('messagesStore')(Messages);
