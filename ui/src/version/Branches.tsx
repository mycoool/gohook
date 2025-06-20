import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import ArrowBack from '@mui/icons-material/ArrowBack';
import CallSplit from '@mui/icons-material/CallSplit';
import CloudSync from '@mui/icons-material/CloudSync';
import Refresh from '@mui/icons-material/Refresh';
import Computer from '@mui/icons-material/Computer';
import CloudQueue from '@mui/icons-material/CloudQueue';
import Delete from '@mui/icons-material/Delete';
import React, {Component} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IBranch} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';
import {Theme} from '@mui/material/styles';

import {WithStyles} from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import createStyles from '@mui/styles/createStyles';

// 添加样式定义
const styles = (theme: Theme) =>
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
                        <Button startIcon={<ArrowBack />} onClick={() => this.goBack()}>
                            返回
                        </Button>
                        <Button
                            id="refresh-branches"
                            color="primary"
                            startIcon={<Refresh />}
                            onClick={() => this.refreshBranches()}>
                            刷新
                        </Button>
                        <Button
                            id="sync-branches"
                            startIcon={<CloudSync />}
                            onClick={() => this.syncBranches()}>
                            同步
                        </Button>
                    </ButtonGroup>
                }
                maxWidth={1000}>
                <Grid size={12}>
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
                        fOnSubmit={() =>
                            this.deleteBranch && this.performDeleteBranch(this.deleteBranch)
                        }
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
                            color: 'white',
                        }}
                    />
                ) : (
                    <Chip label="游离状态" size="small" />
                )}
            </TableCell>
            <TableCell>
                <code className={classes.codeBlock}>{branch.lastCommit}</code>
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
