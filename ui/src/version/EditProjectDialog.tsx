import React, {Component} from 'react';
import {observable} from 'mobx';
import {observer} from 'mobx-react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    Grid,
    CircularProgress,
    Box,
    Divider,
    FormControlLabel,
    IconButton,
    MenuItem,
    Stack,
    Switch,
    Typography,
} from '@mui/material';
import {IProjectSyncConfig, IProjectSyncNodeConfig, ISyncNode, IVersion} from '../types';
import useTranslation from '../i18n/useTranslation';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';

interface IProps {
    open: boolean;
    project: IVersion | null;
    availableNodes: ISyncNode[];
    onClose: () => void;
    onSubmit: (
        originalName: string,
        name: string,
        path: string,
        description: string,
        sync?: IProjectSyncConfig
    ) => Promise<void>;
}

@observer
export default class EditProjectDialog extends Component<IProps> {
    @observable
    private name = '';
    @observable
    private path = '';
    @observable
    private description = '';
    @observable
    private syncEnabled = false;
    @observable
    private syncIgnoreDefaults = true;
    @observable
    private syncIgnorePermissions = false;
    @observable
    private syncIgnorePatterns = '';
    @observable
    private syncIgnoreFile = '';
    @observable
    private syncNodes: Array<{nodeId: string; targetPath: string}> = [];
    @observable
    private submitting = false;

    public componentDidUpdate(prevProps: IProps) {
        if (this.props.open && !prevProps.open && this.props.project) {
            // 对话框打开时，用当前项目数据填充表单
            this.name = this.props.project.name;
            this.path = this.props.project.path;
            this.description = this.props.project.description || '';

            const sync = this.props.project.sync;
            this.syncEnabled = sync?.enabled ?? false;
            this.syncIgnoreDefaults = sync?.ignoreDefaults ?? true;
            this.syncIgnorePermissions = sync?.ignorePermissions ?? false;
            this.syncIgnorePatterns = (sync?.ignorePatterns || []).join('\n');
            this.syncIgnoreFile = sync?.ignoreFile || '';
            this.syncNodes = (sync?.nodes || []).map((n) => ({
                nodeId: n.nodeId,
                targetPath: n.targetPath,
            }));
        }
    }

    public render() {
        const {open, project, onClose} = this.props;
        const {
            name,
            path,
            description,
            submitting,
            syncEnabled,
            syncIgnoreDefaults,
            syncIgnorePermissions,
            syncIgnorePatterns,
            syncIgnoreFile,
            syncNodes,
        } = this;

        if (!project) {
            return null;
        }

        return (
            <EditProjectDialogContent
                open={open}
                project={project}
                availableNodes={this.props.availableNodes}
                onClose={onClose}
                name={name}
                path={path}
                description={description}
                syncEnabled={syncEnabled}
                syncIgnoreDefaults={syncIgnoreDefaults}
                syncIgnorePermissions={syncIgnorePermissions}
                syncIgnorePatterns={syncIgnorePatterns}
                syncIgnoreFile={syncIgnoreFile}
                syncNodes={syncNodes}
                submitting={submitting}
                onNameChange={(value) => (this.name = value)}
                onPathChange={(value) => (this.path = value)}
                onDescriptionChange={(value) => (this.description = value)}
                onSyncEnabledChange={(value) => (this.syncEnabled = value)}
                onSyncIgnoreDefaultsChange={(value) => (this.syncIgnoreDefaults = value)}
                onSyncIgnorePermissionsChange={(value) => (this.syncIgnorePermissions = value)}
                onSyncIgnorePatternsChange={(value) => (this.syncIgnorePatterns = value)}
                onSyncIgnoreFileChange={(value) => (this.syncIgnoreFile = value)}
                onSyncNodesChange={(value) => (this.syncNodes = value)}
                onSubmit={this.handleSubmit}
            />
        );
    }

