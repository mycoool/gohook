import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';
import React, {ChangeEvent, Component} from 'react';
import useTranslation from '../i18n/useTranslation';

interface IProps {
    fClose: VoidFunction;
    fOnSubmit: (oldPassword: string, newPassword: string) => void;
}

interface IState {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
}

// 使用函数组件包装类组件以支持Hook
const ChangePasswordDialogWrapper: React.FC<IProps> = (props) => {
    const {t} = useTranslation();
    return <ChangePasswordDialog {...props} t={t} />;
};

interface IPropsWithTranslation extends IProps {
    t: (key: string, params?: any) => string;
}

class ChangePasswordDialog extends Component<IPropsWithTranslation, IState> {
    public state: IState = {
        oldPassword: '',
        newPassword: '',
        confirmPassword: '',
    };

    public render() {
        const {fClose, fOnSubmit, t} = this.props;
        const {oldPassword, newPassword, confirmPassword} = this.state;

        const oldPasswordPresent = oldPassword.length > 0;
        const newPasswordPresent = newPassword.length > 0;
        const passwordsMatch = newPassword === confirmPassword;
        const canSubmit = oldPasswordPresent && newPasswordPresent && passwordsMatch;

        const submitAndClose = () => {
            fOnSubmit(oldPassword, newPassword);
            fClose();
        };

        return (
            <Dialog
                open={true}
                onClose={fClose}
                aria-labelledby="change-password-dialog-title"
                id="change-password-dialog">
                <DialogTitle id="change-password-dialog-title">
                    {t('user.changePasswordTitle')}
                </DialogTitle>
                <DialogContent>
                    <TextField
                        autoFocus
                        margin="dense"
                        className="old-password"
                        label={t('user.oldPasswordLabel')}
                        type="password"
                        value={oldPassword}
                        onChange={this.handleChange('oldPassword')}
                        fullWidth
                    />
                    <TextField
                        margin="dense"
                        className="new-password"
                        label={t('user.newPasswordLabel')}
                        type="password"
                        value={newPassword}
                        onChange={this.handleChange('newPassword')}
                        fullWidth
                    />
                    <TextField
                        margin="dense"
                        className="confirm-password"
                        label={t('user.confirmPasswordLabel')}
                        type="password"
                        value={confirmPassword}
                        onChange={this.handleChange('confirmPassword')}
                        fullWidth
                        error={
                            newPassword.length > 0 && confirmPassword.length > 0 && !passwordsMatch
                        }
                        helperText={
                            newPassword.length > 0 && confirmPassword.length > 0 && !passwordsMatch
                                ? t('user.passwordMismatch')
                                : ''
                        }
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={fClose} variant="contained" color="secondary">
                        {t('common.cancel')}
                    </Button>
                    <Button
                        className="save-password"
                        disabled={!canSubmit}
                        onClick={submitAndClose}
                        color="primary"
                        variant="contained">
                        {t('user.changePassword')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }

    private handleChange =
        (propertyName: keyof IState) => (event: ChangeEvent<HTMLInputElement>) => {
            this.setState({
                ...this.state,
                [propertyName]: event.target.value,
            });
        };
}

export default ChangePasswordDialogWrapper;
