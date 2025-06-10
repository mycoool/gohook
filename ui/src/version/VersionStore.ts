import axios from 'axios';
import * as config from '../config';
import {action, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IVersion, IBranch, ITag} from '../types';

export class VersionStore {
    @observable
    protected projects: IVersion[] = [];

    @observable
    protected branches: IBranch[] = [];

    @observable
    protected tags: ITag[] = [];

    @observable
    protected currentProject: string | null = null;

    public constructor(private readonly snack: SnackReporter) {}

    protected requestProjects = (): Promise<IVersion[]> =>
        axios
            .get<IVersion[]>(`${config.get('url')}version`)
            .then((response) => response.data);

    protected requestBranches = (projectName: string): Promise<IBranch[]> =>
        axios
            .get<IBranch[]>(`${config.get('url')}version/${projectName}/branches`)
            .then((response) => response.data);

    protected requestTags = (projectName: string): Promise<ITag[]> =>
        axios
            .get<ITag[]>(`${config.get('url')}version/${projectName}/tags`)
            .then((response) => response.data);

    protected requestSwitchBranch = (projectName: string, branch: string): Promise<void> =>
        axios.post(`${config.get('url')}version/${projectName}/switch-branch`, {
            branch: branch
        }).then(() => this.snack('分支切换成功'));

    protected requestSwitchTag = (projectName: string, tag: string): Promise<void> =>
        axios.post(`${config.get('url')}version/${projectName}/switch-tag`, {
            tag: tag
        }).then(() => this.snack('标签切换成功'));

    @action
    public refreshProjects = async (): Promise<void> => {
        this.projects = await this.requestProjects().then((projects) => projects || []);
    };

    @action
    public reloadConfig = async (): Promise<void> => {
        try {
            const response = await axios.post(`${config.get('url')}version/reload-config`);
            this.snack(response.data.message || '配置文件重新加载成功');
            await this.refreshProjects(); // 重新加载后刷新数据
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message : 
                (error as {response?: {data?: {error?: string}}})?.response?.data?.error ?? 
                '未知错误';
            this.snack('重新加载配置文件失败: ' + errorMessage);
        }
    };

    @action
    public addProject = async (name: string, path: string, description: string): Promise<void> => {
        try {
            const response = await axios.post(`${config.get('url')}version/add-project`, {
                name,
                path,
                description
            });
            this.snack(response.data.message || '项目添加成功');
            await this.refreshProjects(); // 添加后刷新项目列表
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message : 
                (error as {response?: {data?: {error?: string}}})?.response?.data?.error ?? 
                '未知错误';
            this.snack('添加项目失败: ' + errorMessage);
            throw error; // 重新抛出错误，让UI组件知道操作失败
        }
    };

    @action
    public deleteProject = async (name: string): Promise<void> => {
        try {
            const response = await axios.delete(`${config.get('url')}version/${name}`);
            this.snack(response.data.message || '项目删除成功');
            await this.refreshProjects(); // 删除后刷新项目列表
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message : 
                (error as {response?: {data?: {error?: string}}})?.response?.data?.error ?? 
                '未知错误';
            this.snack('删除项目失败: ' + errorMessage);
            throw error; // 重新抛出错误，让UI组件知道操作失败
        }
    };

    @action
    public initGit = async (name: string): Promise<void> => {
        try {
            const response = await axios.post(`${config.get('url')}version/${name}/init-git`);
            this.snack(response.data.message || 'Git仓库初始化成功');
            await this.refreshProjects(); // 初始化后刷新项目列表
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message : 
                (error as {response?: {data?: {error?: string}}})?.response?.data?.error ?? 
                '未知错误';
            this.snack('Git仓库初始化失败: ' + errorMessage);
            throw error;
        }
    };

    @action
    public setRemote = async (name: string, remoteUrl: string): Promise<void> => {
        try {
            const response = await axios.post(`${config.get('url')}version/${name}/set-remote`, {
                remoteUrl: remoteUrl
            });
            this.snack(response.data.message || '远程仓库设置成功');
            await this.refreshProjects(); // 设置后刷新项目列表
        } catch (error: unknown) {
            const errorMessage = error instanceof Error ? error.message : 
                (error as {response?: {data?: {error?: string}}})?.response?.data?.error ?? 
                '未知错误';
            this.snack('设置远程仓库失败: ' + errorMessage);
            throw error;
        }
    };

    @action
    public refreshBranches = async (projectName: string): Promise<void> => {
        this.currentProject = projectName;
        this.branches = await this.requestBranches(projectName).then((branches) => branches || []);
    };

    @action
    public refreshTags = async (projectName: string): Promise<void> => {
        this.currentProject = projectName;
        this.tags = await this.requestTags(projectName).then((tags) => tags || []);
    };

    @action
    public switchBranch = async (projectName: string, branch: string): Promise<void> => {
        await this.requestSwitchBranch(projectName, branch);
        await this.refreshBranches(projectName);
        await this.refreshProjects();
    };

    @action
    public switchTag = async (projectName: string, tag: string): Promise<void> => {
        await this.requestSwitchTag(projectName, tag);
        await this.refreshTags(projectName);
        await this.refreshProjects();
    };

    public getProjects = (): IVersion[] => this.projects;

    public getBranches = (): IBranch[] => this.branches;

    public getTags = (): ITag[] => this.tags;

    public getCurrentProject = (): string | null => this.currentProject;

    public getProjectByName = (name: string): IVersion | undefined =>
        this.projects.find(project => project.name === name);

    @action
    public clear = (): void => {
        this.projects = [];
        this.branches = [];
        this.tags = [];
        this.currentProject = null;
    };
} 