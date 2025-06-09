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
import LocalOffer from '@material-ui/icons/LocalOffer';
import React, {Component, SFC} from 'react';
import DefaultPage from '../common/DefaultPage';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {ITag} from '../types';
import {withRouter, RouteComponentProps} from 'react-router-dom';

@observer
class Tags extends Component<RouteComponentProps<{projectName: string}> & Stores<'versionStore'>> {
    @observable
    private switchTag: string | false = false;

    public componentDidMount = () => {
        const projectName = this.props.match.params.projectName;
        this.props.versionStore.refreshTags(projectName);
    };

    public render() {
        const {
            switchTag,
            props: {versionStore, match},
        } = this;
        const projectName = match.params.projectName;
        const tags = versionStore.getTags();
        
        return (
            <DefaultPage
                title={`标签管理 - ${projectName}`}
                rightControl={
                    <div style={{display: 'flex', gap: '8px'}}>
                        <Button
                            startIcon={<ArrowBack />}
                            onClick={() => this.goBack()}>
                            返回
                        </Button>
                        <Button
                            id="refresh-tags"
                            variant="contained"
                            color="primary"
                            onClick={() => this.refreshTags()}>
                            刷新标签
                        </Button>
                    </div>
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
                                    <Row
                                        key={tag.name}
                                        tag={tag}
                                        onSwitch={() => (this.switchTag = tag.name)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>
                {switchTag !== false && (
                    <ConfirmDialog
                        title="确认切换标签"
                        text={`确定要切换到标签 "${switchTag}" 吗？这将使项目进入分离头指针状态。`}
                        fClose={() => (this.switchTag = false)}
                        fOnSubmit={() => this.performSwitchTag(switchTag)}
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
}

interface IRowProps {
    tag: ITag;
    onSwitch: VoidFunction;
}

const Row: SFC<IRowProps> = observer(({tag, onSwitch}) => (
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
            <code style={{fontSize: '0.85em', backgroundColor: '#f5f5f5', padding: '2px 4px', borderRadius: '3px'}}>
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
                    <LocalOffer />
                </IconButton>
            )}
        </TableCell>
    </TableRow>
));

export default withRouter(inject('versionStore')(Tags)); 