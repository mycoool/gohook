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

    useEffect(() => {
        if (!open) return;
        syncTaskStore.loadTasks({...query, includeLogs}).catch(() => undefined);
    }, [open, includeLogs, query, syncTaskStore]);

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
                syncTaskStore.loadTasks({...query, includeLogs}).catch(() => undefined);
            }, 500);
        };

        wsStore.onMessage(handler);
        return () => {
            wsStore.offMessage(handler);
            if (timer) clearTimeout(timer);
        };
    }, [open, includeLogs, query, syncTaskStore, wsStore]);

    const tasks = useMemo(() => syncTaskStore.tasks || [], [syncTaskStore.tasks]);

    const toggleExpanded = (id: number) => {
        setExpanded((prev) => ({...prev, [id]: !prev[id]}));
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

    return (
        <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
            <DialogTitle>{title}</DialogTitle>
            <DialogContent>
                <Box sx={{display: 'flex', justifyContent: 'space-between', mb: 1}}>
                    <Typography variant="caption" color="textSecondary">
                        默认仅展示最近任务；勾选“包含日志”可查看任务日志（可能较大）。
                    </Typography>
                    <Button
                        size="small"
                        variant={includeLogs ? 'contained' : 'outlined'}
                        onClick={() => setIncludeLogs((v) => !v)}
                        disabled={syncTaskStore.loading}>
                        {includeLogs ? '包含日志' : '不含日志'}
                    </Button>
                </Box>

                {syncTaskStore.loading ? (
                    <Box sx={{display: 'flex', justifyContent: 'center', py: 4}}>
                        <CircularProgress />
                    </Box>
                ) : (
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>ID</TableCell>
                                <TableCell>项目</TableCell>
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
                                    <TableCell colSpan={8} align="center">
                                        暂无任务
                                    </TableCell>
                                </TableRow>
                            ) : (
                                tasks.map((t) => (
                                    <React.Fragment key={t.id}>
                                        <TableRow hover>
                                            <TableCell>{t.id}</TableCell>
                                            <TableCell>{t.projectName}</TableCell>
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
                                                {t.logs ? (
                                                    <IconButton
                                                        size="small"
                                                        onClick={() => toggleExpanded(t.id)}
                                                        title="展开/收起日志">
                                                        {expanded[t.id] ? (
                                                            <ExpandLessIcon fontSize="small" />
                                                        ) : (
                                                            <ExpandMoreIcon fontSize="small" />
                                                        )}
                                                    </IconButton>
                                                ) : (
                                                    '--'
                                                )}
                                            </TableCell>
                                        </TableRow>
                                        {t.logs ? (
                                            <TableRow>
                                                <TableCell colSpan={8} sx={{py: 0}}>
                                                    <Collapse
                                                        in={!!expanded[t.id]}
                                                        timeout="auto"
                                                        unmountOnExit>
                                                        <Box sx={{p: 1}}>
                                                            <pre
                                                                style={{
                                                                    margin: 0,
                                                                    maxHeight: 240,
                                                                    overflow: 'auto',
                                                                    whiteSpace: 'pre-wrap',
                                                                }}>
                                                                {t.logs}
                                                            </pre>
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
