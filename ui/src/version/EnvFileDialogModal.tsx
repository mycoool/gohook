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
    MenuItem,
    Select,
    FormControl,
    InputLabel,
    withTheme,
    Theme,
} from '@material-ui/core';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import Editor from 'react-simple-code-editor';
import Prism from 'prismjs';
import 'prismjs/components/prism-toml';
import 'prismjs/components/prism-bash';
import 'prismjs/themes/prism.css';
import './EnvFileDialog.css';

// ENV高亮：自定义实现，精确控制token类型
const highlightEnv = (code: string, isDark: boolean = false) => {
    try {
        // 使用自定义语法解析ENV格式
        return code
            .split('\n')
            .map(line => {
                const trimmed = line.trim();
                
                // 注释行
                if (trimmed.startsWith('#')) {
                    return `<span class="token comment">${escapeHtml(line)}</span>`;
                }
                
                // 空行
                if (trimmed === '') {
                    return line;
                }
                
                // ENV键值对 KEY=VALUE
                const envMatch = line.match(/^(\s*)([A-Za-z_][A-Za-z0-9_]*)(=)(.*)$/);
                if (envMatch) {
                    const [, indent, key, equals, value] = envMatch;
                    const highlightedValue = highlightEnvValue(value);
                    return `${escapeHtml(indent)}<span class="token variable">${escapeHtml(key)}</span><span class="token operator">${escapeHtml(equals)}</span>${highlightedValue}`;
                }
                
                return escapeHtml(line);
            })
            .join('\n');
    } catch (e) {
        return code;
    }
};

// ENV值高亮处理
const highlightEnvValue = (value: string) => {
    const trimmedValue = value.trim();
    
    // 带引号的字符串
    if ((trimmedValue.startsWith('"') && trimmedValue.endsWith('"')) ||
        (trimmedValue.startsWith("'") && trimmedValue.endsWith("'"))) {
        return `<span class="token string">${escapeHtml(value)}</span>`;
    }
    
    // 布尔值
    if (trimmedValue === 'true' || trimmedValue === 'false') {
        return `<span class="token boolean">${escapeHtml(value)}</span>`;
    }
    
    // 数字
    if (/^\d+(\.\d+)?$/.test(trimmedValue)) {
        return `<span class="token number">${escapeHtml(value)}</span>`;
    }
    
    // 无引号值 - 使用builtin类
    return `<span class="token builtin">${escapeHtml(value)}</span>`;
};

// HTML转义函数
const escapeHtml = (text: string): string => {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
};

// TOML高亮，支持自定义样式
const highlightToml = (code: string, isDark: boolean = false) => {
    try {
        return Prism.highlight(code, Prism.languages.toml, 'toml');
    } catch (e) {
        return code;
    }
};

interface IProps {
    open: boolean;
    projectName: string;
    onClose: () => void;
    onGetEnvFile: (name: string) => Promise<{ content: string; exists: boolean; path: string }>;
    onSaveEnvFile: (name: string, content: string) => Promise<void>;
    onDeleteEnvFile: (name: string) => Promise<void>;
    theme?: Theme;
}