    private handleSubmit = async () => {
        const {project, onSubmit, onClose} = this.props;
        const {
            name,
            path,
            description,
            syncEnabled,
            syncIgnoreDefaults,
            syncIgnorePermissions,
            syncIgnorePatterns,
            syncIgnoreFile,
            syncNodes,
        } = this;

        if (!project || !name.trim() || !path.trim()) {
            return;
        }

        const parsePatterns = (value: string): string[] =>
            value
                .split(/[\n,]/g)
                .map((s) => s.trim())
                .filter((s) => s.length > 0);

        const normalizedNodes: IProjectSyncNodeConfig[] = syncNodes
            .map((n) => ({nodeId: n.nodeId.trim(), targetPath: n.targetPath.trim()}))
            .filter((n) => n.nodeId && n.targetPath);

        const sync: IProjectSyncConfig = {
            enabled: syncEnabled,
            ignoreDefaults: syncIgnoreDefaults,
            ignorePermissions: syncIgnorePermissions,
            ignorePatterns: parsePatterns(syncIgnorePatterns),
            ignoreFile: syncIgnoreFile.trim() || undefined,
            nodes: normalizedNodes.length ? normalizedNodes : [],
        };

        this.submitting = true;
        try {
            await onSubmit(project.name, name.trim(), path.trim(), description.trim(), sync);
            onClose();
        } catch (error) {
            console.error('编辑项目失败:', error);
        } finally {
            this.submitting = false;
        }
    };
}

interface EditDialogContentProps {
    open: boolean;
    project: IVersion;
    availableNodes: ISyncNode[];
    onClose: () => void;
    name: string;
    path: string;
    description: string;
    syncEnabled: boolean;
    syncIgnoreDefaults: boolean;
    syncIgnorePermissions: boolean;
    syncIgnorePatterns: string;
    syncIgnoreFile: string;
    syncNodes: Array<{nodeId: string; targetPath: string}>;
    submitting: boolean;
    onNameChange: (value: string) => void;
    onPathChange: (value: string) => void;
    onDescriptionChange: (value: string) => void;
    onSyncEnabledChange: (value: boolean) => void;
    onSyncIgnoreDefaultsChange: (value: boolean) => void;
    onSyncIgnorePermissionsChange: (value: boolean) => void;
    onSyncIgnorePatternsChange: (value: string) => void;
    onSyncIgnoreFileChange: (value: string) => void;
    onSyncNodesChange: (value: Array<{nodeId: string; targetPath: string}>) => void;
    onSubmit: () => void;
}

