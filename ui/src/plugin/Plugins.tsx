import React, {Component} from 'react';
import {Link} from 'react-router-dom';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Settings from '@mui/icons-material/Settings';
import {Switch, Button} from '@mui/material';
import DefaultPage from '../common/DefaultPage';
import CopyableSecret from '../common/CopyableSecret';
import {observer} from 'mobx-react';
import {inject, Stores} from '../inject';
import {IPlugin} from '../types';

@observer
class Plugins extends Component<Stores<'pluginStore' | 'currentUser'>> {
    public componentDidMount() {
        // 只在用户已登录时才进行 API 调用
        if (this.props.currentUser.loggedIn) {
            this.props.pluginStore.refresh();
        }
    }

    public componentDidUpdate(prevProps: Stores<'pluginStore' | 'currentUser'>) {
        // 当用户登录状态改变时，重新加载数据
        if (prevProps.currentUser.loggedIn !== this.props.currentUser.loggedIn && this.props.currentUser.loggedIn) {
            this.props.pluginStore.refresh();
        }
    }

    public render() {
        const {
            props: {pluginStore},
        } = this;
        const plugins = pluginStore.getItems();
        return (
            <DefaultPage title="Plugins" maxWidth={1000}>
                <Grid size={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="plugin-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>ID</TableCell>
                                    <TableCell>Enabled</TableCell>
                                    <TableCell>Name</TableCell>
                                    <TableCell>Token</TableCell>
                                    <TableCell>Details</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {plugins.map((plugin: IPlugin) => (
                                    <Row
                                        key={plugin.token}
                                        id={plugin.id}
                                        token={plugin.token}
                                        name={plugin.name}
                                        enabled={plugin.enabled}
                                        fToggleStatus={() =>
                                            this.props.pluginStore.changeEnabledState(
                                                plugin.id,
                                                !plugin.enabled
                                            )
                                        }
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
            </DefaultPage>
        );
    }
}

interface IRowProps {
    id: number;
    name: string;
    token: string;
    enabled: boolean;
    fToggleStatus: VoidFunction;
}

const Row: React.FC<IRowProps> = observer(({name, id, token, enabled, fToggleStatus}) => (
    <TableRow>
        <TableCell>{id}</TableCell>
        <TableCell>
            <Switch
                checked={enabled}
                onClick={fToggleStatus}
                className="switch"
                data-enabled={enabled}
            />
        </TableCell>
        <TableCell>{name}</TableCell>
        <TableCell>
            <CopyableSecret value={token} style={{display: 'flex', alignItems: 'center'}} />
        </TableCell>
        <TableCell align="right" padding="none">
            <Link to={'/plugins/' + id}>
                <Button>
                    <Settings />
                </Button>
            </Link>
        </TableCell>
    </TableRow>
));

export default inject('pluginStore', 'currentUser')(Plugins);
