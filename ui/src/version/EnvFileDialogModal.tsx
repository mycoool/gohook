import React, {Component} from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Chip,
    Typography,
    Box,
    Alert,
    MenuItem,
    Select,
    FormControl,
    InputLabel,
    Theme,
    SelectChangeEvent,
} from '@mui/material';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import {Controlled as CodeMirror} from 'react-codemirror2';
import 'codemirror/lib/codemirror.css';
import 'codemirror/theme/material.css';
import 'codemirror/mode/properties/properties';
import 'codemirror/mode/shell/shell';
import 'codemirror/mode/toml/toml';
import './EnvFileDialog.css';
import useTranslation from '../i18n/useTranslation';

const ZERO_WIDTH_CHARS_REGEX = /(?:\u200B|\u200C|\u200D|\u2060|\uFEFF)/g;
type EnvConfigFormat = 'env' | 'ini' | 'toml';
type EnvConfigFormatMode = 'auto' | EnvConfigFormat;

function getZeroWidthInfo(content: string): {count: number; lines: number[]} {
    if (!content) {
        return {count: 0, lines: []};
    }

    const linesWithZeroWidth: number[] = [];
    let totalCount = 0;

    const lines = content.split('\n');
    for (let index = 0; index < lines.length; index++) {
        const line = lines[index];
        const matches = line.match(ZERO_WIDTH_CHARS_REGEX);
        if (matches && matches.length > 0) {
            totalCount += matches.length;
            linesWithZeroWidth.push(index + 1);
        }
    }

    return {count: totalCount, lines: linesWithZeroWidth};
}

function removeZeroWidthChars(content: string): string {
    if (!content) {
        return content;
    }
    return content.replace(ZERO_WIDTH_CHARS_REGEX, '');
}

function formatEnvFileContent(content: string): string {
    const normalized = content.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
    const lines = normalized.split('\n').map((line) => line.replace(/[ \t]+$/g, ''));

    const formatted = lines
        .map((line) => {
            const trimmed = line.trim();
            if (trimmed === '' || trimmed.startsWith('#')) {
                return line;
            }

            const envMatch = line.match(/^(\s*)([A-Za-z_][A-Za-z0-9_.-]*)(\s*=\s*)(.*)$/);
            if (!envMatch) {
                return line;
            }

            const [, indent, key, , value] = envMatch;
            return `${indent}${key}=${value}`;
        })
        .join('\n');

    return formatted.endsWith('\n') ? formatted : `${formatted}\n`;
}

interface IProps {
    open: boolean;
    projectName: string;
    onClose: () => void;
    onGetEnvFile: (name: string) => Promise<{content: string; exists: boolean; path: string}>;
    onSaveEnvFile: (name: string, content: string) => Promise<void>;
    onDeleteEnvFile: (name: string) => Promise<void>;
    theme?: Theme;
}

type IInjectedProps = IProps & Stores<'snackManager'>;

interface IPropsWithTranslation extends IInjectedProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

interface IState {
    envFileContent: string;
    originalEnvFileContent: string;
    hasEnvFile: boolean;
    errors: string[];
    formatMode: EnvConfigFormatMode;
    selectedTemplate: string;
    isEditMode: boolean;
}

