import {
    createTheme,
    ThemeProvider,
    StyledEngineProvider,
    Theme,
    ThemeOptions,
    styled,
} from '@mui/material/styles';

// 扩展主题类型
declare module '@mui/material/styles' {
    interface Theme {
        custom: {
            colors: {
                primary: {
                    black: string;
                    darkGray: string;
                    mediumGray: string;
                    lightGray: string;
                };
                background: {
                    white: string;
                    lightGray: string;
                    mediumGray: string;
                    overlay: string;
                };
                border: {
                    light: string;
                    medium: string;
                    dark: string;
                    contrast: string;
                };
                text: {
                    primary: string;
                    secondary: string;
                    disabled: string;
                    onDark: string;
                    onDarkSecondary: string;
                };
                status: {
                    info: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    warning: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    error: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    success: {
                        background: string;
                        border: string;
                        text: string;
                    };
                };
                interactive: {
                    button: {
                        command: string;
                        script: string;
                        hover: string;
                        disabled: string;
                    };
                    input: {
                        background: string;
                        border: string;
                        focus: string;
                        text: string;
                    };
                    code: {
                        background: string;
                        text: string;
                        padding: string;
                        borderRadius: number;
                        fontSize: string;
                    };
                };
            };
        };
    }

    interface ThemeOptions {
        custom?: {
            colors?: {
                primary?: {
                    black?: string;
                    darkGray?: string;
                    mediumGray?: string;
                    lightGray?: string;
                };
                background?: {
                    white?: string;
                    lightGray?: string;
                    mediumGray?: string;
                    overlay?: string;
                };
                border?: {
                    light?: string;
                    medium?: string;
                    dark?: string;
                    contrast?: string;
                };
                text?: {
                    primary?: string;
                    secondary?: string;
                    disabled?: string;
                    onDark?: string;
                    onDarkSecondary?: string;
                };
                status?: {
                    info?: {
                        background?: string;
                        border?: string;
                        text?: string;
                    };
                    warning?: {
                        background?: string;
                        border?: string;
                        text?: string;
                    };
                    error?: {
                        background?: string;
                        border?: string;
                        text?: string;
                    };
                    success?: {
                        background?: string;
                        border?: string;
                        text?: string;
                    };
                };
                interactive?: {
                    button?: {
                        command?: string;
                        script?: string;
                        hover?: string;
                        disabled?: string;
                    };
                    input?: {
                        background?: string;
                        border?: string;
                        focus?: string;
                        text?: string;
                    };
                    code?: {
                        background?: string;
                        text?: string;
                        padding?: string;
                        borderRadius?: number;
                        fontSize?: string;
                    };
                };
            };
        };
    }
}
import {ThemeProvider as StylesThemeProvider} from '@mui/styles';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
import * as React from 'react';
import * as ReactRouter from 'react-router-dom';
import Header from './Header';
import LoadingSpinner from '../common/LoadingSpinner';
import Navigation from './Navigation';
import ScrollUpButton from '../common/ScrollUpButton';
import SettingsDialog from '../common/SettingsDialog';
import SystemSettingsDialogWrapper from '../common/SystemSettingsDialogWrapper';
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
import Logs from '../logs/Logs';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {ConnectionErrorBanner} from '../common/ConnectionErrorBanner';

const {HashRouter, Route, Switch} = ReactRouter;

