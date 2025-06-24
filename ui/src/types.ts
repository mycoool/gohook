export interface IClient {
    id: number;
    token: string;
    name: string;
    lastUsed: string | null;
}

export interface IHook {
    id: string;
    'execute-command': string;
    'command-working-directory'?: string;
    'response-message'?: string;
    'http-methods': string[];
    'response-headers'?: { [key: string]: string };
    'pass-arguments-to-command': IParameter[];
    'pass-environment-to-command': IEnvironmentVariable[];
    'trigger-rule'?: ITriggerRule;
    'include-command-output-in-response'?: boolean;
    'include-command-output-in-response-on-error'?: boolean;
    'parse-parameters-as-json'?: string[];
    'trigger-rule-mismatch-http-response-code'?: number;
    success: boolean;
    'last-execution': string;
    argumentsCount: number;
    environmentCount: number;
    
    // UI字段（用于显示）
    name?: string;
    executeCommand?: string;
    workingDirectory?: string;
    responseMessage?: string;
    httpMethods?: string[];
    triggerRuleDescription?: string;
    lastUsed?: string | null;
    status?: string;
}

export interface IParameter {
    source: 'payload' | 'header' | 'query' | 'string';
    name: string;
}

export interface IEnvironmentVariable {
    name: string;
    source: 'payload' | 'header' | 'query' | 'string';
}

// 触发规则类型定义
export interface IMatchRule {
    type: 'value' | 'regex' | 'payload-hmac-sha1' | 'payload-hmac-sha256' | 'payload-hmac-sha512' | 'ip-whitelist' | 'scalr-signature';
    parameter?: IParameter;
    value?: string;
    regex?: string;
    secret?: string;
    'ip-range'?: string;
}

export interface ITriggerRule {
    match?: IMatchRule;
    and?: ITriggerRule[];
    or?: ITriggerRule[];
    not?: ITriggerRule;
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
    enhook?: boolean;
    hookmode?: 'branch' | 'tag';
    hookbranch?: string; // 具体分支名或'*'表示任意分支
    hooksecret?: string; // webhook密码
}

export interface IBranch {
    name: string;
    isCurrent: boolean;
    lastCommit: string;
    lastCommitTime: string;
    type: 'local' | 'remote' | 'detached';
}

export interface ITag {
    name: string;
    isCurrent: boolean;
    commitHash: string;
    date: string;
    message: string;
}

export interface ITagsResponse {
    tags: ITag[];
    total: number;
    page: number;
    limit: number;
    totalPages: number;
    hasMore: boolean;
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
    username?: string;
    admin: boolean;
    role?: string;
}

export interface IVersionInfo {
    version: string;
    commit: string;
    buildDate: string;
}

// WebSocket realtime message type
export interface IWebSocketMessage {
    type: string;
    timestamp: string;
    data: any;
}

// Hook triggered message
export interface IHookTriggeredMessage {
    hookId: string;
    hookName: string;
    method: string;
    remoteAddr: string;
    success: boolean;
    output?: string;
    error?: string;
}

// GitHook triggered message
export interface IGitHookTriggeredMessage {
    projectName: string;
    action:
        | 'switch-branch'
        | 'switch-tag'
        | 'delete-tag'
        | 'delete-branch'
        | 'skip-branch-switch'
        | 'skip-mode-mismatch';
    target: string;
    success: boolean;
    output?: string;
    error?: string;
    skipped?: boolean;
    message?: string;
}

// Version switch message
export interface IVersionSwitchMessage {
    projectName: string;
    action: 'switch-branch' | 'switch-tag' | 'delete-tag';
    target: string;
    success: boolean;
    error?: string;
}

// Project management message
export interface IProjectManageMessage {
    action: 'add' | 'delete' | 'edit';
    projectName: string;
    projectPath?: string;
    success: boolean;
    error?: string;
}
