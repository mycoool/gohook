import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Delete from '@mui/icons-material/Delete';
import PlayArrow from '@mui/icons-material/PlayArrow';
import Refresh from '@mui/icons-material/Refresh';
import CloudDownload from '@mui/icons-material/CloudDownload';
import Edit from '@mui/icons-material/Edit';
import Add from '@mui/icons-material/Add';
import Settings from '@mui/icons-material/Settings';
import ButtonGroup from '@mui/material/ButtonGroup';
import React, {Component} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IHook} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';
import useTranslation from '../i18n/useTranslation';
import {Theme} from '@mui/material/styles';
import ScriptEditDialog from './ScriptEditDialog';
import HookConfigDialog from './HookConfigDialog';

// 创建一个注入了依赖的包装组件
const ScriptEditDialogWrapper = inject('snackManager')(ScriptEditDialog);
const HookConfigDialogWrapper = inject('snackManager')(HookConfigDialog);

import {WithStyles} from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import createStyles from '@mui/styles/createStyles';

// 添加样式定义 - 优化代码块显示效果
const styles = () =>
    createStyles({
        codeBlock: {
            fontSize: '0.875rem',
            backgroundColor: '#21262d',
            color: '#e6edf3',
            padding: '4px 8px',
            borderRadius: '6px',
            border: '1px solid #30363d',
            fontFamily:
                'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
            fontWeight: 400,
        },
        workingDir: {
            fontSize: '0.8em',
            color: '#8b949e',
            marginTop: '4px',
        },
    });

@observer
class Hooks extends Component<Stores<'hookStore' | 'snackManager'>> {
    @observable
    private deleteId: string | false = false;
    @observable
    private triggerDialog = false;
    @observable
    private editingScriptId: string | false = false;
    @observable
    private configDialog: {open: boolean; hookId?: string; isEdit: boolean} = {
        open: false,
        isEdit: false,
    };

    public componentDidMount = () => this.props.hookStore.refresh();

    public render() {
        const {
            deleteId,
            editingScriptId,
            configDialog,
            props: {hookStore},
        } = this;
        const hooks = hookStore.getItems();
        return (
            <>
                <HooksContainer
                    hooks={hooks}
                    deleteId={deleteId}
                    editingScriptId={editingScriptId}
                    onRefresh={this.refreshHooks}
                    onReloadConfig={this.reloadConfig}
                    onAddHook={() =>
                        (this.configDialog = {open: true, isEdit: false})
                    }
                    onTriggerHook={this.triggerHook}
                    onEditScript={(id) => (this.editingScriptId = id)}
                    onEditConfig={(id) =>
                        (this.configDialog = {open: true, hookId: id, isEdit: true})
                    }
                    onDeleteHook={(id) => (this.deleteId = id)}
                    onCancelDelete={() => (this.deleteId = false)}
                    onConfirmDelete={() => hookStore.remove(deleteId as string)}
                    onCloseScriptEditor={() => (this.editingScriptId = false)}
                    hookStore={hookStore}
                />
                {editingScriptId !== false && (
                    <ScriptEditDialogWrapper
                        open={true}
                        hookId={editingScriptId}
                        onClose={() => (this.editingScriptId = false)}
                        onGetScript={hookStore.getScript}
                        onSaveScript={hookStore.saveScript}
                        onDeleteScript={hookStore.deleteScript}
                    />
                )}
                {configDialog.open && (
                    <HookConfigDialogWrapper
                        open={true}
                        hook={
                            configDialog.hookId
                                ? hookStore.getByID(configDialog.hookId)
                                : undefined
                        }
                        onClose={() => (this.configDialog = {open: false, isEdit: false})}
                        onSave={this.saveHookConfig}
                    />
                )}
            </>
        );
    }

    private saveHookConfig = async (hook: IHook) => {
        try {
            // 这里需要实现保存Hook配置的逻辑
            // 可能需要调用新的API来保存完整的hook配置
            await this.props.hookStore.saveHook(hook);
            this.props.snackManager.snack('Hook配置保存成功');
            this.configDialog = {open: false, isEdit: false};
            this.props.hookStore.refresh();
        } catch (error) {
            this.props.snackManager.snack('Hook配置保存失败: ' + (error as Error).message);
        }
    };

    private refreshHooks = () => {
        this.props.hookStore.refresh();
    };

    private reloadConfig = () => {
        this.props.hookStore.reloadConfig();
    };

    private triggerHook = (id: string) => {
        this.props.hookStore.triggerHook(id);
    };
}

