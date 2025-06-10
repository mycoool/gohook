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
import TextField from '@material-ui/core/TextField';
import Dialog from '@material-ui/core/Dialog';
import DialogTitle from '@material-ui/core/DialogTitle';
import DialogContent from '@material-ui/core/DialogContent';
import DialogActions from '@material-ui/core/DialogActions';
import AccountTree from '@material-ui/icons/AccountTree';
import LocalOffer from '@material-ui/icons/LocalOffer';
import Refresh from '@material-ui/icons/Refresh';
import CloudDownload from '@material-ui/icons/CloudDownload';
import Add from '@material-ui/icons/Add';
import Delete from '@material-ui/icons/Delete';
import Git from '@material-ui/icons/GitHub';
import Link from '@material-ui/icons/Link';
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
    @observable
    private initGitProjectName: string | null = null;
    @observable
    private setRemoteProjectName: string | null = null;
    @observable
    private remoteUrl = '';

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
                                        onInitGit={() => this.initGitProjectName = project.name}
                                        onSetRemote={() => this.setRemoteProjectName = project.name}
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
                {this.initGitProjectName && (
                    <ConfirmDialog
                        title="初始化Git仓库"
                        text={`确定要为项目 "${this.initGitProjectName}" 初始化Git仓库吗？`}
                        fClose={() => this.initGitProjectName = null}
                        fOnSubmit={() => this.handleInitGit()}
                    />
                )}
                {this.setRemoteProjectName && (
                    <Dialog
                        open={true}
                        onClose={() => {
                            this.setRemoteProjectName = null;
                            this.remoteUrl = '';
                        }}
                        maxWidth="sm"
                        fullWidth>
                        <DialogTitle>设置远程仓库</DialogTitle>
                        <DialogContent>
                            <p>为项目 {this.setRemoteProjectName} 设置远程仓库地址：</p>
                            <TextField
                                autoFocus
                                margin="dense"
                                label="远程仓库URL"
                                placeholder="https://github.com/username/repository.git"
                                fullWidth
                                variant="outlined"
                                value={this.remoteUrl}
                                onChange={(e) => this.remoteUrl = e.target.value}
                                helperText="请输入完整的Git远程仓库地址"
                            />
                        </DialogContent>
                        <DialogActions>
                            <Button 
                                onClick={() => {
                                    this.setRemoteProjectName = null;
                                    this.remoteUrl = '';
                                }}
                                color="default">
                                取消
                            </Button>
                            <Button 
                                onClick={() => this.handleSetRemote()}
                                color="primary"
                                variant="contained"
                                disabled={!this.remoteUrl.trim()}>
                                确认设置
                            </Button>
                        </DialogActions>
                    </Dialog>
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

    private handleInitGit = async () => {
        if (this.initGitProjectName) {
            try {
                await this.props.versionStore.initGit(this.initGitProjectName);
                this.refreshProjects(); // 刷新项目列表
            } catch (error) {
                // 错误处理已在Store中完成
            } finally {
                this.initGitProjectName = null;
            }
        }
    };

    private handleSetRemote = async () => {
        if (this.setRemoteProjectName && this.remoteUrl.trim()) {
            try {
                await this.props.versionStore.setRemote(this.setRemoteProjectName, this.remoteUrl.trim());
                this.refreshProjects(); // 刷新项目列表
            } catch (error) {
                // 错误处理已在Store中完成
            } finally {
                this.setRemoteProjectName = null;
                this.remoteUrl = '';
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
    onInitGit: VoidFunction;
    onSetRemote: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({project, onViewBranches, onViewTags, onDelete, onInitGit, onSetRemote}) => {
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
                <div style={{display: 'flex', gap: '8px', flexWrap: 'wrap'}}>
                    {/* Git项目操作按钮 */}
                    {(project.status === 'active') && (
                        <>
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
                                onClick={onSetRemote} 
                                title="设置远程仓库"
                                size="small">
                                <Link />
                            </IconButton>
                        </>
                    )}
                    
                    {/* 非Git项目操作按钮 */}
                    {(project.status === 'not-git' || project.mode === 'none') && (
                        <IconButton 
                            onClick={onInitGit} 
                            title="初始化Git仓库"
                            size="small">
                            <Git />
                        </IconButton>
                    )}
                    
                    {/* 所有项目都有删除按钮 */}
                    <IconButton 
                        onClick={onDelete} 
                        title="删除项目"
                        size="small"
                        style={{color: '#f44336'}}>
                        <Delete />
                    </IconButton>
                </div>
            </TableCell>
        </TableRow>
    );
});

export default withRouter(inject('versionStore')(Versions)); 