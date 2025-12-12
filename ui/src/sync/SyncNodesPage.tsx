import React, {useEffect, useMemo, useState} from 'react';
import {
    Box,
    Button,
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
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import ButtonGroup from '@mui/material/ButtonGroup';
import AddIcon from '@mui/icons-material/Add';
import RefreshIcon from '@mui/icons-material/Refresh';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import LinkOffIcon from '@mui/icons-material/LinkOff';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {inject, Stores} from '../inject';
import {ISyncNode} from '../types';
import {observer} from 'mobx-react';
import SyncNodeDialog from './SyncNodeDialog';
import {SyncNodePayload} from './SyncNodeStore';
import {useLocation} from 'react-router-dom';

type Props = Stores<'syncNodeStore' | 'currentUser'>;

const connectionColor = (status?: string) => {
    switch (status) {
        case 'CONNECTED':
            return 'success';
        case 'DISCONNECTED':
            return 'default';
        case 'UNPAIRED':
            return 'warning';
        default:
            return 'info';
    }
};

const connectionLabel = (node: ISyncNode) => node.connectionStatus || 'UNKNOWN';

const syncColor = (status?: string) => {
    switch ((status || '').toUpperCase()) {
        case 'SUCCESS':
            return 'success';
        case 'FAILED':
            return 'error';
        case 'RUNNING':
            return 'info';
        case 'RETRYING':
        case 'PENDING':
            return 'warning';
        case 'IDLE':
            return 'default';
        default:
            return 'default';
    }
};

const syncLabel = (node: ISyncNode) => (node.syncStatus || 'IDLE').toUpperCase();

const formatTime = (value?: string) => {
    if (!value) {
        return 'N/A';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    return date.toLocaleString();
};

const SyncNodesPage: React.FC<Props> = ({syncNodeStore, currentUser}) => {
    const [dialogOpen, setDialogOpen] = useState(false);
    const [dialogMode, setDialogMode] = useState<'create' | 'edit'>('create');
    const [selectedNode, setSelectedNode] = useState<ISyncNode | null>(null);
    const [deleteTarget, setDeleteTarget] = useState<ISyncNode | null>(null);

    const location = useLocation();
    const highlightId = useMemo(() => {
        const params = new URLSearchParams(location.search);
        const id = Number(params.get('node'));
        return Number.isFinite(id) ? id : null;
    }, [location.search]);

    useEffect(() => {
        if (currentUser.loggedIn && syncNodeStore.all.length === 0 && !syncNodeStore.loading) {
            syncNodeStore.refreshNodes().catch(() => undefined);
        }
    }, [currentUser.loggedIn, syncNodeStore]);

    const nodes = syncNodeStore.all;

    const openCreateDialog = () => {
        setSelectedNode(null);
        setDialogMode('create');
        setDialogOpen(true);
    };

    const openEditDialog = (node: ISyncNode) => {
        setSelectedNode(node);
        setDialogMode('edit');
        setDialogOpen(true);
    };

    const handleSubmit = async (payload: SyncNodePayload, nodeId?: number) => {
        if (dialogMode === 'edit' && nodeId) {
            await syncNodeStore.updateNode(nodeId, payload);
            const updated = syncNodeStore.all.find((n) => n.id === nodeId);
            return updated;
        } else {
            return await syncNodeStore.createNode(payload);
        }
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await syncNodeStore.deleteNode(deleteTarget.id);
        setDeleteTarget(null);
    };

    return (
        <DefaultPage
            title="节点管理"
            maxWidth={1200}
            rightControl={
                <ButtonGroup variant="contained">
                    <Button
                        startIcon={<RefreshIcon />}
                        onClick={() => syncNodeStore.refreshNodes()}
                        disabled={syncNodeStore.loading}>
                        刷新
                    </Button>
                    <Button startIcon={<AddIcon />} onClick={openCreateDialog}>
                        新增节点
                    </Button>
                </ButtonGroup>
            }>
            <Grid size={12}>
                <Paper elevation={6} sx={{mt: 0, width: '100%', overflowX: 'auto'}}>
                    <TableContainer>
                        <Table size="small" sx={{minWidth: 960}}>
                            <TableHead>
                                <TableRow>
                                    <TableCell>名称</TableCell>
                                    <TableCell>地址</TableCell>
                                    <TableCell>连接状态</TableCell>
                                    <TableCell>同步状态</TableCell>
                                    <TableCell>标签</TableCell>
                                    <TableCell align="right">操作</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {nodes.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={6}>
                                            <Typography align="center" color="textSecondary">
                                                {syncNodeStore.loading
                                                    ? '加载节点中...'
                                                    : '暂无节点，请点击“新增节点”'}
                                            </Typography>
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    nodes.map((node) => (
                                        <TableRow
                                            key={node.id}
                                            selected={highlightId === node.id}
                                            hover>
                                            <TableCell>
                                                <Stack spacing={0.5}>
                                                    <Typography variant="subtitle2">
                                                        {node.name}
                                                    </Typography>
                                                    <Typography
                                                        variant="caption"
                                                        color="textSecondary">
                                                        {node.type === 'agent'
                                                            ? 'Sync Agent'
                                                            : 'LEGACY'}
                                                    </Typography>
                                                </Stack>
                                            </TableCell>
                                            <TableCell>{node.address}</TableCell>
                                            <TableCell>
                                                <Chip
                                                    label={connectionLabel(node)}
                                                    variant="outlined"
                                                    size="small"
                                                    color={connectionColor(node.connectionStatus)}
                                                />
                                            </TableCell>
                                            <TableCell>
                                                <Stack spacing={0.5}>
                                                    <Chip
                                                        label={syncLabel(node)}
                                                        size="small"
                                                        color={syncColor(node.syncStatus)}
                                                    />
                                                    {node.lastSyncAt ? (
                                                        <Typography
                                                            variant="caption"
                                                            color="textSecondary">
                                                            {formatTime(node.lastSyncAt)}
                                                        </Typography>
                                                    ) : null}
                                                </Stack>
                                            </TableCell>
                                            <TableCell>
                                                {node.tags?.length
                                                    ? node.tags.map((tag) => (
                                                          <Chip
                                                              key={tag}
                                                              label={tag}
                                                              size="small"
                                                              sx={{mr: 0.5, mb: 0.5}}
                                                          />
                                                      ))
                                                    : '--'}
                                            </TableCell>
                                            <TableCell align="right">
                                                <Tooltip title="重置配对（清空 mTLS 指纹，等待 Agent 重新连接）">
                                                    <span>
                                                        <IconButton
                                                            size="small"
                                                            onClick={() =>
                                                                syncNodeStore.resetPairing(node.id)
                                                            }
                                                            disabled={syncNodeStore.saving}>
                                                            <LinkOffIcon fontSize="small" />
                                                        </IconButton>
                                                    </span>
                                                </Tooltip>
                                                <Tooltip title="编辑节点">
                                                    <IconButton
                                                        size="small"
                                                        onClick={() => openEditDialog(node)}>
                                                        <EditIcon fontSize="small" />
                                                    </IconButton>
                                                </Tooltip>
                                                <Tooltip title="删除节点">
                                                    <IconButton
                                                        size="small"
                                                        color="error"
                                                        onClick={() => setDeleteTarget(node)}>
                                                        <DeleteIcon fontSize="small" />
                                                    </IconButton>
                                                </Tooltip>
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Paper>
            </Grid>

            <SyncNodeDialog
                open={dialogOpen}
                loading={syncNodeStore.saving}
                mode={dialogMode}
                node={selectedNode}
                onClose={() => setDialogOpen(false)}
                onSubmit={handleSubmit}
                onRotateToken={(id) => syncNodeStore.rotateToken(id)}
            />

            {deleteTarget ? (
                <ConfirmDialog
                    title="删除节点"
                    text={`确认删除节点 ${deleteTarget.name} 吗？`}
                    fClose={() => setDeleteTarget(null)}
                    fOnSubmit={handleDelete}
                />
            ) : null}
        </DefaultPage>
    );
};

export default inject('syncNodeStore', 'currentUser')(observer(SyncNodesPage));
