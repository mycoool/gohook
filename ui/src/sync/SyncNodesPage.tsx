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
import ListAltIcon from '@mui/icons-material/ListAlt';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {inject, Stores} from '../inject';
import {ISyncNode} from '../types';
import {observer} from 'mobx-react';
import SyncNodeDialog from './SyncNodeDialog';
import {SyncNodePayload} from './SyncNodeStore';
import {useLocation} from 'react-router-dom';
import SyncTaskDialog from './SyncTaskDialog';
import useTranslation from '../i18n/useTranslation';

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

const connectionLabel = (node: ISyncNode, t: (key: string) => string) => {
    const status = String(node.connectionStatus || 'UNKNOWN').toUpperCase();
    switch (status) {
        case 'CONNECTED':
            return t('syncNodes.connection.connected');
        case 'DISCONNECTED':
            return t('syncNodes.connection.disconnected');
        case 'UNPAIRED':
            return t('syncNodes.connection.unpaired');
        default:
            return t('syncNodes.connection.unknown');
    }
};

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

const syncLabel = (node: ISyncNode, t: (key: string) => string) => {
    const status = String(node.syncStatus || 'IDLE').toUpperCase();
    switch (status) {
        case 'SUCCESS':
            return t('syncNodes.syncStatus.success');
        case 'FAILED':
            return t('syncNodes.syncStatus.failed');
        case 'RUNNING':
            return t('syncNodes.syncStatus.running');
        case 'RETRYING':
            return t('syncNodes.syncStatus.retrying');
        case 'PENDING':
            return t('syncNodes.syncStatus.pending');
        case 'IDLE':
            return t('syncNodes.syncStatus.idle');
        default:
            return status;
    }
};

