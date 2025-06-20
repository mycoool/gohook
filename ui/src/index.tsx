import * as React from 'react';
import { createRoot } from 'react-dom/client';
import 'typeface-roboto';
import {initAxios} from './apiAuth';
import * as config from './config';
import Layout from './layout/Layout';
import {unregister} from './registerServiceWorker';
import {CurrentUser} from './CurrentUser';

import {HookStore} from './hook/HookStore';
import {VersionStore} from './version/VersionStore';
import {WebSocketStore} from './message/WebSocketStore';
import {SnackManager} from './snack/SnackManager';
import {InjectProvider, StoreMapping} from './inject';
import {UserStore} from './user/UserStore';
import {MessagesStore} from './message/MessagesStore';

import {PluginStore} from './plugin/PluginStore';
import {AppConfigStore} from './app/AppConfigStore';
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
    const currentUser = new CurrentUser(snackManager.snack);
    const hookStore = new HookStore(snackManager.snack, () => currentUser.token());
    const versionStore = new VersionStore(snackManager.snack, () => currentUser.token());
    const userStore = new UserStore(snackManager.snack, () => currentUser.token());
    const messagesStore = new MessagesStore(snackManager.snack, () => currentUser.token());
    const appConfigStore = new AppConfigStore(() => currentUser.token(), snackManager.snack);

    const wsStore = new WebSocketStore(snackManager.snack, currentUser);
    const pluginStore = new PluginStore(snackManager.snack);

    return {
        hookStore,
        versionStore,
        snackManager,
        userStore,
        messagesStore,
        currentUser,
        wsStore,
        pluginStore,
        appConfigStore,
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

    const container = document.getElementById('root');
    if (container) {
        const root = createRoot(container);
        root.render(
            <InjectProvider stores={stores}>
                <Layout />
            </InjectProvider>
        );
    }
    unregister();
})();
