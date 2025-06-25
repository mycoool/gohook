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
import Code from '@mui/icons-material/Code';
import Add from '@mui/icons-material/Add';
import Settings from '@mui/icons-material/Settings';
import Tune from '@mui/icons-material/Tune';
import FilterAlt from '@mui/icons-material/FilterAlt';
import Http from '@mui/icons-material/Http';
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
import useTranslation from '../i18n/useTranslation';
import {Theme} from '@mui/material/styles';
import ScriptEditDialog from './ScriptEditDialog';
import AddHookDialog from './AddHookDialog';
import EditBasicDialog from './EditBasicDialog';
import EditParametersDialog from './EditParametersDialog';
import EditTriggersDialog from './EditTriggersDialog';
import EditResponseDialog from './EditResponseDialog';

// 创建一个注入了依赖的包装组件
const ScriptEditDialogWrapper = inject('snackManager')(ScriptEditDialog);

import {WithStyles} from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import createStyles from '@mui/styles/createStyles';

// 添加样式定义 - 优化代码块显示效果
const styles = (theme: Theme) =>
    createStyles({
        codeBlock: {
            backgroundColor: theme.palette.mode === 'dark' ? '#2d2d2d' : '#f5f5f5',
            border: `1px solid ${theme.palette.mode === 'dark' ? '#444' : '#e0e0e0'}`,
            borderRadius: '4px',
            padding: '4px 8px',
            fontSize: '0.85em',
            fontFamily: 'Consolas, Monaco, "Courier New", monospace',
            display: 'inline-block',
            maxWidth: '200px',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
        },
        workingDir: {
            fontSize: '0.75em',
            color: theme.palette.text.secondary,
            marginTop: '4px',
            maxWidth: '200px',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
        },
    });

@observer
class Hooks extends Component<Stores<'hookStore' | 'snackManager' | 'currentUser'>> {
    @observable
    private deleteId: string | false = false;
    @observable
    private triggerDialog = false;
    @observable
    private editingScriptId: string | false = false;
    @observable
    private addDialog = false;
    @observable
    private editBasicDialog: {open: boolean; hookId?: string} = {open: false};
    @observable
    private editParametersDialog: {open: boolean; hookId?: string} = {open: false};
    @observable
    private editTriggersDialog: {open: boolean; hookId?: string} = {open: false};
    @observable
    private editResponseDialog: {open: boolean; hookId?: string} = {open: false};

    public componentDidMount() {
        // 只在用户已登录时才进行 API 调用
        if (this.props.currentUser.loggedIn) {
            this.props.hookStore.refresh();
        }
    }

    public componentDidUpdate(prevProps: Stores<'hookStore' | 'snackManager' | 'currentUser'>) {
        // 当用户登录状态改变时，重新加载数据
        if (prevProps.currentUser.loggedIn !== this.props.currentUser.loggedIn && this.props.currentUser.loggedIn) {
            this.props.hookStore.refresh();
        }
    }

