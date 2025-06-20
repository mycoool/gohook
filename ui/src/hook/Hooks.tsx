import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Delete from '@material-ui/icons/Delete';
import PlayArrow from '@material-ui/icons/PlayArrow';
import Refresh from '@material-ui/icons/Refresh';
import CloudDownload from '@material-ui/icons/CloudDownload';
import ButtonGroup from '@material-ui/core/ButtonGroup';
import React, {Component} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@material-ui/core/Button';
import Chip from '@material-ui/core/Chip';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IHook} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';
import useTranslation from '../i18n/useTranslation';
import {withStyles, WithStyles, Theme, createStyles} from '@material-ui/core/styles';

// 添加样式定义
const styles = (theme: Theme) =>
    createStyles({
        codeBlock: {
            fontSize: '0.85em',
            backgroundColor:
                theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
            color: theme.palette.text.primary,
            padding: '2px 4px',
            borderRadius: '3px',
            border:
                theme.palette.type === 'dark'
                    ? '1px solid rgba(255, 255, 255, 0.2)'
                    : '1px solid rgba(0, 0, 0, 0.2)',
        },
        workingDir: {
            fontSize: '0.8em',
            color: theme.palette.text.secondary,
            marginTop: '4px',
        },
    });

@observer
class Hooks extends Component<Stores<'hookStore'>> {
    @observable
    private deleteId: string | false = false;
    @observable
    private triggerDialog = false;

    public componentDidMount = () => this.props.hookStore.refresh();

    public render() {
        const {
            deleteId,
            props: {hookStore},
        } = this;
        const hooks = hookStore.getItems();
        return (
            <HooksContainer
                hooks={hooks}
                deleteId={deleteId}
                onRefresh={this.refreshHooks}
                onReloadConfig={this.reloadConfig}
                onTriggerHook={this.triggerHook}
                onDeleteHook={(id) => (this.deleteId = id)}
                onCancelDelete={() => (this.deleteId = false)}
                onConfirmDelete={() => hookStore.remove(deleteId as string)}
                hookStore={hookStore}
            />
        );
    }

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
    onRefresh: () => void;
    onReloadConfig: () => void;
    onTriggerHook: (id: string) => void;
    onDeleteHook: (id: string) => void;
    onCancelDelete: () => void;
    onConfirmDelete: () => void;
    hookStore: {getByID: (id: string) => IHook};
}> = ({
    hooks,
    deleteId,
    onRefresh,
    onReloadConfig,
    onTriggerHook,
    onDeleteHook,
    onCancelDelete,
    onConfirmDelete,
    hookStore,
}) => {
    const {t} = useTranslation();

    return (
        <DefaultPage
            title={t('hook.title')}
            rightControl={
                <ButtonGroup variant="contained" color="primary">
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
            <Grid item xs={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="hook-table">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('hook.name')}</TableCell>
                                <TableCell>{t('hook.description')}</TableCell>
                                <TableCell>{t('hook.command')}</TableCell>
                                <TableCell>{t('hook.httpMethods')}</TableCell>
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
    fDelete: VoidFunction;
}

const Row: React.FC<IRowProps> = observer(({hook, fTrigger, fDelete, classes}) => {
    const {t} = useTranslation();

    return (
        <TableRow>
            <TableCell>
                <strong>{hook.name}</strong>
                <br />
                <small style={{color: '#666'}}>ID: {hook.id}</small>
            </TableCell>
            <TableCell style={{maxWidth: 200, wordWrap: 'break-word'}}>
                {hook.description}
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

export default inject('hookStore')(Hooks);