const formatTime = (value: string | undefined, t: (key: string) => string) => {
    if (!value) {
        return t('syncNodes.notAvailable');
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    return date.toLocaleString();
};

const SyncNodesPage: React.FC<Props> = ({syncNodeStore, currentUser}) => {
    const {t} = useTranslation();
    const [dialogOpen, setDialogOpen] = useState(false);
    const [dialogMode, setDialogMode] = useState<'create' | 'edit'>('create');
    const [selectedNode, setSelectedNode] = useState<ISyncNode | null>(null);
    const [deleteTarget, setDeleteTarget] = useState<ISyncNode | null>(null);
    const [taskDialogNode, setTaskDialogNode] = useState<ISyncNode | null>(null);

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
        if (nodeId) {
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
            title={t('syncNodes.pageTitle')}
            maxWidth={1200}
            rightControl={
                <ButtonGroup variant="contained">
                    <Button
                        startIcon={<RefreshIcon />}
                        onClick={() => syncNodeStore.refreshNodes()}
                        disabled={syncNodeStore.loading}>
                        {t('common.refresh')}
                    </Button>
                    <Button startIcon={<AddIcon />} onClick={openCreateDialog}>
                        {t('syncNodes.addNode')}
                    </Button>
                </ButtonGroup>
            }>
            <Grid size={12}>
                <Paper elevation={6} sx={{mt: 0, width: '100%', overflowX: 'auto'}}>
                    <TableContainer>
                        <Table size="small" sx={{minWidth: 960}}>
                            <TableHead>
                                <TableRow>
                                    <TableCell>{t('syncNodes.table.name')}</TableCell>
                                    <TableCell>{t('syncNodes.table.address')}</TableCell>
                                    <TableCell>{t('syncNodes.table.remark')}</TableCell>
                                    <TableCell>{t('syncNodes.table.connection')}</TableCell>
                                    <TableCell>{t('syncNodes.table.sync')}</TableCell>
                                    <TableCell>{t('syncNodes.table.tags')}</TableCell>
                                    <TableCell
                                        align="left"
                                        style={{whiteSpace: 'nowrap', width: 1}}>
                                        {t('common.actions')}
                                    </TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {nodes.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={7}>
                                            <Typography align="center" color="textSecondary">
                                                {syncNodeStore.loading
                                                    ? t('syncNodes.loading')
                                                    : t('syncNodes.empty')}
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
                                                            ? t('syncNodes.typeAgent')
                                                            : t('syncNodes.typeSSH')}
                                                    </Typography>
                                                </Stack>
                                            </TableCell>
                                            <TableCell>
                                                {node.address || t('syncNodes.notAvailable')}
                                            </TableCell>
                                            <TableCell>{node.remark || t('syncNodes.notAvailable')}</TableCell>
                                            <TableCell>
                                                <Chip
                                                    label={connectionLabel(node, t)}
                                                    variant="outlined"
                                                    size="small"
                                                    color={connectionColor(node.connectionStatus)}
                                                />
                                            </TableCell>
                                            <TableCell>
                                                <Stack spacing={0.5}>
                                                    {(() => {
                                                        const detail =
                                                            node.syncStatus === 'FAILED' &&
                                                            node.lastError
                                                                ? `${node.lastTaskProject || ''} ${
                                                                      node.lastTaskTargetPath || ''
                                                                  }\n${node.lastError}${
                                                                      node.lastErrorCode
                                                                          ? `\n[${node.lastErrorCode}]`
                                                                          : ''
                                                                  }`
                                                                : '';
                                                        return (
                                                            <Tooltip
                                                                title={
                                                                    detail ? (
                                                                        <pre
                                                                            style={{
                                                                                margin: 0,
                                                                                whiteSpace:
                                                                                    'pre-wrap',
                                                                            }}>
                                                                            {detail}
                                                                        </pre>
                                                                    ) : (
                                                                        ''
                                                                    )
                                                                }>
                                                                <span>
                                                                    <Chip
                                                                        label={syncLabel(node, t)}
                                                                        size="small"
                                                                        color={syncColor(
                                                                            node.syncStatus
                                                                        )}
                                                                    />
                                                                </span>
                                                            </Tooltip>
                                                        );
                                                    })()}
                                                    {node.lastSyncAt ? (
                                                        <Typography
                                                            variant="caption"
                                                            color="textSecondary">
                                                            {formatTime(node.lastSyncAt, t)}
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
                                                    : t('syncNodes.none')}
                                            </TableCell>
                                            <TableCell
                                                align="left"
                                                style={{whiteSpace: 'nowrap', width: 1}}>
                                                <div
                                                    style={{
                                                        display: 'inline-flex',
                                                        alignItems: 'center',
                                                    }}>
                                                    <Tooltip title={t('syncNodes.tooltips.taskDetails')}>
                                                        <span>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    setTaskDialogNode(node)
                                                                }
                                                                disabled={syncNodeStore.saving}>
                                                                <ListAltIcon fontSize="small" />
                                                            </IconButton>
                                                        </span>
                                                    </Tooltip>
                                                    <Tooltip title={t('syncNodes.tooltips.resetPairing')}>
                                                        <span>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    syncNodeStore.resetPairing(
                                                                        node.id
                                                                    )
                                                                }
                                                                disabled={
                                                                    syncNodeStore.saving ||
                                                                    node.type !== 'agent'
                                                                }>
                                                                <LinkOffIcon fontSize="small" />
                                                            </IconButton>
                                                        </span>
                                                    </Tooltip>
                                                    <Tooltip title={t('syncNodes.tooltips.edit')}>
                                                        <IconButton
                                                            size="small"
                                                            onClick={() => openEditDialog(node)}>
                                                            <EditIcon fontSize="small" />
                                                        </IconButton>
                                                    </Tooltip>
                                                    <Tooltip title={t('syncNodes.tooltips.delete')}>
                                                        <IconButton
                                                            size="small"
                                                            color="error"
                                                            onClick={() => setDeleteTarget(node)}>
                                                            <DeleteIcon fontSize="small" />
                                                        </IconButton>
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

            {taskDialogNode ? (
                <SyncTaskDialog
                    open={!!taskDialogNode}
                    title={t('syncTasks.title', {name: taskDialogNode.name})}
                    query={{nodeId: taskDialogNode.id}}
                    onClose={() => setTaskDialogNode(null)}
                />
            ) : null}

            {deleteTarget ? (
                <ConfirmDialog
                    title={t('syncNodes.deleteTitle')}
                    text={t('syncNodes.deleteConfirm', {name: deleteTarget.name})}
                    fClose={() => setDeleteTarget(null)}
                    fOnSubmit={handleDelete}
                />
            ) : null}
        </DefaultPage>
    );
};

export default inject('syncNodeStore', 'currentUser')(observer(SyncNodesPage));
