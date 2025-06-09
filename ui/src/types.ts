export interface IApplication {
    id: number;
    token: string;
    name: string;
    description: string;
    image: string;
    internal: boolean;
    defaultPriority: number;
    lastUsed: string | null;
}

export interface IClient {
    id: number;
    token: string;
    name: string;
    lastUsed: string | null;
}

export interface IHook {
    id: string;
    name: string;
    description: string;
    executeCommand: string;
    workingDirectory: string;
    responseMessage: string;
    httpMethods: string[];
    triggerRuleDescription: string;
    lastUsed: string | null;
    status: string;
}

export interface IVersion {
    name: string;
    path: string;
    description: string;
    currentBranch: string;
    currentTag: string;
    mode: 'branch' | 'tag' | 'none';
    status: string;
    lastCommit: string;
    lastCommitTime: string;
}

export interface IBranch {
    name: string;
    isCurrent: boolean;
    lastCommit: string;
    lastCommitTime: string;
}

export interface ITag {
    name: string;
    isCurrent: boolean;
    commitHash: string;
    date: string;
    message: string;
}

export interface IPlugin {
    id: number;
    token: string;
    name: string;
    modulePath: string;
    enabled: boolean;
    author?: string;
    website?: string;
    license?: string;
    capabilities: Array<'webhooker' | 'displayer' | 'configurer' | 'messenger' | 'storager'>;
}

export interface IMessage {
    id: number;
    appid: number;
    message: string;
    title: string;
    priority: number;
    date: string;
    image?: string;
    extras?: IMessageExtras;
}

export interface IMessageExtras {
    [key: string]: any; // eslint-disable-line  @typescript-eslint/no-explicit-any
}

export interface IPagedMessages {
    paging: IPaging;
    messages: IMessage[];
}

export interface IPaging {
    next?: string;
    since?: number;
    size: number;
    limit: number;
}

export interface IUser {
    id: number;
    name: string;
    admin: boolean;
}

export interface IVersionInfo {
    version: string;
    commit: string;
    buildDate: string;
}
