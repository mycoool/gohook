import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Chip from '@mui/material/Chip';
import TextField from '@mui/material/TextField';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogActions from '@mui/material/DialogActions';
import CircularProgress from '@mui/material/CircularProgress';
import Refresh from '@mui/icons-material/Refresh';
import CloudDownload from '@mui/icons-material/CloudDownload';
import Add from '@mui/icons-material/Add';
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/AppRegistration';
import AccountTree from '@mui/icons-material/AccountTree';
import LocalOffer from '@mui/icons-material/LocalOffer';
import CloudQueue from '@mui/icons-material/CloudQueue';
import GitHubIcon from '@mui/icons-material/GitHub';
import Settings from '@mui/icons-material/SettingsApplications';
import Link from '@mui/icons-material/Link';
import React, {Component} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import AddProjectDialog from './AddProjectDialog';
import EditProjectDialog from './EditProjectDialog';
import EnvFileDialog from './EnvFileDialogModal';
import GitHookDialog, {GitHookConfig} from './GitHookDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IVersion} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';
import useTranslation from '../i18n/useTranslation';

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
    @observable
    private currentRemoteUrl = '';
    @observable
    private loadingRemote = false;
    @observable
    private envDialogProjectName: string | null = null;
    @observable
    private gitHookDialogProject: IVersion | null = null;
    @observable
    private editProjectDialog: IVersion | null = null;

    public componentDidMount = () => this.props.versionStore.refreshProjects();

    public render() {
        const {versionStore} = this.props;
        const projects = versionStore.getProjects();

        return (
            <>
                <VersionsContainer
                    projects={projects}
                    showAddDialog={this.showAddDialog}
                    deleteProjectName={this.deleteProjectName}
                    initGitProjectName={this.initGitProjectName}
                    setRemoteProjectName={this.setRemoteProjectName}
                    remoteUrl={this.remoteUrl}
                    currentRemoteUrl={this.currentRemoteUrl}
                    loadingRemote={this.loadingRemote}
                    onShowAddDialog={() => (this.showAddDialog = true)}
                    onHideAddDialog={() => (this.showAddDialog = false)}
                    onAddProject={this.handleAddProject}
                    onEditProject={(project) => (this.editProjectDialog = project)}
                    onRefreshProjects={this.refreshProjects}
                    onReloadConfig={this.reloadConfig}
                    onViewBranches={this.handleViewBranches}
                    onViewTags={this.handleViewTags}
                    onEditEnv={this.handleEditEnv}
                    onConfigGitHook={this.handleConfigGitHook}
                    onDelete={(name) => (this.deleteProjectName = name)}
                    onInitGit={(name) => (this.initGitProjectName = name)}
                    onSetRemote={this.handleSetRemote}
                    onRemoteUrlChange={(url) => (this.remoteUrl = url)}
                    onCancelDelete={() => (this.deleteProjectName = null)}
                    onConfirmDelete={this.handleDeleteProject}
                    onCancelInitGit={() => (this.initGitProjectName = null)}
                    onConfirmInitGit={this.handleInitGit}
                    onCancelSetRemote={() => {
                        this.setRemoteProjectName = null;
                        this.remoteUrl = '';
                        this.currentRemoteUrl = '';
                    }}
                    onConfirmSetRemote={this.handleConfirmSetRemote}
                />

                <EnvFileDialog
                    open={this.envDialogProjectName !== null}
                    projectName={this.envDialogProjectName ?? ''}
                    onClose={this.handleCloseEnvDialog}
                    onGetEnvFile={this.handleGetEnvFile}
                    onSaveEnvFile={this.handleSaveEnvFile}
                    onDeleteEnvFile={this.handleDeleteEnvFile}
                />

                <GitHookDialog
                    open={this.gitHookDialogProject !== null}
                    project={this.gitHookDialogProject}
                    onClose={this.handleCloseGitHookDialog}
                    onSave={this.handleSaveGitHookConfig}
                />

                <EditProjectDialog
                    open={this.editProjectDialog !== null}
                    project={this.editProjectDialog}
                    onClose={() => (this.editProjectDialog = null)}
                    onSubmit={this.handleEditProject}
                />
            </>
        );
    }

    private refreshProjects = async () => {
        await this.props.versionStore.refreshProjects();
    };

    private reloadConfig = async () => {
        await this.props.versionStore.reloadConfig();
    };

    private handleAddProject = async (name: string, path: string, description: string) => {
        await this.props.versionStore.addProject(name, path, description);
    };

    private handleDeleteProject = async () => {
        if (this.deleteProjectName) {
            await this.props.versionStore.deleteProject(this.deleteProjectName);
            this.deleteProjectName = null;
        }
    };

    private handleInitGit = async () => {
        if (this.initGitProjectName) {
            await this.props.versionStore.initGit(this.initGitProjectName);
            this.initGitProjectName = null;
        }
    };

    private handleSetRemote = async (name: string) => {
        this.setRemoteProjectName = name;
        this.loadingRemote = true;
        this.currentRemoteUrl = '';
        this.remoteUrl = '';

        try {
            const remoteUrl = await this.props.versionStore.getRemote(name);
            if (remoteUrl) {
                this.currentRemoteUrl = remoteUrl;
                this.remoteUrl = remoteUrl;
            }
        } catch (error) {
            console.warn('Failed to get current remote URL:', error);
        } finally {
            this.loadingRemote = false;
        }
    };

    private handleConfirmSetRemote = async () => {
        if (this.setRemoteProjectName && this.remoteUrl) {
            await this.props.versionStore.setRemote(this.setRemoteProjectName, this.remoteUrl);
            this.setRemoteProjectName = null;
            this.remoteUrl = '';
            this.currentRemoteUrl = '';
        }
    };

    private handleViewBranches = (projectName: string) => {
        this.props.history.push(`/versions/${projectName}/branches`);
    };

    private handleViewTags = (projectName: string) => {
        this.props.history.push(`/versions/${projectName}/tags`);
    };

    private handleEditEnv = (projectName: string) => {
        this.envDialogProjectName = projectName;
    };

    private handleCloseEnvDialog = () => {
        this.envDialogProjectName = null;
    };

    private handleGetEnvFile = async (name: string) =>
        await this.props.versionStore.getEnvFile(name);

    private handleSaveEnvFile = async (name: string, content: string) => {
        await this.props.versionStore.saveEnvFile(name, content);
    };

    private handleDeleteEnvFile = async (name: string) => {
        await this.props.versionStore.deleteEnvFile(name);
    };

    private handleConfigGitHook = (project: IVersion) => {
        this.gitHookDialogProject = project;
    };

    private handleCloseGitHookDialog = () => {
        this.gitHookDialogProject = null;
    };

    private handleSaveGitHookConfig = async (projectName: string, config: GitHookConfig) => {
        await this.props.versionStore.saveGitHookConfig(projectName, config);
    };

    private handleEditProject = async (
        originalName: string,
        name: string,
        path: string,
        description: string
    ) => {
        await this.props.versionStore.editProject(originalName, name, path, description);
    };
}

