import React, {useState} from 'react';
import {
    Box,
    Button,
    Divider,
    FormControlLabel,
    IconButton,
    MenuItem,
    Stack,
    Switch,
    Tab,
    Tabs,
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
    strategy: string;
    ignorePatterns: string;
    ignoreFile: string;
    mirrorFastDelete: boolean;
    mirrorFastFullscanEvery: number | '';
    mirrorCleanEmptyDirs: boolean;
    mirrorSyncEmptyDirs: boolean;
};

interface Props {
    enabled: boolean;
    ignoreDefaults: boolean;
    ignorePermissions: boolean;
    preserveMode: boolean;
    preserveMtime: boolean;
    symlinkPolicy: 'ignore' | 'preserve';
    ignorePatterns: string;
    ignoreFile: string;
    deltaIndexOverlay: boolean;
    deltaMaxFiles: number | '';
    overlayFullScanEvery: number | '';
    overlayFullScanInterval: string;
    syncNodes: SyncNodeRow[];
    availableNodes: ISyncNode[];
    projectPath: string;
    onEnabledChange: (value: boolean) => void;
    onIgnoreDefaultsChange: (value: boolean) => void;
    onIgnorePermissionsChange: (value: boolean) => void;
    onPreserveModeChange: (value: boolean) => void;
    onPreserveMtimeChange: (value: boolean) => void;
    onSymlinkPolicyChange: (value: 'ignore' | 'preserve') => void;
    onIgnorePatternsChange: (value: string) => void;
    onIgnoreFileChange: (value: string) => void;
    onDeltaIndexOverlayChange: (value: boolean) => void;
    onDeltaMaxFilesChange: (value: number | '') => void;
    onOverlayFullScanEveryChange: (value: number | '') => void;
    onOverlayFullScanIntervalChange: (value: string) => void;
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
        strategy: row.strategy?.trim() || 'mirror',
        ignoreFile: row.ignoreFile.trim() || undefined,
        ignorePatterns: parseLines(row.ignorePatterns),
        mirrorFastDelete: row.mirrorFastDelete || undefined,
        mirrorFastFullscanEvery:
            row.mirrorFastFullscanEvery === '' ? undefined : Number(row.mirrorFastFullscanEvery),
        mirrorCleanEmptyDirs: row.mirrorCleanEmptyDirs || undefined,
        mirrorSyncEmptyDirs: row.mirrorSyncEmptyDirs || undefined,
    };
};

export const normalizeNodes = (rows: SyncNodeRow[]): IProjectSyncNodeConfig[] =>
    rows.map(normalizeNodeRow).filter((n) => n.nodeId && n.targetPath);

