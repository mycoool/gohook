import React, {Component} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    Grid,
} from '@mui/material';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import useTranslation from '../i18n/useTranslation';

interface IProps {
    open: boolean;
    onClose: () => void;
    onSubmit: (name: string, path: string, description: string) => Promise<void>;
}

interface IPropsWithTranslation extends IProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

const AddProjectDialogWithTranslation: React.FC<IProps> = (props) => {
    const {t} = useTranslation();
    return <AddProjectDialog {...props} t={t} />;
};

@observer
class AddProjectDialog extends Component<IPropsWithTranslation> {
    @observable
    private name = '';
    @observable
    private path = '';
    @observable
    private description = '';
    @observable
    private submitting = false;

    public componentDidUpdate(prevProps: IProps) {
        if (this.props.open && !prevProps.open) {
            // 对话框打开时重置表单
            this.name = '';
            this.path = '';
            this.description = '';
        }
    }

    public render() {
        const {open, onClose, t} = this.props;
        const {name, path, description, submitting} = this;

        return (
            <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
                <DialogTitle>{t('version.addProjectDialogTitle')}</DialogTitle>
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
                                onChange={(e) => (this.name = e.target.value)}
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
                                onChange={(e) => (this.path = e.target.value)}
                                disabled={submitting}
                                required
                                helperText={t('version.projectPathPlaceholder')}
                                placeholder={t('version.projectPathExample')}
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
                                onChange={(e) => (this.description = e.target.value)}
                                disabled={submitting}
                                helperText={t('version.projectDescriptionPlaceholder')}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Button
                        variant="contained"
                        color="secondary"
                        onClick={onClose}
                        disabled={submitting}>
                        {t('common.cancel')}
                    </Button>
                    <Button
                        onClick={this.handleSubmit}
                        color="primary"
                        variant="contained"
                        disabled={submitting || !this.isFormValid()}>
                        {submitting ? t('version.addingProject') : t('version.addProject')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }

    private isFormValid = (): boolean => this.name.trim() !== '' && this.path.trim() !== '';

    private handleSubmit = async () => {
        if (!this.isFormValid()) return;

        this.submitting = true;
        try {
            await this.props.onSubmit(this.name.trim(), this.path.trim(), this.description.trim());
            this.props.onClose();
        } catch (error) {
            // 错误处理已在Store中完成
        } finally {
            this.submitting = false;
        }
    };
}

export default AddProjectDialogWithTranslation;
