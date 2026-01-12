import React, {useState} from 'react';
import {
    Box,
    Button,
    Divider,
    FormControlLabel,
    IconButton,
    MenuItem,
    Paper,
    Stack,
    Switch,
    Tab,
    Tabs,
    TextField,
    Typography,
    Collapse,
} from '@mui/material';
import {alpha} from '@mui/material/styles';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import SettingsIcon from '@mui/icons-material/Settings';
import CloseIcon from '@mui/icons-material/Close';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import {IProjectSyncNodeConfig, ISyncNode} from '../types';
import useTranslation from '../i18n/useTranslation';

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
    watchEnabled: boolean;
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
    onWatchEnabledChange: (value: boolean) => void;
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

const IgnoreRulesHelp: React.FC<{open: boolean; t: (key: string) => string}> = ({open, t}) => (
    <Collapse in={open}>
        <Paper variant="outlined" sx={{p: 1, mt: 1}}>
            <Stack spacing={0.5}>
                <Typography variant="body2">
                    <code>#</code> {t('syncConfig.ignoreHelp.comment')}
                </Typography>
                <Typography variant="body2">
                    <code>!</code> {t('syncConfig.ignoreHelp.negate')}
                </Typography>
                <Typography variant="body2">
                    <code>*</code> {t('syncConfig.ignoreHelp.singleWildcard')}
                </Typography>
                <Typography variant="body2">
                    <code>**</code> {t('syncConfig.ignoreHelp.multiWildcard')}
                </Typography>
                <Typography variant="body2">
                    <code>?</code> / <code>[abc]</code> {t('syncConfig.ignoreHelp.charGroup')}
                </Typography>
                <Typography variant="body2">
                    <code>dir/</code> {t('syncConfig.ignoreHelp.dirRule')}
                </Typography>
                <Typography variant="body2">
                    <code>/pattern</code> {t('syncConfig.ignoreHelp.rootAnchor')}
                </Typography>
                <Typography variant="body2">{t('syncConfig.ignoreHelp.order')}</Typography>
                <Typography variant="body2">{t('syncConfig.ignoreHelp.anyDepth')}</Typography>
                <Typography variant="body2">{t('syncConfig.ignoreHelp.examples')}</Typography>
            </Stack>
        </Paper>
    </Collapse>
);

