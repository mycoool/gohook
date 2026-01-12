import React, {useEffect, useMemo, useState} from 'react';
import {
    Box,
    Button,
    ButtonGroup,
    Chip,
    IconButton,
    Paper,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Tooltip,
    Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import SettingsIcon from '@mui/icons-material/Settings';
import SyncIcon from '@mui/icons-material/Sync';
import ListAltIcon from '@mui/icons-material/ListAlt';
import DefaultPage from '../common/DefaultPage';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import {ISyncProjectSummary} from '../types';
import SyncConfigDialog from './SyncConfigDialog';
import SyncTaskDialog from './SyncTaskDialog';
import useTranslation from '../i18n/useTranslation';

type Props = Stores<'syncProjectStore' | 'syncNodeStore' | 'currentUser'>;

const parseTime = (value?: string): number => {
    if (!value) return 0;
    const date = new Date(value);
    const ts = date.getTime();
    return Number.isNaN(ts) ? 0 : ts;
};

const formatTime = (value: string | undefined, t: (key: string) => string) => {
    if (!value) return t('syncProjects.notAvailable');
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString();
};

const normalizeTaskStatus = (value?: string) =>
    String(value || '')
        .trim()
        .toLowerCase();

const statusText = (status: string, t: (key: string) => string) => {
    switch (normalizeTaskStatus(status)) {
        case 'running':
            return t('syncProjects.status.running');
        case 'success':
            return t('syncProjects.status.success');
        case 'failed':
            return t('syncProjects.status.failed');
        case 'retrying':
            return t('syncProjects.status.retrying');
        case 'pending':
            return t('syncProjects.status.pending');
        default:
            return t('syncProjects.status.idle');
    }
};

const connText = (conn: string, t: (key: string) => string) => {
    switch (String(conn || '').toUpperCase()) {
        case 'CONNECTED':
            return t('syncProjects.connection.connected');
        case 'DISCONNECTED':
            return t('syncProjects.connection.disconnected');
        case 'UNPAIRED':
            return t('syncProjects.connection.unpaired');
        default:
            return t('syncProjects.connection.unknown');
    }
};

type BadgeColor = 'success' | 'info' | 'warning' | 'error' | 'default';

const badgeForLatest = (
    focusStatus: string,
    opts: {hasNodes: boolean; anyDisconnected: boolean},
    t: (key: string) => string
): {label: string; color: BadgeColor} => {
    if (!opts.hasNodes) return {label: t('syncProjects.badge.noNodes'), color: 'error'};

    const st = normalizeTaskStatus(focusStatus);
    const suffix = opts.anyDisconnected ? t('syncProjects.badge.offlineSuffix') : '';

    switch (st) {
        case 'running':
            return {label: `${t('syncProjects.badge.running')}${suffix}`, color: 'info'};
        case 'success':
            return {
                label: `${t('syncProjects.badge.latest')}${suffix}`,
                color: opts.anyDisconnected ? 'warning' : 'success',
            };
        case 'failed':
            return {label: `${t('syncProjects.badge.failed')}${suffix}`, color: 'error'};
        case 'retrying':
            return {label: `${t('syncProjects.badge.retrying')}${suffix}`, color: 'warning'};
        case 'pending':
            return {label: `${t('syncProjects.badge.pending')}${suffix}`, color: 'warning'};
        default:
            return {
                label: `${t('syncProjects.badge.idle')}${suffix}`,
                color: opts.anyDisconnected ? 'warning' : 'default',
            };
    }
};

const nodeBlockColor = (connected: boolean, lastStatus: string) => {
    if (!connected) return '#9e9e9e'; // disconnected / unknown
    switch (lastStatus) {
        case 'running':
            return '#2196f3';
        case 'success':
            return '#4caf50';
        case 'failed':
            return '#f44336';
        case 'retrying':
            return '#ff9800';
        case 'pending':
            return '#ffb300';
        default:
            return '#bdbdbd';
    }
};

const SyncProjectsPage: React.FC<Props> = ({syncProjectStore, syncNodeStore, currentUser}) => {
    const {t} = useTranslation();
    const [selected, setSelected] = useState<ISyncProjectSummary | null>(null);
    const [taskDialogProject, setTaskDialogProject] = useState<ISyncProjectSummary | null>(null);

    useEffect(() => {
        if (!currentUser.loggedIn) return;
        if (syncNodeStore.all.length === 0 && !syncNodeStore.loading) {
            syncNodeStore.refreshNodes().catch(() => undefined);
        }
        syncProjectStore.refreshProjects().catch(() => undefined);
    }, [currentUser.loggedIn, syncNodeStore, syncProjectStore]);

    const projects = useMemo(() => syncProjectStore.projects || [], [syncProjectStore.projects]);

    const nodeIndex = useMemo(() => {
        const m = new Map<number, {connectionStatus?: string}>();
        for (const n of syncNodeStore.all) {
            m.set(n.id, {connectionStatus: n.connectionStatus});
        }
        return m;
    }, [syncNodeStore.all]);

    const triggerSync = async (p: ISyncProjectSummary) => {
        await syncProjectStore.runProjectSync(p.projectName);
    };

    return (
        <>
            <DefaultPage
                title={t('syncProjects.pageTitle')}
                maxWidth={1200}
                rightControl={
                    <ButtonGroup variant="contained">
                        <Button
                            startIcon={<RefreshIcon />}
                            onClick={() => syncProjectStore.refreshProjects()}
                            disabled={syncProjectStore.loading}>
                            {t('common.refresh')}
                        </Button>
                    </ButtonGroup>
                }>
                <Paper elevation={6} sx={{mt: 0, width: '100%', overflowX: 'auto'}}>
                    <TableContainer>
                        <Table size="small" sx={{minWidth: 980}}>
                            <TableHead>
                                <TableRow>
                                    <TableCell>{t('syncProjects.table.project')}</TableCell>
                                    <TableCell>{t('syncProjects.table.overall')}</TableCell>
                                    <TableCell>{t('syncProjects.table.lastSync')}</TableCell>
                                    <TableCell>{t('syncProjects.table.nodes')}</TableCell>
                                    <TableCell
                                        align="left"
                                        style={{whiteSpace: 'nowrap', width: 1}}>
                                        {t('common.actions')}
                                    </TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {projects.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={5} align="center">
                                            {syncProjectStore.loading
                                                ? t('common.loading')
                                                : t('syncProjects.empty')}
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    projects.map((p) => (
                                        <TableRow key={p.projectName} hover>
                                            <TableCell>
                                                <strong>{p.projectName}</strong>
                                                <br />
                                                <small style={{color: '#666'}}>{p.path}</small>
                                            </TableCell>
                                            <TableCell>
                                                {(() => {
                                                    const nodes = p.nodes || [];
                                                    const anyDisconnected = nodes.some((n) => {
                                                        const conn =
                                                            nodeIndex.get(n.nodeId)
                                                                ?.connectionStatus || 'UNKNOWN';
                                                        return conn !== 'CONNECTED';
                                                    });

                                                    const latestRunning = nodes
                                                        .filter(
                                                            (n) =>
                                                                normalizeTaskStatus(
                                                                    n.lastStatus
                                                                ) === 'running'
                                                        )
                                                        .sort(
                                                            (a, b) =>
                                                                parseTime(b.lastTaskAt) -
                                                                parseTime(a.lastTaskAt)
                                                        )[0];

                                                    const latestAny = nodes
                                                        .slice()
                                                        .sort(
                                                            (a, b) =>
                                                                Math.max(
                                                                    parseTime(b.lastTaskAt),
                                                                    parseTime(b.lastSuccessAt)
                                                                ) -
                                                                Math.max(
                                                                    parseTime(a.lastTaskAt),
                                                                    parseTime(a.lastSuccessAt)
                                                                )
                                                        )[0];

                                                    const focus = latestRunning || latestAny;
                                                    const focusStatus = normalizeTaskStatus(
                                                        focus?.lastStatus
                                                    );
                                                    const focusConn =
                                                        focus?.nodeId != null
                                                            ? nodeIndex.get(focus.nodeId)
                                                                  ?.connectionStatus || 'UNKNOWN'
                                                            : 'UNKNOWN';
                                                    const focusAt = focus?.lastTaskAt
                                                        ? formatTime(focus.lastTaskAt, t)
                                                        : focus?.lastSuccessAt
                                                        ? formatTime(focus.lastSuccessAt, t)
                                                        : '';

                                                    const badge = badgeForLatest(
                                                        focusStatus,
                                                        {
                                                            hasNodes: nodes.length > 0,
                                                            anyDisconnected,
                                                        },
                                                        t
                                                    );

                                                    const latestLine = '';

                                                    const detail = nodes
                                                        .map((n) => {
                                                            const conn =
                                                                nodeIndex.get(n.nodeId)
                                                                    ?.connectionStatus || 'UNKNOWN';
                                                            const st = statusText(
                                                                n.lastStatus || '',
                                                                t
                                                            );
                                                            const when = n.lastTaskAt
                                                                ? formatTime(n.lastTaskAt, t)
                                                                : n.lastSuccessAt
                                                                ? formatTime(n.lastSuccessAt, t)
                                                                : t('syncProjects.notAvailable');
                                                            const err = n.lastError
                                                                ? `\n${n.lastError}${
                                                                      n.lastErrorCode
                                                                          ? `\n[${n.lastErrorCode}]`
                                                                          : ''
                                                                  }`
                                                                : '';
                                                            return `${n.nodeName} (#${
                                                                n.nodeId
                                                            }) [${connText(
                                                                conn,
                                                                t
                                                            )}] [${st}] · ${when}\n${
                                                                n.targetPath
                                                            }${err}`;
                                                        })
                                                        .join('\n\n');

                                                    return (
                                                        <Tooltip
                                                            title={
                                                                detail ? (
                                                                    <pre
                                                                        style={{
                                                                            margin: 0,
                                                                            whiteSpace: 'pre-wrap',
                                                                        }}>
                                                                        {detail}
                                                                    </pre>
                                                                ) : (
                                                                    ''
                                                                )
                                                            }>
                                                            <span>
                                                                <Stack
                                                                    spacing={0.5}
                                                                    alignItems="flex-start">
                                                                    <Chip
                                                                        label={badge.label}
                                                                        size="small"
                                                                        color={badge.color}
                                                                        sx={{
                                                                            height: 20,
                                                                            fontSize: '0.7rem',
                                                                            '& .MuiChip-label': {
                                                                                px: 1,
                                                                            },
                                                                        }}
                                                                    />
                                                                    {latestLine ? (
                                                                        <Typography
                                                                            variant="caption"
                                                                            color="textSecondary"
                                                                            sx={{
                                                                                maxWidth: 320,
                                                                                whiteSpace:
                                                                                    'nowrap',
                                                                                overflow: 'hidden',
                                                                                textOverflow:
                                                                                    'ellipsis',
                                                                            }}>
                                                                            {latestLine}
                                                                        </Typography>
                                                                    ) : null}
                                                                </Stack>
                                                            </span>
                                                        </Tooltip>
                                                    );
                                                })()}
                                            </TableCell>
                                            <TableCell>{formatTime(p.lastSyncAt, t)}</TableCell>
                                            <TableCell>
                                                <Stack
                                                    direction="row"
                                                    spacing={0.75}
                                                    alignItems="center"
                                                    sx={{flexWrap: 'wrap'}}>
                                                    {(p.nodes || []).map((n) => {
                                                        const conn =
                                                            nodeIndex.get(n.nodeId)
                                                                ?.connectionStatus || 'UNKNOWN';
                                                        const connected = conn === 'CONNECTED';
                                                        const st = normalizeTaskStatus(
                                                            n.lastStatus
                                                        );
                                                        const color = nodeBlockColor(connected, st);
                                                        const when = n.lastTaskAt
                                                            ? formatTime(n.lastTaskAt, t)
                                                            : n.lastSuccessAt
                                                            ? formatTime(n.lastSuccessAt, t)
                                                            : t('syncProjects.notAvailable');
                                                        const tooltip = (
                                                            <Box
                                                                sx={{
                                                                    whiteSpace: 'pre-wrap',
                                                                    maxWidth: 520,
                                                                }}>
                                                                <Typography
                                                                    variant="subtitle2"
                                                                    sx={{mb: 0.5}}>
                                                                    {n.nodeName} (#{n.nodeId})
                                                                </Typography>
                                                                <Typography variant="body2">
                                                                    {t(
                                                                        'syncProjects.tooltip.connection'
                                                                    )}
                                                                    :{connText(conn, t)}
                                                                </Typography>
                                                                <Typography variant="body2">
                                                                    {t(
                                                                        'syncProjects.tooltip.status'
                                                                    )}
                                                                    :{statusText(st, t)}
                                                                </Typography>
                                                                <Typography variant="body2">
                                                                    {t('syncProjects.tooltip.time')}
                                                                    : {when}
                                                                </Typography>
                                                                <Typography variant="body2">
                                                                    {t('syncProjects.tooltip.path')}
                                                                    : {n.targetPath}
                                                                </Typography>
                                                                {n.lastError ? (
                                                                    <Typography
                                                                        variant="body2"
                                                                        color="error"
                                                                        sx={{mt: 0.5}}>
                                                                        {n.lastError}
                                                                        {n.lastErrorCode
                                                                            ? ` [${n.lastErrorCode}]`
                                                                            : ''}
                                                                    </Typography>
                                                                ) : null}
                                                            </Box>
                                                        );

                                                        return (
                                                            <Tooltip
                                                                key={n.nodeId}
                                                                title={tooltip}
                                                                placement="top-start">
                                                                <Box
                                                                    sx={{
                                                                        width: 14,
                                                                        height: 14,
                                                                        borderRadius: 0.75,
                                                                        bgcolor: color,
                                                                        border: '1px solid',
                                                                        borderColor:
                                                                            'rgba(0,0,0,0.12)',
                                                                        cursor: 'default',
                                                                    }}
                                                                />
                                                            </Tooltip>
                                                        );
                                                    })}
                                                </Stack>
                                            </TableCell>
                                            <TableCell
                                                align="left"
                                                style={{whiteSpace: 'nowrap', width: 1}}>
                                                <div
                                                    style={{
                                                        display: 'inline-flex',
                                                        alignItems: 'center',
                                                    }}>
                                                    <Tooltip
                                                        title={t(
                                                            'syncProjects.tooltips.taskDetails'
                                                        )}>
                                                        <span>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    setTaskDialogProject(p)
                                                                }
                                                                disabled={syncProjectStore.saving}>
                                                                <ListAltIcon fontSize="small" />
                                                            </IconButton>
                                                        </span>
                                                    </Tooltip>
                                                    <Tooltip
                                                        title={t(
                                                            'syncProjects.tooltips.manageConfig'
                                                        )}>
                                                        <IconButton
                                                            size="small"
                                                            onClick={() => setSelected(p)}>
                                                            <SettingsIcon fontSize="small" />
                                                        </IconButton>
                                                    </Tooltip>
                                                    <Tooltip
                                                        title={t('syncProjects.tooltips.syncNow')}>
                                                        <span>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() => triggerSync(p)}
                                                                disabled={syncProjectStore.saving}>
                                                                <SyncIcon fontSize="small" />
                                                            </IconButton>
                                                        </span>
                                                    </Tooltip>
                                                </div>
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Paper>
            </DefaultPage>

            {taskDialogProject ? (
                <SyncTaskDialog
                    open={!!taskDialogProject}
                    title={t('syncTasks.projectTitle', {name: taskDialogProject.projectName})}
                    query={{projectName: taskDialogProject.projectName}}
                    onClose={() => setTaskDialogProject(null)}
                />
            ) : null}

            {selected ? (
                <SyncConfigDialog
                    open={!!selected}
                    projectName={selected.projectName}
                    projectPath={selected.path}
                    availableNodes={syncNodeStore.all.filter((n) => n.type === 'agent')}
                    initialSync={selected.sync}
                    saving={syncProjectStore.saving}
                    onClose={() => setSelected(null)}
                    onSave={async (sync) => {
                        await syncProjectStore.updateSyncConfig(selected.projectName, sync);
                        setSelected(null);
                    }}
                />
            ) : null}
        </>
    );
};

export default inject(
    'syncProjectStore',
    'syncNodeStore',
    'currentUser'
)(observer(SyncProjectsPage));
