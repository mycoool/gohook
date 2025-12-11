import Divider from '@mui/material/Divider';
import Drawer, {DrawerProps} from '@mui/material/Drawer';
import {Theme, styled} from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import React, {Component} from 'react';
import {Link} from 'react-router-dom';
import {observer} from 'mobx-react';
import {
    Box,
    IconButton,
    Typography,
    ListItemButton,
    ListItemText,
    ListItemAvatar,
    Avatar,
    List,
    ListSubheader,
    Tooltip,
    Chip,
    CircularProgress,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import RefreshIcon from '@mui/icons-material/Refresh';
import {SyncNodeStore} from '../sync/SyncNodeStore';

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
    syncNodeStore: SyncNodeStore;
}

@observer
class Navigation extends Component<IProps, {showRequestNotification: boolean}> {
    public state = {showRequestNotification: false};

    public componentDidMount() {
        if (this.props.loggedIn) {
            this.refreshNodes();
        }
    }

    public componentDidUpdate(prevProps: IProps) {
        if (!prevProps.loggedIn && this.props.loggedIn) {
            this.refreshNodes();
        }
    }

    public render() {
        const {loggedIn, navOpen, setNavOpen} = this.props;

        return (
            <ResponsiveDrawer navOpen={navOpen} setNavOpen={setNavOpen} id="message-navigation">
                <Toolbar />
                {loggedIn ? this.renderSyncNodesSection() : null}
            </ResponsiveDrawer>
        );
    }

    private refreshNodes = () => {
        if (!this.props.loggedIn) {
            return;
        }
        this.props.syncNodeStore.refreshNodes().catch(() => undefined);
    };

    private renderSyncNodesSection() {
        const {syncNodeStore, setNavOpen} = this.props;
        const nodes = syncNodeStore.all;
        const loading = syncNodeStore.loading;

        return (
            <Box mt={2}>
                <Box display="flex" alignItems="center" justifyContent="space-between" px={2}>
                    <Typography
                        variant="subtitle1"
                        color="text.primary"
                        fontWeight={600}
                        sx={{textTransform: 'uppercase', fontSize: 14}}>
                        节点列表
                    </Typography>
                    <Tooltip title="刷新节点列表">
                        <span>
                            <IconButton
                                size="small"
                                onClick={this.refreshNodes}
                                disabled={loading}
                                aria-label="刷新节点"
                                sx={{width: 32, height: 32}}>
                                <Box display="flex" alignItems="center" justifyContent="center">
                                    {loading ? (
                                        <CircularProgress size={16} />
                                    ) : (
                                        <RefreshIcon fontSize="small" />
                                    )}
                                </Box>
                            </IconButton>
                        </span>
                    </Tooltip>
                </Box>
                <Divider sx={{my: 1}} />
                <List dense>
                    {nodes.length === 0 ? (
                        <Box px={3} py={1}>
                            <Typography variant="body2" color="textSecondary">
                                {loading ? '加载中...' : '暂无节点'}
                            </Typography>
                        </Box>
                    ) : (
                        nodes.map((node) => (
                            <StyledLink
                                to={`/sync/nodes?node=${node.id}`}
                                onClick={() => setNavOpen(false)}
                                key={node.id}>
                                <ListItemButton>
                                    <ListItemAvatar>
                                        <Avatar
                                            sx={{
                                                bgcolor: this.getNodeAvatarColor(node.health),
                                                width: 32,
                                                height: 32,
                                                fontSize: 14,
                                            }}>
                                            {node.name.slice(0, 2).toUpperCase()}
                                        </Avatar>
                                    </ListItemAvatar>
                                    <ListItemText
                                        primaryTypographyProps={{noWrap: true}}
                                        primary={node.name}
                                        secondaryTypographyProps={{
                                            noWrap: true,
                                        }}
                                        secondary={node.address}
                                    />
                                    <Chip
                                        label={node.health || node.status}
                                        color={this.getStatusColor(node.health)}
                                        size="small"
                                        sx={{ml: 1}}
                                    />
                                </ListItemButton>
                            </StyledLink>
                        ))
                    )}
                </List>
            </Box>
        );
    }

    private getNodeAvatarColor(health: string) {
        switch (health) {
            case 'HEALTHY':
                return 'success.main';
            case 'DEGRADED':
                return 'warning.main';
            case 'OFFLINE':
            case 'UNKNOWN':
            default:
                return 'text.disabled';
        }
    }

    private getStatusColor(health: string) {
        switch (health) {
            case 'HEALTHY':
                return 'success';
            case 'DEGRADED':
                return 'warning';
            case 'OFFLINE':
                return 'default';
            default:
                return 'info';
        }
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
