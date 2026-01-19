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
import useTranslation from '../i18n/useTranslation';

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
    t: (key: string, params?: Record<string, string | number>) => string;
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
        const {t} = this.props;

        return (
            <Box mt={2}>
                <Box display="flex" alignItems="center" justifyContent="space-between" px={2}>
                    <Typography
                        variant="subtitle1"
                        color="text.primary"
                        fontWeight={600}
                        sx={{textTransform: 'uppercase', fontSize: 14}}>
                        {t('sidebar.nodeList')}
                    </Typography>
                    <Tooltip title={t('sidebar.refreshNodeList')}>
                        <span>
                            <IconButton
                                size="small"
                                onClick={this.refreshNodes}
                                disabled={loading}
                                aria-label={t('sidebar.refreshNodeAria')}
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
                            primary={t('sidebar.mainNode')}
                            secondaryTypographyProps={{
                                noWrap: true,
                            }}
                            secondary={
                                localRuntime
                                    ? localRuntime.hostname || 'gohook'
                                    : loading
                                    ? t('common.loading')
                                    : t('sidebar.noServerInfo')
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
                                {loading ? t('common.loading') : t('sidebar.noNodes')}
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
        const {t} = this.props;
        const updatedAt = runtime?.updatedAt
            ? new Date(runtime.updatedAt).toLocaleString()
            : t('syncNodes.notAvailable');
        const cpu =
            runtime?.cpuPercent != null
                ? `${runtime.cpuPercent.toFixed(1)}%`
                : t('syncNodes.notAvailable');
        const mem =
            runtime?.memUsedPercent != null
                ? `${runtime.memUsedPercent.toFixed(1)}%`
                : t('syncNodes.notAvailable');
        const load1 =
            runtime?.load1 != null ? runtime.load1.toFixed(2) : t('syncNodes.notAvailable');
        const disk =
            runtime?.diskUsedPercent != null
                ? `${runtime.diskUsedPercent.toFixed(1)}%`
                : t('syncNodes.notAvailable');
        const host = runtime?.hostname || '';
        const conn = this.connText(cs);
        return (
            <Box sx={{whiteSpace: 'pre-wrap', maxWidth: 360}}>
                <Typography variant="subtitle2">{host || t('sidebar.nodeFallback')}</Typography>
                <Typography variant="body2">
                    {t('sidebar.connectionLabel')}：{conn}
                </Typography>
                <Typography variant="body2">
                    {t('sidebar.cpuLabel')}：{cpu}
                </Typography>
                <Typography variant="body2">
                    {t('sidebar.memoryLabel')}：{mem}
                </Typography>
                <Typography variant="body2">
                    {t('sidebar.loadLabel')}：{load1}
                </Typography>
                <Typography variant="body2">
                    {t('sidebar.diskLabel')}：{disk}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                    {t('sidebar.updatedLabel')}：{updatedAt}
                </Typography>
            </Box>
        );
    }

    private connText(cs: string) {
        const {t} = this.props;
        switch (cs) {
            case 'CONNECTED':
                return t('syncNodes.connection.connected');
            case 'DISCONNECTED':
                return t('syncNodes.connection.disconnected');
            case 'UNPAIRED':
                return t('syncNodes.connection.unpaired');
            default:
                return t('syncNodes.connection.unknown');
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

const NavigationWithTranslation: React.FC<Omit<IProps, 't'>> = (props) => {
    const {t} = useTranslation();
    return <Navigation {...props} t={t} />;
};

export default NavigationWithTranslation;
export {Navigation, NavigationWithTranslation};