interface IState {
    envFileContent: string;
    originalEnvFileContent: string;
    hasEnvFile: boolean;
    errors: string[];
    isTomlContent: boolean;
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
metrics_port = 9090`
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

@observer
class EnvFileDialogModal extends Component<IProps & Stores<'snackManager'>, IState> {
    state: IState = {
        envFileContent: '',
        originalEnvFileContent: '',
        hasEnvFile: false,
        errors: [],
        isTomlContent: false,
        selectedTemplate: 'empty',
        isEditMode: false,
    };

    componentDidUpdate(prevProps: IProps) {
        if (this.props.open && !prevProps.open && this.props.projectName) {
            this.loadEnvFile();
        }
    }

    loadEnvFile = async () => {
        try {
            const result = await this.props.onGetEnvFile(this.props.projectName);
            if (result.exists) {
                const isToml = detectTomlFormat(result.content);
                this.setState({
                    envFileContent: result.content,
                    originalEnvFileContent: result.content,
                    hasEnvFile: true,
                    isTomlContent: isToml,
                    selectedTemplate: 'empty',
                    isEditMode: true, // 存在文件时进入编辑模式
                });
            } else {
                this.setState({
                    envFileContent: '',
                    originalEnvFileContent: '',
                    hasEnvFile: false,
                    isTomlContent: false,
                    selectedTemplate: 'empty',
                    isEditMode: false, // 不存在文件时进入创建模式
                });
            }
        } catch (error) {
            this.props.snackManager.snack('加载环境文件失败');
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
        const isToml = detectTomlFormat(code);
        this.setState({
            envFileContent: code,
            isTomlContent: isToml,
            selectedTemplate: 'empty',
        });
    };

    handleTemplateChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        const templateKey = event.target.value as string;
        this.setState({
            selectedTemplate: templateKey,
            envFileContent: templates[templateKey as keyof typeof templates],
            isTomlContent: detectTomlFormat(templates[templateKey as keyof typeof templates]),
        });
    };

    handleSave = async () => {
        try {
            await this.props.onSaveEnvFile(this.props.projectName, this.state.envFileContent);
            this.props.snackManager.snack('环境文件保存成功');
            this.setState({
                originalEnvFileContent: this.state.envFileContent,
                hasEnvFile: true,
                errors: [],
            });
            this.props.onClose();
        } catch (error: any) {
            if (error.response?.data?.errors) {
                this.setState({errors: error.response.data.errors});
            } else {
                this.props.snackManager.snack('保存环境文件失败');
            }
        }
    };

    handleDelete = async () => {
        if (window.confirm('确定要删除环境文件吗？')) {
            try {
                await this.props.onDeleteEnvFile(this.props.projectName);
                this.props.snackManager.snack('环境文件删除成功');
                this.setState({
                    envFileContent: '',
                    originalEnvFileContent: '',
                    hasEnvFile: false,
                    errors: [],
                    isTomlContent: false,
                    selectedTemplate: 'empty',
                    isEditMode: false,
                });
                this.props.onClose();
            } catch (error) {
                this.props.snackManager.snack('删除环境文件失败');
            }
        }
    };

    render() {
        const { open, projectName, theme } = this.props;
        const { envFileContent, hasEnvFile, errors, isTomlContent, selectedTemplate, isEditMode } = this.state;

        const formatIndicator = isTomlContent ? 'TOML' : 'ENV';
        const formatColor = isTomlContent ? '#4CAF50' : '#2196F3';
        
        // 检测是否为深色主题
        const isDarkTheme = theme?.palette?.type === 'dark';
        
        // 根据主题选择编辑器样式
        const editorStyles = {
            fontFamily: 'Consolas, Monaco, "Courier New", monospace',
            fontSize: 13,
            minHeight: 400, // 使用minHeight而不是固定height
            maxHeight: 400, // 限制最大高度
            outline: 0,
            background: 'transparent',
            whiteSpace: 'pre' as const,
            color: isDarkTheme ? '#ffffff' : '#000000',
        };
        
        const editorContainerStyle = {
            border: `1px solid ${isDarkTheme ? '#444' : '#e0e0e0'}`,
            borderRadius: 4,
            background: isDarkTheme ? '#1e1e1e' : (isEditMode ? '#f8f8f8' : '#fafafa'),
            maxHeight: 400, // 限制容器最大高度
            overflow: 'auto', // 容器处理滚动
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
                    }
                }}>
                <DialogTitle>
                    <Box display="flex" alignItems="center" justifyContent="space-between">
                        <span>{isEditMode ? '编辑环境文件' : '创建环境文件'} - {projectName}</span>
                        {(isEditMode || envFileContent) && (
                            <Chip
                                label={`格式: ${formatIndicator}`}
                                style={{backgroundColor: formatColor, color: 'white'}}
                                size="small"
                            />
                        )}
                    </Box>
                </DialogTitle>
                <DialogContent style={{ paddingBottom: 0, overflow: 'visible' }}>
                    {/* 模板选择器 - 仅在创建模式显示 */}
                    {!isEditMode && (
                        <Box mb={2}>
                            <FormControl fullWidth variant="outlined" size="small">
                                <InputLabel>选择模板</InputLabel>
                                <Select
                                    value={selectedTemplate}
                                    onChange={this.handleTemplateChange}
                                    label="选择模板">
                                    <MenuItem value="empty">空白</MenuItem>
                                    <MenuItem value="basic_env">基础ENV格式</MenuItem>
                                    <MenuItem value="toml_format">TOML格式</MenuItem>
                                    <MenuItem value="web_app">Web应用配置</MenuItem>
                                    <MenuItem value="microservice">微服务配置</MenuItem>
                                </Select>
                            </FormControl>
                            <Typography variant="caption" color="textSecondary" style={{display: 'block', marginTop: '8px'}}>
                                选择模板将自动填充内容到编辑器中
                            </Typography>
                        </Box>
                    )}

                    {/* 语法高亮编辑器 */}
                    <Box mb={2} style={editorContainerStyle} className={isDarkTheme ? 'prism-dark' : 'prism-light'}>
                        <Editor
                            value={envFileContent}
                            onValueChange={this.handleContentChange}
                            highlight={(code) => isTomlContent ? highlightToml(code, isDarkTheme) : highlightEnv(code, isDarkTheme)}
                            padding={16}
                            style={editorStyles}
                            textareaId="envfile-editor"
                            placeholder={!isEditMode ? 
                                (isTomlContent ? 
                                    "# TOML格式内容\n[section]\nkey = \"value\"\n\n# 选择上方模板快速开始" :
                                    "# 标准ENV格式\nKEY=value\nANOTHER_KEY=another_value\n\n# 选择上方模板快速开始"
                                ) : 
                                (isTomlContent ?
                                    "# TOML格式配置文件" :
                                    "# 环境变量配置文件"
                                )
                            }
                        />
                    </Box>

                    {/* 错误显示 */}
                    {errors.length > 0 && (
                        <Box mt={2}>
                            <Typography variant="subtitle2" color="error">
                                验证错误：
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
                                <strong>检测到格式：</strong>{formatIndicator} 格式内容
                            </Typography>
                        )}
                        <Typography variant="body2" color="textSecondary">
                            文件将始终保存为 <code>.env</code>，但支持两种内容格式
                        </Typography>
                        {!isEditMode && !envFileContent && (
                            <Typography variant="body2" color="primary" style={{marginTop: '8px'}}>
                                💡 提示：选择上方模板可快速开始配置
                            </Typography>
                        )}
                    </Box>
                </DialogContent>
                <DialogActions>
                    {hasEnvFile && (
                        <Button onClick={this.handleDelete} color="secondary">
                            删除文件
                        </Button>
                    )}
                    <Box flexGrow={1} />
                    <Button onClick={this.handleClose}>
                        取消
                    </Button>
                    <Button
                        onClick={this.handleSave}
                        color="primary"
                        variant="contained">
                        保存
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
}

export default inject('snackManager')(withTheme(EnvFileDialogModal)); 