import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Button from '@material-ui/core/Button';
import ButtonGroup from '@material-ui/core/ButtonGroup';
import Chip from '@material-ui/core/Chip';
import AccountTree from '@material-ui/icons/AccountTree';
import LocalOffer from '@material-ui/icons/LocalOffer';
import Refresh from '@material-ui/icons/Refresh';
import CloudDownload from '@material-ui/icons/CloudDownload';
import Add from '@material-ui/icons/Add';
import Delete from '@material-ui/icons/Delete';
import React, {Component, SFC} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import AddProjectDialog from './AddProjectDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IVersion} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';

@observer
class Versions extends Component<RouteComponentProps & Stores<'versionStore'>> {
    @observable
    private showAddDialog = false;
    @observable
    private deleteProjectName: string | null = null;

    public componentDidMount = () => this.props.versionStore.refreshProjects();

    public render() {
        const {versionStore} = this.props;
        const projects = versionStore.getProjects();
        
        return (
            <DefaultPage
                title="版本管理 (VERS)"
                rightControl={
                    <ButtonGroup variant="contained" color="primary">
                        <Button
                            id="add-project"
                            startIcon={<Add />}
                            onClick={() => this.showAddDialog = true}>
                            添加项目
                        </Button>
                        <Button
                            id="refresh-versions"
                            startIcon={<Refresh />}
                            onClick={() => this.refreshProjects()}>
                            刷新
                        </Button>
                        <Button
                            id="reload-config"
                            startIcon={<CloudDownload />}
                            onClick={() => this.reloadConfig()}>
                            重新加载配置
                        </Button>
                    </ButtonGroup>
                }
                maxWidth={1200}>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="version-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>项目名称</TableCell>
                                    <TableCell>描述</TableCell>
                                    <TableCell>当前版本状态</TableCell>
                                    <TableCell>模式</TableCell>
                                    <TableCell>最后提交</TableCell>
                                    <TableCell>操作</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {projects.map((project: IVersion) => (
                                    <Row
                                        key={project.name}
                                        project={project}
                                        onViewBranches={() => this.viewBranches(project.name)}
                                        onViewTags={() => this.viewTags(project.name)}
                                        onDelete={() => this.deleteProjectName = project.name}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {this.showAddDialog && (
                    <AddProjectDialog
                        open={this.showAddDialog}
                        onClose={() => this.showAddDialog = false}
                        onSubmit={this.handleAddProject}
                    />
                )}
                {this.deleteProjectName && (
                    <ConfirmDialog
                        title="确认删除项目"
                        text={`确定要删除项目 "${this.deleteProjectName}" 吗？此操作不可撤销。`}
                        fClose={() => this.deleteProjectName = null}
                        fOnSubmit={() => this.handleDeleteProject()}
                    />
                )}
            </DefaultPage>
        );
    }

    private refreshProjects = () => {
        this.props.versionStore.refreshProjects();
    };

    private reloadConfig = () => {
        this.props.versionStore.reloadConfig();
    };

    private handleAddProject = async (name: string, path: string, description: string) => {
        await this.props.versionStore.addProject(name, path, description);
    };

    private handleDeleteProject = async () => {
        if (this.deleteProjectName) {
            try {
                await this.props.versionStore.deleteProject(this.deleteProjectName);
            } catch (error) {
                // 错误处理已在Store中完成
            } finally {
                this.deleteProjectName = null;
            }
        }
    };

    private viewBranches = (projectName: string) => {
        this.props.history.push(`/versions/${projectName}/branches`);
    };

    private viewTags = (projectName: string) => {
        this.props.history.push(`/versions/${projectName}/tags`);
    };
}

interface IRowProps {
    project: IVersion;
    onViewBranches: VoidFunction;
    onViewTags: VoidFunction;
    onDelete: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({project, onViewBranches, onViewTags, onDelete}) => {
    const getModeChip = (mode: string) => {
        switch (mode) {
            case 'branch':
                return (
                    <Chip
                        label="分支模式"
                        size="small"
                        style={{backgroundColor: '#4caf50', color: 'white'}}
                    />
                );
            case 'tag':
                return (
                    <Chip
                        label="标签模式"
                        size="small"
                        style={{backgroundColor: '#2196f3', color: 'white'}}
                    />
                );
            default:
                return (
                    <Chip
                        label="非Git项目"
                        size="small"
                        style={{backgroundColor: '#9e9e9e', color: 'white'}}
                    />
                );
        }
    };

    const getCurrentVersion = () => {
        if (project.mode === 'branch') {
            return project.currentBranch || '未知分支';
        } else if (project.mode === 'tag') {
            return project.currentTag || '未知标签';
        }
        return '无版本信息';
    };

    const getStatusChip = (status: string) => {
        const statusColor = status === 'active' ? '#4caf50' : '#f44336';
        const statusText = status === 'active' ? '正常' : status === 'not-git' ? '非Git' : '异常';
        
        return (
            <Chip
                label={statusText}
                size="small"
                style={{backgroundColor: statusColor, color: 'white'}}
            />
        );
    };

    return (
        <TableRow>
            <TableCell>
                <strong>{project.name}</strong>
                <br />
                <small style={{color: '#666'}}>{project.path}</small>
            </TableCell>
            <TableCell style={{maxWidth: 200, wordWrap: 'break-word'}}>
                {project.description}
            </TableCell>
            <TableCell>
                <div style={{marginBottom: '4px'}}>
                    <strong>{getCurrentVersion()}</strong>
                </div>
                {getStatusChip(project.status)}
            </TableCell>
            <TableCell>
                {getModeChip(project.mode)}
            </TableCell>
            <TableCell>
                {project.lastCommit && (
                    <div>
                        <div style={{fontSize: '0.85em'}}>
                            <code>{project.lastCommit}</code>
                        </div>
                        <div style={{fontSize: '0.8em', color: '#666'}}>
                            {new Date(project.lastCommitTime).toLocaleString()}
                        </div>
                    </div>
                )}
            </TableCell>
            <TableCell>
                {project.status === 'active' && (
                    <div style={{display: 'flex', gap: '8px'}}>
                        <IconButton 
                            onClick={onViewBranches} 
                            title="管理分支"
                            size="small">
                            <AccountTree />
                        </IconButton>
                        <IconButton 
                            onClick={onViewTags} 
                            title="管理标签"
                            size="small">
                            <LocalOffer />
                        </IconButton>
                        <IconButton 
                            onClick={onDelete} 
                            title="删除项目"
                            size="small">
                            <Delete />
                        </IconButton>
                    </div>
                )}
            </TableCell>
        </TableRow>
    );
});

export default withRouter(inject('versionStore')(Versions)); 