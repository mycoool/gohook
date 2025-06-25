import axios, {AxiosError, AxiosResponse} from 'axios';
import * as config from './config';
import {Base64} from 'js-base64';
import {detect} from 'detect-browser';
import {SnackReporter} from './snack/SnackManager';
import {observable} from 'mobx';
import {IUser} from './types';

const tokenKey = 'gohook-login-key';

export class CurrentUser {
    private tokenCache: string | null = null;
    private reconnectTimeoutId: number | null = null;
    private reconnectTime = 7500;
    @observable
    public loggedIn = false;
    @observable
    public authenticating = true;
    @observable
    public user: IUser = {name: 'unknown', admin: false, id: -1, username: 'unknown', role: 'user'};
    @observable
    public connectionErrorMessage: string | null = null;

    public constructor(private readonly snack: SnackReporter) {}

    public token = (): string => {
        if (this.tokenCache !== null) {
            return this.tokenCache;
        }

        const localStorageToken = window.localStorage.getItem(tokenKey);
        if (localStorageToken) {
            this.tokenCache = localStorageToken;
            return localStorageToken;
        }

        return '';
    };

    private readonly setToken = (token: string) => {
        this.tokenCache = token;
        window.localStorage.setItem(tokenKey, token);
    };

    public register = async (name: string, pass: string): Promise<boolean> =>
        axios
            .create()
            .post(config.get('url') + 'user', {name, pass})
            .then(() => {
                this.snack('User Created. Logging in...');
                this.login(name, pass);
                return true;
            })
            .catch((error: AxiosError) => {
                if (!error || !error.response) {
                    this.snack('No network connection or server unavailable.');
                    return false;
                }
                const {data} = error.response;
                this.snack(
                    `Register failed: ${data?.error ?? 'unknown'}: ${data?.errorDescription ?? ''}`
                );
                return false;
            });

    public login = async (username: string, password: string): Promise<boolean> => {
        this.loggedIn = false;
        this.authenticating = true;
        const browser = detect();
        const name = (browser && browser.name + ' ' + browser.version) || 'unknown browser';

        try {
            const resp = await axios.create().request({
                url: config.get('url') + 'client',
                method: 'POST',
                data: {name},
                // eslint-disable-next-line @typescript-eslint/naming-convention
                headers: {Authorization: 'Basic ' + Base64.encode(username + ':' + password)},
            });

            this.snack(`A client named '${name}' was created for your session.`);
            this.setToken(resp.data.token);

            try {
                await this.tryAuthenticate();
                return true;
            } catch (error) {
                console.log('create client succeeded, but authenticated with given token failed');
                this.authenticating = false;
                this.snack('Authentication failed after client creation');
                return false;
            }
        } catch (error: unknown) {
            this.authenticating = false;

            // 处理不同类型的错误
            const axiosError = error as AxiosError;
            if (!axiosError || !axiosError.response) {
                this.snack('No network connection or server unavailable.');
                return false;
            }

            const {data, status} = axiosError.response;

            if (status === 401) {
                // 用户名或密码错误
                const errorMessage = data?.error || 'Invalid username or password';
                this.snack(`Login failed: ${errorMessage}`);
            } else if (status >= 500) {
                this.snack(`Server error: ${axiosError.response.statusText} (code: ${status})`);
            } else {
                this.snack(`Login failed: ${data?.error || 'Unknown error'}`);
            }

            return false;
        }
    };

    public tryAuthenticate = async (): Promise<AxiosResponse<IUser>> => {
        if (this.token() === '') {
            this.authenticating = false;
            return Promise.reject();
        }

        return (
            axios
                .create()
                // eslint-disable-next-line @typescript-eslint/naming-convention
                .get(config.get('url') + 'current/user', {headers: {'X-GoHook-Key': this.token()}})
                .then((passThrough) => {
                    this.user = passThrough.data;
                    this.loggedIn = true;
                    this.authenticating = false;
                    this.connectionErrorMessage = null;
                    this.reconnectTime = 7500;
                    return passThrough;
                })
                .catch((error: AxiosError) => {
                    this.authenticating = false;
                    if (!error || !error.response) {
                        this.connectionError('No network connection or server unavailable.');
                        return Promise.reject(error);
                    }

                    if (error.response.status >= 500) {
                        this.connectionError(
                            `${error.response.statusText} (code: ${error.response.status}).`
                        );
                        return Promise.reject(error);
                    }

                    this.connectionErrorMessage = null;

                    // 401 错误由 axios 拦截器统一处理，这里只处理其他 4xx 错误
                    if (error.response.status >= 400 && error.response.status < 500 && error.response.status !== 401) {
                        this.logout();
                    }
                    return Promise.reject(error);
                })
        );
    };

    public logout = async () => {
        const token = this.token();

        try {
            if (token) {
                // 优先尝试删除服务器上的会话
                await axios.delete(config.get('url') + 'client/current', {
                    headers: {'X-GoHook-Key': token},
                });
                console.log('Session deletion request sent to server.');
            }
        } catch (error) {
            // 即使请求失败，我们也不关心，因为最终会清理本地状态
            if (process.env.NODE_ENV === 'development') {
                console.warn(
                    'Failed to delete session on server. This can be ignored as user will be logged out locally.',
                    error
                );
            }
        } finally {
            // 无论如何，都确保清理本地状态，完成登出
            window.localStorage.removeItem(tokenKey);
            this.tokenCache = null;
            this.loggedIn = false;
            this.user = {name: 'unknown', admin: false, id: -1, username: 'unknown', role: 'user'};
        }
    };

    public changePassword = (oldPassword: string, newPassword: string) => {
        axios
            .post(
                config.get('url') + 'user/password',
                {
                    oldPassword: oldPassword,
                    newPassword: newPassword,
                },
                {
                    headers: {'X-GoHook-Key': this.token()},
                }
            )
            .then(() => this.snack('Password changed'))
            .catch((error) => {
                this.snack(
                    'Failed to change password: ' + (error.response?.data?.error || error.message)
                );
            });
    };

    public tryReconnect = (quiet = false) => {
        this.tryAuthenticate().catch(() => {
            if (!quiet) {
                this.snack('Reconnect failed');
            }
        });
    };

    private readonly connectionError = (message: string) => {
        this.connectionErrorMessage = message;
        if (this.reconnectTimeoutId !== null) {
            window.clearTimeout(this.reconnectTimeoutId);
        }
        this.reconnectTimeoutId = window.setTimeout(
            () => this.tryReconnect(true),
            this.reconnectTime
        );
        this.reconnectTime = Math.min(this.reconnectTime * 2, 120000);
    };
}
