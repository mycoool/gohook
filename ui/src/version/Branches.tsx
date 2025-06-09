import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Button from '@material-ui/core/Button';
import Chip from '@material-ui/core/Chip';
import ArrowBack from '@material-ui/icons/ArrowBack';
import CallSplit from '@material-ui/icons/CallSplit';
import React, {Component, SFC} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IBranch} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';

@observer
class Branches extends Component<RouteComponentProps<{projectName: string}> & Stores<'versionStore'>> {
    @observable
    private switchBranch: string | false = false;

    public componentDidMount = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshBranches(projectName);
    };

    public render() {
        const {
            switchBranch,
            props: {versionStore, match},
        } = this;
        const projectName = match.params.projectName;
        const branches = versionStore.getBranches();
        
        return (
            <DefaultPage
                title={`分支管理 - ${projectName}`}
                rightControl={
                    <div style={{display: 'flex', gap: '8px'}}>
                        <Button
                            startIcon={<ArrowBack />}
                            onClick={() => this.goBack()}>
                            返回
                        </Button>
                        <Button
                            id="refresh-branches"
                            variant="contained"
                            color="primary"
                            onClick={() => this.refreshBranches()}>
                            刷新分支
                        </Button>
                    </div>
                }
                maxWidth={1000}>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="branch-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>分支名称</TableCell>
                                    <TableCell>状态</TableCell>
                                    <TableCell>最后提交</TableCell>
                                    <TableCell>提交时间</TableCell>
                                    <TableCell>操作</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {branches.map((branch: IBranch) => (
                                    <Row
                                        key={branch.name}
                                        branch={branch}
                                        onSwitch={() => (this.switchBranch = branch.name)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {switchBranch !== false && (
                    <ConfirmDialog
                        title="确认切换分支"
                        text={`确定要切换到分支 "${switchBranch}" 吗？`}
                        fClose={() => (this.switchBranch = false)}
                        fOnSubmit={() => this.performSwitchBranch(switchBranch)}
                    />
                )}
            </DefaultPage>
        );
    }

    private refreshBranches = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshBranches(projectName);
    };

    private goBack = () => {
        this.props.history.push('/versions');
    };

    private performSwitchBranch = (branchName: string) => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.switchBranch(projectName, branchName);
        this.switchBranch = false;
    };
}

interface IRowProps {
    branch: IBranch;
    onSwitch: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({branch, onSwitch}) => (
    <TableRow>
        <TableCell>
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                <strong>{branch.name}</strong>
                {branch.isCurrent && (
                    <Chip
                        label="当前分支"
                        size="small"
                        style={{backgroundColor: '#4caf50', color: 'white'}}
                    />
                )}
            </div>
        </TableCell>
        <TableCell>
            <Chip
                label={branch.isCurrent ? '当前' : '可切换'}
                size="small"
                style={{
                    backgroundColor: branch.isCurrent ? '#4caf50' : '#2196f3',
                    color: 'white'
                }}
            />
        </TableCell>
        <TableCell>
            <code style={{fontSize: '0.85em', backgroundColor: '#f5f5f5', padding: '2px 4px', borderRadius: '3px'}}>
                {branch.lastCommit}
            </code>
        </TableCell>
        <TableCell style={{fontSize: '0.85em'}}>
            {new Date(branch.lastCommitTime).toLocaleString()}
        </TableCell>
        <TableCell>
            {!branch.isCurrent && (
                <IconButton onClick={onSwitch} title="切换到此分支" size="small">
                    <CallSplit />
                </IconButton>
            )}
        </TableCell>
    </TableRow>
));

export default withRouter(inject('versionStore')(Branches)); 