interface IRowProps {
    project: IVersion;
    onViewBranches: (projectName: string) => void;
    onViewTags: (projectName: string) => void;
    onEditEnv: (projectName: string) => void;
    onConfigGitHook: (project: IVersion) => void;
    onEditProject: (project: IVersion) => void;
    onDelete: (projectName: string) => void;
    onInitGit: (projectName: string) => void;
    onSetRemote: (projectName: string) => void;
}

const Row: React.FC<IRowProps> = observer(
    ({
        project,
        onViewBranches,
        onViewTags,
        onEditEnv,
        onConfigGitHook,
        onEditProject,
        onDelete,
        onInitGit,
        onSetRemote,
    }) => {
        const {t} = useTranslation();

        const getModeChip = (mode: string) => {
            switch (mode) {
                case 'branch':
                    return (
                        <Chip
                            label={t('version.branch')}
                            size="small"
                            style={{backgroundColor: '#4caf50', color: 'white'}}
                        />
                    );
                case 'tag':
                    return (
                        <Chip
                            label={t('version.tag')}
                            size="small"
                            style={{backgroundColor: '#2196f3', color: 'white'}}
                        />
                    );
                default:
                    return (
                        <Chip
                            label={t('version.nonGit')}
                            size="small"
                            style={{backgroundColor: '#9e9e9e', color: 'white'}}
                        />
                    );
            }
        };

        const getCurrentVersion = () => {
            if (project.mode === 'branch') {
                return project.currentBranch || t('version.unknownBranch');
            } else if (project.mode === 'tag') {
                return project.currentTag || t('version.unknownTag');
            }
            return t('version.noVersionInfo');
        };

        const getStatusChip = (status: string) => {
            const statusColor = status === 'active' ? '#4caf50' : '#f44336';
            const statusText =
                status === 'active'
                    ? t('version.active')
                    : status === 'not-git'
                    ? t('version.nonGit')
                    : t('version.inactive');

            return (
                <Chip
                    label={statusText}
                    size="small"
                    style={{backgroundColor: statusColor, color: 'white'}}
                />
            );
        };

        const renderActions = () => {
            if (project.mode === 'none') {
                // 非Git项目
                return (
                    <div style={{display: 'flex', alignItems: 'center', gap: '4px'}}>
                        <IconButton
                            size="small"
                            onClick={() => onEditEnv(project.name)}
                            title="编辑环境变量">
                            <Settings />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onInitGit(project.name)}
                            title={t('version.initGit')}>
                            <GitHubIcon />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onDelete(project.name)}
                            title={t('common.delete')}>
                            <Delete />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onEditProject(project)}
                            title={t('version.editProject')}>
                            <Edit />
                        </IconButton>
                    </div>
                );
            } else {
                // Git项目
                return (
                    <div style={{display: 'flex', alignItems: 'center', gap: '4px'}}>
                        <IconButton
                            size="small"
                            onClick={() => onViewBranches(project.name)}
                            title={t('version.branches')}>
                            <AccountTree />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onViewTags(project.name)}
                            title={t('version.tags')}>
                            <LocalOffer />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onEditEnv(project.name)}
                            title="编辑环境变量">
                            <Settings />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onSetRemote(project.name)}
                            title={t('version.setRemote')}>
                            <CloudQueue />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onConfigGitHook(project)}
                            title="配置 GitHook"
                            style={{
                                color: project.enhook ? '#4caf50' : '#666',
                            }}>
                            <Link />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onDelete(project.name)}
                            title={t('common.delete')}>
                            <Delete />
                        </IconButton>
                        <IconButton
                            size="small"
                            onClick={() => onEditProject(project)}
                            title={t('version.editProject')}>
                            <Edit />
                        </IconButton>
                    </div>
                );
            }
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
                <TableCell>{getModeChip(project.mode)}</TableCell>
                <TableCell>
                    {project.mode !== 'none' &&
                        (project.enhook ? (
                            <Chip
                                label={`${
                                    project.hookmode === 'branch'
                                        ? t('version.branch')
                                        : t('version.tag')
                                }${
                                    project.hookmode === 'branch' && project.hookbranch !== '*'
                                        ? `(${project.hookbranch})`
                                        : project.hookmode === 'branch'
                                        ? `(${t('githook.anyBranchShort')})`
                                        : ''
                                }`}
                                size="small"
                                style={{
                                    backgroundColor: '#4caf50',
                                    color: 'white',
                                    fontSize: '0.7rem',
                                    height: '20px',
                                }}
                            />
                        ) : (
                            <Chip
                                label={t('githook.notEnabled')}
                                size="small"
                                style={{
                                    backgroundColor: '#9e9e9e',
                                    color: 'white',
                                    fontSize: '0.7rem',
                                    height: '20px',
                                }}
                            />
                        ))}
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
                <TableCell>{renderActions()}</TableCell>
            </TableRow>
        );
    }
);

