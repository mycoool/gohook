import Grid from '@material-ui/core/Grid';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Button from '@material-ui/core/Button';
import ButtonGroup from '@material-ui/core/ButtonGroup';
import IconButton from '@material-ui/core/IconButton';
import Chip from '@material-ui/core/Chip';
import TextField from '@material-ui/core/TextField';
import Dialog from '@material-ui/core/Dialog';
import DialogTitle from '@material-ui/core/DialogTitle';
import DialogContent from '@material-ui/core/DialogContent';
import DialogContentText from '@material-ui/core/DialogContentText';
import DialogActions from '@material-ui/core/DialogActions';
import CircularProgress from '@material-ui/core/CircularProgress';
import Refresh from '@material-ui/icons/Refresh';
import CloudDownload from '@material-ui/icons/CloudDownload';
import Add from '@material-ui/icons/Add';
import Delete from '@material-ui/icons/Delete';
import AccountTree from '@material-ui/icons/AccountTree';
import LocalOffer from '@material-ui/icons/LocalOffer';
import CloudQueue from '@material-ui/icons/CloudQueue';
import GitHubIcon from '@material-ui/icons/GitHub';
import React, {Component, SFC} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import AddProjectDialog from './AddProjectDialog';
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

    public componentDidMount = () => this.props.versionStore.refreshProjects();

    public render() {
        const {versionStore} = this.props;
        const projects = versionStore.getProjects();
        
        return (
            <VersionsContainer
                projects={projects}
                showAddDialog={this.showAddDialog}
                deleteProjectName={this.deleteProjectName}
                initGitProjectName={this.initGitProjectName}
                setRemoteProjectName={this.setRemoteProjectName}
                remoteUrl={this.remoteUrl}
                currentRemoteUrl={this.currentRemoteUrl}
                loadingRemote={this.loadingRemote}
                onShowAddDialog={() => this.showAddDialog = true}
                onHideAddDialog={() => this.showAddDialog = false}
                onAddProject={this.handleAddProject}
                onRefreshProjects={this.refreshProjects}
                onReloadConfig={this.reloadConfig}
                onViewBranches={this.handleViewBranches}
                onViewTags={this.handleViewTags}
                onDelete={(name) => this.deleteProjectName = name}
                onInitGit={(name) => this.initGitProjectName = name}
                onSetRemote={this.handleSetRemote}
                onRemoteUrlChange={(url) => this.remoteUrl = url}
                onCancelDelete={() => this.deleteProjectName = null}
                onConfirmDelete={this.handleDeleteProject}
                onCancelInitGit={() => this.initGitProjectName = null}
                onConfirmInitGit={this.handleInitGit}
                onCancelSetRemote={() => {
                    this.setRemoteProjectName = null;
                    this.remoteUrl = '';
                    this.currentRemoteUrl = '';
                }}
                onConfirmSetRemote={this.handleConfirmSetRemote}
            />
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
}

interface IRowProps {
    project: IVersion;
    onViewBranches: (projectName: string) => void;
    onViewTags: (projectName: string) => void;
    onDelete: (projectName: string) => void;
    onInitGit: (projectName: string) => void;
    onSetRemote: (projectName: string) => void;
}

const Row: SFC<IRowProps> = observer(({project, onViewBranches, onViewTags, onDelete, onInitGit, onSetRemote}) => {
    const { t } = useTranslation();
    
    const getModeChip = (mode: string) => {
        switch (mode) {
            case 'branch':
                return (
                    <Chip
                        label={t('version.branchMode')}
                        size="small"
                        style={{backgroundColor: '#4caf50', color: 'white'}}
                    />
                );
            case 'tag':
                return (
                    <Chip
                        label={t('version.tagMode')}
                        size="small"
                        style={{backgroundColor: '#2196f3', color: 'white'}}
                    />
                );
            default:
                return (
                    <Chip
                        label={t('version.nonGitProject')}
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
        const statusText = status === 'active' ? t('version.active') : status === 'not-git' ? t('version.nonGit') : t('version.inactive');
        
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
                        onClick={() => onSetRemote(project.name)}
                        title={t('version.setRemote')}>
                        <CloudQueue />
                    </IconButton>
                    <IconButton 
                        size="small"
                        onClick={() => onDelete(project.name)}
                        title={t('common.delete')}>
                        <Delete />
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
                {renderActions()}
            </TableCell>
        </TableRow>
    );
});

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
    onRefreshProjects: () => void;
    onReloadConfig: () => void;
    onViewBranches: (projectName: string) => void;
    onViewTags: (projectName: string) => void;
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
    onRefreshProjects,
    onReloadConfig,
    onViewBranches,
    onViewTags,
    onDelete,
    onInitGit,
    onSetRemote,
    onRemoteUrlChange,
    onCancelDelete,
    onConfirmDelete,
    onCancelInitGit,
    onConfirmInitGit,
    onCancelSetRemote,
    onConfirmSetRemote
}) => {
    const { t } = useTranslation();

    return (
        <DefaultPage
            title={t('version.title')}
            rightControl={
                <ButtonGroup variant="contained" color="primary">
                    <Button
                        id="add-project"
                        startIcon={<Add />}
                        onClick={onShowAddDialog}>
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
            <Grid item xs={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="version-table">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('version.projectName')}</TableCell>
                                <TableCell>{t('version.projectDescription')}</TableCell>
                                <TableCell>{t('version.currentBranch')}/{t('version.currentTag')}</TableCell>
                                <TableCell>{t('common.status')}</TableCell>
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
                    text={t('version.confirmDeleteText', { name: deleteProjectName })}
                    fClose={onCancelDelete}
                    fOnSubmit={onConfirmDelete}
                />
            )}
            {initGitProjectName && (
                <ConfirmDialog
                    title={t('version.confirmInitGit')}
                    text={t('version.confirmInitGitText', { name: initGitProjectName })}
                    fClose={onCancelInitGit}
                    fOnSubmit={onConfirmInitGit}
                />
            )}
            {setRemoteProjectName && (
                <Dialog
                    open={true}
                    onClose={onCancelSetRemote}
                    maxWidth="sm"
                    fullWidth>
                    <DialogTitle>{t('version.setRemote')}</DialogTitle>
                    <DialogContent>
                        <DialogContentText style={{ marginBottom: '16px' }}>
                            {t('version.setRemoteText', { name: setRemoteProjectName })}
                        </DialogContentText>
                        
                        {loadingRemote ? (
                            <div style={{display: 'flex', alignItems: 'center', margin: '16px 0'}}>
                                <CircularProgress size={20} style={{marginRight: '8px'}} />
                                <span style={{color: '#666'}}>{t('version.loadingCurrentRemote')}</span>
                            </div>
                        ) : currentRemoteUrl ? (
                            <div style={{marginBottom: '16px', padding: '12px', backgroundColor: 'rgba(0, 0, 0, 0.05)', borderRadius: '4px'}}>
                                <div style={{fontSize: '0.9em', color: '#666', marginBottom: '4px'}}>
                                    {t('version.currentRemoteUrl')}:
                                </div>
                                <div style={{fontFamily: 'monospace', fontSize: '0.85em', wordBreak: 'break-all'}}>
                                    {currentRemoteUrl}
                                </div>
                            </div>
                        ) : (
                            <div style={{marginBottom: '16px', padding: '12px', backgroundColor: '#fff3cd', borderRadius: '4px', border: '1px solid #ffeaa7'}}>
                                <div style={{fontSize: '0.9em', color: '#856404'}}>
                                    {t('version.noCurrentRemote')}
                                </div>
                            </div>
                        )}
                        
                        <TextField
                            autoFocus
                            margin="dense"
                            label={t('version.remoteUrl')}
                            placeholder={currentRemoteUrl || "https://github.com/username/repository.git"}
                            fullWidth
                            variant="outlined"
                            value={remoteUrl}
                            onChange={(e) => onRemoteUrlChange(e.target.value)}
                            helperText={t('version.remoteUrlPlaceholder')}
                            disabled={loadingRemote}
                        />
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={onCancelSetRemote} disabled={loadingRemote}>
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

export default withRouter(inject('versionStore')(Versions)); 