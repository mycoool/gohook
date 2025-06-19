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
import TextField from '@material-ui/core/TextField';
import InputAdornment from '@material-ui/core/InputAdornment';
import CircularProgress from '@material-ui/core/CircularProgress';
import Typography from '@material-ui/core/Typography';
import Box from '@material-ui/core/Box';
import ArrowBack from '@material-ui/icons/ArrowBack';
import CloudDownload from '@material-ui/icons/CloudDownload';
import Cached from '@material-ui/icons/Cached';
import Refresh from '@material-ui/icons/Refresh';
import Delete from '@material-ui/icons/Delete';
import Search from '@material-ui/icons/Search';
import Clear from '@material-ui/icons/Clear';
import React, {Component} from 'react';
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
    filterContainer: {
        marginBottom: theme.spacing(2),
        padding: theme.spacing(2),
    },
    filterInput: {
        minWidth: 280,
        maxWidth: 300,
    },
    loadingContainer: {
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        padding: theme.spacing(2),
    },
    statsContainer: {
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: theme.spacing(1, 2),
        backgroundColor: theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.05)' : 'rgba(0, 0, 0, 0.05)',
    },
});

type TagsProps = RouteComponentProps<{projectName: string}> & Stores<'versionStore'> & WithStyles<typeof styles>;

@observer
class Tags extends Component<TagsProps> {
    @observable
    private switchTag: string | false = false;

    @observable
    private deleteTag: string | false = false;

    @observable
    private filterText = '';

    @observable
    private filterTimeout: NodeJS.Timeout | null = null;

    public componentDidMount = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName);
        
        // 添加滚动监听器
        window.addEventListener('scroll', this.handleScroll);
    };

    public componentWillUnmount = () => {
        // 清理滚动监听器和定时器
        window.removeEventListener('scroll', this.handleScroll);
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }
    };

    public render() {
        const {
            props: {versionStore, match, classes},
        } = this;
        const projectName = match.params.projectName;
        const tags = versionStore.getTags();
        const tagsTotal = versionStore.getTagsTotal();
        const tagsHasMore = versionStore.getTagsHasMore();
        const tagsLoading = versionStore.getTagsLoading();
        
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
                            id="sync-tags"
                            startIcon={<CloudDownload />}
                            color="primary"
                            onClick={() => this.syncTags()}>
                            同步
                        </Button>
                        <Button
                            id="refresh-tags"
                            startIcon={<Refresh />}
                            color="primary"
                            onClick={() => this.refreshTags()}>
                            刷新
                        </Button>

                    </ButtonGroup>
                }
                maxWidth={1200}>
                <Grid item xs={12}>
                    <Paper elevation={2} className={classes.filterContainer}>
                        <TextField
                            className={classes.filterInput}
                            label="筛选标签"
                            placeholder="输入标签前缀，如 v0.1, v1.0 等"
                            value={this.filterText}
                            onChange={this.handleFilterChange}
                            variant="outlined"
                            size="small"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <Search />
                                    </InputAdornment>
                                ),
                                endAdornment: this.filterText ? (
                                    <InputAdornment position="end">
                                        <IconButton
                                            size="small"
                                            onClick={this.clearFilter}
                                            title="清除筛选">
                                            <Clear />
                                        </IconButton>
                                    </InputAdornment>
                                ) : null,
                            }}
                        />
                    </Paper>
                </Grid>
                <Grid item xs={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        {/* 统计信息 */}
                        <div className={classes.statsContainer}>
                            <Typography variant="body2" color="textSecondary">
                                共 {tagsTotal} 个标签，已显示 {tags.length} 个
                            </Typography>
                            {this.filterText && (
                                <Typography variant="body2" color="textSecondary">
                                    筛选条件: &quot;{this.filterText}&quot;
                                </Typography>
                            )}
                        </div>
                        
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
                        
                        {/* 加载状态 */}
                        {tagsLoading && (
                            <div className={classes.loadingContainer}>
                                <CircularProgress size={24} />
                                <Box ml={1}>
                                    <Typography variant="body2" color="textSecondary">
                                        加载中...
                                    </Typography>
                                </Box>
                            </div>
                        )}
                        
                        {/* 没有更多数据提示 */}
                        {!tagsLoading && !tagsHasMore && tags.length > 0 && (
                            <div className={classes.loadingContainer}>
                                <Typography variant="body2" color="textSecondary">
                                    已显示全部标签
                                </Typography>
                            </div>
                        )}
                        
                        {/* 空状态 */}
                        {!tagsLoading && tags.length === 0 && (
                            <div className={classes.loadingContainer}>
                                <Typography variant="body2" color="textSecondary">
                                    {this.filterText ? '没有找到匹配的标签' : '暂无标签'}
                                </Typography>
                            </div>
                        )}
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
        this.props.versionStore.refreshTags(projectName, this.filterText || undefined);
    };

    private syncTags = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.syncTags(projectName);
    };

    private goBack = () => {
        this.props.history.push('/versions');
    };

    private handleFilterChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const value = event.target.value;
        this.filterText = value;

        // 清除之前的定时器
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }

        // 设置新的定时器，延迟500ms后执行筛选
        this.filterTimeout = setTimeout(() => {
            const projectName = this.props.match.params.projectName;
            this.props.versionStore.refreshTags(projectName, value || undefined);
        }, 500);
    };

    private clearFilter = () => {
        this.filterText = '';
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName);
    };

    private handleScroll = () => {
        // 检查是否滚动到页面底部
        const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
        const windowHeight = window.innerHeight;
        const documentHeight = document.documentElement.scrollHeight;
        
        // 当滚动到距离底部100px时开始加载
        if (scrollTop + windowHeight >= documentHeight - 100) {
            const projectName = this.props.match.params.projectName;
            this.props.versionStore.loadMoreTags(projectName, this.filterText || undefined);
        }
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

const Row: React.FC<IRowProps> = observer(({tag, onSwitch, onDelete, classes}) => (
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

export default (withRouter as any)((inject as any)('versionStore')((withStyles as any)(styles)(Tags))); 