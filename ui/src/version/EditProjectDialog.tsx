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
} from '@mui/material';
import {IVersion} from '../types';
import useTranslation from '../i18n/useTranslation';

interface IProps {
    open: boolean;
    project: IVersion | null;
    onClose: () => void;
    onSubmit: (
        originalName: string,
        name: string,
        path: string,
        description: string
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
    private submitting = false;

    public componentDidUpdate(prevProps: IProps) {
        if (this.props.open && !prevProps.open && this.props.project) {
            // 对话框打开时，用当前项目数据填充表单
            this.name = this.props.project.name;
            this.path = this.props.project.path;
            this.description = this.props.project.description || '';
        }
    }

    public render() {
        const {open, project, onClose} = this.props;
        const {name, path, description, submitting} = this;

        if (!project) {
            return null;
        }

        return (
            <EditProjectDialogContent
                open={open}
                project={project}
                onClose={onClose}
                name={name}
                path={path}
                description={description}
                submitting={submitting}
                onNameChange={(value) => (this.name = value)}
                onPathChange={(value) => (this.path = value)}
                onDescriptionChange={(value) => (this.description = value)}
                onSubmit={this.handleSubmit}
            />
        );
    }

    private handleSubmit = async () => {
        const {project, onSubmit, onClose} = this.props;
        const {name, path, description} = this;

        if (!project || !name.trim() || !path.trim()) {
            return;
        }

        this.submitting = true;
        try {
            await onSubmit(project.name, name.trim(), path.trim(), description.trim());
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
    onClose: () => void;
    name: string;
    path: string;
    description: string;
    submitting: boolean;
    onNameChange: (value: string) => void;
    onPathChange: (value: string) => void;
    onDescriptionChange: (value: string) => void;
    onSubmit: () => void;
}

const EditProjectDialogContent: React.FC<EditDialogContentProps> = ({
    open,
    project,
    onClose,
    name,
    path,
    description,
    submitting,
    onNameChange,
    onPathChange,
    onDescriptionChange,
    onSubmit,
}) => {
    const {t} = useTranslation();

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
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={submitting}>
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
