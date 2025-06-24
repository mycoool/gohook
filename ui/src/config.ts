import {IVersionInfo} from './types';

export interface IConfig {
    url: string;
    register: boolean;
    version: IVersionInfo;
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
declare global {
    interface Window {
        config?: Partial<IConfig>;
    }
}

const config: IConfig = {
    url: 'unset',
    register: false,
    version: {commit: 'unknown', buildDate: 'unknown', version: 'unknown'},
    ...window.config,
};

export function set<Key extends keyof IConfig>(key: Key, value: IConfig[Key]): void {
    config[key] = value;
}

export function get<K extends keyof IConfig>(key: K): IConfig[K] {
    // 在开发模式下，如果url是'unset'，返回空字符串让代理工作
    if (key === 'url' && config[key] === 'unset') {
        return '' as IConfig[K];
    }
    return config[key];
}
