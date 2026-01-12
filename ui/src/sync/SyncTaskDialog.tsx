import React, {useEffect, useMemo, useState} from 'react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Button,
    CircularProgress,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    Typography,
    Chip,
    Stack,
    IconButton,
    Tooltip,
    Box,
    Collapse,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import {TaskQuery} from './SyncTaskStore';
import {ISyncTask, IWebSocketMessage} from '../types';
import useTranslation from '../i18n/useTranslation';

type Props = Stores<'syncTaskStore' | 'wsStore'> & {
    open: boolean;
    title: string;
    query: TaskQuery;
    onClose: () => void;
};

const formatTime = (value?: string) => {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString();
};

const formatBytes = (value?: number) => {
    const bytes = Number(value || 0);
    if (!Number.isFinite(bytes) || bytes <= 0) return '--';
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    let v = bytes;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
        v /= 1024;
        i++;
    }
    return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
};

const formatDuration = (ms?: number) => {
    const v = Number(ms || 0);
    if (!Number.isFinite(v) || v <= 0) return '--';
    if (v < 1000) return `${v}ms`;
    const sec = Math.round(v / 100) / 10;
    if (sec < 60) return `${sec}s`;
    const min = Math.floor(sec / 60);
    const rem = Math.round((sec - min * 60) * 10) / 10;
    return `${min}m ${rem}s`;
};

const statusColor = (status: string) => {
    switch ((status || '').toLowerCase()) {
        case 'success':
            return 'success';
        case 'failed':
            return 'error';
        case 'running':
            return 'info';
        case 'pending':
        case 'retrying':
            return 'warning';
        default:
            return 'default';
    }
};

const hintForCode = (code: string | undefined, translate: (key: string) => string) => {
    switch ((code || '').toUpperCase()) {
        case 'EACCES':
        case 'EPERM':
        case 'EROFS':
            return translate('syncTasks.hints.eacces');
        case 'ENOENT':
            return translate('syncTasks.hints.enoent');
        case 'ENOSPC':
            return translate('syncTasks.hints.enospc');
        case 'INVALID_TARGET':
            return translate('syncTasks.hints.invalidTarget');
        case 'PROTO':
            return translate('syncTasks.hints.proto');
        default:
            return '';
    }
};

