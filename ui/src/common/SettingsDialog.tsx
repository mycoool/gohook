import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import React, {Component} from 'react';
import {observable} from 'mobx';
import {observer} from 'mobx-react';
import {inject, Stores} from '../inject';
import useTranslation from '../i18n/useTranslation';

interface IProps {
    fClose: VoidFunction;
}

// 使用函数组件包装类组件以支持Hook
const SettingsDialogWrapper: React.FC<IProps & Stores<'currentUser'>> = (props) => {
    const {t} = useTranslation();
    return <SettingsDialog {...props} t={t} />;
};

interface IPropsWithTranslation extends IProps, Stores<'currentUser'> {
    t: (key: string, params?: Record<string, string | number>) => string;
}

@observer
class SettingsDialog extends Component<IPropsWithTranslation> {
    @observable
    private oldPassword = '';
    @observable
    private newPassword = '';
    @observable
    private confirmPassword = '';

    public render() {
        const {oldPassword, newPassword, confirmPassword} = this;
        const {fClose, currentUser, t} = this.props;

        const oldPasswordPresent = oldPassword.length > 0;
        const newPasswordPresent = newPassword.length > 0;
        const passwordsMatch = newPassword === confirmPassword && confirmPassword.length > 0;
        const canSubmit = oldPasswordPresent && newPasswordPresent && passwordsMatch;

        const submitAndClose = () => {
            currentUser.changePassword(oldPassword, newPassword);
            fClose();
        };

        return (
            <Dialog
                open={true}
                onClose={fClose}
                aria-labelledby="form-dialog-title"
                id="changepw-dialog">
                <DialogTitle id="form-dialog-title">{t('user.changePasswordTitle')}</DialogTitle>
                <DialogContent>
                    <TextField
                        className="oldpass"
                        autoFocus
                        margin="dense"
                        type="password"
                        label={t('user.oldPasswordLabel')}
                        value={oldPassword}
                        onChange={(e) => (this.oldPassword = e.target.value)}
                        fullWidth
                    />
                    <TextField
                        className="newpass"
                        margin="dense"
                        type="password"
                        label={t('user.newPasswordLabel')}
                        value={newPassword}
                        onChange={(e) => (this.newPassword = e.target.value)}
                        fullWidth
                    />
                    <TextField
                        className="confirmpass"
                        margin="dense"
                        type="password"
                        label={t('user.confirmPasswordLabel')}
                        value={confirmPassword}
                        onChange={(e) => (this.confirmPassword = e.target.value)}
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
                    <Tooltip
                        title={
                            !oldPasswordPresent
                                ? t('user.passwordRequired')
                                : !newPasswordPresent
                                ? t('user.passwordRequired')
                                : !passwordsMatch
                                ? t('user.passwordMismatch')
                                : ''
                        }>
                        <div>
                            <Button
                                className="change"
                                disabled={!canSubmit}
                                onClick={submitAndClose}
                                color="primary"
                                variant="contained">
                                {t('user.changePassword')}
                            </Button>
                        </div>
                    </Tooltip>
                </DialogActions>
            </Dialog>
        );
    }
}

export default inject('currentUser')(SettingsDialogWrapper);