// 分离容器组件以使用Hook
const HooksContainer: React.FC<{
    hooks: IHook[];
    deleteId: string | false;
    editingScriptId: string | false;
    onRefresh: () => void;
    onReloadConfig: () => void;
    onAddHook: () => void;
    onTriggerHook: (id: string) => void;
    onEditScript: (id: string) => void;
    onEditConfig: (id: string) => void;
    onDeleteHook: (id: string) => void;
    onCancelDelete: () => void;
    onConfirmDelete: () => void;
    onCloseScriptEditor: () => void;
    hookStore: {getByID: (id: string) => IHook};
}> = ({
    hooks,
    deleteId,
    editingScriptId,
    onRefresh,
    onReloadConfig,
    onAddHook,
    onTriggerHook,
    onEditScript,
    onEditConfig,
    onDeleteHook,
    onCancelDelete,
    onConfirmDelete,
    onCloseScriptEditor,
    hookStore,
}) => {
    const {t} = useTranslation();

    return (
        <DefaultPage
            title={t('hook.title')}
            rightControl={
                <ButtonGroup variant="contained" color="primary">
                    <Button id="add-hook" startIcon={<Add />} onClick={onAddHook}>
                        添加Hook
                    </Button>
                    <Button id="refresh-hooks" startIcon={<Refresh />} onClick={onRefresh}>
                        {t('common.refresh')}
                    </Button>
                    <Button
                        id="reload-config"
                        startIcon={<CloudDownload />}
                        onClick={onReloadConfig}>
                        {t('hook.reloadConfig')}
                    </Button>
                </ButtonGroup>
            }
            maxWidth={1200}>
            <Grid size={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="hook-table">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('hook.name')}</TableCell>
                                <TableCell>{t('hook.command')}</TableCell>
                                <TableCell>{t('hook.httpMethods')}</TableCell>
                                <TableCell>{t('hook.parameters')}</TableCell>
                                <TableCell>{t('hook.triggerRule')}</TableCell>
                                <TableCell>{t('hook.status')}</TableCell>
                                <TableCell>{t('hook.lastUsed')}</TableCell>
                                <TableCell align="center">{t('common.actions')}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {hooks.map((hook: IHook) => (
                                <StyledRow
                                    key={hook.id}
                                    hook={hook}
                                    fTrigger={() => onTriggerHook(hook.id)}
                                    fEditScript={() => onEditScript(hook.id)}
                                    fEditConfig={() => onEditConfig(hook.id)}
                                    fDelete={() => onDeleteHook(hook.id)}
                                />
                            ))}
                        </TableBody>
                    </Table>
                </Paper>
            </Grid>
            {deleteId !== false && (
                <ConfirmDialog
                    title={t('hook.confirmDelete')}
                    text={t('hook.confirmDeleteText', {name: hookStore.getByID(deleteId).name})}
                    fClose={onCancelDelete}
                    fOnSubmit={onConfirmDelete}
                />
            )}
        </DefaultPage>
    );
};

// 更新接口定义
interface IRowProps extends WithStyles<typeof styles> {
    hook: IHook;
    fTrigger: VoidFunction;
    fEditScript: VoidFunction;
    fEditConfig: VoidFunction;
    fDelete: VoidFunction;
}

const Row: React.FC<IRowProps> = observer(({hook, fTrigger, fEditScript, fEditConfig, fDelete, classes}) => {
    const {t} = useTranslation();

    return (
        <TableRow>
            <TableCell>
                <strong>{hook.name}</strong>
                <br />
                <small style={{color: '#666'}}>ID: {hook.id}</small>
            </TableCell>
            <TableCell>
                <code className={classes.codeBlock}>{hook.executeCommand}</code>
                {hook.workingDirectory && (
                    <div className={classes.workingDir}>
                        {t('hook.workingDir')}: {hook.workingDirectory}
                    </div>
                )}
            </TableCell>
            <TableCell>
                {hook.httpMethods.map((method) => (
                    <Chip
                        key={method}
                        label={method}
                        size="small"
                        style={{
                            marginRight: '4px',
                            marginBottom: '2px',
                            backgroundColor: getMethodColor(method),
                            color: 'white',
                            fontSize: '0.7em',
                        }}
                    />
                ))}
            </TableCell>
            <TableCell>
                <Chip
                    label={`参数: ${hook.argumentsCount || 0}`}
                    size="small"
                    style={{
                        marginRight: '4px',
                        marginBottom: '2px',
                        backgroundColor: '#2196f3',
                        color: 'white',
                        fontSize: '0.7em',
                    }}
                />
                <Chip
                    label={`环境变量: ${hook.environmentCount || 0}`}
                    size="small"
                    style={{
                        marginRight: '4px',
                        marginBottom: '2px',
                        backgroundColor: '#4caf50',
                        color: 'white',
                        fontSize: '0.7em',
                    }}
                />
            </TableCell>
            <TableCell style={{maxWidth: 150, wordWrap: 'break-word', fontSize: '0.85em'}}>
                {hook.triggerRuleDescription}
            </TableCell>
            <TableCell>
                <Chip
                    label={hook.status === 'active' ? t('version.active') : t('version.inactive')}
                    size="small"
                    style={{
                        backgroundColor: hook.status === 'active' ? '#4caf50' : '#f44336',
                        color: 'white',
                    }}
                />
            </TableCell>
            <TableCell>
                <LastUsedCell lastUsed={hook.lastUsed} />
            </TableCell>
            <TableCell align="center" padding="none">
                <IconButton
                    onClick={fTrigger}
                    className="trigger"
                    title={t('hook.triggerHook')}
                    size="small">
                    <PlayArrow />
                </IconButton>
                <IconButton
                    onClick={fEditScript}
                    className="edit-script"
                    title="编辑脚本"
                    size="small">
                    <Edit />
                </IconButton>
                <IconButton
                    onClick={fEditConfig}
                    className="edit-config"
                    title="编辑配置"
                    size="small">
                    <Settings />
                </IconButton>
                <IconButton
                    onClick={fDelete}
                    className="delete"
                    title={t('hook.deleteHook')}
                    size="small">
                    <Delete />
                </IconButton>
            </TableCell>
        </TableRow>
    );
});

// 根据HTTP方法返回对应的颜色
function getMethodColor(method: string): string {
    switch (method.toUpperCase()) {
        case 'GET':
            return '#4caf50';
        case 'POST':
            return '#2196f3';
        case 'PUT':
            return '#ff9800';
        case 'DELETE':
            return '#f44336';
        case 'PATCH':
            return '#9c27b0';
        default:
            return '#607d8b';
    }
}

// 使用 withStyles 包装 Row 组件
const StyledRow = withStyles(styles)(Row);

export default inject('hookStore', 'snackManager')(Hooks);
