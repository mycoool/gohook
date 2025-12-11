import {StoreMapping} from './inject';
import {reaction} from 'mobx';
import * as Notifications from './snack/browserNotification';

export const registerReactions = (stores: StoreMapping) => {
    const clearAll = () => {
        stores.messagesStore.clearAll();
        stores.userStore.clear();
        stores.wsStore.close();
        stores.syncNodeStore.clear();
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
