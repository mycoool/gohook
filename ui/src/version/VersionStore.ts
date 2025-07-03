import axios from 'axios';
import * as config from '../config';
import {action, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IVersion, IBranch, ITag, ITagsResponse} from '../types';
import {GitHookConfig} from './GitHookDialog';

export class VersionStore {
    @observable
    protected projects: IVersion[] = [];

    @observable
    protected branches: IBranch[] = [];

    @observable
    protected tags: ITag[] = [];

    @observable
    protected tagsTotal = 0;

    @observable
    protected tagsPage = 1;

    @observable
    protected tagsHasMore = false;

    @observable
    protected tagsLoading = false;

    @observable
    protected currentProject: string | null = null;

    public constructor(
        private readonly snack: SnackReporter,
        private readonly tokenProvider: () => string
    ) {}

    protected requestProjects = (): Promise<IVersion[]> =>
        axios
            .get<IVersion[]>(`${config.get('url')}version`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            })
            .then((response) => response.data);

    protected requestBranches = (projectName: string): Promise<IBranch[]> =>
        axios
            .get<IBranch[]>(`${config.get('url')}version/${projectName}/branches`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            })
            .then((response) => response.data);

    protected requestTags = (
        projectName: string,
        filter?: string,
        messageFilter?: string,
        page = 1
    ): Promise<ITagsResponse> =>
        axios
            .get<ITagsResponse>(`${config.get('url')}version/${projectName}/tags`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
                params: {
                    page: page.toString(),
                    limit: '20',
                    ...(filter ? {filter} : {}),
                    ...(messageFilter ? {messageFilter} : {}),
                },
            })
            .then((response) => response.data);

    protected requestSwitchBranch = (projectName: string, branch: string): Promise<void> =>
        axios
            .post(
                `${config.get('url')}version/${projectName}/switch-branch`,
                {
                    branch: branch,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            )
            .then(() => this.snack('分支切换成功'));

    protected requestSwitchTag = (projectName: string, tag: string): Promise<void> =>
        axios
            .post(
                `${config.get('url')}version/${projectName}/switch-tag`,
                {
                    tag: tag,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            )
            .then(() => this.snack('标签切换成功'));

    @action
    public refreshProjects = async (): Promise<void> => {
        this.projects = await this.requestProjects().then((projects) => projects || []);
    };

    @action
    public reloadConfig = async (): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/reload-config`,
                {},
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '加载项目成功');
            await this.refreshProjects(); // 加载后刷新数据
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('加载项目失败: ' + errorMessage);
        }
    };

    @action
    public addProject = async (name: string, path: string, description: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/add-project`,
                {
                    name,
                    path,
                    description,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '项目添加成功');
            await this.refreshProjects(); // 添加后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('添加项目失败: ' + errorMessage);
            throw error; // 重新抛出错误，让UI组件知道操作失败
        }
    };

    @action
    public editProject = async (
        originalName: string,
        name: string,
        path: string,
        description: string
    ): Promise<void> => {
        try {
            const response = await axios.put(
                `${config.get('url')}version/${originalName}`,
                {
                    name,
                    path,
                    description,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '项目编辑成功');
            await this.refreshProjects(); // 编辑后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('编辑项目失败: ' + errorMessage);
            throw error; // 重新抛出错误，让UI组件知道操作失败
        }
    };

    @action
    public deleteProject = async (name: string): Promise<void> => {
        try {
            const response = await axios.delete(`${config.get('url')}version/${name}`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            this.snack(response.data.message || '项目删除成功');
            await this.refreshProjects(); // 删除后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('删除项目失败: ' + errorMessage);
            throw error; // 重新抛出错误，让UI组件知道操作失败
        }
    };

    @action
    public initGit = async (name: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/${name}/init-git`,
                {},
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || 'Git仓库初始化成功');
            await this.refreshProjects(); // 初始化后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('Git仓库初始化失败: ' + errorMessage);
            throw error;
        }
    };

    @action
    public setRemote = async (name: string, remoteUrl: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/${name}/set-remote`,
                {
                    remoteUrl: remoteUrl,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '远程仓库设置成功');
            await this.refreshProjects(); // 设置后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('设置远程仓库失败: ' + errorMessage);
            throw error;
        }
    };

    public getRemote = async (name: string): Promise<string> => {
        try {
            const response = await axios.get<{url: string}>(
                `${config.get('url')}version/${name}/remote`,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            return response.data.url;
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '获取远程仓库地址失败';
            this.snack('获取远程仓库地址失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    // 环境变量文件管理方法

    public getEnvFile = async (
        name: string
    ): Promise<{content: string; exists: boolean; path: string}> => {
        try {
            const response = await axios.get<{content: string; exists: boolean; path: string}>(
                `${config.get('url')}version/${name}/env`,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            return response.data;
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '获取环境变量文件失败';
            this.snack('获取环境变量文件失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    public saveEnvFile = async (name: string, content: string): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/${name}/env`,
                {
                    content: content,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || '环境变量文件保存成功');
        } catch (error: unknown) {
            if (error && typeof error === 'object' && 'response' in error) {
                const axiosError = error as {
                    response?: {data?: {error?: string; details?: string[]}};
                };
                const errorData = axiosError.response?.data;

                if (errorData?.details && Array.isArray(errorData.details)) {
                    // 格式验证失败，显示详细错误信息
                    const details = errorData.details.join('\n');
                    this.snack(`环境变量文件格式验证失败:\n${details}`);
                    throw new Error(`格式验证失败:\n${details}`);
                } else {
                    const errorMessage = errorData?.error ?? '保存环境变量文件失败';
                    this.snack('保存环境变量文件失败: ' + errorMessage);
                    throw new Error(errorMessage);
                }
            } else {
                const errorMessage = error instanceof Error ? error.message : '未知错误';
                this.snack('保存环境变量文件失败: ' + errorMessage);
                throw new Error(errorMessage);
            }
        }
    };

    public deleteEnvFile = async (name: string): Promise<void> => {
        try {
            const response = await axios.delete(`${config.get('url')}version/${name}/env`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            this.snack(response.data.message || '环境变量文件删除成功');
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '删除环境变量文件失败';
            this.snack('删除环境变量文件失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    // GitHook配置管理方法
    @action
    public saveGitHookConfig = async (
        projectName: string,
        gitHookConfig: GitHookConfig
    ): Promise<void> => {
        try {
            const response = await axios.post(
                `${config.get('url')}version/${projectName}/githook`,
                {
                    enhook: gitHookConfig.enhook,
                    hookmode: gitHookConfig.hookmode,
                    hookbranch: gitHookConfig.hookbranch,
                    hooksecret: gitHookConfig.hooksecret,
                },
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack(response.data.message || 'GitHook配置保存成功');
            await this.refreshProjects(); // 保存后刷新项目列表
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '保存GitHook配置失败';
            this.snack('保存GitHook配置失败: ' + errorMessage);
            throw new Error(errorMessage);
        }
    };

    @action
    public refreshBranches = async (projectName: string): Promise<void> => {
        this.currentProject = projectName;
        this.branches = await this.requestBranches(projectName).then((branches) => branches || []);
    };

    @action
    public refreshTags = async (
        projectName: string,
        filter?: string,
        messageFilter?: string
    ): Promise<void> => {
        this.currentProject = projectName;
        this.tagsLoading = true;
        this.tagsPage = 1;

        try {
            const response = await this.requestTags(projectName, filter, messageFilter, 1);
            this.tags = response.tags || [];
            this.tagsTotal = response.total;
            this.tagsPage = response.page;
            this.tagsHasMore = response.hasMore;
        } catch (error) {
            this.tags = [];
            this.tagsTotal = 0;
            this.tagsHasMore = false;
        } finally {
            this.tagsLoading = false;
        }
    };

    @action
    public loadMoreTags = async (
        projectName: string,
        filter?: string,
        messageFilter?: string
    ): Promise<void> => {
        if (this.tagsLoading || !this.tagsHasMore) {
            return;
        }

        this.tagsLoading = true;
        const nextPage = this.tagsPage + 1;

        try {
            const response = await this.requestTags(projectName, filter, messageFilter, nextPage);
            this.tags = [...this.tags, ...(response.tags || [])];
            this.tagsTotal = response.total;
            this.tagsPage = response.page;
            this.tagsHasMore = response.hasMore;
        } catch (error) {
            // 加载失败时不更新状态
            console.error('加载更多标签失败:', error);
        } finally {
            this.tagsLoading = false;
        }
    };

    @action
    public switchBranch = async (projectName: string, branch: string): Promise<void> => {
        await this.requestSwitchBranch(projectName, branch);
        await this.refreshBranches(projectName);
        await this.refreshProjects();
    };

    @action
    public syncBranches = async (projectName: string): Promise<void> => {
        try {
            await axios.post(
                `${config.get('url')}version/${projectName}/sync-branches`,
                {},
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack('分支同步成功');
            await this.refreshBranches(projectName);
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('分支同步失败: ' + errorMessage);
            throw error;
        }
    };

    @action
    public switchTag = async (projectName: string, tag: string): Promise<void> => {
        await this.requestSwitchTag(projectName, tag);
        await this.refreshTags(projectName);
        await this.refreshProjects();
    };

    @action
    public syncTags = async (projectName: string): Promise<void> => {
        await axios.post(
            `${config.get('url')}version/${projectName}/sync-tags`,
            {},
            {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            }
        );
        this.snack('标签同步成功');
        await this.refreshTags(projectName);
    };

    @action
    public deleteBranch = async (projectName: string, branchName: string): Promise<void> => {
        try {
            await axios.delete(
                `${config.get('url')}version/${projectName}/branches/${branchName}`,
                {
                    headers: {'X-GoHook-Key': this.tokenProvider()},
                }
            );
            this.snack('分支删除成功');
            await this.refreshBranches(projectName);
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('分支删除失败: ' + errorMessage);
            throw error;
        }
    };

    @action
    public deleteTag = async (projectName: string, tagName: string): Promise<void> => {
        try {
            await axios.delete(`${config.get('url')}version/${projectName}/tags/${tagName}`, {
                headers: {'X-GoHook-Key': this.tokenProvider()},
            });
            this.snack('标签删除成功');
            await this.refreshTags(projectName);
        } catch (error: unknown) {
            const errorMessage =
                error instanceof Error
                    ? error.message
                    : (error as {response?: {data?: {error?: string}}})?.response?.data?.error ??
                      '未知错误';
            this.snack('标签删除失败: ' + errorMessage);
            throw error;
        }
    };

    public getProjects = (): IVersion[] => this.projects;

    public getBranches = (): IBranch[] => this.branches;

    public getTags = (): ITag[] => this.tags;

    public getTagsTotal = (): number => this.tagsTotal;

    public getTagsHasMore = (): boolean => this.tagsHasMore;

    public getTagsLoading = (): boolean => this.tagsLoading;

    public getCurrentProject = (): string | null => this.currentProject;

    public getProjectByName = (name: string): IVersion | undefined =>
        this.projects.find((project) => project.name === name);

    @action
    public clear = (): void => {
        this.projects = [];
        this.branches = [];
        this.tags = [];
        this.tagsTotal = 0;
        this.tagsPage = 1;
        this.tagsHasMore = false;
        this.tagsLoading = false;
        this.currentProject = null;
    };
}
