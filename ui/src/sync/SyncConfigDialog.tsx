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
    const [enabled, setEnabled] = useState(false);
    const [ignoreDefaults, setIgnoreDefaults] = useState(true);
    const [ignorePermissions, setIgnorePermissions] = useState(false);
    const [ignorePatterns, setIgnorePatterns] = useState('');
    const [ignoreFile, setIgnoreFile] = useState('');
    const [syncNodes, setSyncNodes] = useState<SyncNodeRow[]>([]);

    useEffect(() => {
        if (!open) return;
        const sync = initialSync;
        setEnabled(sync?.enabled ?? true);
        setIgnoreDefaults(sync?.ignoreDefaults ?? true);
        setIgnorePermissions(sync?.ignorePermissions ?? false);
        setIgnorePatterns(joinLines(sync?.ignorePatterns));
        setIgnoreFile(sync?.ignoreFile || '');
        setSyncNodes(
            (sync?.nodes || []).map((n) => ({
                nodeId: n.nodeId,
                targetPath: n.targetPath,
                ignorePatterns: joinLines(n.ignorePatterns),
                ignoreFile: n.ignoreFile || '',
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
            ignoreDefaults,
            ignorePermissions,
            ignorePatterns: parseLines(ignorePatterns),
            ignoreFile: ignoreFile.trim() || undefined,
            nodes: normalizeNodes(syncNodes),
        };
        await onSave(sync);
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>同步管理 - {projectName}</DialogTitle>
            <DialogContent>
                <SyncConfigEditor
                    enabled={enabled}
                    ignoreDefaults={ignoreDefaults}
                    ignorePermissions={ignorePermissions}
                    ignorePatterns={ignorePatterns}
                    ignoreFile={ignoreFile}
                    syncNodes={syncNodes}
                    availableNodes={availableNodes}
                    projectPath={projectPath}
                    onEnabledChange={setEnabled}
                    onIgnoreDefaultsChange={setIgnoreDefaults}
                    onIgnorePermissionsChange={setIgnorePermissions}
                    onIgnorePatternsChange={setIgnorePatterns}
                    onIgnoreFileChange={setIgnoreFile}
                    onSyncNodesChange={setSyncNodes}
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={saving} variant="contained" color="secondary">
                    取消
                </Button>
                <Button
                    onClick={handleSave}
                    disabled={saving}
                    variant="contained"
                    color="primary"
                    startIcon={saving ? <CircularProgress size={16} /> : undefined}>
                    保存
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default SyncConfigDialog;
