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
    marginTop: 64,
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
            primary: {
                main: '#3f51b5', // 深紫蓝色，与按钮保持一致
            },
            background: {
                default: '#fafafa',
                paper: '#ffffff',
            },
            text: {
                primary: '#212121',
                secondary: '#757575',
            },
        },
        components: {
            MuiAppBar: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#3f51b5 !important', // 强制设置深蓝色背景，与按钮一致
                        color: '#ffffff',
                        borderBottom: '1px solid #e0e0e0',
                        boxShadow: '0 2px 4px rgba(0, 0, 0, 0.1)',
                        // 导航文字样式 - 英文全大写加粗
                        '& .MuiButton-text': {
                            textTransform: 'uppercase',
                            fontWeight: 'bold',
                            letterSpacing: '0.5px',
                        },
                    },
                },
            },
            MuiDrawer: {
                styleOverrides: {
                    paper: {
                        backgroundColor: '#ffffff',
                        borderRight: '1px solid #e0e0e0',
                    },
                },
            },
            MuiButton: {
                styleOverrides: {
                    root: {
                        borderRadius: '6px',
                        textTransform: 'none',
                        fontWeight: 500,
                        '&.MuiButton-containedPrimary': {
                            backgroundColor: '#3f51b5',
                            color: '#ffffff',
                            '&:hover': {
                                backgroundColor: '#283593',
                            },
                        },
                        '&.MuiButton-containedSecondary': {
                            backgroundColor: '#757575',
                            color: '#ffffff',
                            '&:hover': {
                                backgroundColor: '#616161',
                            },
                        },
                        '&.MuiButton-outlined': {
                            borderColor: '#e0e0e0',
                            color: '#3f51b5',
                            '&:hover': {
                                borderColor: '#3f51b5',
                                backgroundColor: 'rgba(48, 63, 159, 0.04)',
                            },
                        },
                        // 次要按钮样式 - 用于对话框等
                        '&.MuiButton-textSecondary': {
                            backgroundColor: '#f5f5f5',
                            color: '#666666',
                            border: '1px solid #d0d0d0',
                            '&:hover': {
                                backgroundColor: '#eeeeee',
                                borderColor: '#bdbdbd',
                            },
                        },
                        // 导航按钮样式
                        '&.MuiButton-text': {
                            color: '#ffffff', // 确保导航栏文字为白色
                            '&:hover': {
                                backgroundColor: 'rgba(255, 255, 255, 0.1)',
                            },
                        },
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        borderRadius: '6px',
                        color: '#616161',
                        '&:hover': {
                            backgroundColor: 'rgba(0, 0, 0, 0.04)',
                        },
                        // AppBar中的图标按钮样式
                        '.MuiAppBar-root &': {
                            color: '#ffffff',
                            '&:hover': {
                                backgroundColor: 'rgba(255, 255, 255, 0.1)',
                            },
                        },
                    },
                },
            },
            // 优化代码块显示
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#ffffff',
                        '& code': {
                            backgroundColor: '#f5f5f5',
                            color: '#333333',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.875rem',
                            fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                            border: '1px solid #e0e0e0',
                        },
                    },
                },
            },
            // 表格样式优化
            MuiTable: {
                styleOverrides: {
                    root: {
                        '& .MuiTableHead-root': {
                            '& .MuiTableCell-root': {
                                backgroundColor: '#787878',
                                fontWeight: 600,
                                color: '#ffffff',
                                borderBottom: 'none',
                                padding: '16px',
                            },
                        },
                        '& .MuiTableBody-root': {
                            '& .MuiTableCell-root': {
                                borderBottom: '1px solid #e0e0e0',
                                color: '#333333',
                                padding: '16px',
                            },
                            '& .MuiTableRow-root': {
                                backgroundColor: '#f5f5f5',
                                '&:hover': {
                                    backgroundColor: '#eeeeee',
                                },
                            },
                        },
                    },
                },
            },
            // 按钮组高度优化
            MuiButtonGroup: {
                styleOverrides: {
                    root: {
                        '& .MuiButton-root': {
                            minHeight: '40px',
                            padding: '10px 20px',
                            fontSize: '0.875rem',
                            fontWeight: 500,
                        },
                    },
                },
            },
        },
    }),
    dark: createTheme({
        palette: {
            mode: 'dark',
            primary: {
                main: '#3f51b5', // 使用与AppBar一致的深蓝色
            },
            background: {
                default: '#303030', // 更温和的深灰色背景
                paper: '#424242', // 卡片背景色
            },
            text: {
                primary: '#ffffff',
                secondary: '#b0bec5',
            },
        },
        components: {
            MuiAppBar: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#3f51b5 !important', // 强制设置，与深色主题按钮保持一致
                        borderBottom: '1px solid #616161',
                        boxShadow: '0 2px 4px rgba(0, 0, 0, 0.3)',
                        // 导航文字样式 - 英文全大写加粗
                        '& .MuiButton-text': {
                            textTransform: 'uppercase',
                            fontWeight: 'bold',
                            letterSpacing: '0.5px',
                        },
                    },
                },
            },
            MuiDrawer: {
                styleOverrides: {
                    paper: {
                        backgroundColor: '#424242', // 左侧导航背景色
                        borderRight: '1px solid #616161',
                    },
                },
            },
            MuiButton: {
                styleOverrides: {
                    root: {
                        borderRadius: '6px',
                        textTransform: 'none',
                        fontWeight: 500,
                        '&.MuiButton-containedPrimary': {
                            backgroundColor: '#3f51b5',
                            color: '#ffffff',
                            '&:hover': {
                                backgroundColor: '#283593',
                            },
                        },
                        '&.MuiButton-containedSecondary': {
                            backgroundColor: '#757575',
                            color: '#ffffff',
                            '&:hover': {
                                backgroundColor: '#616161',
                            },
                        },
                        '&.MuiButton-outlined': {
                            borderColor: '#616161',
                            color: '#ffffff',
                            '&:hover': {
                                borderColor: '#3f51b5',
                                backgroundColor: 'rgba(63, 81, 181, 0.1)',
                            },
                        },
                        // 次要按钮样式 - 用于对话框等 (深色主题)
                        '&.MuiButton-textSecondary': {
                            backgroundColor: '#424242',
                            color: '#b0bec5',
                            border: '1px solid #616161',
                            '&:hover': {
                                backgroundColor: '#555555',
                                borderColor: '#757575',
                                color: '#ffffff',
                            },
                        },
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        borderRadius: '6px',
                        color: '#ffffff',
                        '&:hover': {
                            backgroundColor: 'rgba(255, 255, 255, 0.1)',
                        },
                    },
                },
            },
            // 优化代码块显示
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#424242', // 卡片背景色
                        '& code': {
                            backgroundColor: '#616161',
                            color: '#ffffff',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.875rem',
                            fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                            border: '1px solid #757575',
                        },
                    },
                },
            },
            // 表格样式优化 - 深色主题
            MuiTable: {
                styleOverrides: {
                    root: {
                        '& .MuiTableHead-root': {
                            '& .MuiTableCell-root': {
                                backgroundColor: '#383838',
                                fontWeight: 600,
                                color: '#ffffff',
                                borderBottom: 'none',
                                padding: '16px',
                            },
                        },
                        '& .MuiTableBody-root': {
                            '& .MuiTableCell-root': {
                                borderBottom: '1px solid #555555',
                                color: '#ffffff',
                                padding: '16px',
                            },
                            '& .MuiTableRow-root': {
                                backgroundColor: '#424242',
                                '&:hover': {
                                    backgroundColor: '#484848',
                                },
                            },
                        },
                    },
                },
            },
            // 按钮组高度优化 - 深色主题
            MuiButtonGroup: {
                styleOverrides: {
                    root: {
                        '& .MuiButton-root': {
                            minHeight: '40px',
                            padding: '10px 20px',
                            fontSize: '0.875rem',
                            fontWeight: 500,
                        },
                    },
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
