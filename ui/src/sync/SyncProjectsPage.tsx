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

type Props = Stores<'syncProjectStore' | 'syncNodeStore' | 'currentUser'>;

const formatTime = (value?: string) => {
    if (!value) return 'N/A';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString();
};

const statusColor = (status: string) => {
    switch (status) {
        case 'HEALTHY':
            return 'success';
        case 'DEGRADED':
            return 'warning';
        case 'SYNCING':
            return 'info';
        case 'MISCONFIGURED':
            return 'error';
        default:
            return 'default';
    }
};

const SyncProjectsPage: React.FC<Props> = ({syncProjectStore, syncNodeStore, currentUser}) => {
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

    const triggerSync = async (p: ISyncProjectSummary) => {
        await syncProjectStore.runProjectSync(p.projectName);
    };

    return (
        <>
            <DefaultPage
                title="同步管理"
                maxWidth={1200}
                rightControl={
                    <ButtonGroup variant="contained">
                        <Button
                            startIcon={<RefreshIcon />}
                            onClick={() => syncProjectStore.refreshProjects()}
                            disabled={syncProjectStore.loading}>
                            刷新
                        </Button>
                    </ButtonGroup>
                }>
                <Paper elevation={6} sx={{mt: 0, width: '100%', overflowX: 'auto'}}>
                    <TableContainer>
                        <Table size="small" sx={{minWidth: 960}}>
                            <TableHead>
                                <TableRow>
                                    <TableCell>项目</TableCell>
                                    <TableCell>目录</TableCell>
                                    <TableCell>状态</TableCell>
                                    <TableCell>最后同步</TableCell>
                                    <TableCell>节点</TableCell>
                                    <TableCell align="right">操作</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {projects.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={6} align="center">
                                            {syncProjectStore.loading
                                                ? '加载中...'
                                                : '暂无同步项目（请在版本管理中开启同步）'}
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    projects.map((p) => (
                                        <TableRow key={p.projectName} hover>
                                            <TableCell>
                                                <Typography variant="body2" fontWeight={600}>
                                                    {p.projectName}
                                                </Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Typography variant="body2">{p.path}</Typography>
                                            </TableCell>
                                            <TableCell>
                                                <Tooltip
                                                    title={
                                                        p.status === 'DEGRADED'
                                                            ? (p.nodes || [])
                                                                  .filter(
                                                                      (n) =>
                                                                          String(
                                                                              n.lastStatus || ''
                                                                          ).toLowerCase() ===
                                                                              'failed' &&
                                                                          n.lastError
                                                                  )
                                                                  .map(
                                                                      (n) =>
                                                                          `${n.nodeName} (${
                                                                              n.targetPath
                                                                          })\n${n.lastError}${
                                                                              n.lastErrorCode
                                                                                  ? `\n[${n.lastErrorCode}]`
                                                                                  : ''
                                                                          }`
                                                                  )
                                                                  .join('\n\n')
                                                            : ''
                                                    }>
                                                    <span>
                                                        <Chip
                                                            label={p.status}
                                                            size="small"
                                                            color={statusColor(p.status)}
                                                        />
                                                    </span>
                                                </Tooltip>
                                            </TableCell>
                                            <TableCell>{formatTime(p.lastSyncAt)}</TableCell>
                                            <TableCell>
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    alignItems="center">
                                                    <Chip
                                                        size="small"
                                                        label={`${p.nodes?.length || 0}`}
                                                        variant="outlined"
                                                    />
                                                    <Typography
                                                        variant="caption"
                                                        color="textSecondary">
                                                        已绑定
                                                    </Typography>
                                                </Stack>
                                            </TableCell>
                                            <TableCell align="right">
                                                <Tooltip title="查看任务/错误详情">
                                                    <span>
                                                        <IconButton
                                                            size="small"
                                                            onClick={() => setTaskDialogProject(p)}
                                                            disabled={syncProjectStore.saving}>
                                                            <ListAltIcon fontSize="small" />
                                                        </IconButton>
                                                    </span>
                                                </Tooltip>
                                                <Tooltip title="管理同步配置">
                                                    <IconButton
                                                        size="small"
                                                        onClick={() => setSelected(p)}>
                                                        <SettingsIcon fontSize="small" />
                                                    </IconButton>
                                                </Tooltip>
                                                <Tooltip title="立即同步（手动触发）">
                                                    <span>
                                                        <IconButton
                                                            size="small"
                                                            onClick={() => triggerSync(p)}
                                                            disabled={syncProjectStore.saving}>
                                                            <SyncIcon fontSize="small" />
                                                        </IconButton>
                                                    </span>
                                                </Tooltip>
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
                    title={`任务详情 - ${taskDialogProject.projectName}`}
                    query={{projectName: taskDialogProject.projectName, limit: 50}}
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
