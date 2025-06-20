import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Switch from '@material-ui/core/Switch';
import TextField from '@material-ui/core/TextField';
import Tooltip from '@material-ui/core/Tooltip';
import React, {ChangeEvent, Component} from 'react';
import useTranslation from '../i18n/useTranslation';

interface IProps {
    name?: string;
    admin?: boolean;
    fClose: VoidFunction;
    fOnSubmit: (name: string, pass: string, admin: boolean) => void;
    isEdit?: boolean;
}

interface IState {
    name: string;
    pass: string;
    admin: boolean;
}

// 使用函数组件包装类组件以支持Hook
const AddEditDialogWrapper: React.FC<IProps> = (props) => {
    const {t} = useTranslation();
    return <AddEditDialog {...props} t={t} />;
};

interface IPropsWithTranslation extends IProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

class AddEditDialog extends Component<IPropsWithTranslation, IState> {
    public state = {
        name: this.props.name ?? '',
        pass: '',
        admin: this.props.admin ?? false,
    };

    public render() {
        const {fClose, fOnSubmit, isEdit, t} = this.props;
        const {name, pass, admin} = this.state;
        const namePresent = this.state.name.length !== 0;
        const passPresent = this.state.pass.length !== 0 || isEdit;
        const submitAndClose = () => {
            fOnSubmit(name, pass, admin);
            fClose();
        };
        return (
            <Dialog
                open={true}
                onClose={fClose}
                aria-labelledby="form-dialog-title"
                id="add-edit-user-dialog">
                <DialogTitle id="form-dialog-title">
                    {isEdit
                        ? t('user.editUserTitle', {name: this.props.name ?? ''})
                        : t('user.addUserTitle')}
                </DialogTitle>
                <DialogContent>
                    <TextField
                        autoFocus
                        margin="dense"
                        className="name"
                        label={t('user.nameLabel')}
                        type="text"
                        value={name}
                        onChange={this.handleChange('name')}
                        fullWidth
                    />
                    <TextField
                        margin="dense"
                        className="password"
                        type="password"
                        value={pass}
                        fullWidth
                        label={isEdit ? t('user.passwordEditLabel') : t('user.passwordLabel')}
                        onChange={this.handleChange('pass')}
                    />
                    <FormControlLabel
                        control={
                            <Switch
                                checked={admin}
                                className="admin-rights"
                                onChange={this.handleChecked('admin')}
                                value="admin"
                            />
                        }
                        label={t('user.adminRightsLabel')}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={fClose}>{t('common.cancel')}</Button>
                    <Tooltip
                        placement={'bottom-start'}
                        title={
                            namePresent
                                ? passPresent
                                    ? ''
                                    : t('user.passwordRequired')
                                : t('user.nameRequired')
                        }>
                        <div>
                            <Button
                                className="save-create"
                                disabled={!passPresent || !namePresent}
                                onClick={submitAndClose}
                                color="primary"
                                variant="contained">
                                {isEdit ? t('user.save') : t('user.create')}
                            </Button>
                        </div>
                    </Tooltip>
                </DialogActions>
            </Dialog>
        );
    }

    private handleChange =
        (propertyName: 'name' | 'pass') => (event: ChangeEvent<HTMLInputElement>) => {
            this.setState({
                ...this.state,
                [propertyName]: event.target.value,
            });
        };

    private handleChecked = (propertyName: 'admin') => (event: ChangeEvent<HTMLInputElement>) => {
        this.setState({
            ...this.state,
            [propertyName]: event.target.checked,
        });
    };
}

export default AddEditDialogWrapper;