const VersionsContainer: React.FC<{
    projects: IVersion[];
    showAddDialog: boolean;
    deleteProjectName: string | null;
    initGitProjectName: string | null;
    setRemoteProjectName: string | null;
    remoteUrl: string;
    currentRemoteUrl: string;
    loadingRemote: boolean;
    onShowAddDialog: () => void;
    onHideAddDialog: () => void;
    onAddProject: (name: string, path: string, description: string) => Promise<void>;
    onEditProject: (project: IVersion) => void;
    onRefreshProjects: () => void;
    onReloadConfig: () => void;
    onViewBranches: (projectName: string) => void;
    onViewTags: (projectName: string) => void;
    onEditEnv: (projectName: string) => void;
    onConfigGitHook: (project: IVersion) => void;
    onDelete: (projectName: string) => void;
    onInitGit: (projectName: string) => void;
    onSetRemote: (projectName: string) => void;
    onRemoteUrlChange: (url: string) => void;
    onCancelDelete: () => void;
    onConfirmDelete: () => void;
    onCancelInitGit: () => void;
    onConfirmInitGit: () => void;
    onCancelSetRemote: () => void;
    onConfirmSetRemote: () => void;
}> = ({
    projects,
    showAddDialog,
    deleteProjectName,
    initGitProjectName,
    setRemoteProjectName,
    remoteUrl,
    currentRemoteUrl,
    loadingRemote,
    onShowAddDialog,
    onHideAddDialog,
    onAddProject,
    onEditProject,
    onRefreshProjects,
    onReloadConfig,
    onViewBranches,
    onViewTags,
    onEditEnv,
    onConfigGitHook,
    onDelete,
    onInitGit,
    onSetRemote,
    onRemoteUrlChange,
    onCancelDelete,
    onConfirmDelete,
    onCancelInitGit,
    onConfirmInitGit,
    onCancelSetRemote,
    onConfirmSetRemote,
}) => {
    const {t} = useTranslation();

    return (
        <DefaultPage
            title={t('version.title')}
            rightControl={
                <ButtonGroup variant="contained" color="primary">
                    <Button id="add-project" startIcon={<Add />} onClick={onShowAddDialog}>
                        {t('version.addProject')}
                    </Button>
                    <Button
                        id="refresh-versions"
                        startIcon={<Refresh />}
                        onClick={onRefreshProjects}>
                        {t('common.refresh')}
                    </Button>
                    <Button
                        id="reload-config"
                        startIcon={<CloudDownload />}
                        onClick={onReloadConfig}>
                        {t('version.reloadConfig')}
                    </Button>
                </ButtonGroup>
            }
            maxWidth={1200}>
            <Grid size={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="version-table">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('version.projectName')}</TableCell>
                                <TableCell>{t('version.projectDescription')}</TableCell>
                                <TableCell>
                                    {t('version.currentBranch')}/{t('version.currentTag')}
                                </TableCell>
                                <TableCell>{t('version.gitStatus')}</TableCell>
                                <TableCell>{t('version.hookStatus')}</TableCell>
                                <TableCell>{t('version.lastCommit')}</TableCell>
                                <TableCell>{t('common.actions')}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {projects.map((project) => (
                                <Row
                                    key={project.name}
                                    project={project}
                                    onViewBranches={() => onViewBranches(project.name)}
                                    onViewTags={() => onViewTags(project.name)}
                                    onEditEnv={() => onEditEnv(project.name)}
                                    onConfigGitHook={() => onConfigGitHook(project)}
                                    onEditProject={() => onEditProject(project)}
                                    onDelete={() => onDelete(project.name)}
                                    onInitGit={() => onInitGit(project.name)}
                                    onSetRemote={() => onSetRemote(project.name)}
                                />
                            ))}
                        </TableBody>
                    </Table>
                </Paper>
            </Grid>
            {showAddDialog && (
                <AddProjectDialog
                    open={showAddDialog}
                    onClose={onHideAddDialog}
                    onSubmit={onAddProject}
                />
            )}
            {deleteProjectName && (
                <ConfirmDialog
                    title={t('version.confirmDelete')}
                    text={t('version.confirmDeleteText', {name: deleteProjectName})}
                    fClose={onCancelDelete}
                    fOnSubmit={onConfirmDelete}
                />
            )}
            {initGitProjectName && (
                <ConfirmDialog
                    title={t('version.confirmInitGit')}
                    text={t('version.confirmInitGitText', {name: initGitProjectName})}
                    fClose={onCancelInitGit}
                    fOnSubmit={onConfirmInitGit}
                />
            )}
            {setRemoteProjectName && (
                <Dialog open={true} onClose={onCancelSetRemote} maxWidth="sm" fullWidth>
                    <DialogTitle>{t('version.setRemote')}</DialogTitle>
                    <DialogContent>
                        <DialogContentText style={{marginBottom: '16px'}}>
                            {t('version.setRemoteText', {name: setRemoteProjectName})}
                        </DialogContentText>

                        {loadingRemote ? (
                            <div style={{display: 'flex', alignItems: 'center', margin: '16px 0'}}>
                                <CircularProgress size={20} style={{marginRight: '8px'}} />
                                <span style={{color: '#666'}}>
                                    {t('version.loadingCurrentRemote')}
                                </span>
                            </div>
                        ) : currentRemoteUrl ? (
                            <div
                                style={{
                                    marginBottom: '16px',
                                    padding: '12px',
                                    backgroundColor: 'rgba(0, 0, 0, 0.05)',
                                    borderRadius: '4px',
                                }}>
                                <div
                                    style={{fontSize: '0.9em', color: '#666', marginBottom: '4px'}}>
                                    {t('version.currentRemoteUrl')}:
                                </div>
                                <div
                                    style={{
                                        fontFamily: 'monospace',
                                        fontSize: '0.85em',
                                        wordBreak: 'break-all',
                                    }}>
                                    {currentRemoteUrl}
                                </div>
                            </div>
                        ) : (
                            <div
                                style={{
                                    marginBottom: '16px',
                                    padding: '12px',
                                    backgroundColor: '#fff3cd',
                                    borderRadius: '4px',
                                    border: '1px solid #ffeaa7',
                                }}>
                                <div style={{fontSize: '0.9em', color: '#856404'}}>
                                    {t('version.noCurrentRemote')}
                                </div>
                            </div>
                        )}

                        <TextField
                            autoFocus
                            margin="dense"
                            label={t('version.remoteUrl')}
                            placeholder={
                                currentRemoteUrl || 'https://github.com/username/repository.git'
                            }
                            fullWidth
                            variant="outlined"
                            value={remoteUrl}
                            onChange={(e) => onRemoteUrlChange(e.target.value)}
                            helperText={t('version.remoteUrlPlaceholder')}
                            disabled={loadingRemote}
                        />
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={onCancelSetRemote} disabled={loadingRemote} variant="contained" color="secondary">
                            {t('common.cancel')}
                        </Button>
                        <Button
                            onClick={onConfirmSetRemote}
                            color="primary"
                            variant="contained"
                            disabled={!remoteUrl.trim() || loadingRemote}>
                            {t('common.confirm')}
                        </Button>
                    </DialogActions>
                </Dialog>
            )}
        </DefaultPage>
    );
};

export default (withRouter as any)((inject as any)('versionStore')(Versions));
