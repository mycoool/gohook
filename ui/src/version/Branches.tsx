import ButtonGroup from '@material-ui/core/ButtonGroup';
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
import CloudDownload from '@material-ui/icons/CloudDownload';
import Refresh from '@material-ui/icons/Refresh';
import Computer from '@material-ui/icons/Computer';
import CloudQueue from '@material-ui/icons/CloudQueue';
import Delete from '@material-ui/icons/Delete';
import React, {Component} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IBranch} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';
import {withStyles, WithStyles, Theme, createStyles} from '@material-ui/core/styles';

// 添加样式定义
const styles = (theme: Theme) => createStyles({
    codeBlock: {
        fontSize: '0.85em',
        backgroundColor: theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
        color: theme.palette.text.primary,
        padding: '2px 4px',
        borderRadius: '3px',
        border: theme.palette.type === 'dark' ? '1px solid rgba(255, 255, 255, 0.2)' : '1px solid rgba(0, 0, 0, 0.2)',
    },
});

type BranchesProps = RouteComponentProps<{projectName: string}> & Stores<'versionStore'>;

@observer
class Branches extends Component<BranchesProps> {
    @observable
    private switchBranch: string | false = false;

    @observable
    private deleteBranch: string | false = false;

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
                    <ButtonGroup variant="contained" color="primary">
                        <Button
                            startIcon={<ArrowBack />}
                            onClick={() => this.goBack()}>
                            返回
                        </Button>
                        <Button
                            id="sync-branches"
                            startIcon={<CloudDownload />}
                            onClick={() => this.syncBranches()}>
                            同步
                        </Button>
                        <Button
                            id="refresh-branches"
                            color="primary"
                            startIcon={<Refresh />}
                            onClick={() => this.refreshBranches()}>
                            刷新
                        </Button>
                    </ButtonGroup>
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
                                    <StyledRow
                                        key={branch.name}
                                        branch={branch}
                                        onSwitch={() => (this.switchBranch = branch.name)}
                                        onDelete={() => (this.deleteBranch = branch.name)}
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
                {this.deleteBranch !== false && (
                    <ConfirmDialog
                        title="确认删除分支"
                        text={`确定要删除分支 "${this.deleteBranch}" 吗？此操作不可撤销。`}
                        fClose={() => (this.deleteBranch = false)}
                        fOnSubmit={() => this.deleteBranch && this.performDeleteBranch(this.deleteBranch)}
                    />
                )}
            </DefaultPage>
        );
    }

    private syncBranches = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.syncBranches(projectName);
    };

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

    private performDeleteBranch = (branchName: string) => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.deleteBranch(projectName, branchName);
        this.deleteBranch = false;
    };
}

interface IRowProps extends WithStyles<typeof styles> {
    branch: IBranch;
    onSwitch: VoidFunction;
    onDelete: VoidFunction;
}

const Row: React.FC<IRowProps> = observer(({branch, onSwitch, onDelete, classes}) => {
    const renderBranchName = () => {
        let icon = null;
        let title = '';

        if (branch.type === 'local') {
            icon = <Computer fontSize="small" />;
            title = '本地分支';
        } else if (branch.type === 'remote') {
            icon = <CloudQueue fontSize="small" />;
            title = '远程分支';
        }

        return (
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                {icon && <span title={title}>{icon}</span>}
                <strong>{branch.name}</strong>
            </div>
        );
    };

    return (
        <TableRow>
            <TableCell>
                <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                    {renderBranchName()}
                    {branch.isCurrent && branch.type === 'local' && (
                        <Chip
                            label="当前分支"
                            size="small"
                            style={{backgroundColor: '#4caf50', color: 'white'}}
                        />
                    )}
                </div>
            </TableCell>
            <TableCell>
                {branch.type !== 'detached' ? (
                    <Chip
                        label={branch.isCurrent ? '当前' : '可切换'}
                        size="small"
                        style={{
                            backgroundColor: branch.isCurrent ? '#4caf50' : '#2196f3',
                            color: 'white'
                        }}
                    />
                ) : (
                    <Chip label="游离状态" size="small" />
                )}
            </TableCell>
            <TableCell>
                <code className={classes.codeBlock}>
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
                {branch.type === 'local' && !branch.isCurrent && (
                    <IconButton onClick={onDelete} title="删除分支" size="small">
                        <Delete />
                    </IconButton>
                )}
            </TableCell>
        </TableRow>
    );
});

// 使用 withStyles 包装 Row 组件
const StyledRow = withStyles(styles)(Row);

export default (withRouter as any)((inject('versionStore') as any)(Branches)); 