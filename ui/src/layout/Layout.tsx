import { createTheme, ThemeProvider, StyledEngineProvider, Theme, styled } from '@mui/material/styles';
import { ThemeProvider as StylesThemeProvider } from '@mui/styles';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
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
import EnvFileDialog from '../version/EnvFileDialog';
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





const {HashRouter, Route, Switch} = ReactRouter;

const MainContent = styled('main')(({ theme }) => ({
    flexGrow: 1,
    marginTop: 12,
    padding: '32px', // 减少内边距从32px到16px
    marginLeft: 250, // 为固定导航栏留出空间
    [theme.breakpoints.down('sm')]: {
        marginTop: 0,
        marginLeft: 0, // 小屏幕下不需要左边距
        padding: '8px', // 小屏幕下进一步减少内边距
    },
}));

const localStorageThemeKey = 'gohook-theme';
type ThemeKey = 'dark' | 'light';
const themeMap: Record<ThemeKey, Theme> = {
    light: createTheme({
        palette: {
            mode: 'light',
        },
    }),
    dark: createTheme({
        palette: {
            mode: 'dark',
            primary: {
                main: '#2196f3', // 蓝色主题
            },
            background: {
                default: '#0d1117', // GitHub深色主题背景
                paper: '#161b22',
            },
            text: {
                primary: '#f0f6fc',
                secondary: '#8b949e',
            },
        },
        components: {
            MuiAppBar: {
                styleOverrides: {
                    root: ({ theme }) => ({
                        backgroundColor: '#ffffff',
                        borderBottom: '1px solid #e0e0e0',
                        boxShadow: '0 1px 0 rgba(0, 0, 0, 0.1)',
                        ...theme.applyStyles('dark', {
                            backgroundColor: '#161b22',
                            borderBottom: '1px solid #21262d',
                            boxShadow: '0 1px 0 rgba(33, 38, 45, 1)',
                        }),
                    }),
                },
            },
            MuiDrawer: {
                styleOverrides: {
                    paper: ({ theme }) => ({
                        backgroundColor: '#ffffff',
                        borderRight: '1px solid #e0e0e0',
                        ...theme.applyStyles('dark', {
                            backgroundColor: '#0d1117',
                            borderRight: '1px solid #21262d',
                        }),
                    }),
                },
            },
            MuiButton: {
                styleOverrides: {
                    root: ({ theme }) => ({
                        borderRadius: '8px',
                        textTransform: 'none',
                        fontWeight: 500,
                        ...theme.applyStyles('dark', {
                            '&.MuiButton-containedPrimary': {
                                backgroundColor: '#238be6',
                                color: '#ffffff',
                                '&:hover': {
                                    backgroundColor: '#1976d2',
                                },
                            },
                            '&.MuiButton-containedSecondary': {
                                backgroundColor: '#6c757d',
                                color: '#ffffff',
                                '&:hover': {
                                    backgroundColor: '#5a6268',
                                },
                            },
                            '&.MuiButton-outlined': {
                                borderColor: '#30363d',
                                color: '#f0f6fc',
                                '&:hover': {
                                    borderColor: '#58a6ff',
                                    backgroundColor: 'rgba(88, 166, 255, 0.1)',
                                },
                            },
                        }),
                    }),
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: ({ theme }) => ({
                        borderRadius: '6px',
                        ...theme.applyStyles('dark', {
                            color: '#f0f6fc',
                            '&:hover': {
                                backgroundColor: 'rgba(240, 246, 252, 0.1)',
                            },
                        }),
                    }),
                },
            },
            // 优化代码块显示
            MuiPaper: {
                styleOverrides: {
                    root: ({ theme }) => ({
                        ...theme.applyStyles('dark', {
                            backgroundColor: '#161b22',
                            '& code': {
                                backgroundColor: '#21262d',
                                color: '#e6edf3',
                                padding: '2px 6px',
                                borderRadius: '4px',
                                fontSize: '0.875rem',
                                fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                            },
                        }),
                    }),
                },
            },
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
    Stores<'currentUser' | 'snackManager'>
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
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={theme}>
                    <StylesThemeProvider theme={theme}>
                        {React.createElement(
                            HashRouter as any,
                            null,
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
                                    <MainContent>
                                        {React.createElement(
                                            Switch as any,
                                            null,
                                            authenticating
                                                ? React.createElement(
                                                      Route as any,
                                                      {path: '/'},
                                                      React.createElement(LoadingSpinner)
                                                  )
                                                : null,
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/login',
                                                render: loginRoute,
                                            }),
                                            loggedIn
                                                ? null
                                                : React.createElement(CustomRedirect, {to: '/login'}),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/',
                                                component: Messages,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/messages/:id',
                                                component: Messages,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/versions',
                                                component: Versions,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/versions/:projectName/branches',
                                                component: Branches,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/versions/:projectName/tags',
                                                component: Tags,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/versions/:projectName/env',
                                                component: EnvFileDialog,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/hooks',
                                                component: Hooks,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/users',
                                                component: Users,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/plugins',
                                                component: Plugins,
                                            }),
                                            React.createElement(Route as any, {
                                                exact: true,
                                                path: '/plugins/:id',
                                                component: PluginDetailView,
                                            })
                                        )}
                                    </MainContent>
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
                    </StylesThemeProvider>
                </ThemeProvider>
            </StyledEngineProvider>
        );
    }

    private toggleTheme() {
        this.currentTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
        localStorage.setItem(localStorageThemeKey, this.currentTheme);
    }
}

export default inject('currentUser', 'snackManager')(Layout);
