import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Delete from '@material-ui/icons/Delete';
import PlayArrow from '@material-ui/icons/PlayArrow';
import React, {Component, SFC} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@material-ui/core/Button';
import Chip from '@material-ui/core/Chip';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IProject} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';

@observer
class Projects extends Component<Stores<'projectStore'>> {
    @observable
    private deleteId: string | false = false;
    @observable
    private triggerDialog = false;

    public componentDidMount = () => this.props.projectStore.refresh();

    public render() {
        const {
            deleteId,
            props: {projectStore},
        } = this;
        const projects = projectStore.getItems();
        return (
            <DefaultPage
                title="Projects"
                rightControl={
                    <Button
                        id="create-project"
                        variant="contained"
                        color="primary"
                        onClick={() => this.refreshProjects()}>
                        Refresh Projects
                    </Button>
                }
                maxWidth={1200}>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="project-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Name</TableCell>
                                    <TableCell>Description</TableCell>
                                    <TableCell>Command</TableCell>
                                    <TableCell>HTTP Methods</TableCell>
                                    <TableCell>Trigger Rules</TableCell>
                                    <TableCell>Status</TableCell>
                                    <TableCell>Last Used</TableCell>
                                    <TableCell />
                                    <TableCell />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {projects.map((project: IProject) => (
                                    <Row
                                        key={project.id}
                                        project={project}
                                        fTrigger={() => this.triggerProject(project.id)}
                                        fDelete={() => (this.deleteId = project.id)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {deleteId !== false && (
                    <ConfirmDialog
                        title="Confirm Delete"
                        text={'Delete project ' + projectStore.getByID(deleteId).name + '?'}
                        fClose={() => (this.deleteId = false)}
                        fOnSubmit={() => projectStore.remove(deleteId)}
                    />
                )}
            </DefaultPage>
        );
    }

    private refreshProjects = () => {
        this.props.projectStore.refresh();
    };

    private triggerProject = (id: string) => {
        this.props.projectStore.triggerProject(id);
    };
}

interface IRowProps {
    project: IProject;
    fTrigger: VoidFunction;
    fDelete: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({project, fTrigger, fDelete}) => (
    <TableRow>
        <TableCell>
            <strong>{project.name}</strong>
            <br />
            <small style={{color: '#666'}}>ID: {project.id}</small>
        </TableCell>
        <TableCell style={{maxWidth: 200, wordWrap: 'break-word'}}>
            {project.description}
        </TableCell>
        <TableCell>
            <code style={{fontSize: '0.85em', backgroundColor: '#f5f5f5', padding: '2px 4px', borderRadius: '3px'}}>
                {project.executeCommand}
            </code>
            {project.workingDirectory && (
                <div style={{fontSize: '0.8em', color: '#666', marginTop: '4px'}}>
                    Working Dir: {project.workingDirectory}
                </div>
            )}
        </TableCell>
        <TableCell>
            {project.httpMethods.map((method) => (
                <Chip
                    key={method}
                    label={method}
                    size="small"
                    style={{
                        marginRight: '4px',
                        marginBottom: '2px',
                        backgroundColor: getMethodColor(method),
                        color: 'white',
                        fontSize: '0.7em'
                    }}
                />
            ))}
        </TableCell>
        <TableCell style={{maxWidth: 150, wordWrap: 'break-word', fontSize: '0.85em'}}>
            {project.triggerRuleDescription}
        </TableCell>
        <TableCell>
            <Chip
                label={project.status}
                size="small"
                style={{
                    backgroundColor: project.status === 'active' ? '#4caf50' : '#f44336',
                    color: 'white'
                }}
            />
        </TableCell>
        <TableCell>
            <LastUsedCell lastUsed={project.lastUsed} />
        </TableCell>
        <TableCell align="right" padding="none">
            <IconButton onClick={fTrigger} className="trigger" title="Trigger Webhook">
                <PlayArrow />
            </IconButton>
        </TableCell>
        <TableCell align="right" padding="none">
            <IconButton onClick={fDelete} className="delete" title="Delete Project">
                <Delete />
            </IconButton>
        </TableCell>
    </TableRow>
));

// 根据HTTP方法返回对应的颜色
function getMethodColor(method: string): string {
    switch (method.toUpperCase()) {
        case 'GET':
            return '#4caf50';
        case 'POST':
            return '#2196f3';
        case 'PUT':
            return '#ff9800';
        case 'DELETE':
            return '#f44336';
        case 'PATCH':
            return '#9c27b0';
        default:
            return '#607d8b';
    }
}

export default inject('projectStore')(Projects); 