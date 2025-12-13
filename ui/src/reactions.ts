import {StoreMapping} from './inject';
import {reaction} from 'mobx';
import * as Notifications from './snack/browserNotification';
import {IWebSocketMessage} from './types';

export const registerReactions = (stores: StoreMapping) => {
    const scheduleRefresh = (() => {
        let timer: ReturnType<typeof setTimeout> | null = null;
        let nodes = false;
        let projects = false;

        return (opts: {nodes?: boolean; projects?: boolean}) => {
            nodes = nodes || !!opts.nodes;
            projects = projects || !!opts.projects;
            if (timer) return;
            timer = setTimeout(() => {
                const doNodes = nodes;
                const doProjects = projects;
                timer = null;
                nodes = false;
                projects = false;
                if (doNodes) {
                    stores.syncNodeStore.refreshNodes().catch(() => undefined);
                }
                if (doProjects) {
                    stores.syncProjectStore.refreshProjects().catch(() => undefined);
                }
            }, 500);
        };
    })();

    const handleSyncStreamEvent = (message: IWebSocketMessage) => {
        switch (message.type) {
            case 'sync_node_event':
                scheduleRefresh({nodes: true, projects: true});
                break;
            case 'sync_task_event':
                scheduleRefresh({nodes: true, projects: true});
                break;
            case 'sync_project_event':
                scheduleRefresh({projects: true});
                break;
            default:
                break;
        }
    };

    const clearAll = () => {
        stores.messagesStore.clearAll();
        stores.userStore.clear();
        stores.wsStore.offMessage(handleSyncStreamEvent);
        stores.wsStore.close();
        stores.syncNodeStore.clear();
        stores.syncProjectStore.clear();
    };
    const loadAll = () => {
        stores.wsStore.listen((message) => {
            stores.messagesStore.publishSingleMessage(message);
            Notifications.notifyNewMessage(message);
            if (message.priority >= 4) {
                const src = 'static/notification.ogg';
                const audio = new Audio(src);
                audio.play();
            }
        });
        stores.wsStore.onMessage(handleSyncStreamEvent);
        // 刷新用户列表（如果当前用户是管理员）
        if (stores.currentUser.user.admin || stores.currentUser.user.role === 'admin') {
            stores.userStore.refresh();
        }
        stores.syncNodeStore.refreshNodes().catch(() => undefined);
    };

    reaction(
        () => stores.currentUser.loggedIn,
        (loggedIn) => {
            if (loggedIn) {
                loadAll();
            } else {
                clearAll();
            }
        }
    );

    reaction(
        () => stores.currentUser.connectionErrorMessage,
        (connectionErrorMessage) => {
            if (!connectionErrorMessage) {
                clearAll();
                loadAll();
            }
        }
    );
};
