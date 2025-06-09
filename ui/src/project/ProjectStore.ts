import axios from 'axios';
import * as config from '../config';
import {action, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IProject} from '../types';

export class ProjectStore {
    @observable
    protected items: IProject[] = [];

    public constructor(private readonly snack: SnackReporter) {}

    protected requestItems = (): Promise<IProject[]> =>
        axios
            .get<IProject[]>(`${config.get('url')}project`)
            .then((response) => response.data);

    protected requestDelete = (id: string): Promise<void> =>
        axios.delete(`${config.get('url')}project/${id}`).then(() => 
            this.snack('Project deleted')
        );

    @action
    public remove = async (id: string): Promise<void> => {
        await this.requestDelete(id);
        await this.refresh();
    };

    @action
    public refresh = async (): Promise<void> => {
        this.items = await this.requestItems().then((items) => items || []);
    };

    @action
    public triggerProject = async (id: string): Promise<void> => {
        await axios.post(`${config.get('url')}project/${id}/trigger`);
        this.snack('Project webhook triggered successfully');
    };

    @action
    public getProjectDetails = async (id: string): Promise<IProject> => {
        const response = await axios.get<IProject>(`${config.get('url')}project/${id}`);
        return response.data;
    };

    public getName = (id: string): string => {
        const project = this.getByIDOrUndefined(id);
        return project !== undefined ? project.name : 'unknown';
    };

    public getByIDOrUndefined = (id: string): IProject | undefined => 
        this.items.find(project => project.id === id);

    public getByID = (id: string): IProject => {
        const project = this.getByIDOrUndefined(id);
        if (project === undefined) {
            throw new Error(`Project with id ${id} not found`);
        }
        return project;
    };

    public getItems = (): IProject[] => this.items;

    @action
    public clear = (): void => {
        this.items = [];
    };
} 