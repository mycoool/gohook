import axios from 'axios';
import {CurrentUser} from './CurrentUser';
import {SnackReporter} from './snack/SnackManager';

export const initAxios = (currentUser: CurrentUser, snack: SnackReporter) => {
    axios.interceptors.request.use(
        async (config) => {
            // Exclude renewal and login endpoints from checks to avoid loops
            if (config.url?.endsWith('/client/renew') || config.url?.endsWith('/client')) {
                config.headers['X-GoHook-Key'] = currentUser.token();
                return config;
            }

            // If token is already expired, let the request go through.
            // It will fail with a 401, and the response interceptor will handle the logout.
            // This prevents spamming the renew endpoint with an invalid token.
            if (currentUser.isTokenExpired()) {
                config.headers['X-GoHook-Key'] = currentUser.token();
                return config;
            }

            if (currentUser.isTokenExpiring()) {
                await currentUser.renewToken();
            }

            // Ensure all requests carry the most up-to-date token
            config.headers['X-GoHook-Key'] = currentUser.token();
            return config;
        },
        (error) => {
            return Promise.reject(error);
        }
    );

    axios.interceptors.response.use(undefined, (error) => {
        if (!error.response) {
            snack('GoHook server is not reachable, try refreshing the page.');
            return Promise.reject(error);
        }

        const status = error.response.status;

        if (status === 401) {
            // Call the full logout procedure to ensure all state is cleaned up
            // before redirecting.
            currentUser.logout();
            snack('登录已过期，请重新登录');
            setTimeout(() => {
                // The /#/login path is necessary for the hash router
                window.location.href = '/#/login';
                // Reload to ensure a clean state
                window.location.reload();
            }, 1500);
        } else if (status === 400 || status === 403 || status === 500) {
            snack(error.response.data.error + ': ' + error.response.data.errorDescription);
        }

        return Promise.reject(error);
    });
};
