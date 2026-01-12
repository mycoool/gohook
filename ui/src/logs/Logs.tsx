import React, {Component} from 'react';
import {
    Box,
    Typography,
    Card,
    CardContent,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    Chip,
    IconButton,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    TextField,
    Button,
    CircularProgress,
    Tooltip,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Stack,
    InputAdornment,
} from '@mui/material';
import {
    Refresh as RefreshIcon,
    Download as DownloadIcon,
    Delete as DeleteIcon,
    Visibility as VisibilityIcon,
    Clear as ClearIcon,
} from '@mui/icons-material';
import {observer} from 'mobx-react';
import LogStore, {LogEntry} from './LogStore';
import useTranslation from '../i18n/useTranslation';

interface LogsState {
    selectedLog: LogEntry | null;
    detailDialogOpen: boolean;
    cleanupDialogOpen: boolean;
    cleanupDays: number;
}

interface LogsProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

@observer
class Logs extends Component<LogsProps, LogsState> {
    private logStore = new LogStore();

    constructor(props: LogsProps) {
        super(props);
        this.state = {
            selectedLog: null,
            detailDialogOpen: false,
            cleanupDialogOpen: false,
            cleanupDays: 30,
        };
    }

    componentDidMount() {
        this.logStore.loadLogs();
        // 添加滚动监听器
        window.addEventListener('scroll', this.handleScroll);
    }

    componentWillUnmount() {
        this.logStore.destroy();
        // 移除滚动监听器
        window.removeEventListener('scroll', this.handleScroll);
    }

    handleScroll = () => {
        // 检查是否接近页面底部
        const threshold = 100; // 距离底部100px时开始加载
        const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
        const windowHeight = window.innerHeight;
        const documentHeight = document.documentElement.scrollHeight;

        if (
            scrollTop + windowHeight >= documentHeight - threshold &&
            this.logStore.hasMore &&
            !this.logStore.loading
        ) {
            this.handleLoadMore();
        }
    };

    handleRefresh = () => {
        this.logStore.refreshLogs();
    };

    handleExport = () => {
        this.logStore.exportLogs();
    };

    handleCleanup = () => {
        this.setState({cleanupDialogOpen: true});
    };

    handleCleanupConfirm = async () => {
        try {
            await this.logStore.clearLogs(this.state.cleanupDays);
            this.setState({cleanupDialogOpen: false});
        } catch (error) {
            console.error('清理日志失败:', error);
        }
    };

    handleViewDetail = (log: LogEntry) => {
        this.setState({selectedLog: log, detailDialogOpen: true});
    };

    handleLoadMore = () => {
        this.logStore.loadMore();
    };

    getStatusChip = (log: LogEntry) => {
        // 对于有success字段的日志（hook、user、project类型）
        if (log.success !== undefined) {
            return (
                <Chip
                    size="small"
                    label={log.success ? this.props.t('logs.success') : this.props.t('logs.failed')}
                    color={log.success ? 'success' : 'error'}
                />
            );
        }
        // 对于系统日志，使用level字段
        if (log.level) {
            const colors: Record<string, 'default' | 'info' | 'warning' | 'error'> = {
                INFO: 'info',
                WARN: 'warning',
                ERROR: 'error',
                DEBUG: 'default',
                info: 'info',
                warn: 'warning',
                error: 'error',
                debug: 'default',
            };
            const levelKey = log.level.toUpperCase();
            const translationKey = `logs.logLevel.${levelKey.toLowerCase()}`;
            const translatedLevel = this.props.t(translationKey);

            return (
                <Chip
                    size="small"
                    label={translatedLevel !== translationKey ? translatedLevel : levelKey}
                    color={colors[levelKey] || 'default'}
                />
            );
        }
        return <span>-</span>;
    };

    getTypeChip = (type: string) => {
        const colors: Record<string, 'primary' | 'secondary' | 'default'> = {
            hook: 'primary',
            system: 'secondary',
            user: 'default',
            project: 'default',
        };

        // 确保type在已知类型列表中
        const validTypes = ['hook', 'system', 'user', 'project'];
        const displayType = validTypes.includes(type) ? type : 'system';

        return (
            <Chip
                size="small"
                label={this.props.t(`logs.${displayType}Logs`)}
                color={colors[displayType] || 'default'}
            />
        );
    };