const SyncTaskDialog: React.FC<Props> = ({open, title, query, onClose, syncTaskStore, wsStore}) => {
    const {t: translate} = useTranslation();
    const [includeLogs, setIncludeLogs] = useState(false);
    const [expanded, setExpanded] = useState<Record<number, boolean>>({});
    const [logPages, setLogPages] = useState<Record<number, number>>({});
    const [logLoading, setLogLoading] = useState<Record<number, boolean>>({});
    const [pageCursors, setPageCursors] = useState<number[]>([0]);
    const logPageSize = 5;
    const taskPageSize = 5;
    const emptyValue = translate('syncTasks.emptyValue');

    const queryKey = useMemo(
        () =>
            [
                query.projectName || '',
                String(query.nodeId || ''),
                query.status || '',
                String(query.limit || ''),
            ].join('|'),
        [query.projectName, query.nodeId, query.status, query.limit]
    );

    const beforeId = pageCursors[pageCursors.length - 1] || 0;
    const effectiveQuery = useMemo(
        () => ({
            projectName: query.projectName,
            nodeId: query.nodeId,
            status: query.status,
            includeLogs,
            limit: taskPageSize,
            ...(beforeId > 0 ? {beforeId} : {}),
        }),
        [query.projectName, query.nodeId, query.status, includeLogs, taskPageSize, beforeId]
    );

    useEffect(() => {
        if (!open) return;
        syncTaskStore.loadTasks(effectiveQuery).catch(() => undefined);
    }, [open, effectiveQuery, syncTaskStore]);

    useEffect(() => {
        if (!open) return;
        let timer: ReturnType<typeof setTimeout> | null = null;

        const handler = (message: IWebSocketMessage) => {
            if (message.type !== 'sync_task_event') return;
            const data = (message.data || {}) as {projectName?: string; nodeId?: number};
            if (query.projectName && data.projectName && data.projectName !== query.projectName) {
                return;
            }
            if (query.nodeId && data.nodeId && Number(data.nodeId) !== Number(query.nodeId)) {
                return;
            }
            if (timer) return;
            timer = setTimeout(() => {
                timer = null;
                syncTaskStore.loadTasks(effectiveQuery, {silent: true}).catch(() => undefined);
            }, 500);
        };

        wsStore.onMessage(handler);
        return () => {
            wsStore.offMessage(handler);
            if (timer) clearTimeout(timer);
        };
    }, [open, effectiveQuery, query.projectName, query.nodeId, syncTaskStore, wsStore]);

    const tasks = useMemo(() => syncTaskStore.tasks || [], [syncTaskStore.tasks]);
    const showProjectColumn = !query.projectName;

    const toggleExpanded = (task: ISyncTask) => {
        const id = Number(task.id);
        if (!Number.isFinite(id) || id <= 0) return;
        const next = !expanded[id];
        setExpanded((prev) => ({...prev, [id]: next}));
        if (next) {
            setLogPages((pages) => ({...pages, [id]: 0}));
            if (!includeLogs && !task.logs && !logLoading[id]) {
                setLogLoading((m) => ({...m, [id]: true}));
                syncTaskStore
                    .loadTask(id, {includeLogs: true})
                    .catch(() => undefined)
                    .finally(() => setLogLoading((m) => ({...m, [id]: false})));
            }
        }
    };

    const copyText = async (text: string) => {
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            // ignore
        }
    };

    const renderError = (task: ISyncTask) => {
        if (!task.lastError) return emptyValue;
        const hint = hintForCode(task.errorCode, translate);
        const content = hint ? `${task.lastError}\n${hint}` : task.lastError;
        return (
            <Stack direction="row" spacing={1} alignItems="center">
                <Tooltip
                    title={
                        <Box
                            sx={{
                                whiteSpace: 'pre-wrap',
                                maxWidth: 720,
                                maxHeight: 240,
                                overflow: 'auto',
                                m: 0,
                            }}>
                            {content}
                        </Box>
                    }
                    placement="top-start"
                    PopperProps={{
                        modifiers: [
                            {name: 'flip', enabled: true},
                            {name: 'preventOverflow', enabled: true},
                        ],
                    }}
                    componentsProps={{
                        tooltip: {
                            sx: {
                                maxWidth: 760,
                            },
                        },
                    }}>
                    <Typography
                        variant="body2"
                        sx={{
                            maxWidth: 360,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                        }}>
                        {task.lastError}
                    </Typography>
                </Tooltip>
                <Tooltip title={translate('syncTasks.copyError')}>
                    <IconButton size="small" onClick={() => copyText(content)}>
                        <ContentCopyIcon fontSize="inherit" />
                    </IconButton>
                </Tooltip>
            </Stack>
        );
    };

    const goOlder = () => {
        if (!tasks.length) return;
        const minID = Math.min(
            ...tasks.map((task) => Number(task.id)).filter((v) => Number.isFinite(v))
        );
        if (!Number.isFinite(minID) || minID <= 0) return;
        setPageCursors((prev) => [...prev, minID]);
    };

    const goNewer = () => {
        setPageCursors((prev) => (prev.length > 1 ? prev.slice(0, prev.length - 1) : prev));
    };

    const resetToLatest = () => {
        setPageCursors([0]);
    };

    const clearRecords = async () => {
        const scope = query.projectName
            ? translate('syncTasks.scope.project', {name: query.projectName})
            : query.nodeId
            ? translate('syncTasks.scope.node', {id: query.nodeId})
            : translate('syncTasks.scope.all');
        if (!window.confirm(translate('syncTasks.clearConfirm', {scope}))) {
            return;
        }
        try {
            await syncTaskStore.clearTasks(query);
            setExpanded({});
            setLogPages({});
            setLogLoading({});
            resetToLatest();
        } catch {
            window.alert(translate('syncTasks.clearFailed'));
        } finally {
            syncTaskStore
                .loadTasks({
                    projectName: query.projectName,
                    nodeId: query.nodeId,
                    status: query.status,
                    includeLogs,
                    limit: taskPageSize,
                })
                .catch(() => undefined);
        }
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
            <DialogTitle>{title}</DialogTitle>
            <DialogContent>
                <Box sx={{display: 'flex', justifyContent: 'space-between', mb: 1}}>
                    <Typography variant="caption" color="textSecondary">
                        {translate('syncTasks.recentHint', {count: taskPageSize})}
                    </Typography>
                    <Stack direction="row" spacing={1} alignItems="center">
                        {syncTaskStore.refreshing ? <CircularProgress size={14} /> : null}
                        <Button
                            size="small"
                            variant={includeLogs ? 'contained' : 'outlined'}
                            onClick={() => setIncludeLogs((v) => !v)}
                            disabled={syncTaskStore.loading}>
                            {includeLogs
                                ? translate('syncTasks.includeLogs')
                                : translate('syncTasks.excludeLogs')}
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            color="error"
                            onClick={clearRecords}
                            disabled={syncTaskStore.loading}>
                            {translate('syncTasks.clearRecords')}
                        </Button>
                    </Stack>
                </Box>

                <Box sx={{display: 'flex', justifyContent: 'space-between', mb: 1}}>
                    <Typography variant="caption" color="textSecondary">
                        {pageCursors.length > 1
                            ? translate('syncTasks.pageLabelWithBefore', {
                                  page: pageCursors.length,
                                  beforeId,
                              })
                            : translate('syncTasks.pageLabel', {page: pageCursors.length})}
                    </Typography>
                    <Stack direction="row" spacing={1}>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={resetToLatest}
                            disabled={syncTaskStore.loading || pageCursors.length <= 1}>
                            {translate('syncTasks.latest')}
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={goNewer}
                            disabled={syncTaskStore.loading || pageCursors.length <= 1}>
                            {translate('syncTasks.newer')}
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={goOlder}
                            disabled={syncTaskStore.loading || tasks.length < taskPageSize}>
                            {translate('syncTasks.older')}
                        </Button>
                    </Stack>
                </Box>

                {syncTaskStore.loading && tasks.length === 0 ? (
                    <Box sx={{display: 'flex', justifyContent: 'center', py: 4}}>
                        <CircularProgress />
                    </Box>
                ) : (
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>{translate('syncTasks.table.id')}</TableCell>
                                {showProjectColumn ? (
                                    <TableCell>{translate('syncTasks.table.project')}</TableCell>
                                ) : null}
                                <TableCell>{translate('syncTasks.table.node')}</TableCell>
                                <TableCell>{translate('syncTasks.table.status')}</TableCell>
                                <TableCell>{translate('syncTasks.table.time')}</TableCell>
                                <TableCell>{translate('syncTasks.table.transfer')}</TableCell>
                                <TableCell>{translate('syncTasks.table.error')}</TableCell>
                                <TableCell>{translate('syncTasks.table.logs')}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {tasks.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={showProjectColumn ? 8 : 7} align="center">
                                        {translate('syncTasks.empty')}
                                    </TableCell>
                                </TableRow>
                            ) : (
                                tasks.map((task) => (
                                    <React.Fragment key={task.id}>
                                        <TableRow hover>
                                            <TableCell>{task.id}</TableCell>
                                            {showProjectColumn ? (
                                                <TableCell>{task.projectName}</TableCell>
                                            ) : null}
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {task.nodeName} (#{task.nodeId})
                                                </Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    alignItems="center">
                                                    <Chip
                                                        size="small"
                                                        label={String(task.status).toUpperCase()}
                                                        color={statusColor(task.status)}
                                                    />
                                                    {task.errorCode ? (
                                                        <Chip
                                                            size="small"
                                                            variant="outlined"
                                                            label={task.errorCode}
                                                        />
                                                    ) : null}
                                                </Stack>
                                            </TableCell>
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {formatTime(task.updatedAt)}
                                                </Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {task.blocks
                                                        ? translate('syncTasks.blocks', {
                                                              count: task.blocks,
                                                          })
                                                        : emptyValue}
                                                </Typography>
                                                <Typography variant="caption" color="textSecondary">
                                                    {formatBytes(task.bytes)} ·{' '}
                                                    {formatDuration(task.durationMs)}
                                                </Typography>
                                            </TableCell>
                                            <TableCell>{renderError(task)}</TableCell>
                                            <TableCell>
                                                <IconButton
                                                    size="small"
                                                    onClick={() => toggleExpanded(task)}
                                                    title={
                                                        includeLogs
                                                            ? translate('syncTasks.logsToggle')
                                                            : translate('syncTasks.logsLazyLoad')
                                                    }>
                                                    {expanded[task.id] ? (
                                                        <ExpandLessIcon fontSize="small" />
                                                    ) : (
                                                        <ExpandMoreIcon fontSize="small" />
                                                    )}
                                                </IconButton>
                                            </TableCell>
                                        </TableRow>
                                        {expanded[task.id] || task.logs ? (
                                            <TableRow>
                                                <TableCell
                                                    colSpan={showProjectColumn ? 8 : 7}
                                                    sx={{py: 0}}>
                                                    <Collapse
                                                        in={!!expanded[task.id]}
                                                        timeout="auto"
                                                        unmountOnExit>
                                                        <Box sx={{p: 1}}>
                                                            {(() => {
                                                                if (logLoading[task.id]) {
                                                                    return (
                                                                        <Box
                                                                            sx={{
                                                                                display: 'flex',
                                                                                justifyContent:
                                                                                    'center',
                                                                                py: 2,
                                                                            }}>
                                                                            <CircularProgress
                                                                                size={20}
                                                                            />
                                                                        </Box>
                                                                    );
                                                                }
                                                                const raw = String(
                                                                    task.logs || ''
                                                                ).replace(/\n$/, '');
                                                                const lines =
                                                                    raw === ''
                                                                        ? []
                                                                        : raw.split(/\r?\n/);
                                                                const totalPages = Math.max(
                                                                    1,
                                                                    Math.ceil(
                                                                        lines.length / logPageSize
                                                                    )
                                                                );
                                                                const pageIndex = Math.min(
                                                                    Math.max(
                                                                        logPages[task.id] ?? 0,
                                                                        0
                                                                    ),
                                                                    totalPages - 1
                                                                );
                                                                const end = Math.max(
                                                                    0,
                                                                    lines.length -
                                                                        pageIndex * logPageSize
                                                                );
                                                                const start = Math.max(
                                                                    0,
                                                                    end - logPageSize
                                                                );
                                                                const view = lines
                                                                    .slice(start, end)
                                                                    .join('\n');

                                                                return (
                                                                    <>
                                                                        <Stack
                                                                            direction="row"
                                                                            spacing={1}
                                                                            alignItems="center"
                                                                            justifyContent="space-between"
                                                                            sx={{mb: 1}}>
                                                                            <Typography
                                                                                variant="caption"
                                                                                color="textSecondary">
                                                                                {translate(
                                                                                    'syncTasks.logPageHint',
                                                                                    {
                                                                                        count: logPageSize,
                                                                                        page: Math.max(
                                                                                            1,
                                                                                            totalPages -
                                                                                                pageIndex
                                                                                        ),
                                                                                        total: totalPages,
                                                                                    }
                                                                                )}
                                                                            </Typography>
                                                                            <Stack
                                                                                direction="row"
                                                                                spacing={1}
                                                                                alignItems="center">
                                                                                <Button
                                                                                    size="small"
                                                                                    variant="outlined"
                                                                                    disabled={
                                                                                        pageIndex >=
                                                                                        totalPages -
                                                                                            1
                                                                                    }
                                                                                    onClick={() =>
                                                                                        setLogPages(
                                                                                            (
                                                                                                pages
                                                                                            ) => ({
                                                                                                ...pages,
                                                                                                [task.id]:
                                                                                                    (pages[
                                                                                                        task
                                                                                                            .id
                                                                                                    ] ??
                                                                                                        0) +
                                                                                                    1,
                                                                                            })
                                                                                        )
                                                                                    }>
                                                                                    {translate(
                                                                                        'syncTasks.older'
                                                                                    )}
                                                                                </Button>
                                                                                <Button
                                                                                    size="small"
                                                                                    variant="outlined"
                                                                                    disabled={
                                                                                        pageIndex <=
                                                                                        0
                                                                                    }
                                                                                    onClick={() =>
                                                                                        setLogPages(
                                                                                            (
                                                                                                pages
                                                                                            ) => ({
                                                                                                ...pages,
                                                                                                [task.id]:
                                                                                                    Math.max(
                                                                                                        (pages[
                                                                                                            task
                                                                                                                .id
                                                                                                        ] ??
                                                                                                            0) -
                                                                                                            1,
                                                                                                        0
                                                                                                    ),
                                                                                            })
                                                                                        )
                                                                                    }>
                                                                                    {translate(
                                                                                        'syncTasks.newer'
                                                                                    )}
                                                                                </Button>
                                                                            </Stack>
                                                                        </Stack>
                                                                        <pre
                                                                            style={{
                                                                                margin: 0,
                                                                                maxHeight: 240,
                                                                                overflow: 'auto',
                                                                                whiteSpace:
                                                                                    'pre-wrap',
                                                                            }}>
                                                                            {view || emptyValue}
                                                                        </pre>
                                                                    </>
                                                                );
                                                            })()}
                                                        </Box>
                                                    </Collapse>
                                                </TableCell>
                                            </TableRow>
                                        ) : null}
                                    </React.Fragment>
                                ))
                            )}
                        </TableBody>
                    </Table>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} variant="contained" color="secondary">
                    {translate('common.close')}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default inject('syncTaskStore', 'wsStore')(observer(SyncTaskDialog));
