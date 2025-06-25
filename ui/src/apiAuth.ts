import axios from 'axios';
import {CurrentUser} from './CurrentUser';
import {SnackReporter} from './snack/SnackManager';

export const initAxios = (currentUser: CurrentUser, snack: SnackReporter) => {
    axios.interceptors.request.use((config) => {
        config.headers['X-GoHook-Key'] = currentUser.token();
        return config;
    });

    axios.interceptors.response.use(undefined, (error) => {
        if (!error.response) {
            snack('GoHook server is not reachable, try refreshing the page.');
            return Promise.reject(error);
        }

        const status = error.response.status;

        if (status === 401) {
            window.localStorage.removeItem('gohook-login-key');
            snack('登录已过期，请重新登录');
            setTimeout(() => {
                window.location.href = '/#/login';
            }, 1500);
        } else if (status === 400 || status === 403 || status === 500) {
            snack(error.response.data.error + ': ' + error.response.data.errorDescription);
        }

        return Promise.reject(error);
    });
};