    getActionTranslation = (action: string, type?: string) => {
        // 根据日志类型选择不同的翻译源
        if (type === 'project') {
            // 项目操作翻译
            const translation = this.props.t(`logs.projectActions.${action}`);
            if (translation !== `logs.projectActions.${action}`) {
                return translation;
            }
        }

        // 用户操作翻译
        const userActionTranslation = this.props.t(`logs.userActions.${action}`);
        if (userActionTranslation !== `logs.userActions.${action}`) {
            return userActionTranslation;
        }

        // 如果都没有找到翻译，返回原始action
        return action;
    };

    getResourceTranslation = (resource: string) => {
        // 尝试翻译资源类型
        const translation = this.props.t(`logs.resources.${resource}`);
        if (translation !== `logs.resources.${resource}`) {
            return translation;
        }

        // 如果没有找到翻译，返回原始resource
        return resource;
    };

    renderFilters = () => {
        const {filters} = this.logStore;

        return (
            <Card sx={{mb: 2}}>
                <CardContent>
                    <Typography variant="h6" gutterBottom>
                        {this.props.t('logs.filterByType')}
                    </Typography>
                    <Stack spacing={2}>
                        <Box sx={{display: 'flex', gap: 2, flexWrap: 'wrap'}}>
                            <FormControl sx={{minWidth: 200}}>
                                <InputLabel>{this.props.t('logs.filterByType')}</InputLabel>
                                <Select
                                    value={filters.type || ''}
                                    onChange={(e) => {
                                        const newType = e.target.value || undefined;
                                        this.logStore.setFilters({
                                            type: newType,
                                        });
                                    }}>
                                    <MenuItem value="">{this.props.t('logs.allLogs')}</MenuItem>
                                    <MenuItem value="hook">
                                        {this.props.t('logs.hookLogs')}
                                    </MenuItem>
                                    <MenuItem value="system">
                                        {this.props.t('logs.systemLogs')}
                                    </MenuItem>
                                    <MenuItem value="user">
                                        {this.props.t('logs.userLogs')}
                                    </MenuItem>
                                    <MenuItem value="project">
                                        {this.props.t('logs.projectLogs')}
                                    </MenuItem>
                                </Select>
                            </FormControl>
                            <FormControl sx={{minWidth: 200}}>
                                <InputLabel>{this.props.t('logs.level')}</InputLabel>
                                <Select
                                    value={filters.level || ''}
                                    onChange={(e) => {
                                        const newLevel = e.target.value || undefined;
                                        this.logStore.setFilters({
                                            level: newLevel,
                                        });
                                    }}>
                                    <MenuItem value="">
                                        {this.props.t('logs.statusFilter.all')}
                                    </MenuItem>
                                    <MenuItem value="info">
                                        {this.props.t('logs.logLevel.info')}
                                    </MenuItem>
                                    <MenuItem value="warn">
                                        {this.props.t('logs.logLevel.warn')}
                                    </MenuItem>
                                    <MenuItem value="error">
                                        {this.props.t('logs.logLevel.error')}
                                    </MenuItem>
                                    <MenuItem value="debug">
                                        {this.props.t('logs.logLevel.debug')}
                                    </MenuItem>
                                </Select>
                            </FormControl>
                            <TextField
                                label={this.props.t('logs.startDate')}
                                type="datetime-local"
                                value={filters.startDate || ''}
                                onChange={(e) => {
                                    const newStartDate = e.target.value || undefined;
                                    this.logStore.setFilters({
                                        startDate: newStartDate,
                                    });
                                }}
                                InputLabelProps={{shrink: true}}
                                sx={{minWidth: 200}}
                            />
                            <TextField
                                label={this.props.t('logs.endDate')}
                                type="datetime-local"
                                value={filters.endDate || ''}
                                onChange={(e) => {
                                    const newEndDate = e.target.value || undefined;
                                    this.logStore.setFilters({
                                        endDate: newEndDate,
                                    });
                                }}
                                InputLabelProps={{shrink: true}}
                                sx={{minWidth: 200}}
                            />
                        </Box>
                        <Box sx={{display: 'flex', gap: 2, alignItems: 'center'}}>
                            <TextField
                                fullWidth
                                label={this.props.t('logs.searchPlaceholder')}
                                value={filters.search || ''}
                                onChange={(e) => {
                                    const newSearch = e.target.value || undefined;
                                    this.logStore.setFilters({search: newSearch});
                                }}
                                placeholder={this.props.t('logs.searchPlaceholder')}
                                autoComplete="off"
                                name="log-search-filter"
                                InputProps={{
                                    endAdornment: (
                                        <InputAdornment position="end">
                                            <Tooltip title={this.props.t('logs.clearFilters')}>
                                                <IconButton
                                                    onClick={() => this.logStore.clearFilters()}
                                                    edge="end"
                                                    size="small"
                                                    sx={{
                                                        visibility:
                                                            filters.type ||
                                                            filters.level ||
                                                            filters.startDate ||
                                                            filters.endDate ||
                                                            filters.search
                                                                ? 'visible'
                                                                : 'hidden',
                                                        transition: 'all 0.2s ease',
                                                        '&:hover': {
                                                            backgroundColor: 'action.hover',
                                                            transform: 'scale(1.1)',
                                                        },
                                                    }}>
                                                    <ClearIcon fontSize="small" />
                                                </IconButton>
                                            </Tooltip>
                                        </InputAdornment>
                                    ),
                                }}
                            />
                        </Box>
                    </Stack>
                </CardContent>
            </Card>
        );
    };

