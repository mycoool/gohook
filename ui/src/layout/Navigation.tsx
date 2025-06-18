import Divider from '@material-ui/core/Divider';
import Drawer from '@material-ui/core/Drawer';
import {StyleRules, Theme, WithStyles, withStyles} from '@material-ui/core/styles';
import React, {Component} from 'react';
import {Link} from 'react-router-dom';
import {observer} from 'mobx-react';
import {mayAllowPermission, requestPermission} from '../snack/browserNotification';
import {
    Button,
    Hidden,
    IconButton,
    Typography,
    ListItem,
    ListItemText,
    ListItemAvatar,
    Avatar,
} from '@material-ui/core';
import {DrawerProps} from '@material-ui/core/Drawer/Drawer';
import CloseIcon from '@material-ui/icons/Close';
import PeopleIcon from '@material-ui/icons/People';

const styles = (theme: Theme): StyleRules<'root' | 'drawerPaper' | 'toolbar' | 'link'> => ({
    root: {
        height: '100%',
    },
    drawerPaper: {
        position: 'relative',
        width: 250,
        minHeight: '100%',
        height: '100vh',
    },
    toolbar: theme.mixins.toolbar,
    link: {
        color: 'inherit',
        textDecoration: 'none',
    },
});

type Styles = WithStyles<'root' | 'drawerPaper' | 'toolbar' | 'link'>;

interface IProps {
    loggedIn: boolean;
    navOpen: boolean;
    setNavOpen: (open: boolean) => void;
    user?: { admin: boolean; role?: string };
}

@observer
class Navigation extends Component<
    IProps & Styles,
    {showRequestNotification: boolean}
> {
    public state = {showRequestNotification: mayAllowPermission()};

    public render() {
        const {classes, loggedIn, navOpen, setNavOpen, user} = this.props;
        const {showRequestNotification} = this.state;

        const isAdmin = user?.admin ?? user?.role === 'admin';

        return (
            <ResponsiveDrawer
                classes={{root: classes.root, paper: classes.drawerPaper}}
                navOpen={navOpen}
                setNavOpen={setNavOpen}
                id="message-navigation">
                <div className={classes.toolbar} />
                {React.createElement(Link as any, {
                    className: classes.link, 
                    to: "/", 
                    onClick: () => setNavOpen(false),
                }, (
                    <ListItem button disabled={!loggedIn} className="all">
                        <ListItemText primary="All Projects" />
                    </ListItem>
                ))}
                <Divider />
                <Typography align="center" style={{marginTop: 10}}>
                    {showRequestNotification ? (
                        <Button
                            onClick={() => {
                                requestPermission();
                                this.setState({showRequestNotification: false});
                            }}>
                            启用通知
                        </Button>
                    ) : null}
                </Typography>
            </ResponsiveDrawer>
        );
    }
}

const ResponsiveDrawer: React.FC<
    DrawerProps & {navOpen: boolean; setNavOpen: (open: boolean) => void}
> = ({navOpen, setNavOpen, children, ...rest}) => (
    <>
        <Hidden smUp implementation="css">
            <Drawer variant="temporary" open={navOpen} {...rest}>
                <IconButton onClick={() => setNavOpen(false)}>
                    <CloseIcon />
                </IconButton>
                {children}
            </Drawer>
        </Hidden>
        <Hidden xsDown implementation="css">
            <Drawer variant="permanent" {...rest}>
                {children}
            </Drawer>
        </Hidden>
    </>
);

export default withStyles(styles, {withTheme: true})(Navigation);
