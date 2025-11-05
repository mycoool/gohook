import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Chip from '@mui/material/Chip';
import TextField from '@mui/material/TextField';
import InputAdornment from '@mui/material/InputAdornment';
import CircularProgress from '@mui/material/CircularProgress';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import ArrowBack from '@mui/icons-material/ArrowBack';
import CloudSync from '@mui/icons-material/CloudSync';
import Cached from '@mui/icons-material/Cached';
import Refresh from '@mui/icons-material/Refresh';
import Delete from '@mui/icons-material/Delete';
import Search from '@mui/icons-material/Search';
import Clear from '@mui/icons-material/Clear';
import React, {Component} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import ConfirmDialogWithOptions from '../common/ConfirmDialogWithOptions';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {ITag} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';
import {withStyles, WithStyles, createStyles} from '@mui/styles';
import {Theme} from '@mui/material/styles';
import useTranslation from '../i18n/useTranslation';

// 临时保留@mui/styles以确保构建通过，使用固定样式避免主题兼容性问题
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
        filterContainer: {
            marginBottom: '16px',
            padding: '16px',
        },
        filterInput: {
            minWidth: 280,
            maxWidth: 300,
        },
        loadingContainer: {
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            padding: '16px',
        },
        statsContainer: {
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '8px 16px',
            backgroundColor: 'rgba(0, 0, 0, 0.05)',
        },
    });

type TagsProps = RouteComponentProps<{projectName: string}> &
    Stores<'versionStore'> &
    WithStyles<typeof styles>;

interface TagsPropsWithTranslation extends TagsProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

@observer
class Tags extends Component<TagsPropsWithTranslation> {
    @observable
    private switchTag: string | false = false;

    @observable
    private deleteTag: string | false = false;

    @observable
    private filterText = '';

    @observable
    private messageFilterText = '';

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
            props: {versionStore, match, classes, t},
        } = this;
        const projectName = match.params.projectName;
        const tags = versionStore.getTags();
        const tagsTotal = versionStore.getTagsTotal();
        const tagsHasMore = versionStore.getTagsHasMore();
        const tagsLoading = versionStore.getTagsLoading();

        return (
            <DefaultPage
                title={t('version.tagManagementTitle', {name: projectName})}
                rightControl={
                    <ButtonGroup variant="contained" color="primary">
                        <Button startIcon={<ArrowBack />} onClick={() => this.goBack()}>
                            {t('common.back')}
                        </Button>
                        <Button
                            id="refresh-tags"
                            startIcon={<Refresh />}
                            color="primary"
                            onClick={() => this.refreshTags()}>
                            {t('common.refresh')}
                        </Button>
                        <Button
                            id="sync-tags"
                            startIcon={<CloudSync />}
                            color="primary"
                            onClick={() => this.syncTags()}>
                            {t('version.syncTags')}
                        </Button>
                    </ButtonGroup>
                }
                maxWidth={1200}>
                <Grid size={12}>
                    <Paper elevation={2} className={classes.filterContainer}>
                        <div style={{display: 'flex', gap: '16px', flexWrap: 'wrap'}}>
                            <TextField
                                className={classes.filterInput}
                                label={t('version.filterTagLabel')}
                                placeholder={t('version.filterTagPlaceholder')}
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
                                                title={t('version.clearFilter')}>
                                                <Clear />
                                            </IconButton>
                                        </InputAdornment>
                                    ) : null,
                                }}
                            />
                            <TextField
                                className={classes.filterInput}
                                label={t('version.filterTagMessageLabel')}
                                placeholder={t('version.filterTagMessagePlaceholder')}
                                value={this.messageFilterText}
                                onChange={this.handleMessageFilterChange}
                                variant="outlined"
                                size="small"
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <Search />
                                        </InputAdornment>
                                    ),
                                    endAdornment: this.messageFilterText ? (
                                        <InputAdornment position="end">
                                            <IconButton
                                                size="small"
                                                onClick={this.clearMessageFilter}
                                                title={t('version.clearMessageFilter')}>
                                                <Clear />
                                            </IconButton>
                                        </InputAdornment>
                                    ) : null,
                                }}
                            />
                        </div>
                    </Paper>
                </Grid>
                <Grid size={12}>
                    <Paper elevation={6} style={{overflowX: 'auto'}}>
                        {/* 统计信息 */}
                        <div className={classes.statsContainer}>
                            <Typography variant="body2" color="textSecondary">
                                {t('version.tagStats', {total: tagsTotal, displayed: tags.length})}
                            </Typography>
                            {(this.filterText || this.messageFilterText) && (
                                <Typography variant="body2" color="textSecondary">
                                    {t('version.filterConditionsLabel')}
                                    {this.filterText &&
                                        ` ${t('version.tagNameContains', {
                                            value: this.filterText,
                                        })}`}
                                    {this.filterText && this.messageFilterText && ', '}
                                    {this.messageFilterText &&
                                        ` ${t('version.tagMessageContains', {
                                            value: this.messageFilterText,
                                        })}`}
                                </Typography>
                            )}
                        </div>

                        <Table id="tag-table">
                            <TableHead>
                                <TableRow>
                                    <TableCell>{t('version.tagName')}</TableCell>
                                    <TableCell>{t('common.status')}</TableCell>
                                    <TableCell>{t('version.commitHash')}</TableCell>
                                    <TableCell>{t('version.createdAt')}</TableCell>
                                    <TableCell>{t('version.description')}</TableCell>
                                    <TableCell>{t('common.actions')}</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {tags.map((tag: ITag) => (
                                    <StyledRow
                                        key={tag.name}
                                        tag={tag}
                                        onSwitch={() => (this.switchTag = tag.name)}
                                        onDelete={() => (this.deleteTag = tag.name)}
                                        t={t}
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
                                        {t('common.loading')}
                                    </Typography>
                                </Box>
                            </div>
                        )}

                        {/* 没有更多数据提示 */}
                        {!tagsLoading && !tagsHasMore && tags.length > 0 && (
                            <div className={classes.loadingContainer}>
                                <Typography variant="body2" color="textSecondary">
                                    {t('version.allTagsLoaded')}
                                </Typography>
                            </div>
                        )}

                        {/* 空状态 */}
                        {!tagsLoading && tags.length === 0 && (
                            <div className={classes.loadingContainer}>
                                <Typography variant="body2" color="textSecondary">
                                    {this.filterText || this.messageFilterText
                                        ? t('version.noMatchingTags')
                                        : t('version.noTags')}
                                </Typography>
                            </div>
                        )}
                    </Paper>
                </Grid>
                {this.switchTag !== false && (
                    <ConfirmDialogWithOptions
                        title={t('version.confirmSwitchTagTitle')}
                        text={t('version.confirmSwitchTagText', {name: this.switchTag})}
                        fClose={() => (this.switchTag = false)}
                        fOnSubmit={(force) =>
                            this.switchTag && this.performSwitchTag(this.switchTag, force)
                        }
                        forceOptionLabel={t('version.forceSwitchLabel')}
                        forceOptionDescription={t('version.forceSwitchDescription')}
                        warningText={t('version.forceSwitchWarning')}
                    />
                )}
                {this.deleteTag !== false && (
                    <ConfirmDialog
                        title={t('version.confirmDeleteTagTitle')}
                        text={t('version.confirmDeleteTagText', {name: this.deleteTag})}
                        fClose={() => (this.deleteTag = false)}
                        fOnSubmit={() => this.deleteTag && this.performDeleteTag(this.deleteTag)}
                    />
                )}
            </DefaultPage>
        );
    }

    private refreshTags = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(
            projectName,
            this.filterText || undefined,
            this.messageFilterText || undefined
        );
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
            this.props.versionStore.refreshTags(
                projectName,
                value || undefined,
                this.messageFilterText || undefined
            );
        }, 500);
    };

    private clearFilter = () => {
        this.filterText = '';
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(
            projectName,
            undefined,
            this.messageFilterText || undefined
        );
    };

    private handleMessageFilterChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const value = event.target.value;
        this.messageFilterText = value;

        // 清除之前的定时器
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }

        // 设置新的定时器，延迟500ms后执行筛选
        this.filterTimeout = setTimeout(() => {
            const projectName = this.props.match.params.projectName;
            this.props.versionStore.refreshTags(
                projectName,
                this.filterText || undefined,
                value || undefined
            );
        }, 500);
    };

    private clearMessageFilter = () => {
        this.messageFilterText = '';
        if (this.filterTimeout) {
            clearTimeout(this.filterTimeout);
        }
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName, this.filterText || undefined, undefined);
    };

    private handleScroll = () => {
        // 检查是否滚动到页面底部
        const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
        const windowHeight = window.innerHeight;
        const documentHeight = document.documentElement.scrollHeight;

        // 当滚动到距离底部100px时开始加载
        if (scrollTop + windowHeight >= documentHeight - 100) {
            const projectName = this.props.match.params.projectName;
            this.props.versionStore.loadMoreTags(
                projectName,
                this.filterText || undefined,
                this.messageFilterText || undefined
            );
        }
    };

    private performSwitchTag = (tagName: string, force: boolean) => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.switchTag(projectName, tagName, force);
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
    t: TagsPropsWithTranslation['t'];
}