    renderDetailDialog = () => {
        const {selectedLog} = this.state;
        if (!selectedLog) return null;

        return (
            <Dialog
                open={this.state.detailDialogOpen}
                onClose={() => this.setState({detailDialogOpen: false})}
                maxWidth="md"
                fullWidth>
                <DialogTitle>
                    <Box display="flex" justifyContent="space-between" alignItems="center">
                        <Typography variant="h6">
                            {this.props.t('logs.details')} - {selectedLog.type}
                        </Typography>
                        <Box display="flex" gap={1}>
                            {this.getTypeChip(selectedLog.type)}
                            {this.getStatusChip(selectedLog)}
                        </Box>
                    </Box>
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        {/* Basic Information */}
                        <Box>
                            <Typography variant="subtitle2" gutterBottom>
                                {this.props.t('logs.basicInfo')}
                            </Typography>
                            <Paper sx={{p: 2, bgcolor: 'background.default'}}>
                                <Box display="grid" gridTemplateColumns="repeat(2, 1fr)" gap={2}>
                                    <Box>
                                        <Typography variant="caption" color="textSecondary">
                                            {this.props.t('logs.logId')}
                                        </Typography>
                                        <Typography variant="body2">{selectedLog.id}</Typography>
                                    </Box>
                                    <Box>
                                        <Typography variant="caption" color="textSecondary">
                                            {this.props.t('logs.timestamp')}
                                        </Typography>
                                        <Typography variant="body2">
                                            {selectedLog.timestamp
                                                ? new Date(selectedLog.timestamp).toLocaleString()
                                                : 'Invalid Date'}
                                        </Typography>
                                    </Box>
                                    {selectedLog.username && (
                                        <Box>
                                            <Typography variant="caption" color="textSecondary">
                                                {this.props.t('logs.user')}
                                            </Typography>
                                            <Typography variant="body2">
                                                {selectedLog.username}
                                            </Typography>
                                        </Box>
                                    )}
                                    {selectedLog.action && (
                                        <Box>
                                            <Typography variant="caption" color="textSecondary">
                                                {this.props.t('logs.action')}
                                            </Typography>
                                            <Typography variant="body2">
                                                {this.getActionTranslation(
                                                    selectedLog.action,
                                                    selectedLog.type
                                                )}
                                            </Typography>
                                        </Box>
                                    )}
                                    {selectedLog.resource && (
                                        <Box>
                                            <Typography variant="caption" color="textSecondary">
                                                {this.props.t('logs.resource')}
                                            </Typography>
                                            <Typography variant="body2">
                                                {this.getResourceTranslation(selectedLog.resource)}
                                            </Typography>
                                        </Box>
                                    )}
                                    {selectedLog.ipAddress && (
                                        <Box>
                                            <Typography variant="caption" color="textSecondary">
                                                {this.props.t('logs.ipAddress')}
                                            </Typography>
                                            <Typography variant="body2">
                                                {selectedLog.ipAddress}
                                            </Typography>
                                        </Box>
                                    )}
                                </Box>
                            </Paper>
                        </Box>

                        {/* 消息/描述 */}
                        {(selectedLog.message || selectedLog.description) && (
                            <Box>
                                <Typography variant="subtitle2">
                                    {this.props.t('logs.message')}
                                </Typography>
                                <Paper
                                    sx={{
                                        p: 2,
                                        bgcolor: (theme) =>
                                            theme.palette.mode === 'dark' ? '#1a1a1a' : '#f8f9fa',
                                        border: 1,
                                        borderColor: 'divider',
                                    }}>
                                    <Typography variant="body2">
                                        {selectedLog.message || selectedLog.description}
                                    </Typography>
                                </Paper>
                            </Box>
                        )}

                        {/* Hook特定信息 */}
                        {selectedLog.type === 'hook' && (
                            <Box>
                                <Typography variant="subtitle2" gutterBottom>
                                    {this.props.t('logs.hookInfo')}
                                </Typography>
                                <Paper sx={{p: 2, bgcolor: 'background.default'}}>
                                    <Box
                                        display="grid"
                                        gridTemplateColumns="repeat(2, 1fr)"
                                        gap={2}>
                                        {selectedLog.hookName && (
                                            <Box>
                                                <Typography variant="caption" color="textSecondary">
                                                    {this.props.t('logs.hookName')}
                                                </Typography>
                                                <Typography variant="body2">
                                                    {selectedLog.hookName}
                                                </Typography>
                                            </Box>
                                        )}
                                        {selectedLog.method && (
                                            <Box>
                                                <Typography variant="caption" color="textSecondary">
                                                    {this.props.t('logs.requestMethod')}
                                                </Typography>
                                                <Typography variant="body2">
                                                    {selectedLog.method}
                                                </Typography>
                                            </Box>
                                        )}
                                        {selectedLog.remoteAddr && (
                                            <Box>
                                                <Typography variant="caption" color="textSecondary">
                                                    {this.props.t('logs.remoteAddress')}
                                                </Typography>
                                                <Typography variant="body2">
                                                    {selectedLog.remoteAddr}
                                                </Typography>
                                            </Box>
                                        )}
                                        {selectedLog.duration !== undefined && (
                                            <Box>
                                                <Typography variant="caption" color="textSecondary">
                                                    {this.props.t('logs.executionDuration')}
                                                </Typography>
                                                <Typography variant="body2">
                                                    {selectedLog.duration}ms
                                                </Typography>
                                            </Box>
                                        )}
                                    </Box>
                                </Paper>
                            </Box>
                        )}

                        {selectedLog.output && (
                            <Box>
                                <Typography variant="subtitle2">
                                    {this.props.t('logs.output')}
                                </Typography>
                                <Paper
                                    sx={{
                                        p: 2,
                                        bgcolor: (theme) =>
                                            theme.palette.mode === 'dark' ? '#1a1a1a' : '#f8f9fa',
                                        color: (theme) =>
                                            theme.palette.mode === 'dark' ? '#d0d7de' : '#24292f',
                                        border: 1,
                                        borderColor: (theme) =>
                                            theme.palette.mode === 'dark' ? '#30363d' : '#d0d7de',
                                        fontFamily:
                                            'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                                        fontSize: '0.85rem',
                                        maxHeight: 200,
                                        overflow: 'auto',
                                        '& pre': {
                                            margin: 0,
                                            whiteSpace: 'pre-wrap',
                                            wordBreak: 'break-word',
                                        },
                                        // 应用环境文件编辑器的滚动条样式
                                        '&::-webkit-scrollbar': {
                                            width: '8px',
                                            height: '8px',
                                        },
                                        '&::-webkit-scrollbar-track': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#2d2d2d'
                                                    : '#f1f3f4',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#424242'
                                                    : '#c1c8cd',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb:hover': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#484f58'
                                                    : '#a8b3ba',
                                        },
                                        scrollbarWidth: 'thin',
                                        scrollbarColor: (theme) =>
                                            theme.palette.mode === 'dark'
                                                ? '#30363d #2d2d2d'
                                                : '#c1c8cd #f1f3f4',
                                    }}>
                                    <pre>{selectedLog.output}</pre>
                                </Paper>
                            </Box>
                        )}

                        {selectedLog.error && (
                            <Box>
                                <Typography variant="subtitle2">
                                    {this.props.t('logs.error')}
                                </Typography>
                                <Paper
                                    sx={{
                                        p: 2,
                                        bgcolor: 'error.light',
                                        color: 'error.contrastText',
                                        border: 1,
                                        borderColor: 'error.main',
                                        fontFamily:
                                            'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                                        fontSize: '0.85rem',
                                        maxHeight: 200,
                                        overflow: 'auto',
                                        '& pre': {
                                            margin: 0,
                                            whiteSpace: 'pre-wrap',
                                            wordBreak: 'break-word',
                                        },
                                        // 应用环境文件编辑器的滚动条样式
                                        '&::-webkit-scrollbar': {
                                            width: '8px',
                                            height: '8px',
                                        },
                                        '&::-webkit-scrollbar-track': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#2d2d2d'
                                                    : '#f1f3f4',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#424242'
                                                    : '#c1c8cd',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb:hover': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#484f58'
                                                    : '#a8b3ba',
                                        },
                                        scrollbarWidth: 'thin',
                                        scrollbarColor: (theme) =>
                                            theme.palette.mode === 'dark'
                                                ? '#30363d #2d2d2d'
                                                : '#c1c8cd #f1f3f4',
                                    }}>
                                    <pre>{selectedLog.error}</pre>
                                </Paper>
                            </Box>
                        )}

