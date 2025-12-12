import React, {useMemo, useState} from 'react';
import {
    Box,
    Button,
    Divider,
    FormControlLabel,
    IconButton,
    MenuItem,
    Stack,
    Switch,
    TextField,
    Typography,
    Collapse,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import SettingsIcon from '@mui/icons-material/Settings';
import CloseIcon from '@mui/icons-material/Close';
import {IProjectSyncNodeConfig, ISyncNode} from '../types';

export type SyncNodeRow = {
    nodeId: string;
    targetPath: string;
    ignorePatterns: string;
    ignoreFile: string;
};

interface Props {
    enabled: boolean;
    ignoreDefaults: boolean;
    ignorePermissions: boolean;
    ignorePatterns: string;
    ignoreFile: string;
    syncNodes: SyncNodeRow[];
    availableNodes: ISyncNode[];
    projectPath: string;
    onEnabledChange: (value: boolean) => void;
    onIgnoreDefaultsChange: (value: boolean) => void;
    onIgnorePermissionsChange: (value: boolean) => void;
    onIgnorePatternsChange: (value: string) => void;
    onIgnoreFileChange: (value: string) => void;
    onSyncNodesChange: (value: SyncNodeRow[]) => void;
}

const normalizeNodeRow = (row: SyncNodeRow): IProjectSyncNodeConfig => {
    const parseLines = (value: string): string[] =>
        value
            .split('\n')
            .map((s) => s.trim())
            .filter((s) => s.length > 0);
    return {
        nodeId: row.nodeId.trim(),
        targetPath: row.targetPath.trim(),
        ignoreFile: row.ignoreFile.trim() || undefined,
        ignorePatterns: parseLines(row.ignorePatterns),
    };
};

export const normalizeNodes = (rows: SyncNodeRow[]): IProjectSyncNodeConfig[] =>
    rows.map(normalizeNodeRow).filter((n) => n.nodeId && n.targetPath);

const SyncConfigEditor: React.FC<Props> = ({
    enabled,
    ignoreDefaults,
    ignorePermissions,
    ignorePatterns,
    ignoreFile,
    syncNodes,
    availableNodes,
    projectPath,
    onEnabledChange,
    onIgnoreDefaultsChange,
    onIgnorePermissionsChange,
    onIgnorePatternsChange,
    onIgnoreFileChange,
    onSyncNodesChange,
}) => {
    const [expanded, setExpanded] = useState<Record<number, boolean>>({});

    const defaultNodeId = useMemo(
        () => (availableNodes.length ? String(availableNodes[0].id) : ''),
        [availableNodes]
    );

    const addSyncNodeRow = () => {
        onSyncNodesChange([
            ...syncNodes,
            {
                nodeId: defaultNodeId,
                targetPath: projectPath || '',
                ignorePatterns: '',
                ignoreFile: '',
            },
        ]);
    };

    const updateNodeRow = (idx: number, patch: Partial<SyncNodeRow>) => {
        onSyncNodesChange(syncNodes.map((row, i) => (i === idx ? {...row, ...patch} : row)));
    };

    const removeNodeRow = (idx: number) => {
        onSyncNodesChange(syncNodes.filter((_, i) => i !== idx));
    };

    const toggleExpanded = (idx: number) => {
        setExpanded((prev) => ({...prev, [idx]: !prev[idx]}));
    };

    return (
        <>
            <Divider sx={{my: 1}} />
            <Stack direction="row" alignItems="center" justifyContent="space-between">
                <Typography variant="subtitle1">同步</Typography>
                <FormControlLabel
                    control={
                        <Switch
                            checked={enabled}
                            onChange={(_, checked) => onEnabledChange(checked)}
                        />
                    }
                    label="启用"
                />
            </Stack>

            {enabled ? (
                <>
                    <Box mt={1}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={ignoreDefaults}
                                    onChange={(_, checked) => onIgnoreDefaultsChange(checked)}
                                />
                            }
                            label="启用默认忽略列表 (.git、runtime、tmp)"
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={ignorePermissions}
                                    onChange={(_, checked) => onIgnorePermissionsChange(checked)}
                                />
                            }
                            label="忽略权限变更 (chmod/chown)"
                        />
                    </Box>

                    <TextField
                        margin="dense"
                        label="忽略规则（Syncthing 风格，支持 # 注释 与 ! 反选）"
                        value={ignorePatterns}
                        onChange={(e) => onIgnorePatternsChange(e.target.value)}
                        fullWidth
                        multiline
                        minRows={4}
                        placeholder={`# example\n.env\n!keep.env\nnode_modules/**\n*.log`}
                        helperText="无“/”的规则默认匹配任意目录层级；支持 **。"
                    />

                    <TextField
                        margin="dense"
                        label="忽略文件路径（可选）"
                        value={ignoreFile}
                        onChange={(e) => onIgnoreFileChange(e.target.value)}
                        fullWidth
                        placeholder=".stignore"
                        helperText="路径相对于主节点项目目录；内容同样支持 # 与 !。"
                    />

                    <Box mt={2}>
                        <Stack direction="row" alignItems="center" justifyContent="space-between">
                            <Typography variant="subtitle2">同步节点</Typography>
                            <Button
                                size="small"
                                startIcon={<AddIcon />}
                                onClick={addSyncNodeRow}
                                disabled={availableNodes.length === 0}>
                                添加节点
                            </Button>
                        </Stack>
                        {availableNodes.length === 0 ? (
                            <Typography variant="caption" color="textSecondary">
                                还没有可用节点，请先在“节点管理”里创建节点。
                            </Typography>
                        ) : null}
                        <Box sx={{mt: 1}}>
                            {syncNodes.map((row, idx) => (
                                <Box
                                    key={`${row.nodeId}-${idx}`}
                                    sx={{
                                        border: '1px solid rgba(0,0,0,0.08)',
                                        borderRadius: 1,
                                        p: 1,
                                        mb: 1,
                                    }}>
                                    <Box
                                        sx={{
                                            display: 'flex',
                                            gap: 1,
                                            alignItems: 'center',
                                        }}>
                                        <TextField
                                            select
                                            label="节点"
                                            value={row.nodeId}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {nodeId: e.target.value})
                                            }
                                            size="small"
                                            sx={{minWidth: 180}}>
                                            {availableNodes.map((n) => (
                                                <MenuItem key={n.id} value={String(n.id)}>
                                                    {n.name}
                                                </MenuItem>
                                            ))}
                                        </TextField>
                                        <TextField
                                            label="目标目录"
                                            value={row.targetPath}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {targetPath: e.target.value})
                                            }
                                            size="small"
                                            fullWidth
                                            placeholder={projectPath || '/www/wwwroot/app'}
                                        />
                                        <IconButton
                                            size="small"
                                            onClick={() => toggleExpanded(idx)}
                                            title="节点同步设置">
                                            {expanded[idx] ? (
                                                <CloseIcon fontSize="small" />
                                            ) : (
                                                <SettingsIcon fontSize="small" />
                                            )}
                                        </IconButton>
                                        <IconButton
                                            size="small"
                                            onClick={() => removeNodeRow(idx)}
                                            title="移除"
                                            color="error">
                                            <DeleteIcon fontSize="small" />
                                        </IconButton>
                                    </Box>

                                    <Collapse in={!!expanded[idx]} timeout="auto" unmountOnExit>
                                        <Box mt={1}>
                                            <TextField
                                                margin="dense"
                                                label="节点额外忽略规则（可选）"
                                                value={row.ignorePatterns}
                                                onChange={(e) =>
                                                    updateNodeRow(idx, {
                                                        ignorePatterns: e.target.value,
                                                    })
                                                }
                                                fullWidth
                                                multiline
                                                minRows={3}
                                                placeholder={`# only for this node\n*.local\n!important.local`}
                                                helperText="与项目级规则合并后按顺序生效（支持 ! 反选）。"
                                            />
                                            <TextField
                                                margin="dense"
                                                label="节点忽略文件路径（可选）"
                                                value={row.ignoreFile}
                                                onChange={(e) =>
                                                    updateNodeRow(idx, {ignoreFile: e.target.value})
                                                }
                                                fullWidth
                                                placeholder=".stignore.node"
                                            />
                                        </Box>
                                    </Collapse>
                                </Box>
                            ))}
                        </Box>
                    </Box>
                </>
            ) : null}
        </>
    );
};

export default SyncConfigEditor;
