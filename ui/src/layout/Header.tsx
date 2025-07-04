import AppBar from '@mui/material/AppBar';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import {Theme, Breakpoint, styled, useTheme} from '@mui/material/styles';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import {PropTypes, useMediaQuery} from '@mui/material';
import AccountCircle from '@mui/icons-material/AccountCircle';
import ExitToApp from '@mui/icons-material/ExitToApp';
import Highlight from '@mui/icons-material/Highlight';
import GitHubIcon from '@mui/icons-material/GitHub';
import MenuIcon from '@mui/icons-material/Menu';
import Apps from '@mui/icons-material/Apps';
import SupervisorAccount from '@mui/icons-material/SupervisorAccount';
import Link from '@mui/icons-material/Link';
import Settings from '@mui/icons-material/Settings';

import AccountTree from '@mui/icons-material/AccountTree';
import History from '@mui/icons-material/History';
import React, {Component, CSSProperties} from 'react';
import {Link as RouterLink} from 'react-router-dom';
import {observer} from 'mobx-react';
import useTranslation from '../i18n/useTranslation';
import LanguageSwitcher from '../i18n/LanguageSwitcher';
import EnvironmentIndicator from './EnvironmentIndicator';

// 使用现代的 useMediaQuery 实现响应式宽度检测
const withWidth = () => (WrappedComponent: React.ComponentType<any>) => {
    const WrappedWithWidth: React.FC<any> = (props) => {
        const isXs = useMediaQuery((theme: Theme) => theme.breakpoints.only('xs'));
        const isSm = useMediaQuery((theme: Theme) => theme.breakpoints.only('sm'));
        const isMd = useMediaQuery((theme: Theme) => theme.breakpoints.only('md'));
        const isLg = useMediaQuery((theme: Theme) => theme.breakpoints.only('lg'));

        let width: Breakpoint = 'xl';
        if (isXs) width = 'xs';
        else if (isSm) width = 'sm';
        else if (isMd) width = 'md';
        else if (isLg) width = 'lg';

        return <WrappedComponent {...props} width={width} />;
    };
    WrappedWithWidth.displayName = `withWidth(${
        WrappedComponent.displayName || WrappedComponent.name
    })`;
    return WrappedWithWidth;
};

const StyledAppBar = styled(AppBar)(({theme}) => ({
    zIndex: theme.zIndex.drawer + 1,
    backgroundColor: '#3f51b5 !important',
    '&.MuiAppBar-colorPrimary': {
        backgroundColor: '#3f51b5 !important',
    },
    '&.MuiAppBar-root': {
        backgroundColor: '#3f51b5 !important',
    },
    [theme.breakpoints.down('md')]: {
        paddingBottom: 10,
    },
}));

const StyledToolbar = styled(Toolbar)(({theme}) => ({
    justifyContent: 'space-between',
    [theme.breakpoints.down('md')]: {
        flexWrap: 'wrap',
    },
}));

const MenuButtons = styled('div')(({theme}) => ({
    display: 'flex',
    gap: '4px', // 按钮之间的间距
    [theme.breakpoints.down('lg')]: {
        flex: 1,
    },
    justifyContent: 'center',
    [theme.breakpoints.down('md')]: {
        flexBasis: '100%',
        marginTop: 5,
        order: 1,
        justifyContent: 'space-between',
        gap: '2px',
    },
}));

const Title = styled('div')(({theme}) => ({
    [theme.breakpoints.up('md')]: {
        flex: 1,
    },
    display: 'flex',
    alignItems: 'center',
}));

const TitleName = styled(Typography)({
    paddingRight: 10,
});

const StyledLink = styled(RouterLink)({
    color: 'inherit',
    textDecoration: 'none',
});

const StyledA = styled('a')({
    color: 'inherit',
    textDecoration: 'none',
});

interface IProps {
    loggedIn: boolean;
    name: string;
    admin: boolean;
    version: string;
    toggleTheme: VoidFunction;
    showSettings: VoidFunction;
    showSystemSettings: VoidFunction;
    logout: VoidFunction;
    style: CSSProperties;
    width: Breakpoint;
    setNavOpen: (open: boolean) => void;
}

@observer
class Header extends Component<IProps> {
    public render() {
        const {version, name, loggedIn, admin, toggleTheme, logout, style, setNavOpen, width} =
            this.props;

        const position = width === 'xs' ? 'sticky' : 'fixed';

        return (
            <StyledAppBar
                position={position}
                style={{
                    ...style,
                    backgroundColor: '#3f51b5',
                    // 强制覆盖所有可能的背景色样式
                    backgroundImage: 'none',
                }}
                sx={{
                    backgroundColor: '#3f51b5 !important',
                    '&.MuiAppBar-colorPrimary': {
                        backgroundColor: '#3f51b5 !important',
                    },
                }}>
                <StyledToolbar>
                    <Title>
                        <StyledLink to="/">
                            <TitleName variant="h5" color="inherit">
                                GoHook
                            </TitleName>
                        </StyledLink>
                        {loggedIn && <EnvironmentIndicator />}
                        <StyledA
                            href={'https://github.com/mycoool/gohook/releases/tag/v' + version}>
                            <Typography variant="button" color="inherit">
                                v{version}
                            </Typography>
                        </StyledA>
                    </Title>
                    {loggedIn && this.renderButtons(name, admin, logout, width, setNavOpen)}
                    <div>
                        <LanguageSwitcher />

                        <IconButton onClick={toggleTheme} color="inherit" size="large">
                            <Highlight />
                        </IconButton>

                        <StyledA
                            href="https://github.com/mycoool/gohook"
                            target="_blank"
                            rel="noopener noreferrer">
                            <IconButton color="inherit" size="large">
                                <GitHubIcon />
                            </IconButton>
                        </StyledA>
                    </div>
                </StyledToolbar>
            </StyledAppBar>
        );
    }