// 预定义模板
const templates = {
    empty: '',
    basic_env: `# 基础环境配置
APP_NAME=MyApplication
APP_VERSION=1.0.0
APP_DEBUG=true
APP_ENV=development

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=username
DB_PASSWORD=password

# 外部服务
API_KEY=your-api-key-here
SECRET_KEY=your-secret-key-here`,

    toml_format: `# TOML格式配置
# 应用程序设置
[app]
name = "MyApplication"
version = "1.0.0"
debug = true
environment = "development"

# 数据库配置
[database]
host = "localhost"
port = 5432
name = "myapp"
username = "user"
password = "password"
max_connections = 100

# API配置
[api]
endpoints = ["https://api1.example.com", "https://api2.example.com"]
timeout = 30
retry_count = 3

# 功能开关
[features]
enable_cache = true
enable_logging = true
enable_metrics = false

# 嵌套配置
app.cache.ttl = 3600
app.cache.size = 1024`,

    web_app: `# Web应用配置
PORT=3000
HOST=0.0.0.0
NODE_ENV=development

# 数据库
DATABASE_URL=postgresql://localhost:5432/webapp
REDIS_URL=redis://localhost:6379

# 认证
JWT_SECRET=your-jwt-secret-here
SESSION_SECRET=your-session-secret-here

# 第三方服务
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# 文件存储
UPLOAD_PATH=./uploads
MAX_FILE_SIZE=10485760`,

    microservice: `# 微服务配置
[service]
name = "user-service"
version = "1.0.0"
port = 8080

[database]
driver = "postgres"
host = "localhost"
port = 5432
name = "userdb"
user = "postgres"
password = "postgres"

[redis]
host = "localhost"
port = 6379
password = ""

[logging]
level = "info"
format = "json"

[monitoring]
enable_metrics = true
metrics_port = 9090`,
};

// 检测TOML格式
function detectTomlFormat(content: string): boolean {
    const lines = content.split('\n');

    for (const line of lines) {
        const trimmedLine = line.trim();

        if (trimmedLine === '' || trimmedLine.startsWith('#')) {
            continue;
        }

        if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
            return true;
        }

        if (trimmedLine.includes('"""')) {
            return true;
        }

        if (/^\w+\s*=\s*\[.*\]/.test(trimmedLine)) {
            return true;
        }

        if (/^\w+\.\w+\s*=/.test(trimmedLine)) {
            return true;
        }
    }

    return false;
}

function detectIniFormat(content: string): boolean {
    const lines = content.split('\n');

    for (const line of lines) {
        const trimmedLine = line.trim();

        if (trimmedLine === '') {
            continue;
        }

        // INI commonly uses ';' for comments
        if (trimmedLine.startsWith(';')) {
            return true;
        }

        // INI allows section headers as well, but TOML detection should run first.
        if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
            continue;
        }

        // key: value or key = value, with broader key charset (spaces are common)
        if (/^[^=:#[];][^=:#]*\s*[:=]\s*.*$/.test(trimmedLine)) {
            const key = trimmedLine.split(/[:=]/, 1)[0];
            // if key contains spaces, treat as ini (env keys do not)
            if (/\s/.test(key)) {
                return true;
            }
        }
    }

    return false;
}

function detectEnvConfigFormat(content: string): EnvConfigFormat {
    if (detectTomlFormat(content)) {
        return 'toml';
    }
    if (detectIniFormat(content)) {
        return 'ini';
    }
    return 'env';
}

function getFormatIndicator(format: EnvConfigFormat): string {
    if (format === 'toml') return 'TOML';
    if (format === 'ini') return 'INI';
    return 'ENV';
}

function getFormatColor(format: EnvConfigFormat): string {
    if (format === 'toml') return '#4CAF50';
    if (format === 'ini') return '#9C27B0';
    return '#2196F3';
}

function getCodeMirrorMode(format: EnvConfigFormat): string {
    if (format === 'toml') return 'toml';
    // Use properties mode for both INI and ENV-like key/value configs.
    return 'properties';
}

@observer
class EnvFileDialogModal extends Component<IPropsWithTranslation, IState> {
    state: IState = {
        envFileContent: '',
        originalEnvFileContent: '',
        hasEnvFile: false,
        errors: [],
        formatMode: 'auto',
        selectedTemplate: 'empty',
        isEditMode: false,
    };

    componentDidUpdate(prevProps: IPropsWithTranslation) {
        if (this.props.open && !prevProps.open && this.props.projectName) {
            this.loadEnvFile();
        }
    }

    loadEnvFile = async () => {
        try {
            const result = await this.props.onGetEnvFile(this.props.projectName);
            if (result.exists) {
                this.setState({
                    envFileContent: result.content,
                    originalEnvFileContent: result.content,
                    hasEnvFile: true,
                    formatMode: 'auto',
                    selectedTemplate: 'empty',
                    isEditMode: true, // 存在文件时进入编辑模式
                });
            } else {
                this.setState({
                    envFileContent: '',
                    originalEnvFileContent: '',
                    hasEnvFile: false,
                    formatMode: 'auto',
                    selectedTemplate: 'empty',
                    isEditMode: false, // 不存在文件时进入创建模式
                });
            }
        } catch (error) {
            this.props.snackManager.snack(this.props.t('version.env.loadError'));
        }
    };

