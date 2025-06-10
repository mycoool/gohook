import Button from '@material-ui/core/Button';
import Container from '@material-ui/core/Container';
import Grid from '@material-ui/core/Grid';
import TextField from '@material-ui/core/TextField';
import Paper from '@material-ui/core/Paper';
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

    public render() {
        const {username, password} = this;
        return (
            <LoginContainer
                username={username}
                password={password}
                onUsernameChange={(value: string) => (this.username = value)}
                onPasswordChange={(value: string) => (this.password = value)}
                onLogin={this.login}
                disabled={!!this.props.currentUser.connectionErrorMessage}
            />
        );
    }


    private login = async () => {
        await this.props.currentUser.login(this.username, this.password);
    };
}

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

// 登录容器组件，使用Hook
const LoginContainer: React.FC<{
    username: string;
    password: string;
    onUsernameChange: (value: string) => void;
    onPasswordChange: (value: string) => void;
    onLogin: () => void;
    disabled: boolean;
}> = ({
    username,
    password,
    onUsernameChange,
    onPasswordChange,
    onLogin,
    disabled,
}) => {
    const { t } = useTranslation();

    return (
        <DefaultPage 
            title={t('auth.login')} 
            maxWidth={340}>
            <Grid item xs={12} style={{textAlign: 'center'}}>
                <Container>
                    <Paper style={{padding: '30px 20px', marginTop: 30}}>
                        <LoginForm
                            username={username}
                            password={password}
                            onUsernameChange={onUsernameChange}
                            onPasswordChange={onPasswordChange}
                            onLogin={onLogin}
                            disabled={disabled}
                        />
                    </Paper>
                </Container>
            </Grid>
        </DefaultPage>
    );
};

export default inject('currentUser')(Login);
