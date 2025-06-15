import {createTheme, ThemeProvider, Theme, WithStyles, withStyles} from '@material-ui/core';
import CssBaseline from '@material-ui/core/CssBaseline';
import * as React from 'react';
import * as ReactRouter from 'react-router-dom';
import Header from './Header';
import LoadingSpinner from '../common/LoadingSpinner';
import Navigation from './Navigation';
import ScrollUpButton from '../common/ScrollUpButton';
import SettingsDialog from '../common/SettingsDialog';
import SnackBarHandler from '../snack/SnackBarHandler';
import * as config from '../config';
import Versions from '../version/Versions';
import Branches from '../version/Branches';
import Tags from '../version/Tags';
import Hooks from '../hook/Hooks';

import Plugins from '../plugin/Plugins';
import PluginDetailView from '../plugin/PluginDetailView';
import Login from '../user/Login';
import Messages from '../message/Messages';
import RealtimeMessages from '../message/RealtimeMessages';
import Users from '../user/Users';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {ConnectionErrorBanner} from '../common/ConnectionErrorBanner';

const { HashRouter, Route, Switch } = ReactRouter;

const styles = (theme: Theme) => ({
    content: {
        margin: '0 auto',
        marginTop: 64,
        padding: theme.spacing(4),
        width: '100%',
        [theme.breakpoints.down('xs')]: {
            marginTop: 0,
        },
    },
});

const localStorageThemeKey = 'gohook-theme';
type ThemeKey = 'dark' | 'light';
const themeMap: Record<ThemeKey, Theme> = {
    light: createTheme({
        palette: {
            type: 'light',
        },
    }),
    dark: createTheme({
        palette: {
            type: 'dark',
        },
    }),
};

const isThemeKey = (value: string | null): value is ThemeKey =>
    value === 'light' || value === 'dark';

// 自定义重定向组件
const CustomRedirect: React.FC<{to: string}> = ({to}) => {
    React.useEffect(() => {
        window.location.hash = to.startsWith('#') ? to : `#${to}`;
    }, [to]);
    return null;
};

@observer
class Layout extends React.Component<
    WithStyles<'content'> & Stores<'currentUser' | 'snackManager'>
> {
    @observable
    private currentTheme: ThemeKey = 'dark';
    @observable
    private showSettings = false;
    @observable
    private navOpen = false;

    private setNavOpen(open: boolean) {
        this.navOpen = open;
    }

    public componentDidMount() {
        const localStorageTheme = window.localStorage.getItem(localStorageThemeKey);
        if (isThemeKey(localStorageTheme)) {
            this.currentTheme = localStorageTheme;
        } else {
            window.localStorage.setItem(localStorageThemeKey, this.currentTheme);
        }
    }

    public render() {
        const {showSettings, currentTheme} = this;
        const {
            classes,
            currentUser: {
                loggedIn,
                authenticating,
                user: {name, admin, role},
                logout,
                tryReconnect,
                connectionErrorMessage,
            },
        } = this.props;
        const theme = themeMap[currentTheme];
        const loginRoute = () => (loggedIn ? <CustomRedirect to="/" /> : <Login />);
        const versionInfo = config.get('version');
        return (
            <ThemeProvider theme={theme}>
                {React.createElement(HashRouter as any, null,
                    <div>
                        {!connectionErrorMessage ? null : (
                            <ConnectionErrorBanner
                                height={64}
                                retry={() => tryReconnect()}
                                message={connectionErrorMessage}
                            />
                        )}
                        <div style={{display: 'flex', flexDirection: 'column'}}>
                            <CssBaseline />
                            <Header
                                style={{top: !connectionErrorMessage ? 0 : 64}}
                                admin={admin}
                                name={name}
                                version={versionInfo.version}
                                loggedIn={loggedIn}
                                toggleTheme={this.toggleTheme.bind(this)}
                                showSettings={() => (this.showSettings = true)}
                                logout={logout}
                                setNavOpen={this.setNavOpen.bind(this)}
                            />
                            <div style={{display: 'flex'}}>
                                <Navigation
                                    loggedIn={loggedIn}
                                    navOpen={this.navOpen}
                                    setNavOpen={this.setNavOpen.bind(this)}
                                    user={{admin, role}}
                                />
                                <main className={classes.content}>
                                    {React.createElement(Switch as any, null,
                                        authenticating ? React.createElement(Route as any, { path: "/" }, React.createElement(LoadingSpinner)) : null,
                                        React.createElement(Route as any, { exact: true, path: "/login", render: loginRoute }),
                                        loggedIn ? null : React.createElement(CustomRedirect, { to: "/login" }),
                                        React.createElement(Route as any, { exact: true, path: "/", component: Messages }),
                                        React.createElement(Route as any, { exact: true, path: "/messages/:id", component: Messages }),
                                        React.createElement(Route as any, { exact: true, path: "/versions", component: Versions }),
                                        React.createElement(Route as any, { exact: true, path: "/versions/:projectName/branches", component: Branches }),
                                        React.createElement(Route as any, { exact: true, path: "/versions/:projectName/tags", component: Tags }),
                                        React.createElement(Route as any, { exact: true, path: "/hooks", component: Hooks }),
                                        React.createElement(Route as any, { exact: true, path: "/users", component: Users }),
                                        React.createElement(Route as any, { exact: true, path: "/plugins", component: Plugins }),
                                        React.createElement(Route as any, { exact: true, path: "/plugins/:id", component: PluginDetailView })
                                    )}
                                </main>
                            </div>
                            {showSettings && (
                                <SettingsDialog fClose={() => (this.showSettings = false)} />
                            )}
                            <ScrollUpButton />
                            <SnackBarHandler />
                            {loggedIn && <RealtimeMessages />}
                        </div>
                    </div>
                )}
            </ThemeProvider>
        );
    }

    private toggleTheme() {
        this.currentTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
        localStorage.setItem(localStorageThemeKey, this.currentTheme);
    }
}

export default withStyles(styles, {withTheme: true})(inject('currentUser', 'snackManager')(Layout));