    public render() {
        const {
            deleteId,
            editingScriptId,
            addDialog,
            editBasicDialog,
            editParametersDialog,
            editTriggersDialog,
            editResponseDialog,
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
                    onAddHook={() => (this.addDialog = true)}
                    onTriggerHook={this.triggerHook}
                    onEditScript={(id) => (this.editingScriptId = id)}
                    onEditBasic={(hook) => (this.editBasicDialog = {open: true, hookId: hook.id})}
                    onEditParameters={(hook) =>
                        (this.editParametersDialog = {open: true, hookId: hook.id})
                    }
                    onEditTriggers={(hook) =>
                        (this.editTriggersDialog = {open: true, hookId: hook.id})
                    }
                    onEditResponse={(hook) =>
                        (this.editResponseDialog = {open: true, hookId: hook.id})
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
                        onUpdateExecuteCommand={hookStore.updateExecuteCommand}
                        onGetHookDetails={hookStore.getHookDetails}
                    />
                )}
                <AddHookDialog
                    open={addDialog}
                    onClose={() => (this.addDialog = false)}
                    onSave={this.createHook}
                />
                <EditBasicDialog
                    open={editBasicDialog.open}
                    hookId={editBasicDialog.hookId}
                    onClose={() => (this.editBasicDialog = {open: false})}
                    onSave={this.updateHookBasic}
                    onGetHookDetails={this.props.hookStore.getHookDetails}
                />
                <EditParametersDialog
                    open={editParametersDialog.open}
                    hookId={editParametersDialog.hookId}
                    onClose={() => (this.editParametersDialog = {open: false})}
                    onSave={this.updateHookParameters}
                    onGetHookDetails={this.props.hookStore.getHookDetails}
                />
                <EditTriggersDialog
                    open={editTriggersDialog.open}
                    hookId={editTriggersDialog.hookId}
                    onClose={() => (this.editTriggersDialog = {open: false})}
                    onSave={this.updateHookTriggers}
                    onGetHookDetails={this.props.hookStore.getHookDetails}
                />
                <EditResponseDialog
                    open={editResponseDialog.open}
                    hookId={editResponseDialog.hookId}
                    onClose={() => (this.editResponseDialog = {open: false})}
                    onSave={this.updateHookResponse}
                    onGetHookDetails={this.props.hookStore.getHookDetails}
                />
            </>
        );
    }

    private createHook = async (hookData: {
        id: string;
        'execute-command': string;
        'command-working-directory': string;
        'response-message': string;
    }) => {
        try {
            await this.props.hookStore.createHook(hookData);
            this.props.hookStore.refresh();
        } catch (error) {
            // 错误已在store中处理
        }
    };

    private updateHookBasic = async (
        hookId: string,
        basicData: {
            'execute-command': string;
            'command-working-directory': string;
            'response-message': string;
        }
    ) => {
        try {
            await this.props.hookStore.updateHookBasic(hookId, basicData);
            this.props.hookStore.refresh();
        } catch (error) {
            // 错误已在store中处理
        }
    };

    private updateHookParameters = async (hookId: string, parametersData: any) => {
        try {
            await this.props.hookStore.updateHookParameters(hookId, parametersData);
            this.props.hookStore.refresh();
        } catch (error) {
            // 错误已在store中处理
        }
    };

    private updateHookTriggers = async (hookId: string, triggersData: any) => {
        try {
            await this.props.hookStore.updateHookTriggers(hookId, triggersData);
            this.props.hookStore.refresh();
        } catch (error) {
            // 错误已在store中处理
        }
    };

    private updateHookResponse = async (hookId: string, responseData: any) => {
        try {
            await this.props.hookStore.updateHookResponse(hookId, responseData);
            this.props.hookStore.refresh();
        } catch (error) {
            // 错误已在store中处理
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
    onEditBasic: (hook: IHook) => void;
    onEditParameters: (hook: IHook) => void;
    onEditTriggers: (hook: IHook) => void;
    onEditResponse: (hook: IHook) => void;
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
    onEditBasic,
    onEditParameters,
    onEditTriggers,
    onEditResponse,
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
                                <TableCell>触发规则</TableCell>
                                <TableCell>{t('hook.status')}</TableCell>
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
                                    fEditBasic={() => onEditBasic(hook)}
                                    fEditParameters={() => onEditParameters(hook)}
                                    fEditTriggers={() => onEditTriggers(hook)}
                                    fEditResponse={() => onEditResponse(hook)}
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
    fEditBasic: VoidFunction;
    fEditParameters: VoidFunction;
    fEditTriggers: VoidFunction;
    fEditResponse: VoidFunction;
    fDelete: VoidFunction;
}

// 智能显示执行命令
const formatExecuteCommand = (executeCommand: string): string => {
    if (!executeCommand) return '';

    // 检查是否是脚本文件路径（包含.sh, .py, .js, .ts, .bat, .cmd等扩展名）
    const scriptExtensions = ['.sh', '.py', '.js', '.ts', '.bat', '.cmd', '.ps1', '.pl', '.rb'];
    const isScript = scriptExtensions.some((ext) => executeCommand.toLowerCase().includes(ext));

    if (isScript) {
        // 如果是脚本路径，只显示文件名
        const parts = executeCommand.split(/[/\\]/);
        return parts[parts.length - 1] || executeCommand;
    } else {
        // 如果是命令，进行智能缩短
        const words = executeCommand.trim().split(/\s+/);
        if (words.length <= 3) {
            return executeCommand; // 短命令直接显示
        }

        // 长命令显示前3个词 + "..."
        return words.slice(0, 3).join(' ') + '...';
    }
};

// 获取执行命令的完整工具提示
const getExecuteCommandTooltip = (executeCommand: string, workingDirectory: string): string => {
    let tooltip = `完整命令: ${executeCommand}`;
    if (workingDirectory) {
        tooltip += `\n工作目录: ${workingDirectory}`;
    }
    return tooltip;
};

const Row: React.FC<IRowProps> = observer(
    ({
        hook,
        fTrigger,
        fEditScript,
        fEditBasic,
        fEditParameters,
        fEditTriggers,
        fEditResponse,
        fDelete,
        classes,
    }) => {
        const {t} = useTranslation();

        return (
            <TableRow>
                <TableCell>
                    <strong>{hook.name}</strong>
                    <br />
                    <small style={{color: '#666'}}>ID: {hook.id}</small>
                </TableCell>
                <TableCell>
                    <code
                        className={classes.codeBlock}
                        title={getExecuteCommandTooltip(
                            hook.executeCommand || '',
                            hook.workingDirectory || ''
                        )}>
                        {formatExecuteCommand(hook.executeCommand || '')}
                    </code>
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
                <TableCell>
                    <Chip
                        label={`规则: ${countTriggerRules(hook['trigger-rule'])}`}
                        size="small"
                        style={{
                            backgroundColor:
                                countTriggerRules(hook['trigger-rule']) > 0 ? '#ff9800' : '#9e9e9e',
                            color: 'white',
                            fontSize: '0.7em',
                        }}
                    />
                </TableCell>
                <TableCell>
                    <Chip
                        label={
                            hook.status === 'active' ? t('version.active') : t('version.inactive')
                        }
                        size="small"
                        style={{
                            backgroundColor: hook.status === 'active' ? '#4caf50' : '#f44336',
                            color: 'white',
                        }}
                    />
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
                        <Code />
                    </IconButton>
                    <IconButton
                        onClick={fEditBasic}
                        className="edit-basic"
                        title="编辑基本信息"
                        size="small">
                        <Settings />
                    </IconButton>
                    <IconButton
                        onClick={fEditParameters}
                        className="edit-parameters"
                        title="编辑参数配置"
                        size="small">
                        <Tune />
                    </IconButton>
                    <IconButton
                        onClick={fEditTriggers}
                        className="edit-triggers"
                        title="编辑触发规则"
                        size="small">
                        <FilterAlt />
                    </IconButton>
                    <IconButton
                        onClick={fEditResponse}
                        className="edit-response"
                        title="编辑响应配置"
                        size="small">
                        <Http />
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
    }
);

// 计算触发规则条数
function countTriggerRules(triggerRule: any): number {
    if (!triggerRule) {
        return 0;
    }

    let count = 0;

    // 如果有直接的match规则
    if (triggerRule.match) {
        count++;
    }

    // 递归计算and规则
    if (triggerRule.and && Array.isArray(triggerRule.and)) {
        triggerRule.and.forEach((rule: any) => {
            count += countTriggerRules(rule);
        });
    }

    // 递归计算or规则
    if (triggerRule.or && Array.isArray(triggerRule.or)) {
        triggerRule.or.forEach((rule: any) => {
            count += countTriggerRules(rule);
        });
    }

    // 递归计算not规则
    if (triggerRule.not) {
        count += countTriggerRules(triggerRule.not);
    }

    return count;
}

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

export default inject('hookStore', 'snackManager', 'currentUser')(Hooks);
