import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Button from '@material-ui/core/Button';
import ButtonGroup from '@material-ui/core/ButtonGroup';
import Chip from '@material-ui/core/Chip';
import ArrowBack from '@material-ui/icons/ArrowBack';
import Cached from '@material-ui/icons/Cached';
import Refresh from '@material-ui/icons/Refresh';
import Delete from '@material-ui/icons/Delete';
import React, {Component, SFC} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {ITag} from '../types';
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

@observer
class Tags extends Component<RouteComponentProps<{projectName: string}> & Stores<'versionStore'>> {
    @observable
    private switchTag: string | false = false;

    @observable
    private deleteTag: string | false = false;

    public componentDidMount = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName);
    };

    public render() {
        const {
            props: {versionStore, match},
        } = this;
        const projectName = match.params.projectName;
        const tags = versionStore.getTags();
        
        return (
            <DefaultPage
                title={`标签管理 - ${projectName}`}
                rightControl={
                    <ButtonGroup variant="contained" color="primary">
                        <Button
                            startIcon={<ArrowBack />}
                            onClick={() => this.goBack()}>
                            返回
                        </Button>
                        <Button
                            id="refresh-tags"
                            startIcon={<Refresh />}
                            color="primary"
                            onClick={() => this.refreshTags()}>
                            刷新标签
                        </Button>
                    </ButtonGroup>
                }
                maxWidth={1200}>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        <Table id="tag-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>标签名称</TableCell>
                                    <TableCell>状态</TableCell>
                                    <TableCell>提交哈希</TableCell>
                                    <TableCell>创建时间</TableCell>
                                    <TableCell>说明</TableCell>
                                    <TableCell>操作</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {tags.map((tag: ITag) => (
                                    <StyledRow
                                        key={tag.name}
                                        tag={tag}
                                        onSwitch={() => (this.switchTag = tag.name)}
                                        onDelete={() => (this.deleteTag = tag.name)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {this.switchTag !== false && (
                    <ConfirmDialog
                        title="确认切换标签"
                        text={`确定要切换到标签 "${this.switchTag}" 吗？这将使项目进入分离头指针状态。`}
                        fClose={() => (this.switchTag = false)}
                        fOnSubmit={() => this.switchTag && this.performSwitchTag(this.switchTag)}
                    />
                )}
                {this.deleteTag !== false && (
                    <ConfirmDialog
                        title="确认删除标签"
                        text={`确定要删除标签 "${this.deleteTag}" 吗？此操作将同时删除本地和远程标签，不可撤销。`}
                        fClose={() => (this.deleteTag = false)}
                        fOnSubmit={() => this.deleteTag && this.performDeleteTag(this.deleteTag)}
                    />
                )}
            </DefaultPage>
        );
    }

    private refreshTags = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName);
    };

    private goBack = () => {
        this.props.history.push('/versions');
    };

    private performSwitchTag = (tagName: string) => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.switchTag(projectName, tagName);
        this.switchTag = false;
    };

    private performDeleteTag = (tagName: string) => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.deleteTag(projectName, tagName);
        this.deleteTag = false;
    };
}

interface IRowProps extends WithStyles<typeof styles> {
    tag: ITag;
    onSwitch: VoidFunction;
    onDelete: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({tag, onSwitch, onDelete, classes}) => (
    <TableRow>
        <TableCell>
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                <strong>{tag.name}</strong>
                {tag.isCurrent && (
                    <Chip
                        label="当前标签"
                        size="small"
                        style={{backgroundColor: '#2196f3', color: 'white'}}
                    />
                )}
            </div>
        </TableCell>
        <TableCell>
            <Chip
                label={tag.isCurrent ? '当前' : '可切换'}
                size="small"
                style={{
                    backgroundColor: tag.isCurrent ? '#2196f3' : '#4caf50',
                    color: 'white'
                }}
            />
        </TableCell>
        <TableCell>
            <code className={classes.codeBlock}>
                {tag.commitHash}
            </code>
        </TableCell>
        <TableCell style={{fontSize: '0.85em'}}>
            {new Date(tag.date).toLocaleString()}
        </TableCell>
        <TableCell style={{maxWidth: 200, wordWrap: 'break-word', fontSize: '0.85em'}}>
            {tag.message || '无说明'}
        </TableCell>
        <TableCell>
            {!tag.isCurrent && (
                <IconButton onClick={onSwitch} title="切换到此标签" size="small">
                    <Cached />
                </IconButton>
            )}
            {!tag.isCurrent && (
                <IconButton onClick={onDelete} title="删除标签" size="small">
                    <Delete />
                </IconButton>
            )}
        </TableCell>
    </TableRow>
));

// 使用 withStyles 包装 Row 组件
const StyledRow = withStyles(styles)(Row);

export default withRouter(inject('versionStore')(Tags)); 