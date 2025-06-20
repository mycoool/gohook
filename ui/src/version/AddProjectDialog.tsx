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

interface IProps {
    open: boolean;
    onClose: () => void;
    onSubmit: (name: string, path: string, description: string) => Promise<void>;
}

@observer
export default class AddProjectDialog extends Component<IProps> {
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
        const {open, onClose} = this.props;
        const {name, path, description, submitting} = this;

        return (
            <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
                <DialogTitle>添加新项目</DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={12}>
                            <TextField
                                autoFocus
                                margin="dense"
                                label="项目名称"
                                type="text"
                                fullWidth
                                variant="outlined"
                                value={name}
                                onChange={(e) => (this.name = e.target.value)}
                                disabled={submitting}
                                required
                                helperText="项目的唯一标识符"
                            />
                        </Grid>
                        <Grid size={12}>
                            <TextField
                                margin="dense"
                                label="项目路径"
                                type="text"
                                fullWidth
                                variant="outlined"
                                value={path}
                                onChange={(e) => (this.path = e.target.value)}
                                disabled={submitting}
                                required
                                helperText="项目在服务器上的绝对路径"
                                placeholder="/www/wwwroot/my-project"
                            />
                        </Grid>
                        <Grid size={12}>
                            <TextField
                                margin="dense"
                                label="项目描述"
                                type="text"
                                fullWidth
                                variant="outlined"
                                multiline
                                rows={3}
                                value={description}
                                onChange={(e) => (this.description = e.target.value)}
                                disabled={submitting}
                                helperText="项目的详细描述（可选）"
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Button onClick={onClose} disabled={submitting}>
                        取消
                    </Button>
                    <Button
                        onClick={this.handleSubmit}
                        color="primary"
                        variant="contained"
                        disabled={submitting || !this.isFormValid()}>
                        {submitting ? '添加中...' : '添加项目'}
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