const SyncConfigEditor: React.FC<Props> = ({
    enabled,
    watchEnabled,
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
    onWatchEnabledChange,
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
    const {t} = useTranslation();
    const [expanded, setExpanded] = useState<Record<number, boolean>>({});
    const [tab, setTab] = useState(0);
    const [showIgnoreHelp, setShowIgnoreHelp] = useState(false);

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
                <Tab label={t('syncConfig.tabs.sync')} />
                <Tab label={t('syncConfig.tabs.permissions')} />
                <Tab label={t('syncConfig.tabs.nodes')} />
                <Tab label={t('syncConfig.tabs.advanced')} />
            </Tabs>

            {tab === 0 ? (
                <Box mt={1}>
                    <Stack direction="row" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">{t('syncConfig.sections.sync')}</Typography>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={enabled}
                                    onChange={(_, checked) => onEnabledChange(checked)}
                                />
                            }
                            label={t('syncConfig.sync.enable')}
                        />
                    </Stack>

                    {enabled ? (
                        <>
                            <Box mt={1}>
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={watchEnabled}
                                            onChange={(_, checked) => onWatchEnabledChange(checked)}
                                        />
                                    }
                                    label={t('syncConfig.sync.watch')}
                                />
                                <Typography variant="caption" color="textSecondary">
                                    {t('syncConfig.sync.watchHelp')}
                                </Typography>
                            </Box>

                            <Box mt={1}>
                                <Stack
                                    direction="row"
                                    alignItems="center"
                                    justifyContent="space-between">
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={ignoreDefaults}
                                                onChange={(_, checked) =>
                                                    onIgnoreDefaultsChange(checked)
                                                }
                                            />
                                        }
                                        label={t('syncConfig.sync.ignoreDefaults')}
                                    />
                                    <Button
                                        size="small"
                                        startIcon={<HelpOutlineIcon />}
                                        onClick={() => setShowIgnoreHelp((v) => !v)}>
                                        {t('syncConfig.sync.rulesHelp')}
                                    </Button>
                                </Stack>
                            </Box>
                            <IgnoreRulesHelp open={showIgnoreHelp} t={t} />

                            <TextField
                                margin="dense"
                                label={t('syncConfig.sync.ignorePatterns')}
                                value={ignorePatterns}
                                onChange={(e) => onIgnorePatternsChange(e.target.value)}
                                fullWidth
                                multiline
                                minRows={4}
                                placeholder={t('syncConfig.sync.ignorePlaceholder')}
                                helperText={t('syncConfig.sync.ignoreHelp')}
                            />

                            <TextField
                                margin="dense"
                                label={t('syncConfig.sync.ignoreFile')}
                                value={ignoreFile}
                                onChange={(e) => onIgnoreFileChange(e.target.value)}
                                fullWidth
                                placeholder={t('syncConfig.sync.ignoreFilePlaceholder')}
                                helperText={t('syncConfig.sync.ignoreFileHelp')}
                            />
                        </>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            {t('syncConfig.sync.disabledHint')}
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 1 ? (
                <Box mt={1}>
                    <Typography variant="subtitle1">
                        {t('syncConfig.sections.permissions')}
                    </Typography>
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
                                label={t('syncConfig.permissions.ignoreAll')}
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
                                        label={t('syncConfig.permissions.preserveMode')}
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
                                        label={t('syncConfig.permissions.preserveMtime')}
                                    />
                                </Box>
                            </Collapse>

                            <TextField
                                margin="dense"
                                select
                                label={t('syncConfig.permissions.symlinkPolicy')}
                                value={symlinkPolicy}
                                onChange={(e) =>
                                    onSymlinkPolicyChange(
                                        e.target.value === 'preserve' ? 'preserve' : 'ignore'
                                    )
                                }
                                fullWidth>
                                <MenuItem value="ignore">
                                    {t('syncConfig.permissions.symlinkIgnore')}
                                </MenuItem>
                                <MenuItem value="preserve">
                                    {t('syncConfig.permissions.symlinkPreserve')}
                                </MenuItem>
                            </TextField>
                            <Typography variant="caption" color="textSecondary">
                                {t('syncConfig.permissions.symlinkHint')}
                            </Typography>
                        </Box>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            {t('syncConfig.permissions.disabledHint')}
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 3 ? (
                <Box mt={1}>
                    <Typography variant="subtitle1">{t('syncConfig.sections.advanced')}</Typography>
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
                                label={t('syncConfig.advanced.overlayEnable')}
                            />
                            <TextField
                                margin="dense"
                                label={t('syncConfig.advanced.deltaMaxFiles')}
                                type="number"
                                value={deltaMaxFiles}
                                onChange={(e) =>
                                    onDeltaMaxFilesChange(
                                        e.target.value === '' ? '' : Number(e.target.value)
                                    )
                                }
                                fullWidth
                                placeholder={t('syncConfig.advanced.deltaMaxFilesPlaceholder')}
                            />
                            <TextField
                                margin="dense"
                                label={t('syncConfig.advanced.overlayFullScanEvery')}
                                type="number"
                                value={overlayFullScanEvery}
                                onChange={(e) =>
                                    onOverlayFullScanEveryChange(
                                        e.target.value === '' ? '' : Number(e.target.value)
                                    )
                                }
                                fullWidth
                                placeholder={t(
                                    'syncConfig.advanced.overlayFullScanEveryPlaceholder'
                                )}
                                helperText={t('syncConfig.advanced.overlayFullScanEveryHelp')}
                            />
                            <TextField
                                margin="dense"
                                label={t('syncConfig.advanced.overlayFullScanInterval')}
                                value={overlayFullScanInterval}
                                onChange={(e) => onOverlayFullScanIntervalChange(e.target.value)}
                                fullWidth
                                placeholder={t(
                                    'syncConfig.advanced.overlayFullScanIntervalPlaceholder'
                                )}
                            />
                        </Box>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                            {t('syncConfig.advanced.disabledHint')}
                        </Typography>
                    )}
                </Box>
            ) : null}

            {tab === 2 ? (
                <Box mt={2}>
                    <Stack direction="row" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle2">{t('syncConfig.nodes.title')}</Typography>
                        <Button
                            size="small"
                            startIcon={<AddIcon />}
                            onClick={addSyncNodeRow}
                            disabled={availableNodes.length === 0}>
                            {t('syncConfig.nodes.add')}
                        </Button>
                    </Stack>
                    {availableNodes.length === 0 ? (
                        <Typography variant="caption" color="textSecondary">
                            {t('syncConfig.nodes.empty')}
                        </Typography>
                    ) : null}
                    <Box sx={{mt: 1}}>
                        {syncNodes.map((row, idx) => (
                            <Paper
                                key={`${row.nodeId}-${idx}`}
                                variant="outlined"
                                sx={{
                                    borderRadius: 1,
                                    p: 1,
                                    mb: 1,
                                    borderColor: (theme) =>
                                        theme.palette.mode === 'dark'
                                            ? 'rgba(255,255,255,0.18)'
                                            : 'rgba(0,0,0,0.14)',
                                    backgroundColor: (theme) =>
                                        alpha(
                                            theme.palette.background.paper,
                                            theme.palette.mode === 'dark' ? 0.35 : 0.85
                                        ),
                                    transition:
                                        'border-color 120ms ease, background-color 120ms ease',
                                    '&:hover': {
                                        borderColor: (theme) =>
                                            theme.palette.mode === 'dark'
                                                ? 'rgba(255,255,255,0.28)'
                                                : 'rgba(0,0,0,0.24)',
                                        backgroundColor: (theme) =>
                                            alpha(
                                                theme.palette.background.paper,
                                                theme.palette.mode === 'dark' ? 0.5 : 0.95
                                            ),
                                    },
                                }}>
                                <Box
                                    sx={{
                                        display: 'flex',
                                        gap: 1,
                                        alignItems: 'center',
                                    }}>
                                    <TextField
                                        select
                                        label={t('syncConfig.nodes.nodeLabel')}
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
                                        label={t('syncConfig.nodes.targetPath')}
                                        value={row.targetPath}
                                        onChange={(e) =>
                                            updateNodeRow(idx, {targetPath: e.target.value})
                                        }
                                        size="small"
                                        fullWidth
                                        placeholder={
                                            projectPath ||
                                            t('syncConfig.nodes.targetPathPlaceholder')
                                        }
                                    />
                                    <IconButton
                                        size="small"
                                        onClick={() => toggleExpanded(idx)}
                                        title={t('syncConfig.nodes.settings')}>
                                        {expanded[idx] ? (
                                            <CloseIcon fontSize="small" />
                                        ) : (
                                            <SettingsIcon fontSize="small" />
                                        )}
                                    </IconButton>
                                    <IconButton
                                        size="small"
                                        onClick={() => removeNodeRow(idx)}
                                        title={t('syncConfig.nodes.remove')}
                                        color="error">
                                        <DeleteIcon fontSize="small" />
                                    </IconButton>
                                </Box>

                                <Collapse in={!!expanded[idx]} timeout="auto" unmountOnExit>
                                    <Box mt={1}>
                                        <TextField
                                            margin="dense"
                                            select
                                            label={t('syncConfig.nodes.strategy')}
                                            value={row.strategy || 'mirror'}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {strategy: e.target.value})
                                            }
                                            fullWidth
                                            helperText={t('syncConfig.nodes.strategyHelp')}>
                                            <MenuItem value="mirror">mirror</MenuItem>
                                            <MenuItem value="overlay">overlay</MenuItem>
                                        </TextField>
                                        <TextField
                                            margin="dense"
                                            label={t('syncConfig.nodes.extraIgnore')}
                                            value={row.ignorePatterns}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {
                                                    ignorePatterns: e.target.value,
                                                })
                                            }
                                            fullWidth
                                            multiline
                                            minRows={3}
                                            placeholder={t(
                                                'syncConfig.nodes.extraIgnorePlaceholder'
                                            )}
                                            helperText={t('syncConfig.nodes.extraIgnoreHelp')}
                                        />
                                        <Box sx={{display: 'flex', justifyContent: 'flex-end'}}>
                                            <Button
                                                size="small"
                                                startIcon={<HelpOutlineIcon />}
                                                onClick={() => setShowIgnoreHelp((v) => !v)}>
                                                {t('syncConfig.sync.rulesHelp')}
                                            </Button>
                                        </Box>
                                        <IgnoreRulesHelp open={showIgnoreHelp} t={t} />
                                        <TextField
                                            margin="dense"
                                            label={t('syncConfig.nodes.ignoreFile')}
                                            value={row.ignoreFile}
                                            onChange={(e) =>
                                                updateNodeRow(idx, {ignoreFile: e.target.value})
                                            }
                                            fullWidth
                                            placeholder={t(
                                                'syncConfig.nodes.ignoreFilePlaceholder'
                                            )}
                                        />
                                        {row.strategy === 'mirror' ? (
                                            <Box mt={1}>
                                                <Typography variant="subtitle2">
                                                    {t('syncConfig.nodes.mirror.title')}
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
                                                    label={t(
                                                        'syncConfig.nodes.mirror.syncEmptyDirs'
                                                    )}
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
                                                    label={t('syncConfig.nodes.mirror.fastDelete')}
                                                />
                                                <TextField
                                                    margin="dense"
                                                    label={t(
                                                        'syncConfig.nodes.mirror.fullscanEvery'
                                                    )}
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
                                                    placeholder={t(
                                                        'syncConfig.nodes.mirror.fullscanEveryPlaceholder'
                                                    )}
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
                                                    label={t(
                                                        'syncConfig.nodes.mirror.cleanEmptyDirs'
                                                    )}
                                                />
                                            </Box>
                                        ) : null}
                                    </Box>
                                </Collapse>
                            </Paper>
                        ))}
                    </Box>
                </Box>
            ) : null}
        </>
    );
};

export default SyncConfigEditor;
