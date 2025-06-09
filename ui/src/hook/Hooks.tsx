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
import React, {Component, SFC} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@material-ui/core/Button';
import Chip from '@material-ui/core/Chip';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IHook} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';

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
            <DefaultPage
                title="Hooks"
                rightControl={
                    <ButtonGroup variant="contained" color="primary">
                        <Button
                            id="refresh-hooks"
                            startIcon={<Refresh />}
                            onClick={() => this.refreshHooks()}>
                            刷新
                        </Button>
                        <Button
                            id="reload-config"
                            startIcon={<CloudDownload />}
                            onClick={() => this.reloadConfig()}>
                            重新加载配置
                        </Button>
                    </ButtonGroup>
                }
                maxWidth={1200}>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="hook-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Name</TableCell>
                                    <TableCell>Description</TableCell>
                                    <TableCell>Command</TableCell>
                                    <TableCell>HTTP Methods</TableCell>
                                    <TableCell>Trigger Rules</TableCell>
                                    <TableCell>Status</TableCell>
                                    <TableCell>Last Used</TableCell>
                                    <TableCell />
                                    <TableCell />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {hooks.map((hook: IHook) => (
                                    <Row
                                        key={hook.id}
                                        hook={hook}
                                        fTrigger={() => this.triggerHook(hook.id)}
                                        fDelete={() => (this.deleteId = hook.id)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {deleteId !== false && (
                    <ConfirmDialog
                        title="Confirm Delete"
                        text={'Delete hook ' + hookStore.getByID(deleteId).name + '?'}
                        fClose={() => (this.deleteId = false)}
                        fOnSubmit={() => hookStore.remove(deleteId)}
                    />
                )}
            </DefaultPage>
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

interface IRowProps {
    hook: IHook;
    fTrigger: VoidFunction;
    fDelete: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({hook, fTrigger, fDelete}) => (
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
            <code style={{fontSize: '0.85em', backgroundColor: '#f5f5f5', padding: '2px 4px', borderRadius: '3px'}}>
                {hook.executeCommand}
            </code>
            {hook.workingDirectory && (
                <div style={{fontSize: '0.8em', color: '#666', marginTop: '4px'}}>
                    Working Dir: {hook.workingDirectory}
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
                        fontSize: '0.7em'
                    }}
                />
            ))}
        </TableCell>
        <TableCell style={{maxWidth: 150, wordWrap: 'break-word', fontSize: '0.85em'}}>
            {hook.triggerRuleDescription}
        </TableCell>
        <TableCell>
            <Chip
                label={hook.status}
                size="small"
                style={{
                    backgroundColor: hook.status === 'active' ? '#4caf50' : '#f44336',
                    color: 'white'
                }}
            />
        </TableCell>
        <TableCell>
            <LastUsedCell lastUsed={hook.lastUsed} />
        </TableCell>
        <TableCell align="right" padding="none">
            <IconButton onClick={fTrigger} className="trigger" title="Trigger Hook">
                <PlayArrow />
            </IconButton>
        </TableCell>
        <TableCell align="right" padding="none">
            <IconButton onClick={fDelete} className="delete" title="Delete Hook">
                <Delete />
            </IconButton>
        </TableCell>
    </TableRow>
));

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

export default inject('hookStore')(Hooks); 