const MainContent = styled('main')(({theme}) => ({
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
        custom: {
            colors: {
                primary: {
                    black: '#000000',
                    darkGray: '#424242',
                    mediumGray: '#616161',
                    lightGray: '#9e9e9e',
                },
                background: {
                    white: '#ffffff',
                    lightGray: '#f5f5f5',
                    mediumGray: '#e0e0e0',
                    overlay: '#f8f8f8',
                },
                border: {
                    light: '#e0e0e0',
                    medium: '#d0d0d0',
                    dark: '#bdbdbd',
                    contrast: '#757575',
                },
                text: {
                    primary: '#212121',
                    secondary: '#757575',
                    disabled: '#bdbdbd',
                    onDark: '#ffffff',
                    onDarkSecondary: '#e0e0e0',
                },
                status: {
                    info: {
                        background: '#e3f2fd',
                        border: '#1976d2',
                        text: '#1565c0',
                    },
                    warning: {
                        background: '#fff3e0',
                        border: '#f57c00',
                        text: '#e65100',
                    },
                    error: {
                        background: '#ffebee',
                        border: '#d32f2f',
                        text: '#c62828',
                    },
                    success: {
                        background: '#e8f5e8',
                        border: '#388e3c',
                        text: '#2e7d32',
                    },
                },
                interactive: {
                    button: {
                        command: '#4caf50',
                        script: '#2196f3',
                        hover: '#f5f5f5',
                        disabled: '#e0e0e0',
                    },
                    input: {
                        background: '#ffffff',
                        border: '#d0d0d0',
                        focus: '#3f51b5',
                        text: '#212121',
                    },
                    code: {
                        background: '#f5f5f5',
                        text: '#333333',
                        padding: '8px',
                        borderRadius: 4,
                        fontSize: '0.875rem',
                    },
                },
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
            // Dialog 对话框样式优化 - 浅色主题
            MuiDialog: {
                styleOverrides: {
                    root: {
                        '& .MuiBackdrop-root': {
                            backgroundColor: 'rgba(0, 0, 0, 0.5) !important',
                            backdropFilter: 'none !important',
                            filter: 'none !important',
                        },
                    },
                    paper: {
                        backgroundColor: '#ffffff !important', // 强制白色背景
                        color: '#000000 !important', // 强制黑色文字
                        opacity: '1 !important',
                        filter: 'none !important',
                        backdropFilter: 'none !important',
                        backgroundImage: 'none !important', // 移除背景图片overlay
                        '--Paper-overlay': 'none !important', // 禁用Paper-overlay CSS变量
                    },
                },
            },
            // 优化代码块显示
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#ffffff',
                        backgroundImage: 'none !important', // 移除overlay背景图片
                        '--Paper-overlay': 'none !important', // 禁用Paper-overlay CSS变量
                        '& code': {
                            backgroundColor: '#f5f5f5',
                            color: '#333333',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.875rem',
                            fontFamily:
                                'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
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
                paper: '#424242', // 恢复深色背景
            },
            text: {
                primary: '#ffffff',
                secondary: '#b0bec5',
            },
        },
        custom: {
            colors: {
                primary: {
                    black: '#000000',
                    darkGray: '#2c2c2c',
                    mediumGray: '#424242',
                    lightGray: '#616161',
                },
                background: {
                    white: '#ffffff',
                    lightGray: '#f5f5f5',
                    mediumGray: '#424242',
                    overlay: '#383838',
                },
                border: {
                    light: '#616161',
                    medium: '#757575',
                    dark: '#424242',
                    contrast: '#9e9e9e',
                },
                text: {
                    primary: '#ffffff',
                    secondary: '#b0bec5',
                    disabled: '#757575',
                    onDark: '#e0e0e0',
                    onDarkSecondary: '#b0bec5',
                },
                status: {
                    info: {
                        background: '#1976d2',
                        border: '#1565c0',
                        text: '#ffffff',
                    },
                    warning: {
                        background: '#f57c00',
                        border: '#e65100',
                        text: '#ffffff',
                    },
                    error: {
                        background: '#d32f2f',
                        border: '#c62828',
                        text: '#ffffff',
                    },
                    success: {
                        background: '#388e3c',
                        border: '#2e7d32',
                        text: '#ffffff',
                    },
                },
                interactive: {
                    button: {
                        command: '#4caf50',
                        script: '#2196f3',
                        hover: '#616161',
                        disabled: '#424242',
                    },
                    input: {
                        background: '#2c2c2c',
                        border: '#757575',
                        focus: '#3f51b5',
                        text: '#e0e0e0',
                    },
                    code: {
                        background: '#1e1e1e',
                        text: '#e0e0e0',
                        padding: '8px',
                        borderRadius: 4,
                        fontSize: '0.875rem',
                    },
                },
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
                        backgroundColor: '#424242', // 恢复深色背景
                        borderRight: '1px solid #616161',
                    },
                },
            },
            MuiButton: {
                styleOverrides: {
                    root: {
                        borderRadius: '4px',
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
            // Dialog 对话框样式优化 - 强制移除透明度，使用深色背景+亮色文字
            MuiDialog: {
                styleOverrides: {
                    root: {
                        // 移除所有可能的透明度和滤镜效果
                        '& .MuiBackdrop-root': {
                            backgroundColor: 'rgba(0, 0, 0, 0.5) !important',
                            backdropFilter: 'none !important',
                            filter: 'none !important',
                        },
                        // 确保对话框容器不透明
                        '& .MuiDialog-container': {
                            backgroundColor: 'transparent !important',
                        },
                    },
                    paper: {
                        backgroundColor: '#424242 !important', // 强制深色背景
                        color: '#e0e0e0 !important', // 使用温和的浅灰色文字
                        opacity: '1 !important', // 强制不透明
                        filter: 'none !important', // 移除任何滤镜
                        backdropFilter: 'none !important', // 移除背景滤镜
                        backgroundImage: 'none !important', // 移除背景图片overlay
                        '--Paper-overlay': 'none !important', // 禁用Paper-overlay CSS变量
                        boxShadow:
                            '0px 11px 15px -7px rgba(0,0,0,0.2), 0px 24px 38px 3px rgba(0,0,0,0.14), 0px 9px 46px 8px rgba(0,0,0,0.12) !important',
                        // 强制所有文字元素为温和的灰色
                        '& *': {
                            color: '#e0e0e0 !important',
                        },
                        '& .MuiTypography-root': {
                            color: '#e0e0e0 !important',
                        },
                        '& .MuiDialogTitle-root': {
                            color: '#f5f5f5 !important', // 标题稍微亮一点
                            backgroundColor: 'transparent !important',
                        },
                        '& .MuiDialogContent-root': {
                            color: '#e0e0e0 !important',
                            backgroundColor: 'transparent !important',
                        },
                        '& .MuiDialogActions-root': {
                            backgroundColor: 'transparent !important',
                        },
                        '& .MuiInputLabel-root': {
                            color: '#b0bec5 !important', // 输入标签保持原有颜色
                        },
                        '& .MuiOutlinedInput-input': {
                            color: '#e0e0e0 !important', // 输入框文字也使用温和灰色
                            backgroundColor: '#616161 !important',
                        },
                        '& .MuiTextField-root': {
                            '& .MuiOutlinedInput-root': {
                                backgroundColor: '#616161 !important',
                                '& fieldset': {
                                    borderColor: '#757575 !important',
                                },
                            },
                        },
                        // 优化按钮文字颜色
                        '& .MuiButton-root': {
                            '&.MuiButton-contained': {
                                color: '#ffffff !important', // 主要按钮保持白色文字
                            },
                            '&:not(.MuiButton-contained)': {
                                color: '#e0e0e0 !important', // 次要按钮使用温和灰色
                            },
                        },
                    },
                },
            },
            // 优化代码块显示
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundColor: '#424242', // 恢复深色背景
                        color: '#ffffff',
                        backgroundImage: 'none !important', // 移除overlay背景图片
                        '--Paper-overlay': 'none !important', // 禁用Paper-overlay CSS变量
                        '& code': {
                            backgroundColor: '#616161',
                            color: '#ffffff',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.875rem',
                            fontFamily:
                                'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
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
class Layout extends React.Component<Stores<'currentUser' | 'snackManager'>> {
    @observable
    private currentTheme: ThemeKey = 'dark';
    @observable
    private showSettings = false;

    @observable
    private showSystemSettings = false;

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
        const {showSettings, showSystemSettings, currentTheme} = this;
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
                                    <style>{`
                                        .MuiDialog-paper {
                                            background-color: ${
                                                currentTheme === 'dark' ? '#424242' : '#ffffff'
                                            } !important;
                                            color: ${
                                                currentTheme === 'dark' ? '#e0e0e0' : '#000000'
                                            } !important;
                                            opacity: 1 !important;
                                            filter: none !important;
                                            backdrop-filter: none !important;
                                            --Paper-overlay: none !important;
                                            background-image: none !important;
                                        }
                                        .MuiDialog-paper * {
                                            color: ${
                                                currentTheme === 'dark' ? '#e0e0e0' : '#000000'
                                            } !important;
                                        }
                                        .MuiDialog-paper .MuiDialogTitle-root {
                                            color: ${
                                                currentTheme === 'dark' ? '#f5f5f5' : '#000000'
                                            } !important;
                                        }
                                        .MuiBackdrop-root {
                                            backdrop-filter: none !important;
                                            filter: none !important;
                                        }
                                        /* 全局禁用 Paper overlay 效果 */
                                        .MuiPaper-root {
                                            --Paper-overlay: none !important;
                                        }
                                        .MuiDialog-paper {
                                            --Paper-overlay: none !important;
                                        }
                                    `}</style>
                                    <Header
                                        style={{top: !connectionErrorMessage ? 0 : 64}}
                                        admin={admin}
                                        name={name}
                                        version={versionInfo.version}
                                        loggedIn={loggedIn}
                                        toggleTheme={this.toggleTheme.bind(this)}
                                        showSettings={() => (this.showSettings = true)}
                                        showSystemSettings={() => (this.showSystemSettings = true)}
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
                                                    : React.createElement(CustomRedirect, {
                                                          to: '/login',
                                                      }),
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
                                                }),
                                                React.createElement(Route as any, {
                                                    exact: true,
                                                    path: '/logs',
                                                    component: Logs,
                                                })
                                            )}
                                        </MainContent>
                                    </div>
                                    {showSettings && (
                                        <SettingsDialog
                                            fClose={() => (this.showSettings = false)}
                                        />
                                    )}
                                    {showSystemSettings && admin && (
                                        <SystemSettingsDialogWrapper
                                            open={showSystemSettings}
                                            onClose={() => (this.showSystemSettings = false)}
                                            token={this.props.currentUser.token()}
                                        />
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