    handleClose = () => {
        this.setState({
            envFileContent: this.state.originalEnvFileContent,
            errors: [],
            selectedTemplate: 'empty',
            // 重置状态，但保持isEditMode根据文件存在性决定
        });
        this.props.onClose();
    };

    handleContentChange = (code: string) => {
        this.setState({
            envFileContent: code,
            selectedTemplate: 'empty',
        });
    };

    handleTemplateChange = (event: SelectChangeEvent<string>) => {
        const templateKey = event.target.value as string;
        this.setState({
            selectedTemplate: templateKey,
            envFileContent: templates[templateKey as keyof typeof templates],
            formatMode: 'auto',
        });
    };

    handleSave = async () => {
        try {
            const zeroWidthInfo = getZeroWidthInfo(this.state.envFileContent);
            const shouldRemove =
                zeroWidthInfo.count > 0
                    ? window.confirm(
                          this.props.t('version.env.zeroWidthConfirmRemoveBeforeSave', {
                              count: zeroWidthInfo.count,
                          })
                      )
                    : false;

            const contentToSave = shouldRemove
                ? removeZeroWidthChars(this.state.envFileContent)
                : this.state.envFileContent;

            await this.props.onSaveEnvFile(this.props.projectName, contentToSave);
            this.props.snackManager.snack(this.props.t('version.env.saveSuccess'));
            this.setState({
                envFileContent: contentToSave,
                originalEnvFileContent: contentToSave,
                hasEnvFile: true,
                errors: [],
                formatMode: this.state.formatMode,
            });
            this.props.onClose();
        } catch (error: any) {
            if (error.response?.data?.errors) {
                this.setState({errors: error.response.data.errors});
            } else {
                this.props.snackManager.snack(this.props.t('version.env.saveError'));
            }
        }
    };

    handleFormat = () => {
        const format =
            this.state.formatMode === 'auto'
                ? detectEnvConfigFormat(this.state.envFileContent)
                : this.state.formatMode;

        const formatted =
            format === 'env'
                ? formatEnvFileContent(this.state.envFileContent)
                : // For non-env formats, keep content intact except trimming line endings and trailing whitespace.
                  this.state.envFileContent
                      .replace(/\r\n/g, '\n')
                      .replace(/\r/g, '\n')
                      .split('\n')
                      .map((line) => line.replace(/[ \t]+$/g, ''))
                      .join('\n')
                      .replace(/(?:\n)?$/, '\n');

        this.setState({
            envFileContent: formatted,
            selectedTemplate: 'empty',
        });
        this.props.snackManager.snack(this.props.t('version.env.formatSuccess'));
    };

    handleDelete = async () => {
        if (window.confirm(this.props.t('version.env.deleteConfirm'))) {
            try {
                await this.props.onDeleteEnvFile(this.props.projectName);
                this.props.snackManager.snack(this.props.t('version.env.deleteSuccess'));
                this.setState({
                    envFileContent: '',
                    originalEnvFileContent: '',
                    hasEnvFile: false,
                    errors: [],
                    formatMode: 'auto',
                    selectedTemplate: 'empty',
                    isEditMode: false,
                });
                this.props.onClose();
            } catch (error) {
                this.props.snackManager.snack(this.props.t('version.env.deleteError'));
            }
        }
    };

