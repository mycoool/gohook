import * as React from 'react';
import * as ReactDOM from 'react-dom';
import 'typeface-roboto';
import {initAxios} from './apiAuth';
import * as config from './config';
import Layout from './layout/Layout';
import {unregister} from './registerServiceWorker';
import {CurrentUser} from './CurrentUser';
import {AppStore} from './application/AppStore';
import {HookStore} from './hook/HookStore';
import {VersionStore} from './version/VersionStore';
import {WebSocketStore} from './message/WebSocketStore';
import {SnackManager} from './snack/SnackManager';
import {InjectProvider, StoreMapping} from './inject';
import {UserStore} from './user/UserStore';
import {MessagesStore} from './message/MessagesStore';

import {PluginStore} from './plugin/PluginStore';
import {registerReactions} from './reactions';
import './i18n';

const devUrl = 'http://localhost:3000/';

const {port, hostname, protocol, pathname} = window.location;
const slashes = protocol.concat('//');
const path = pathname.endsWith('/') ? pathname : pathname.substring(0, pathname.lastIndexOf('/'));
const url = slashes.concat(port ? hostname.concat(':', port) : hostname) + path;
const urlWithSlash = url.endsWith('/') ? url : url.concat('/');

const prodUrl = urlWithSlash;

const initStores = (): StoreMapping => {
    const snackManager = new SnackManager();
    const appStore = new AppStore(snackManager.snack);
    const hookStore = new HookStore(snackManager.snack);
    const versionStore = new VersionStore(snackManager.snack);
    const userStore = new UserStore(snackManager.snack);
    const messagesStore = new MessagesStore(appStore, snackManager.snack);
    const currentUser = new CurrentUser(snackManager.snack);

    const wsStore = new WebSocketStore(snackManager.snack, currentUser);
    const pluginStore = new PluginStore(snackManager.snack);
    appStore.onDelete = () => messagesStore.clearAll();

    return {
        appStore,
        hookStore,
        versionStore,
        snackManager,
        userStore,
        messagesStore,
        currentUser,
        wsStore,
        pluginStore,
    };
};

(function clientJS() {
    if (process.env.NODE_ENV === 'production') {
        config.set('url', prodUrl);
    } else {
        config.set('url', devUrl);
        config.set('register', true);
    }
    const stores = initStores();
    initAxios(stores.currentUser, stores.snackManager.snack);

    registerReactions(stores);

    stores.currentUser.tryAuthenticate().catch(() => {});

    window.onbeforeunload = () => {
        stores.wsStore.close();
    };

    ReactDOM.render(
        <InjectProvider stores={stores}>
            <Layout />
        </InjectProvider>,
        document.getElementById('root')
    );
    unregister();
})();
