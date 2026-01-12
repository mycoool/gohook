import React, {useEffect, useState} from 'react';
import {
    Button,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
} from '@mui/material';
import {IProjectSyncConfig, ISyncNode} from '../types';
import SyncConfigEditor, {SyncNodeRow, normalizeNodes} from './SyncConfigEditor';
import useTranslation from '../i18n/useTranslation';

interface Props {
    open: boolean;
    projectName: string;
    projectPath: string;
    availableNodes: ISyncNode[];
    initialSync?: IProjectSyncConfig;
    saving: boolean;
    onClose: () => void;
    onSave: (sync: IProjectSyncConfig) => Promise<void>;
}

const joinLines = (lines?: string[]) => (lines?.length ? lines.join('\n') : '');

const SyncConfigDialog: React.FC<Props> = ({
    open,
    projectName,
    projectPath,
    availableNodes,
    initialSync,
    saving,
    onClose,
    onSave,
}) => {
    const {t} = useTranslation();
    const [enabled, setEnabled] = useState(false);
    const [watchEnabled, setWatchEnabled] = useState(true);
    const [ignoreDefaults, setIgnoreDefaults] = useState(true);
    const [ignorePermissions, setIgnorePermissions] = useState(false);
    const [preserveMode, setPreserveMode] = useState(true);
    const [preserveMtime, setPreserveMtime] = useState(true);
    const [symlinkPolicy, setSymlinkPolicy] = useState<'ignore' | 'preserve'>('ignore');
    const [ignorePatterns, setIgnorePatterns] = useState('');
    const [ignoreFile, setIgnoreFile] = useState('');
    const [deltaIndexOverlay, setDeltaIndexOverlay] = useState(true);
    const [deltaMaxFiles, setDeltaMaxFiles] = useState<number | ''>(5000);
    const [overlayFullScanEvery, setOverlayFullScanEvery] = useState<number | ''>('');
    const [overlayFullScanInterval, setOverlayFullScanInterval] = useState('1h');
    const [syncNodes, setSyncNodes] = useState<SyncNodeRow[]>([]);

    useEffect(() => {
        if (!open) return;
        const sync = initialSync;
        setEnabled(sync?.enabled ?? true);
        setWatchEnabled(sync?.watchEnabled ?? true);
        setIgnoreDefaults(sync?.ignoreDefaults ?? true);
        setIgnorePermissions(sync?.ignorePermissions ?? false);
        setPreserveMode(sync?.preserveMode ?? true);
        setPreserveMtime(sync?.preserveMtime ?? true);
        setSymlinkPolicy((sync?.symlinkPolicy as any) === 'preserve' ? 'preserve' : 'ignore');
        setIgnorePatterns(joinLines(sync?.ignorePatterns));
        setIgnoreFile(sync?.ignoreFile || '');
        setDeltaIndexOverlay(sync?.deltaIndexOverlay ?? true);
        setDeltaMaxFiles(sync?.deltaMaxFiles ?? 5000);
        setOverlayFullScanEvery(sync?.overlayFullScanEvery ?? '');
        setOverlayFullScanInterval(sync?.overlayFullScanInterval || '1h');
        setSyncNodes(
            (sync?.nodes || []).map((n) => ({
                nodeId: n.nodeId,
                targetPath: n.targetPath,
                strategy: n.strategy || 'mirror',
                ignorePatterns: joinLines(n.ignorePatterns),
                ignoreFile: n.ignoreFile || '',
                mirrorFastDelete: n.mirrorFastDelete ?? false,
                mirrorFastFullscanEvery: n.mirrorFastFullscanEvery ?? '',
                mirrorCleanEmptyDirs: n.mirrorCleanEmptyDirs ?? false,
                mirrorSyncEmptyDirs: n.mirrorSyncEmptyDirs ?? false,
            }))
        );
    }, [open, initialSync]);

    const parseLines = (value: string): string[] =>
        value
            .split('\n')
            .map((s) => s.trim())
            .filter((s) => s.length > 0);

    const handleSave = async () => {
        const sync: IProjectSyncConfig = {
            enabled,
            watchEnabled: watchEnabled ? undefined : false,
            ignoreDefaults,
            ignorePermissions,
            preserveMode,
            preserveMtime,
            symlinkPolicy,
            ignorePatterns: parseLines(ignorePatterns),
            ignoreFile: ignoreFile.trim() || undefined,
            deltaIndexOverlay,
            deltaMaxFiles: deltaMaxFiles === '' ? undefined : Number(deltaMaxFiles),
            overlayFullScanEvery:
                overlayFullScanEvery === '' ? undefined : Number(overlayFullScanEvery),
            overlayFullScanInterval: overlayFullScanInterval.trim() || undefined,
            nodes: normalizeNodes(syncNodes),
        };
        await onSave(sync);
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{t('syncConfig.title', {name: projectName})}</DialogTitle>
            <DialogContent>
                <SyncConfigEditor
                    enabled={enabled}
                    watchEnabled={watchEnabled}
                    ignoreDefaults={ignoreDefaults}
                    ignorePermissions={ignorePermissions}
                    preserveMode={preserveMode}
                    preserveMtime={preserveMtime}
                    symlinkPolicy={symlinkPolicy}
                    ignorePatterns={ignorePatterns}
                    ignoreFile={ignoreFile}
                    deltaIndexOverlay={deltaIndexOverlay}
                    deltaMaxFiles={deltaMaxFiles}
                    overlayFullScanEvery={overlayFullScanEvery}
                    overlayFullScanInterval={overlayFullScanInterval}
                    syncNodes={syncNodes}
                    availableNodes={availableNodes}
                    projectPath={projectPath}
                    onEnabledChange={setEnabled}
                    onWatchEnabledChange={setWatchEnabled}
                    onIgnoreDefaultsChange={setIgnoreDefaults}
                    onIgnorePermissionsChange={setIgnorePermissions}
                    onPreserveModeChange={setPreserveMode}
                    onPreserveMtimeChange={setPreserveMtime}
                    onSymlinkPolicyChange={setSymlinkPolicy}
                    onIgnorePatternsChange={setIgnorePatterns}
                    onIgnoreFileChange={setIgnoreFile}
                    onDeltaIndexOverlayChange={setDeltaIndexOverlay}
                    onDeltaMaxFilesChange={setDeltaMaxFiles}
                    onOverlayFullScanEveryChange={setOverlayFullScanEvery}
                    onOverlayFullScanIntervalChange={setOverlayFullScanInterval}
                    onSyncNodesChange={setSyncNodes}
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={saving} variant="contained" color="secondary">
                    {t('common.cancel')}
                </Button>
                <Button
                    onClick={handleSave}
                    disabled={saving}
                    variant="contained"
                    color="primary"
                    startIcon={saving ? <CircularProgress size={16} /> : undefined}>
                    {t('common.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default SyncConfigDialog;
