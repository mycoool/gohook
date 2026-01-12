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
    List,
    ListSubheader,
    Tooltip,
    CircularProgress,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import RefreshIcon from '@mui/icons-material/Refresh';
import {SyncNodeStore} from '../sync/SyncNodeStore';
import {ISyncNodeRuntime} from '../types';

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

const displayAddr = (addr?: string) => {
    const raw = String(addr || '').trim();
    if (!raw) return '';
    if (raw.startsWith('[')) {
        const idx = raw.indexOf(']');
        if (idx > 1) return raw.slice(1, idx);
        return raw;
    }
    const lastColon = raw.lastIndexOf(':');
    if (lastColon > 0) {
        const port = raw.slice(lastColon + 1);
        if (/^\d+$/.test(port)) return raw.slice(0, lastColon);
    }
    return raw;
};

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
        const localRuntime = syncNodeStore.local;

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
                    <ListItemButton sx={{alignItems: 'flex-start'}}>
                        <ListItemText
                            primaryTypographyProps={{noWrap: true}}
                            primary="MAIN-NODE"
                            secondaryTypographyProps={{
                                noWrap: true,
                            }}
                            secondary={
                                localRuntime
                                    ? localRuntime.hostname || 'gohook'
                                    : loading
                                    ? '加载中...'
                                    : '暂无服务器信息'
                            }
                            sx={{minWidth: 0, mr: 1}}
                        />
                        <Tooltip
                            title={this.runtimeTooltip(
                                localRuntime,
                                localRuntime ? 'CONNECTED' : 'DISCONNECTED'
                            )}
                            placement="right">
                            <Box
                                sx={{
                                    ml: 1,
                                    width: 76,
                                    display: 'flex',
                                    flexDirection: 'column',
                                    gap: 0.25,
                                    flexShrink: 0,
                                }}>
                                {this.renderRuntimeMiniChart(
                                    localRuntime,
                                    localRuntime ? 'CONNECTED' : 'DISCONNECTED'
                                )}
                            </Box>
                        </Tooltip>
                    </ListItemButton>
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
                                <ListItemButton sx={{alignItems: 'flex-start'}}>
                                    <ListItemText
                                        primaryTypographyProps={{noWrap: true}}
                                        primary={node.name}
                                        secondaryTypographyProps={{
                                            noWrap: true,
                                        }}
                                        secondary={displayAddr(node.address)}
                                        sx={{minWidth: 0, mr: 1}}
                                    />
                                    <Tooltip
                                        title={this.runtimeTooltip(
                                            node.runtime,
                                            node.connectionStatus
                                        )}
                                        placement="right">
                                        <Box
                                            sx={{
                                                ml: 1,
                                                width: 76,
                                                display: 'flex',
                                                flexDirection: 'column',
                                                gap: 0.25,
                                                flexShrink: 0,
                                            }}>
                                            {this.renderRuntimeMiniChart(
                                                node.runtime,
                                                node.connectionStatus
                                            )}
                                        </Box>
                                    </Tooltip>
                                </ListItemButton>
                            </StyledLink>
                        ))
                    )}
                </List>
            </Box>
        );
    }

    private runtimeTooltip(runtime?: ISyncNodeRuntime, connectionStatus?: string) {
        const cs = String(connectionStatus || 'UNKNOWN').toUpperCase();
        const updatedAt = runtime?.updatedAt ? new Date(runtime.updatedAt).toLocaleString() : 'N/A';
        const cpu = runtime?.cpuPercent != null ? `${runtime.cpuPercent.toFixed(1)}%` : 'N/A';
        const mem =
            runtime?.memUsedPercent != null ? `${runtime.memUsedPercent.toFixed(1)}%` : 'N/A';
        const load1 = runtime?.load1 != null ? runtime.load1.toFixed(2) : 'N/A';
        const disk =
            runtime?.diskUsedPercent != null ? `${runtime.diskUsedPercent.toFixed(1)}%` : 'N/A';
        const host = runtime?.hostname || '';
        const conn = this.connText(cs);
        return (
            <Box sx={{whiteSpace: 'pre-wrap', maxWidth: 360}}>
                <Typography variant="subtitle2">{host || '节点'}</Typography>
                <Typography variant="body2">连接：{conn}</Typography>
                <Typography variant="body2">CPU：{cpu}</Typography>
                <Typography variant="body2">内存：{mem}</Typography>
                <Typography variant="body2">Load1：{load1}</Typography>
                <Typography variant="body2">磁盘：{disk}</Typography>
                <Typography variant="caption" color="textSecondary">
                    更新：{updatedAt}
                </Typography>
            </Box>
        );
    }

    private connText(cs: string) {
        switch (cs) {
            case 'CONNECTED':
                return '在线';
            case 'DISCONNECTED':
                return '离线';
            case 'UNPAIRED':
                return '未配对';
            default:
                return '未知';
        }
    }

    private barColorByPercent(label: string, p: number) {
        let warn = 81;
        let critical = 91;
        switch (label) {
            case 'C':
                warn = 80;
                critical = 90;
                break;
            case 'M':
                warn = 75;
                critical = 90;
                break;
            case 'D':
                warn = 70;
                critical = 85;
                break;
            case 'L':
                warn = 60;
                critical = 80;
                break;
            default:
                break;
        }
        if (p >= critical) return '#f44336';
        if (p >= warn) return '#ff9800';
        return '#4caf50';
    }

    private renderMetricBar(label: string, value: number | null, disabled: boolean) {
        const v =
            value == null || !Number.isFinite(value) ? null : Math.max(0, Math.min(100, value));
        const fill = v == null ? 0 : v;
        const fillColor = disabled ? '#9e9e9e' : this.barColorByPercent(label, fill);
        return (
            <Box sx={{display: 'flex', alignItems: 'center', gap: 0.5}}>
                <Typography
                    variant="caption"
                    sx={{
                        width: 14,
                        lineHeight: 1,
                        color: 'rgba(255,255,255,0.7)',
                        fontSize: 9,
                    }}>
                    {label}
                </Typography>
                <Box
                    sx={{
                        flex: 1,
                        height: 6,
                        borderRadius: 1,
                        bgcolor: 'rgba(255,255,255,0.12)',
                        overflow: 'hidden',
                    }}>
                    <Box
                        sx={{
                            height: '100%',
                            width: `${fill}%`,
                            bgcolor: fillColor,
                            opacity: disabled ? 0.6 : 1,
                        }}
                    />
                </Box>
            </Box>
        );
    }

    private renderRuntimeMiniChart(runtime?: ISyncNodeRuntime, connectionStatus?: string) {
        const cs = String(connectionStatus || '').toUpperCase();
        const disabled = cs !== 'CONNECTED';

        const cpu = runtime?.cpuPercent != null ? Number(runtime.cpuPercent) : null;
        const mem = runtime?.memUsedPercent != null ? Number(runtime.memUsedPercent) : null;
        const disk = runtime?.diskUsedPercent != null ? Number(runtime.diskUsedPercent) : null;

        // Load1: normalize roughly to 0-100. Without core count, cap at 4.0 => 100%.
        const load1 = runtime?.load1 != null ? Number(runtime.load1) : null;
        const loadPercent = (() => {
            if (runtime?.load1Percent != null) {
                return Number(runtime.load1Percent);
            }
            if (load1 == null) {
                return null;
            }
            const cores = runtime?.cpuCores != null ? Number(runtime.cpuCores) : 0;
            const divisor = cores > 0 ? cores : 4;
            return (load1 / divisor) * 100;
        })();

        return (
            <>
                {this.renderMetricBar('C', cpu, disabled)}
                {this.renderMetricBar('M', mem, disabled)}
                {this.renderMetricBar('D', disk, disabled)}
                {this.renderMetricBar('L', loadPercent, disabled)}
            </>
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
