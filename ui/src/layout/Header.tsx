import AppBar from '@material-ui/core/AppBar';
import Button from '@material-ui/core/Button';
import IconButton from '@material-ui/core/IconButton';
import {createStyles, Theme, WithStyles, withStyles} from '@material-ui/core/styles';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import {Hidden, PropTypes, withWidth} from '@material-ui/core';
import {Breakpoint} from '@material-ui/core/styles/createBreakpoints';
import AccountCircle from '@material-ui/icons/AccountCircle';
import ExitToApp from '@material-ui/icons/ExitToApp';
import Highlight from '@material-ui/icons/Highlight';
import GitHubIcon from '@material-ui/icons/GitHub';
import MenuIcon from '@material-ui/icons/Menu';
import Apps from '@material-ui/icons/Apps';
import SupervisorAccount from '@material-ui/icons/SupervisorAccount';
import Link from '@material-ui/icons/Link';

import AccountTree from '@material-ui/icons/AccountTree';
import React, {Component, CSSProperties} from 'react';
import {Link as RouterLink} from 'react-router-dom';
import {observer} from 'mobx-react';
import useTranslation from '../i18n/useTranslation';
import LanguageSwitcher from '../i18n/LanguageSwitcher';

const styles = (theme: Theme) =>
    createStyles({
        appBar: {
            zIndex: theme.zIndex.drawer + 1,
            [theme.breakpoints.down('xs')]: {
                paddingBottom: 10,
            },
        },
        toolbar: {
            justifyContent: 'space-between',
            [theme.breakpoints.down('xs')]: {
                flexWrap: 'wrap',
            },
        },
        menuButtons: {
            display: 'flex',
            [theme.breakpoints.down('sm')]: {
                flex: 1,
            },
            justifyContent: 'center',
            [theme.breakpoints.down('xs')]: {
                flexBasis: '100%',
                marginTop: 5,
                order: 1,
                justifyContent: 'space-between',
            },
        },
        title: {
            [theme.breakpoints.up('md')]: {
                flex: 1,
            },
            display: 'flex',
            alignItems: 'center',
        },
        titleName: {
            paddingRight: 10,
        },
        link: {
            color: 'inherit',
            textDecoration: 'none',
        },
    });

type Styles = WithStyles<'link' | 'menuButtons' | 'toolbar' | 'titleName' | 'title' | 'appBar'>;

interface IProps extends Styles {
    loggedIn: boolean;
    name: string;
    admin: boolean;
    version: string;
    toggleTheme: VoidFunction;
    showSettings: VoidFunction;
    logout: VoidFunction;
    style: CSSProperties;
    width: Breakpoint;
    setNavOpen: (open: boolean) => void;
}

@observer
class Header extends Component<IProps> {
    public render() {
        const {
            classes,
            version,
            name,
            loggedIn,
            admin,
            toggleTheme,
            logout,
            style,
            setNavOpen,
            width,
        } = this.props;

        const position = width === 'xs' ? 'sticky' : 'fixed';

        return (
            <AppBar position={position} style={style} className={classes.appBar}>
                <Toolbar className={classes.toolbar}>
                    <div className={classes.title}>
                        <RouterLink to="/" className={classes.link}>
                            <Typography variant="h5" className={classes.titleName} color="inherit">
                                GoHook
                            </Typography>
                        </RouterLink>
                        <a
                            href={'https://github.com/mycoool/gohook/releases/tag/v' + version}
                            className={classes.link}>
                            <Typography variant="button" color="inherit">
                                @{version}
                            </Typography>
                        </a>
                    </div>
                    {loggedIn && this.renderButtons(name, admin, logout, width, setNavOpen)}
                    <div>
                        <LanguageSwitcher />
                        
                        <IconButton onClick={toggleTheme} color="inherit">
                            <Highlight />
                        </IconButton>

                        <a
                            href="https://github.com/mycoool/gohook"
                            className={classes.link}
                            target="_blank"
                            rel="noopener noreferrer">
                            <IconButton color="inherit">
                                <GitHubIcon />
                            </IconButton>
                        </a>
                    </div>
                </Toolbar>
            </AppBar>
        );
    }

    private renderButtons(
        name: string,
        admin: boolean,
        logout: VoidFunction,
        width: Breakpoint,
        setNavOpen: (open: boolean) => void
    ) {
        const {classes, showSettings} = this.props;
        return (
            <div className={classes.menuButtons}>
                <Hidden smUp implementation="css">
                    <ResponsiveButtonWithTranslation
                        icon={<MenuIcon />}
                        onClick={() => setNavOpen(true)}
                        translationKey="nav.menu"
                        fallbackLabel="menu"
                        width={width}
                        color="inherit"
                    />
                </Hidden>
                <RouterLink className={classes.link} to="/versions" id="navigate-versions">
                    <ResponsiveButtonWithTranslation
                        icon={<AccountTree />}
                        translationKey="nav.versions"
                        fallbackLabel="versions"
                            width={width}
                            color="inherit"
                        />
                </RouterLink>
                <RouterLink className={classes.link} to="/hooks" id="navigate-hooks">
                    <ResponsiveButtonWithTranslation
                        icon={<Link />}
                        translationKey="nav.hooks"
                        fallbackLabel="hooks"
                        width={width}
                        color="inherit"
                    />
                </RouterLink>

                <RouterLink className={classes.link} to="/plugins" id="navigate-plugins">
                    <ResponsiveButtonWithTranslation
                        icon={<Apps />}
                        translationKey="nav.plugins"
                        fallbackLabel="plugins"
                        width={width}
                        color="inherit"
                    />
                </RouterLink>
                {admin && (
                    <RouterLink className={classes.link} to="/users" id="navigate-users">
                        <ResponsiveButtonWithTranslation
                            icon={<SupervisorAccount />}
                            translationKey="nav.users"
                            fallbackLabel="users"
                            width={width}
                            color="inherit"
                        />
                    </RouterLink>
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
            </div>
        );
    }
}

// 支持翻译的响应式按钮组件
const ResponsiveButtonWithTranslation: React.FC<{
    width: Breakpoint;
    color: PropTypes.Color;
    translationKey: string;
    fallbackLabel: string;
    customLabel?: string;
    id?: string;
    onClick?: () => void;
    icon: React.ReactNode;
}> = ({width, icon, translationKey, fallbackLabel, customLabel, ...rest}) => {
    const { t } = useTranslation();
    
    // 如果有自定义标签（如用户名），优先使用
    const label = customLabel ?? t(translationKey) ?? fallbackLabel;
    
    if (width === 'xs' || width === 'sm') {
        return <IconButton {...rest}>{icon}</IconButton>;
    }
    return (
        <Button startIcon={icon} {...rest}>
            {label}
        </Button>
    );
};

export default withWidth()(withStyles(styles, {withTheme: true})(Header));