const SyncConfigEditor: React.FC<Props> = ({
    enabled,
    ignoreDefaults,
    ignorePermissions,
    preserveMode,
    preserveMtime,
    symlinkPolicy,
    ignorePatterns,
    ignoreFile,
    deltaIndexOverlay,
    deltaMaxFiles,
    overlayFullScanEvery,
    overlayFullScanInterval,
    syncNodes,
    availableNodes,
    projectPath,
    onEnabledChange,
    onIgnoreDefaultsChange,
    onIgnorePermissionsChange,
    onPreserveModeChange,
    onPreserveMtimeChange,
    onSymlinkPolicyChange,
    onIgnorePatternsChange,
    onIgnoreFileChange,
    onDeltaIndexOverlayChange,
    onDeltaMaxFilesChange,
    onOverlayFullScanEveryChange,
    onOverlayFullScanIntervalChange,
    onSyncNodesChange,
}) => {
    const [expanded, setExpanded] = useState<Record<number, boolean>>({});
    const [tab, setTab] = useState(0);

    const pickDefaultNodeId = (): string => {
        const picked = new Set(syncNodes.map((r) => r.nodeId).filter((v) => v));
        for (const n of availableNodes) {
            const id = String(n.id);
            if (!picked.has(id)) {
                return id;
            }
        }
        return availableNodes.length ? String(availableNodes[0].id) : '';
    };

    const addSyncNodeRow = () => {
        onSyncNodesChange([
            ...syncNodes,
            {
                nodeId: pickDefaultNodeId(),
                targetPath: projectPath || '',
                strategy: 'mirror',
                ignorePatterns: '',
                ignoreFile: '',
                mirrorFastDelete: false,
                mirrorFastFullscanEvery: '',
                mirrorCleanEmptyDirs: false,
                mirrorSyncEmptyDirs: false,
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
            <Tabs
                value={tab}
                onChange={(_, v) => setTab(v)}
                variant="scrollable"
                scrollButtons="auto">
                <Tab label="同步" />
                <Tab label="权限" />
                <Tab label="节点" />
                <Tab label="高级" />
            </Tabs>

            {tab === 0 ? (
                <Box mt={1}>
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
                                            onChange={(_, checked) =>
                                                onIgnoreDefaultsChange(checked)
                                            }
                                        />
                                    }
                                    label="启用默认忽略列表 (.git、runtime)"
                                />
                            </Box>

                            <TextField
                                margin="dense"
                                label="忽略规则（支持 # 注释 与 ! 反选）"
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
                        </>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            当前同步已关闭。开启后可配置忽略规则与同步节点。
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 1 ? (
                <Box mt={1}>
                    <Typography variant="subtitle1">权限</Typography>
                    {enabled ? (
                        <Box mt={1}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={ignorePermissions}
                                        onChange={(_, checked) =>
                                            onIgnorePermissionsChange(checked)
                                        }
                                    />
                                }
                                label="忽略权限与时间（不做 chmod/chown/mtime）"
                            />

                            <Collapse in={!ignorePermissions}>
                                <Box mt={1}>
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={preserveMode}
                                                onChange={(_, checked) =>
                                                    onPreserveModeChange(checked)
                                                }
                                            />
                                        }
                                        label="保留权限位（chmod）"
                                    />
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={preserveMtime}
                                                onChange={(_, checked) =>
                                                    onPreserveMtimeChange(checked)
                                                }
                                            />
                                        }
                                        label="保留修改时间（mtime）"
                                    />
                                </Box>
                            </Collapse>

                            <TextField
                                margin="dense"
                                select
                                label="符号链接策略"
                                value={symlinkPolicy}
                                onChange={(e) =>
                                    onSymlinkPolicyChange(
                                        e.target.value === 'preserve' ? 'preserve' : 'ignore'
                                    )
                                }
                                fullWidth>
                                <MenuItem value="ignore">忽略（默认）</MenuItem>
                                <MenuItem value="preserve">保留为 symlink</MenuItem>
                            </TextField>
                            <Typography variant="caption" color="textSecondary">
                                注意：Windows 下创建 symlink 可能需要管理员/开发者模式权限。
                            </Typography>
                        </Box>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            当前同步已关闭。开启后可配置权限与符号链接策略。
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 3 ? (
                <Box mt={1}>
                    <Typography variant="subtitle1">高级</Typography>
                    {enabled ? (
                        <Box mt={1}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={deltaIndexOverlay}
                                        onChange={(_, checked) =>
                                            onDeltaIndexOverlayChange(checked)
                                        }
                                    />
                                }
                                label="启用 Overlay 增量索引（高性能，需配合基线全量校验）"
                            />
                            <TextField
                                margin="dense"
                                label="增量索引最大文件数（超过自动回退全量）"
                                type="number"
                                value={deltaMaxFiles}
                                onChange={(e) =>
                                    onDeltaMaxFilesChange(
                                        e.target.value === '' ? '' : Number(e.target.value)
                                    )
                                }
                                fullWidth
                                placeholder="5000"
                            />
                            <TextField
                                margin="dense"
                                label="Overlay 基线全量索引：每 N 次任务强制全量（可选）"
                                type="number"
                                value={overlayFullScanEvery}
                                onChange={(e) =>
                                    onOverlayFullScanEveryChange(
                                        e.target.value === '' ? '' : Number(e.target.value)
                                    )
                                }
                                fullWidth
                                placeholder="10"
                                helperText="用于纠偏：子节点误删/损坏文件、watcher 漏事件等。"
                            />
                            <TextField
                                margin="dense"
                                label="Overlay 基线全量索引：最大间隔（duration，可选）"
                                value={overlayFullScanInterval}
                                onChange={(e) => onOverlayFullScanIntervalChange(e.target.value)}
                                fullWidth
                                placeholder="30m"
                            />
                        </Box>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            当前同步已关闭。开启后可配置增量索引与基线全量校验策略。
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 2 ? (
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
                                            select
                                            label="策略"
                                            value={row.strategy || 'mirror'}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {strategy: e.target.value})
                                            }
                                            fullWidth
                                            helperText="mirror：对齐源并删除额外文件；overlay：仅覆盖/新增，不删除目标端额外文件。">
                                            <MenuItem value="mirror">mirror</MenuItem>
                                            <MenuItem value="overlay">overlay</MenuItem>
                                        </TextField>
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
                                        {row.strategy === 'mirror' ? (
                                            <Box mt={1}>
                                                <Typography variant="subtitle2">
                                                    Mirror 优化（可选）
                                                </Typography>
                                                <FormControlLabel
                                                    control={
                                                        <Switch
                                                            checked={row.mirrorSyncEmptyDirs}
                                                            onChange={(_, checked) =>
                                                                updateNodeRow(idx, {
                                                                    mirrorSyncEmptyDirs: checked,
                                                                })
                                                            }
                                                        />
                                                    }
                                                    label="同步空目录（创建源端目录结构）"
                                                />
                                                <FormControlLabel
                                                    control={
                                                        <Switch
                                                            checked={row.mirrorFastDelete}
                                                            onChange={(_, checked) =>
                                                                updateNodeRow(idx, {
                                                                    mirrorFastDelete: checked,
                                                                })
                                                            }
                                                        />
                                                    }
                                                    label="快速删除（基于 manifest，性能更好）"
                                                />
                                                <TextField
                                                    margin="dense"
                                                    label="每 N 次强制全量扫描删除（可选）"
                                                    type="number"
                                                    value={row.mirrorFastFullscanEvery}
                                                    onChange={(e) =>
                                                        updateNodeRow(idx, {
                                                            mirrorFastFullscanEvery:
                                                                e.target.value === ''
                                                                    ? ''
                                                                    : Number(e.target.value),
                                                        })
                                                    }
                                                    fullWidth
                                                    placeholder="10"
                                                    disabled={!row.mirrorFastDelete}
                                                />
                                                <FormControlLabel
                                                    control={
                                                        <Switch
                                                            checked={row.mirrorCleanEmptyDirs}
                                                            onChange={(_, checked) =>
                                                                updateNodeRow(idx, {
                                                                    mirrorCleanEmptyDirs: checked,
                                                                })
                                                            }
                                                        />
                                                    }
                                                    label="清理空目录（删除文件后向上尝试移除空父目录）"
                                                />
                                            </Box>
                                        ) : null}
                                    </Box>
                                </Collapse>
                            </Box>
                        ))}
                    </Box>
                </Box>
            ) : null}
        </>
    );
};

export default SyncConfigEditor;
