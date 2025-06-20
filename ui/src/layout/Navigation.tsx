import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import {Theme, styled} from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import React, {Component} from 'react';
import {Link} from 'react-router-dom';
import {observer} from 'mobx-react';
import {mayAllowPermission, requestPermission} from '../snack/browserNotification';
import {
    Button,
    IconButton,
    Typography,
    ListItem,
    ListItemButton,
    ListItemText,
    ListItemAvatar,
    Avatar,
} from '@mui/material';
import {DrawerProps} from '@mui/material/Drawer';
import CloseIcon from '@mui/icons-material/Close';
import PeopleIcon from '@mui/icons-material/People';

const StyledDrawer = styled(Drawer)(({theme}) => ({
    height: '100%',
    '& .MuiDrawer-paper': {
        position: 'fixed',
        width: 250,
        height: '100vh',
        top: 10, // Header的高度
        left: 0,
        zIndex: theme.zIndex.drawer,
    },
}));

const Toolbar = styled('div')(({theme}) => theme.mixins.toolbar);

const StyledLink = styled(Link)({
    color: 'inherit',
    textDecoration: 'none',
});

interface IProps {
    loggedIn: boolean;
    navOpen: boolean;
    setNavOpen: (open: boolean) => void;
    user?: {admin: boolean; role?: string};
}

@observer
class Navigation extends Component<IProps, {showRequestNotification: boolean}> {
    public state = {showRequestNotification: mayAllowPermission()};

    public render() {
        const {loggedIn, navOpen, setNavOpen, user} = this.props;
        const {showRequestNotification} = this.state;

        const isAdmin = user?.admin ?? user?.role === 'admin';

        return (
            <ResponsiveDrawer navOpen={navOpen} setNavOpen={setNavOpen} id="message-navigation">
                <Toolbar />
                {React.createElement(
                    StyledLink as any,
                    {
                        to: '/',
                        onClick: () => setNavOpen(false),
                    },
                    <ListItemButton disabled={!loggedIn} className="all">
                        <ListItemText primary="All Projects" />
                    </ListItemButton>
                )}
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
> = ({navOpen, setNavOpen, children, ...rest}) => {
    const isSmUp = useMediaQuery((theme: Theme) => theme.breakpoints.up('sm'));
    const isXsDown = useMediaQuery((theme: Theme) => theme.breakpoints.down('sm'));

    return (
        <>
            {!isSmUp && (
                <Drawer
                    variant="temporary"
                    open={navOpen}
                    onClose={() => setNavOpen(false)}
                    ModalProps={{
                        keepMounted: true, // 提高移动端性能
                    }}
                    {...rest}
                    sx={{
                        '& .MuiDrawer-paper': {
                            position: 'fixed',
                            top: 0,
                            height: '100vh',
                            width: 250,
                        },
                    }}>
                    <IconButton onClick={() => setNavOpen(false)} size="large">
                        <CloseIcon />
                    </IconButton>
                    {children}
                </Drawer>
            )}
            {!isXsDown && (
                <StyledDrawer variant="permanent" {...rest}>
                    {children}
                </StyledDrawer>
            )}
        </>
    );
};

export default Navigation;