const EditProjectDialogContent: React.FC<EditDialogContentProps> = ({
    open,
    project,
    availableNodes,
    onClose,
    name,
    path,
    description,
    syncEnabled,
    syncIgnoreDefaults,
    syncIgnorePermissions,
    syncIgnorePatterns,
    syncIgnoreFile,
    syncNodes,
    submitting,
    onNameChange,
    onPathChange,
    onDescriptionChange,
    onSyncEnabledChange,
    onSyncIgnoreDefaultsChange,
    onSyncIgnorePermissionsChange,
    onSyncIgnorePatternsChange,
    onSyncIgnoreFileChange,
    onSyncNodesChange,
    onSubmit,
}) => {
    const {t} = useTranslation();

    const addSyncNodeRow = () => {
        const defaultNodeId = availableNodes.length ? String(availableNodes[0].id) : '';
        onSyncNodesChange([...syncNodes, {nodeId: defaultNodeId, targetPath: ''}]);
    };

    const updateNodeRow = (idx: number, patch: Partial<{nodeId: string; targetPath: string}>) => {
        onSyncNodesChange(
            syncNodes.map((row, i) => (i === idx ? {...row, ...patch} : row))
        );
    };

    const removeNodeRow = (idx: number) => {
        onSyncNodesChange(syncNodes.filter((_, i) => i !== idx));
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {t('version.editProject')} - {project.name}
            </DialogTitle>
            <DialogContent>
                <Grid container spacing={2}>
                    <Grid size={12}>
                        <TextField
                            autoFocus
                            margin="dense"
                            label={t('version.projectName')}
                            type="text"
                            fullWidth
                            variant="outlined"
                            value={name}
                            onChange={(e) => onNameChange(e.target.value)}
                            disabled={submitting}
                            required
                            helperText={t('version.projectNamePlaceholder')}
                        />
                    </Grid>
                    <Grid size={12}>
                        <TextField
                            margin="dense"
                            label={t('version.projectPath')}
                            type="text"
                            fullWidth
                            variant="outlined"
                            value={path}
                            onChange={(e) => onPathChange(e.target.value)}
                            disabled={submitting}
                            required
                            helperText={t('version.projectPathPlaceholder')}
                            placeholder="/www/wwwroot/my-project"
                        />
                    </Grid>
                    <Grid size={12}>
                        <TextField
                            margin="dense"
                            label={t('version.projectDescription')}
                            type="text"
                            fullWidth
                            variant="outlined"
                            multiline
                            rows={3}
                            value={description}
                            onChange={(e) => onDescriptionChange(e.target.value)}
                            disabled={submitting}
                            helperText={t('version.projectDescriptionPlaceholder')}
                            placeholder={t('version.projectDescriptionPlaceholder')}
                        />
                    </Grid>

                    <Grid size={12}>
                        <Divider sx={{my: 1}} />
                        <Stack direction="row" alignItems="center" justifyContent="space-between">
                            <Typography variant="subtitle1">同步</Typography>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={syncEnabled}
                                        onChange={(_, checked) => onSyncEnabledChange(checked)}
                                    />
                                }
                                label="启用"
                            />
                        </Stack>
                    </Grid>

                    {syncEnabled ? (
                        <>
                            <Grid size={12}>
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={syncIgnoreDefaults}
                                            onChange={(_, checked) => onSyncIgnoreDefaultsChange(checked)}
                                        />
                                    }
                                    label="启用默认忽略列表 (.git、runtime、tmp)"
                                />
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={syncIgnorePermissions}
                                            onChange={(_, checked) =>
                                                onSyncIgnorePermissionsChange(checked)
                                            }
                                        />
                                    }
                                    label="忽略权限变更 (chmod/chown)"
                                />
                            </Grid>

                            <Grid size={12}>
                                <TextField
                                    margin="dense"
                                    label="忽略模式（每行一个）"
                                    value={syncIgnorePatterns}
                                    onChange={(e) => onSyncIgnorePatternsChange(e.target.value)}
                                    fullWidth
                                    multiline
                                    minRows={3}
                                    placeholder=".env\nnode_modules/**\n*.log"
                                    helperText="用于过滤同步的文件/目录（glob）"
                                />
                            </Grid>

                            <Grid size={12}>
                                <TextField
                                    margin="dense"
                                    label="忽略文件路径"
                                    value={syncIgnoreFile}
                                    onChange={(e) => onSyncIgnoreFileChange(e.target.value)}
                                    fullWidth
                                    placeholder="sync.ignore"
                                />
                            </Grid>

                            <Grid size={12}>
                                <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{mt: 1}}>
                                    <Typography variant="subtitle2">同步节点</Typography>
                                    <Button
                                        size="small"
                                        startIcon={<AddIcon />}
                                        onClick={addSyncNodeRow}
                                        disabled={availableNodes.length === 0}>
                                        添加节点
                                    </Button>
                                </Stack>
                                {availableNodes.length === 0 ? (
                                    <Typography variant="caption" color="textSecondary">
                                        还没有可用节点，请先在“节点管理”里创建节点。
                                    </Typography>
                                ) : null}
                                <Box sx={{mt: 1}}>
                                    {syncNodes.map((row, idx) => (
                                        <Box
                                            key={`${row.nodeId}-${idx}`}
                                            sx={{display: 'flex', gap: 1, alignItems: 'center', mb: 1}}>
                                            <TextField
                                                select
                                                label="节点"
                                                value={row.nodeId}
                                                onChange={(e) => updateNodeRow(idx, {nodeId: e.target.value})}
                                                size="small"
                                                sx={{minWidth: 160}}>
                                                {availableNodes.map((n) => (
                                                    <MenuItem key={n.id} value={String(n.id)}>
                                                        {n.name}
                                                    </MenuItem>
                                                ))}
                                            </TextField>
                                            <TextField
                                                label="目标目录"
                                                value={row.targetPath}
                                                onChange={(e) => updateNodeRow(idx, {targetPath: e.target.value})}
                                                size="small"
                                                fullWidth
                                                placeholder="/www/wwwroot/app"
                                            />
                                            <IconButton
                                                size="small"
                                                onClick={() => removeNodeRow(idx)}
                                                title="移除">
                                                <DeleteIcon fontSize="small" />
                                            </IconButton>
                                        </Box>
                                    ))}
                                </Box>
                            </Grid>
                        </>
                    ) : null}
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button
                    onClick={onClose}
                    disabled={submitting}
                    variant="contained"
                    color="secondary">
                    {t('common.cancel')}
                </Button>
                <Button
                    onClick={onSubmit}
                    color="primary"
                    variant="contained"
                    disabled={submitting || !name.trim() || !path.trim()}
                    startIcon={submitting ? <CircularProgress size={16} /> : undefined}>
                    {submitting ? t('version.editingProject') : t('common.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
};