const Row: React.FC<IRowProps> = observer(({tag, onSwitch, onDelete, classes, t}) => (
    <TableRow>
        <TableCell>
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                <strong>{tag.name}</strong>
                {tag.isCurrent && (
                    <Chip
                        label={t('version.currentTagLabel')}
                        size="small"
                        style={{backgroundColor: '#2196f3', color: 'white'}}
                    />
                )}
            </div>
        </TableCell>
        <TableCell>
            <Chip
                label={
                    tag.isCurrent ? t('version.tagStatusCurrent') : t('version.tagStatusSwitchable')
                }
                size="small"
                style={{
                    backgroundColor: tag.isCurrent ? '#2196f3' : '#4caf50',
                    color: 'white',
                }}
            />
        </TableCell>
        <TableCell>
            <code className={classes.codeBlock}>{tag.commitHash}</code>
        </TableCell>
        <TableCell style={{fontSize: '0.85em'}}>{new Date(tag.date).toLocaleString()}</TableCell>
        <TableCell style={{maxWidth: 200, wordWrap: 'break-word', fontSize: '0.85em'}}>
            {tag.message || t('version.noDescription')}
        </TableCell>
        <TableCell>
            {!tag.isCurrent && (
                <IconButton onClick={onSwitch} title={t('version.switchToTag')} size="small">
                    <Cached />
                </IconButton>
            )}
            {!tag.isCurrent && (
                <IconButton onClick={onDelete} title={t('version.deleteTag')} size="small">
                    <Delete />
                </IconButton>
            )}
        </TableCell>
    </TableRow>
));

// 使用 withStyles 包装 Row 组件
const StyledRow = withStyles(styles)(Row);

const TagsWithTranslation: React.FC<TagsProps> = (props) => {
    const {t} = useTranslation();
    return <Tags {...props} t={t} />;
};

export default (withRouter as any)(
    (inject as any)('versionStore')((withStyles as any)(styles)(TagsWithTranslation))
);
