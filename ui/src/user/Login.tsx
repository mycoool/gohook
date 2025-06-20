import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import Grid from '@mui/material/Grid';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';
import React, {Component} from 'react';
import {observable} from 'mobx';
import {observer} from 'mobx-react';
import {inject, Stores} from '../inject';
import DefaultPage from '../common/DefaultPage';
import useTranslation from '../i18n/useTranslation';

@observer
class Login extends Component<Stores<'currentUser'>> {
    @observable
    private username = '';
    @observable
    private password = '';
    @observable
    private isLogging = false;

    public render() {
        const {username, password, isLogging} = this;
        return (
            <LoginContainer
                username={username}
                password={password}
                onUsernameChange={(value: string) => (this.username = value)}
                onPasswordChange={(value: string) => (this.password = value)}
                onLogin={this.login}
                disabled={!!this.props.currentUser.connectionErrorMessage || isLogging}
                isLogging={isLogging}
            />
        );
    }

    private login = async (event?: React.FormEvent) => {
        if (event) {
            event.preventDefault();
        }

        if (this.isLogging) {
            return;
        }

        this.isLogging = true;

        try {
            await this.props.currentUser.login(this.username, this.password);
        } catch (error) {
            console.error('Login error:', error);
        } finally {
            this.isLogging = false;
        }
    };
}

// 分离表单组件以使用Hook
const LoginForm: React.FC<{
    username: string;
    password: string;
    onUsernameChange: (value: string) => void;
    onPasswordChange: (value: string) => void;
    onLogin: (event?: React.FormEvent) => void;
    disabled: boolean;
    isLogging: boolean;
}> = ({username, password, onUsernameChange, onPasswordChange, onLogin, disabled, isLogging}) => {
    const {t} = useTranslation();

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        onLogin(e);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !disabled) {
            e.preventDefault();
            onLogin();
        }
    };

    return (
        <form onSubmit={handleSubmit} id="login-form">
            <TextField
                autoFocus
                className="name"
                label={t('auth.username')}
                margin="dense"
                fullWidth
                autoComplete="username"
                value={username}
                onChange={(e) => onUsernameChange(e.target.value)}
                onKeyDown={handleKeyDown}
                disabled={disabled}
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
                onKeyDown={handleKeyDown}
                disabled={disabled}
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
                onClick={() => onLogin()}>
                {isLogging ? t('auth.logging_in') || 'Logging in...' : t('auth.login')}
            </Button>
        </form>
    );
};

// 登录容器组件，使用Hook
const LoginContainer: React.FC<{
    username: string;
    password: string;
    onUsernameChange: (value: string) => void;
    onPasswordChange: (value: string) => void;
    onLogin: (event?: React.FormEvent) => void;
    disabled: boolean;
    isLogging: boolean;
}> = ({username, password, onUsernameChange, onPasswordChange, onLogin, disabled, isLogging}) => {
    const {t} = useTranslation();

    return (
        <DefaultPage title={t('auth.login')} maxWidth={340} centerTitle={true}>
            <Grid size={12} style={{textAlign: 'center'}}>
                <Container>
                    <Paper style={{padding: '30px 20px', marginTop: 30}}>
                        <LoginForm
                            username={username}
                            password={password}
                            onUsernameChange={onUsernameChange}
                            onPasswordChange={onPasswordChange}
                            onLogin={onLogin}
                            disabled={disabled}
                            isLogging={isLogging}
                        />
                    </Paper>
                </Container>
            </Grid>
        </DefaultPage>
    );
};

export default inject('currentUser')(Login);
