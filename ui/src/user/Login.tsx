import Button from '@material-ui/core/Button';
import Container from '@material-ui/core/Container';
import Grid from '@material-ui/core/Grid';
import TextField from '@material-ui/core/TextField';
import React, {Component} from 'react';
import {observable} from 'mobx';
import {observer} from 'mobx-react';
import {inject, Stores} from '../inject';
import RegistrationDialog from './Register';
import DefaultPage from '../common/DefaultPage';
import * as config from '../config';
import useTranslation from '../i18n/useTranslation';

@observer
class Login extends Component<Stores<'currentUser'>> {
    @observable
    private username = '';
    @observable
    private password = '';
    @observable
    private registerDialog = false;

    public render() {
        const {username, password, registerDialog} = this;
        return (
            <DefaultPage 
                title="Login" 
                rightControl={this.registerButton()} 
                maxWidth={250}>
                <Grid item xs={12} style={{textAlign: 'center'}}>
                    <Container>
                        <LoginForm
                            username={username}
                            password={password}
                            onUsernameChange={(value) => (this.username = value)}
                            onPasswordChange={(value) => (this.password = value)}
                            onLogin={this.login}
                            disabled={!!this.props.currentUser.connectionErrorMessage}
                        />
                    </Container>
                </Grid>
                {registerDialog && (
                    <RegistrationDialog
                        fClose={() => (this.registerDialog = false)}
                        fOnSubmit={this.props.currentUser.register}
                    />
                )}
            </DefaultPage>
        );
    }

    private registerButton = () => {
        if (config.get('register')) {
            return (
                <RegisterButtonWithTranslation 
                    onClick={() => (this.registerDialog = true)}
                />
            );
        }
        return null;
    };

    private preventDefault = (e: React.FormEvent<HTMLFormElement>) => e.preventDefault();

    private login = async () => {
        await this.props.currentUser.login(this.username, this.password);
    };
}

// 注册按钮组件，使用翻译
const RegisterButtonWithTranslation: React.FC<{
    onClick: () => void;
}> = ({ onClick }) => {
    const { t } = useTranslation();
    
    return (
        <Button
            id="register"
            variant="contained"
            color="primary"
            onClick={onClick}>
            {t('auth.register')}
        </Button>
    );
};

// 分离表单组件以使用Hook
const LoginForm: React.FC<{
    username: string;
    password: string;
    onUsernameChange: (value: string) => void;
    onPasswordChange: (value: string) => void;
    onLogin: () => void;
    disabled: boolean;
}> = ({username, password, onUsernameChange, onPasswordChange, onLogin, disabled}) => {
    const { t } = useTranslation();

    const preventDefault = (e: React.FormEvent<HTMLFormElement>) => e.preventDefault();

    return (
        <form onSubmit={preventDefault} id="login-form">
            <TextField
                autoFocus
                className="name"
                label={t('auth.username')}
                margin="dense"
                fullWidth
                autoComplete="username"
                value={username}
                onChange={(e) => onUsernameChange(e.target.value)}
            />
            <TextField
                type="password"
                className="password"
                label={t('auth.password')}
                margin="normal"
                fullWidth
                autoComplete="current-password"
                value={password}
                onChange={(e) => onPasswordChange(e.target.value)}
            />
            <Button
                type="submit"
                variant="contained"
                size="large"
                className="login"
                color="primary"
                fullWidth
                disabled={disabled}
                style={{marginTop: 15, marginBottom: 5}}
                onClick={onLogin}>
                {t('auth.login')}
            </Button>
        </form>
    );
};

export default inject('currentUser')(Login);
