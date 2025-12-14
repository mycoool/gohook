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

const hintForCode = (code?: string) => {
    switch ((code || '').toUpperCase()) {
        case 'EACCES':
        case 'EPERM':
        case 'EROFS':
            return '可能原因：目标目录无写入权限/只读文件系统（chown/chmod 或更换 targetPath）';
        case 'ENOENT':
            return '可能原因：目标目录不存在（检查 targetPath 或上级目录权限）';
        case 'ENOSPC':
            return '可能原因：磁盘空间不足（清理空间）';
        case 'INVALID_TARGET':
            return '可能原因：targetPath 配置不合法（不能为空或 /）';
        case 'PROTO':
            return '可能原因：连接/协议异常（检查主节点与 Agent 版本）';
        default:
            return '';
    }
};

const SyncTaskDialog: React.FC<Props> = ({open, title, query, onClose, syncTaskStore, wsStore}) => {
    const [includeLogs, setIncludeLogs] = useState(false);
    const [expanded, setExpanded] = useState<Record<number, boolean>>({});
    const [logPages, setLogPages] = useState<Record<number, number>>({});
    const [logLoading, setLogLoading] = useState<Record<number, boolean>>({});
    const [pageCursors, setPageCursors] = useState<number[]>([0]);
    const logPageSize = 5;
    const taskPageSize = 5;

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

    const toggleExpanded = (t: ISyncTask) => {
        const id = Number(t.id);
        if (!Number.isFinite(id) || id <= 0) return;
        const next = !expanded[id];
        setExpanded((prev) => ({...prev, [id]: next}));
        if (next) {
            setLogPages((pages) => ({...pages, [id]: 0}));
            if (!includeLogs && !t.logs && !logLoading[id]) {
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

    const renderError = (t: ISyncTask) => {
        if (!t.lastError) return '--';
        const hint = hintForCode(t.errorCode);
        const content = hint ? `${t.lastError}\n${hint}` : t.lastError;
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
                        {t.lastError}
                    </Typography>
                </Tooltip>
                <Tooltip title="复制错误">
                    <IconButton size="small" onClick={() => copyText(content)}>
                        <ContentCopyIcon fontSize="inherit" />
                    </IconButton>
                </Tooltip>
            </Stack>
        );
    };

    const goOlder = () => {
        if (!tasks.length) return;
        const minID = Math.min(...tasks.map((t) => Number(t.id)).filter((v) => Number.isFinite(v)));
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
            ? `项目 ${query.projectName}`
            : query.nodeId
            ? `节点 #${query.nodeId}`
            : '全部';
        if (!window.confirm(`确认清空 ${scope} 的任务记录（默认仅清空 success/failed）？`)) {
            return;
        }
        try {
            await syncTaskStore.clearTasks(query);
            setExpanded({});
            setLogPages({});
            setLogLoading({});
            resetToLatest();
        } catch {
            window.alert('清空失败（需要管理员权限或服务端异常）');
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
                        默认展示最近 {taskPageSize} 条任务；翻页可查看更早记录。
                    </Typography>
                    <Stack direction="row" spacing={1} alignItems="center">
                        {syncTaskStore.refreshing ? <CircularProgress size={14} /> : null}
                        <Button
                            size="small"
                            variant={includeLogs ? 'contained' : 'outlined'}
                            onClick={() => setIncludeLogs((v) => !v)}
                            disabled={syncTaskStore.loading}>
                            {includeLogs ? '包含日志' : '不含日志'}
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            color="error"
                            onClick={clearRecords}
                            disabled={syncTaskStore.loading}>
                            清空记录
                        </Button>
                    </Stack>
                </Box>

                <Box sx={{display: 'flex', justifyContent: 'space-between', mb: 1}}>
                    <Typography variant="caption" color="textSecondary">
                        第 {pageCursors.length} 页
                        {pageCursors.length > 1 ? `（beforeId=${beforeId}）` : ''}
                    </Typography>
                    <Stack direction="row" spacing={1}>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={resetToLatest}
                            disabled={syncTaskStore.loading || pageCursors.length <= 1}>
                            最新
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={goNewer}
                            disabled={syncTaskStore.loading || pageCursors.length <= 1}>
                            下一页
                        </Button>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={goOlder}
                            disabled={syncTaskStore.loading || tasks.length < taskPageSize}>
                            上一页
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
                                <TableCell>ID</TableCell>
                                {showProjectColumn ? <TableCell>项目</TableCell> : null}
                                <TableCell>节点</TableCell>
                                <TableCell>状态</TableCell>
                                <TableCell>时间</TableCell>
                                <TableCell>传输</TableCell>
                                <TableCell>错误</TableCell>
                                <TableCell>日志</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {tasks.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={showProjectColumn ? 8 : 7} align="center">
                                        暂无任务
                                    </TableCell>
                                </TableRow>
                            ) : (
                                tasks.map((t) => (
                                    <React.Fragment key={t.id}>
                                        <TableRow hover>
                                            <TableCell>{t.id}</TableCell>
                                            {showProjectColumn ? (
                                                <TableCell>{t.projectName}</TableCell>
                                            ) : null}
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {t.nodeName} (#{t.nodeId})
                                                </Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    alignItems="center">
                                                    <Chip
                                                        size="small"
                                                        label={String(t.status).toUpperCase()}
                                                        color={statusColor(t.status)}
                                                    />
                                                    {t.errorCode ? (
                                                        <Chip
                                                            size="small"
                                                            variant="outlined"
                                                            label={t.errorCode}
                                                        />
                                                    ) : null}
                                                </Stack>
                                            </TableCell>
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {formatTime(t.updatedAt)}
                                                </Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Typography variant="body2">
                                                    {t.blocks ? `${t.blocks} blocks` : '--'}
                                                </Typography>
                                                <Typography variant="caption" color="textSecondary">
                                                    {formatBytes(t.bytes)} ·{' '}
                                                    {formatDuration(t.durationMs)}
                                                </Typography>
                                            </TableCell>
                                            <TableCell>{renderError(t)}</TableCell>
                                            <TableCell>
                                                <IconButton
                                                    size="small"
                                                    onClick={() => toggleExpanded(t)}
                                                    title={
                                                        includeLogs
                                                            ? '展开/收起日志'
                                                            : '展开后按需加载日志'
                                                    }>
                                                    {expanded[t.id] ? (
                                                        <ExpandLessIcon fontSize="small" />
                                                    ) : (
                                                        <ExpandMoreIcon fontSize="small" />
                                                    )}
                                                </IconButton>
                                            </TableCell>
                                        </TableRow>
                                        {expanded[t.id] || t.logs ? (
                                            <TableRow>
                                                <TableCell
                                                    colSpan={showProjectColumn ? 8 : 7}
                                                    sx={{py: 0}}>
                                                    <Collapse
                                                        in={!!expanded[t.id]}
                                                        timeout="auto"
                                                        unmountOnExit>
                                                        <Box sx={{p: 1}}>
                                                            {(() => {
                                                                if (logLoading[t.id]) {
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
                                                                    t.logs || ''
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
                                                                        logPages[t.id] ?? 0,
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
                                                                                默认显示最新{' '}
                                                                                {logPageSize} 行 ·
                                                                                第{' '}
                                                                                {Math.max(
                                                                                    1,
                                                                                    totalPages -
                                                                                        pageIndex
                                                                                )}
                                                                                /{totalPages} 页
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
                                                                                                [t.id]:
                                                                                                    (pages[
                                                                                                        t
                                                                                                            .id
                                                                                                    ] ??
                                                                                                        0) +
                                                                                                    1,
                                                                                            })
                                                                                        )
                                                                                    }>
                                                                                    上一页
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
                                                                                                [t.id]:
                                                                                                    Math.max(
                                                                                                        (pages[
                                                                                                            t
                                                                                                                .id
                                                                                                        ] ??
                                                                                                            0) -
                                                                                                            1,
                                                                                                        0
                                                                                                    ),
                                                                                            })
                                                                                        )
                                                                                    }>
                                                                                    下一页
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
                                                                            {view || '--'}
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
                    关闭
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default inject('syncTaskStore', 'wsStore')(observer(SyncTaskDialog));