    private renderButtons(
        name: string,
        admin: boolean,
        logout: VoidFunction,
        width: Breakpoint,
        setNavOpen: (open: boolean) => void
    ) {
        const {showSettings, showSystemSettings} = this.props;
        return (
            <MenuButtons>
                {width === 'xs' && (
                    <ResponsiveButtonWithTranslation
                        icon={<MenuIcon />}
                        onClick={() => setNavOpen(true)}
                        translationKey="nav.menu"
                        fallbackLabel="menu"
                        width={width}
                        color="inherit"
                    />
                )}
                <StyledLink to="/versions" id="navigate-versions">
                    <ResponsiveButtonWithTranslation
                        icon={<AccountTree />}
                        translationKey="nav.versions"
                        fallbackLabel="versions"
                        width={width}
                        color="inherit"
                    />
                </StyledLink>
                <StyledLink to="/hooks" id="navigate-hooks">
                    <ResponsiveButtonWithTranslation
                        icon={<Link />}
                        translationKey="nav.hooks"
                        fallbackLabel="hooks"
                        width={width}
                        color="inherit"
                    />
                </StyledLink>

                {/* 临时隐藏插件导航 - 待后续插件功能完善后再启用 */}
                {false && (
                    <StyledLink to="/plugins" id="navigate-plugins">
                        <ResponsiveButtonWithTranslation
                            icon={<Apps />}
                            translationKey="nav.plugins"
                            fallbackLabel="plugins"
                            width={width}
                            color="inherit"
                        />
                    </StyledLink>
                )}

                <StyledLink to="/logs" id="navigate-logs">
                    <ResponsiveButtonWithTranslation
                        icon={<History />}
                        translationKey="nav.logs"
                        fallbackLabel="logs"
                        width={width}
                        color="inherit"
                    />
                </StyledLink>
                {admin && (
                    <StyledLink to="/users" id="navigate-users">
                        <ResponsiveButtonWithTranslation
                            icon={<SupervisorAccount />}
                            translationKey="nav.users"
                            fallbackLabel="users"
                            width={width}
                            color="inherit"
                        />
                    </StyledLink>
                )}
                {admin && (
                    <ResponsiveButtonWithTranslation
                        icon={<Settings />}
                        translationKey="nav.systemSettings"
                        fallbackLabel="System Settings"
                        onClick={showSystemSettings}
                        id="system-settings"
                        width={width}
                        color="inherit"
                    />
                )}
                <ResponsiveButtonWithTranslation
                    icon={<AccountCircle />}
                    translationKey="nav.settings"
                    fallbackLabel={name}
                    customLabel={name}
                    onClick={showSettings}
                    id="changepw"
                    width={width}
                    color="inherit"
                />
                <ResponsiveButtonWithTranslation
                    icon={<ExitToApp />}
                    translationKey="nav.logout"
                    fallbackLabel="Logout"
                    onClick={logout}
                    id="logout"
                    width={width}
                    color="inherit"
                />
            </MenuButtons>
        );
    }
}

// 支持翻译的响应式按钮组件
const ResponsiveButtonWithTranslation: React.FC<{
    width: Breakpoint;
    color: 'inherit' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
    translationKey: string;
    fallbackLabel: string;
    customLabel?: string;
    id?: string;
    onClick?: () => void;
    icon: React.ReactNode;
}> = ({width, icon, translationKey, fallbackLabel, customLabel, ...rest}) => {
    const {t} = useTranslation();

    // 如果有自定义标签（如用户名），优先使用
    const label = customLabel ?? t(translationKey) ?? fallbackLabel;

    // 只在超小屏幕时显示纯图标，其他情况都显示图标+文字
    if (width === 'xs') {
        return (
            <IconButton {...rest} size="large">
                {icon}
            </IconButton>
        );
    }
    return (
        <Button
            startIcon={icon}
            {...rest}
            sx={{
                textTransform: 'uppercase', // 英文全大写
                fontWeight: 'bold', // 加粗
                letterSpacing: '0.5px', // 字母间距
                minWidth: 'auto',
                padding: '6px 12px',
            }}>
            {label}
        </Button>
    );
};

export default withWidth()(Header);