                        {selectedLog.details && (
                            <Box>
                                <Typography variant="subtitle2">
                                    {this.props.t('logs.details')}
                                </Typography>
                                <Paper
                                    sx={{
                                        p: 2,
                                        bgcolor: 'background.paper',
                                        border: 1,
                                        borderColor: 'divider',
                                        fontFamily:
                                            'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                                        fontSize: '0.85rem',
                                        maxHeight: 300,
                                        overflow: 'auto',
                                        '& pre': {
                                            margin: 0,
                                            color: 'text.primary',
                                            whiteSpace: 'pre-wrap',
                                            wordBreak: 'break-word',
                                        },
                                        // 应用环境文件编辑器的滚动条样式
                                        '&::-webkit-scrollbar': {
                                            width: '8px',
                                            height: '8px',
                                        },
                                        '&::-webkit-scrollbar-track': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#2d2d2d'
                                                    : '#f1f3f4',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#424242'
                                                    : '#c1c8cd',
                                            borderRadius: '4px',
                                        },
                                        '&::-webkit-scrollbar-thumb:hover': {
                                            backgroundColor: (theme) =>
                                                theme.palette.mode === 'dark'
                                                    ? '#484f58'
                                                    : '#a8b3ba',
                                        },
                                        scrollbarWidth: 'thin',
                                        scrollbarColor: (theme) =>
                                            theme.palette.mode === 'dark'
                                                ? '#30363d #2d2d2d'
                                                : '#c1c8cd #f1f3f4',
                                    }}>
                                    <pre>{JSON.stringify(selectedLog.details, null, 2)}</pre>
                                </Paper>
                            </Box>
                        )}
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button
                        onClick={() => this.setState({detailDialogOpen: false})}
                        variant="contained"
                        color="secondary">
                        {this.props.t('common.close')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    };

    renderCleanupDialog = () => {
        return (
            <Dialog
                open={this.state.cleanupDialogOpen}
                onClose={() => this.setState({cleanupDialogOpen: false})}>
                <DialogTitle>{this.props.t('logs.clearLogs')}</DialogTitle>
                <DialogContent>
                    <Typography sx={{mb: 2}}>
                        {this.props.t('logs.confirmCleanupText', {days: this.state.cleanupDays})}
                    </Typography>
                    <TextField
                        fullWidth
                        type="number"
                        label={this.props.t('logs.cleanupOlderThan')}
                        value={this.state.cleanupDays}
                        onChange={(e) =>
                            this.setState({cleanupDays: parseInt(e.target.value) || 30})
                        }
                    />
                </DialogContent>
                <DialogActions>
                    <Button
                        onClick={() => this.setState({cleanupDialogOpen: false})}
                        variant="contained"
                        color="secondary">
                        {this.props.t('common.cancel')}
                    </Button>
                    <Button onClick={this.handleCleanupConfirm} color="error">
                        {this.props.t('logs.confirmCleanup')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    };

    render() {
        const {logs, loading, total, hasMore} = this.logStore;

        return (
            <main style={{margin: '0 auto', maxWidth: 1200}}>
                <Typography variant="h4" gutterBottom>
                    {this.props.t('logs.title')}
                </Typography>

                <Box
                    sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        mb: 2,
                    }}>
                    <Typography variant="h6">
                        {this.props.t('logs.totalItems', {count: total})}
                    </Typography>
                    <Box sx={{display: 'flex', gap: 1}}>
                        <Tooltip title={this.props.t('logs.refreshLogs')}>
                            <span>
                                <IconButton onClick={this.handleRefresh} disabled={loading}>
                                    <RefreshIcon />
                                </IconButton>
                            </span>
                        </Tooltip>
                        <Tooltip title={this.props.t('logs.exportLogs')}>
                            <IconButton onClick={this.handleExport}>
                                <DownloadIcon />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={this.props.t('logs.clearLogs')}>
                            <IconButton onClick={this.handleCleanup} color="error">
                                <DeleteIcon />
                            </IconButton>
                        </Tooltip>
                    </Box>
                </Box>

                {this.renderFilters()}

                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableCell>{this.props.t('logs.logId')}</TableCell>
                                <TableCell>{this.props.t('logs.timestamp')}</TableCell>
                                <TableCell>{this.props.t('logs.filterByType')}</TableCell>
                                <TableCell>{this.props.t('common.status')}</TableCell>
                                <TableCell>{this.props.t('logs.message')}</TableCell>
                                <TableCell>{this.props.t('logs.user')}</TableCell>
                                <TableCell>{this.props.t('logs.resource')}</TableCell>
                                <TableCell>{this.props.t('common.actions')}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {logs.map((log) => (
                                <TableRow key={`${log.type}-${log.id}-${log.timestamp}`} hover>
                                    <TableCell>{log.id}</TableCell>
                                    <TableCell>
                                        <Tooltip
                                            title={
                                                log.timestamp
                                                    ? new Date(log.timestamp).toLocaleString()
                                                    : 'Invalid Date'
                                            }>
                                            <span>
                                                {log.timestamp && !isNaN(Date.parse(log.timestamp))
                                                    ? new Date(log.timestamp).toLocaleTimeString()
                                                    : 'Invalid Date'}
                                            </span>
                                        </Tooltip>
                                    </TableCell>
                                    <TableCell>{this.getTypeChip(log.type)}</TableCell>
                                    <TableCell>{this.getStatusChip(log)}</TableCell>
                                    <TableCell style={{maxWidth: 300, wordWrap: 'break-word'}}>
                                        {log.message}
                                    </TableCell>
                                    <TableCell>{log.username || log.userId || '-'}</TableCell>
                                    <TableCell>{log.hookName || log.projectName || '-'}</TableCell>
                                    <TableCell>
                                        <IconButton
                                            size="small"
                                            onClick={() => this.handleViewDetail(log)}>
                                            <VisibilityIcon />
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))}
                            {loading && (
                                <TableRow>
                                    <TableCell colSpan={8} sx={{textAlign: 'center', p: 2}}>
                                        <CircularProgress size={24} sx={{mr: 1}} />
                                        {this.props.t('common.loading')}...
                                    </TableCell>
                                </TableRow>
                            )}
                            {!loading && logs.length === 0 && (
                                <TableRow>
                                    <TableCell
                                        colSpan={8}
                                        sx={{
                                            textAlign: 'center',
                                            p: 2,
                                            color: 'text.secondary',
                                        }}>
                                        {this.props.t('logs.noMoreLogs')}
                                    </TableCell>
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </Paper>

                {loading && logs.length > 0 && (
                    <Box sx={{textAlign: 'center', mt: 2}}>
                        <CircularProgress size={24} sx={{mr: 1}} />
                        {this.props.t('logs.loadMore')}...
                    </Box>
                )}

                {this.renderDetailDialog()}
                {this.renderCleanupDialog()}
            </main>
        );
    }
}

// 创建包装组件以使用翻译功能
const LogsWithTranslation: React.FC = () => {
    const {t} = useTranslation();

    // 传递翻译函数作为属性
    return <Logs t={t} />;
};

export default LogsWithTranslation;