    render() {
        const {open, projectName, theme, t} = this.props;
        const {envFileContent, hasEnvFile, errors, formatMode, selectedTemplate, isEditMode} =
            this.state;

        const zeroWidthInfo = getZeroWidthInfo(envFileContent);

        const effectiveFormat =
            formatMode === 'auto' ? detectEnvConfigFormat(envFileContent) : formatMode;
        const formatIndicator = getFormatIndicator(effectiveFormat);
        const formatColor = getFormatColor(effectiveFormat);

        // 检测是否为深色主题 - 从localStorage获取
        const isDarkTheme = localStorage.getItem('gohook-theme') === 'dark';

        const editorContainerStyle = {
            border: `1px solid ${isDarkTheme ? '#444' : '#e0e0e0'}`,
            borderRadius: 4,
            overflow: 'hidden',
        };

        return (
            <Dialog
                open={open}
                onClose={this.handleClose}
                maxWidth="md"
                fullWidth
                scroll="paper"
                PaperProps={{
                    style: {
                        maxHeight: '85vh', // 限制对话框最大高度
                        height: 'auto',
                        // 移除深色灰色背景，使用默认主题背景以提高文字可读性
                        color: isDarkTheme ? '#ffffff' : '#000000',
                    },
                }}>
                <DialogTitle>
                    <Box display="flex" alignItems="center" justifyContent="space-between">
                        <span>
                            {isEditMode ? t('version.env.edit') : t('version.env.create')} -{' '}
                            {projectName}
                        </span>
                        {(isEditMode || envFileContent) && (
                            <Chip
                                label={t('version.env.formatChip', {format: formatIndicator})}
                                style={{backgroundColor: formatColor, color: 'white'}}
                                size="small"
                            />
                        )}
                    </Box>
                </DialogTitle>
                <DialogContent style={{paddingBottom: 0, overflow: 'visible'}}>
                    {/* 模板选择器 - 仅在创建模式显示 */}
                    {!isEditMode && (
                        <Box mb={2}>
                            <FormControl fullWidth variant="outlined" size="small">
                                <InputLabel>{t('version.env.templateLabel')}</InputLabel>
                                <Select
                                    value={selectedTemplate}
                                    onChange={this.handleTemplateChange}
                                    label={t('version.env.templateLabel')}>
                                    <MenuItem value="empty">
                                        {t('version.env.templateEmpty')}
                                    </MenuItem>
                                    <MenuItem value="basic_env">
                                        {t('version.env.templateBasic')}
                                    </MenuItem>
                                    <MenuItem value="toml_format">
                                        {t('version.env.templateToml')}
                                    </MenuItem>
                                    <MenuItem value="web_app">
                                        {t('version.env.templateWebApp')}
                                    </MenuItem>
                                    <MenuItem value="microservice">
                                        {t('version.env.templateMicroservice')}
                                    </MenuItem>
                                </Select>
                            </FormControl>
                            <Typography
                                variant="caption"
                                color="textSecondary"
                                style={{display: 'block', marginTop: '8px'}}>
                                {t('version.env.templatePlaceholder')}
                            </Typography>
                        </Box>
                    )}

                    <Box mb={2} display="flex" gap={2} alignItems="center">
                        <FormControl variant="outlined" size="small" style={{minWidth: 160}}>
                            <InputLabel>{t('version.env.contentFormat')}</InputLabel>
                            <Select
                                value={formatMode}
                                onChange={(event) => {
                                    const value = event.target.value as EnvConfigFormatMode;
                                    this.setState({formatMode: value});
                                }}
                                label={t('version.env.contentFormat')}>
                                <MenuItem value="auto">{t('version.env.formatAuto')}</MenuItem>
                                <MenuItem value="env">{t('version.env.formatEnv')}</MenuItem>
                                <MenuItem value="ini">{t('version.env.formatIni')}</MenuItem>
                                <MenuItem value="toml">{t('version.env.formatToml')}</MenuItem>
                            </Select>
                        </FormControl>
                        <Typography variant="caption" color="textSecondary">
                            {t('version.env.formatHint')}
                        </Typography>
                    </Box>

                    {/* 语法高亮编辑器 */}
                    <Box
                        mb={2}
                        style={editorContainerStyle}
                        className={`env-codemirror ${
                            isDarkTheme ? 'env-codemirror--dark' : 'env-codemirror--light'
                        }`}>
                        <CodeMirror
                            value={envFileContent}
                            options={{
                                mode: getCodeMirrorMode(effectiveFormat),
                                theme: isDarkTheme ? 'material' : 'default',
                                lineNumbers: true,
                                lineWrapping: true,
                                tabSize: 2,
                                indentWithTabs: false,
                            }}
                            onBeforeChange={(_, __, value) => {
                                this.handleContentChange(value);
                            }}
                        />
                    </Box>

                    {zeroWidthInfo.count > 0 && (
                        <Box mb={2}>
                            <Alert
                                severity="warning"
                                action={
                                    <Button
                                        color="inherit"
                                        size="small"
                                        onClick={() => {
                                            const sanitized = removeZeroWidthChars(envFileContent);
                                            this.setState({
                                                envFileContent: sanitized,
                                            });
                                            this.props.snackManager.snack(
                                                t('version.env.zeroWidthRemoved', {
                                                    count: zeroWidthInfo.count,
                                                })
                                            );
                                        }}>
                                        {t('version.env.zeroWidthRemove')}
                                    </Button>
                                }>
                                <Typography variant="body2">
                                    {t('version.env.zeroWidthDetected', {
                                        count: zeroWidthInfo.count,
                                    })}
                                </Typography>
                                {zeroWidthInfo.lines.length > 0 && (
                                    <Typography
                                        variant="caption"
                                        color="textSecondary"
                                        style={{display: 'block', marginTop: '4px'}}>
                                        {t('version.env.zeroWidthLines', {
                                            lines:
                                                zeroWidthInfo.lines.length > 12
                                                    ? `${zeroWidthInfo.lines
                                                          .slice(0, 12)
                                                          .join(', ')}...`
                                                    : zeroWidthInfo.lines.join(', '),
                                        })}
                                    </Typography>
                                )}
                            </Alert>
                        </Box>
                    )}

                    {/* 错误显示 */}
                    {errors.length > 0 && (
                        <Box mt={2}>
                            <Typography variant="subtitle2" color="error">
                                {t('version.env.validationTitle')}
                            </Typography>
                            {errors.map((error, index) => (
                                <Typography key={index} variant="body2" color="error">
                                    • {error}
                                </Typography>
                            ))}
                        </Box>
                    )}

                    {/* 提示信息 - 固定在底部 */}
                    <Box mt={2} mb={1}>
                        {(isEditMode || envFileContent) && (
                            <Typography variant="body2" color="textSecondary">
                                <strong>{t('version.env.detectedFormat')}</strong>
                                {t('version.env.detectedFormatValue', {format: formatIndicator})}
                            </Typography>
                        )}
                        <Typography variant="body2" color="textSecondary">
                            {t('version.env.saveInfoPrefix')} <code>.env</code>{' '}
                            {t('version.env.saveInfoSuffix')}
                        </Typography>
                        {!isEditMode && !envFileContent && (
                            <Typography variant="body2" color="primary" style={{marginTop: '8px'}}>
                                💡 {t('version.env.templateHint')}
                            </Typography>
                        )}
                    </Box>
                </DialogContent>
                <DialogActions style={{paddingLeft: 24, paddingRight: 24}}>
                    {hasEnvFile && (
                        <Button onClick={this.handleDelete} variant="contained" color="error">
                            {t('version.env.deleteFile')}
                        </Button>
                    )}
                    <Box flexGrow={1} />
                    <Button
                        onClick={this.handleFormat}
                        variant="contained"
                        disabled={!envFileContent}>
                        {t('version.env.format')}
                    </Button>
                    <Button onClick={this.handleClose} variant="contained" color="secondary">
                        {t('common.cancel')}
                    </Button>
                    <Button onClick={this.handleSave} color="primary" variant="contained">
                        {t('version.env.save')}
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
}

const EnvFileDialogModalWithTranslation: React.FC<IInjectedProps> = (props) => {
    const {t} = useTranslation();
    return <EnvFileDialogModal {...props} t={t} />;
};

const EnvFileDialogModalInjected = inject('snackManager')(EnvFileDialogModalWithTranslation);

export default EnvFileDialogModalInjected as React.ComponentType<IProps>;
