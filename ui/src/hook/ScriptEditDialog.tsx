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
    SelectChangeEvent,
} from '@mui/material';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import Editor from 'react-simple-code-editor';
import Prism from 'prismjs';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-javascript';
import 'prismjs/components/prism-json';
import 'prismjs/components/prism-yaml';
import 'prismjs/themes/prism.css';
import '../version/EnvFileDialog.css';

// 脚本类型定义
type ScriptType = 'bash' | 'javascript' | 'json' | 'yaml';

// 检测脚本类型
const detectScriptType = (content: string): ScriptType => {
    const trimmedContent = content.trim();
    
    // 检测 JSON
    if (trimmedContent.startsWith('{') || trimmedContent.startsWith('[')) {
        try {
            JSON.parse(trimmedContent);
            return 'json';
        } catch (e) {
            // 不是有效的 JSON
        }
    }
    
    // 检测 YAML (简单检测)
    if (trimmedContent.includes(': ') || trimmedContent.includes('- ')) {
        return 'yaml';
    }
    
    // 检测 JavaScript
    if (trimmedContent.includes('function') || 
        trimmedContent.includes('=>') || 
        trimmedContent.includes('const ') || 
        trimmedContent.includes('let ') ||
        trimmedContent.includes('var ')) {
        return 'javascript';
    }
    
    // 默认为 bash
    return 'bash';
};

// HTML 转义函数
const escapeHtml = (text: string) => {
    const map: {[key: string]: string} = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, (m) => map[m]);
};

// 脚本高亮函数
const highlightScript = (code: string, scriptType: ScriptType, isDark: boolean = false) => {
    try {
        switch (scriptType) {
            case 'bash':
                return Prism.highlight(code, Prism.languages.bash, 'bash');
            case 'javascript':
                return Prism.highlight(code, Prism.languages.javascript, 'javascript');
            case 'json':
                return Prism.highlight(code, Prism.languages.json, 'json');
            case 'yaml':
                return Prism.highlight(code, Prism.languages.yaml, 'yaml');
            default:
                return escapeHtml(code);
        }
    } catch (e) {
        return escapeHtml(code);
    }
};

// 预定义模板
const templates = {
    empty: '',
    bash_simple: `#!/bin/bash
# 简单的 Bash 脚本示例

echo "Hook 被触发: $HOOK_ID"
echo "请求方法: $HOOK_METHOD"
echo "远程地址: $HOOK_REMOTE_ADDR"

# 执行你的逻辑
echo "执行完成"`,

    bash_git_deploy: `#!/bin/bash
# Git 部署脚本示例

set -e  # 遇到错误立即退出

echo "开始部署..."

# 进入项目目录
cd /path/to/your/project

# 拉取最新代码
git pull origin main

# 安装依赖 (根据项目类型选择)
# npm install
# yarn install
# composer install
# pip install -r requirements.txt

# 构建项目 (如果需要)
# npm run build
# yarn build

# 重启服务 (根据实际情况选择)
# systemctl restart your-service
# pm2 restart app
# docker-compose restart

echo "部署完成"`,

    javascript_simple: `// JavaScript 脚本示例
const hookId = process.env.HOOK_ID;
const method = process.env.HOOK_METHOD;
const remoteAddr = process.env.HOOK_REMOTE_ADDR;

console.log(\`Hook 被触发: \${hookId}\`);
console.log(\`请求方法: \${method}\`);
console.log(\`远程地址: \${remoteAddr}\`);

// 执行你的逻辑
console.log("执行完成");`,

    javascript_webhook_handler: `// Webhook 处理脚本示例
const fs = require('fs');
const path = require('path');

// 从环境变量获取 webhook 数据
const hookId = process.env.HOOK_ID;
const payload = process.env.HOOK_PAYLOAD;

try {
    // 解析 payload (如果是 JSON)
    const data = payload ? JSON.parse(payload) : {};
    
    console.log('收到 Webhook:', {
        hookId,
        event: data.event_type || 'unknown',
        repository: data.repository?.name || 'unknown'
    });
    
    // 记录到日志文件
    const logEntry = {
        timestamp: new Date().toISOString(),
        hookId,
        data
    };
    
    fs.appendFileSync(
        path.join(__dirname, 'webhook.log'),
        JSON.stringify(logEntry) + '\\n'
    );
    
    console.log('处理完成');
} catch (error) {
    console.error('处理失败:', error.message);
    process.exit(1);
}`,

    json_config: `{
    "name": "webhook-config",
    "version": "1.0.0",
    "description": "Webhook 配置示例",
    "settings": {
        "timeout": 30,
        "retries": 3,
        "log_level": "info"
    },
    "notifications": {
        "email": {
            "enabled": true,
            "recipients": ["admin@example.com"]
        },
        "slack": {
            "enabled": false,
            "webhook_url": ""
        }
    },
    "actions": [
        {
            "name": "deploy",
            "command": "./deploy.sh",
            "working_dir": "/app"
        },
        {
            "name": "test",
            "command": "npm test",
            "working_dir": "/app"
        }
    ]
}`,

    yaml_config: `# Webhook YAML 配置示例
name: webhook-config
version: 1.0.0
description: Webhook 配置示例

settings:
  timeout: 30
  retries: 3
  log_level: info

notifications:
  email:
    enabled: true
    recipients:
      - admin@example.com
  slack:
    enabled: false
    webhook_url: ""

actions:
  - name: deploy
    command: ./deploy.sh
    working_dir: /app
  - name: test
    command: npm test
    working_dir: /app`
};

interface IProps {
    open: boolean;
    hookId: string;
    onClose: () => void;
    onGetScript: (hookId: string) => Promise<{content: string; exists: boolean; path: string}>;
    onSaveScript: (hookId: string, content: string) => Promise<void>;
    onDeleteScript: (hookId: string) => Promise<void>;
}

interface IState {
    scriptContent: string;
    originalScriptContent: string;
    hasScript: boolean;
    scriptType: ScriptType;
    selectedTemplate: string;
    isEditMode: boolean;
    errors: string[];
    scriptPath?: string;
}

@observer
class ScriptEditDialog extends Component<IProps & Stores<'snackManager'>, IState> {
    state: IState = {
        scriptContent: '',
        originalScriptContent: '',
        hasScript: false,
        scriptType: 'bash',
        selectedTemplate: 'empty',
        isEditMode: false,
        errors: [],
        scriptPath: undefined,
    };

    componentDidMount() {
        if (this.props.open) {
            this.loadScript();
        }
    }

    componentDidUpdate(prevProps: IProps) {
        if (this.props.open && !prevProps.open) {
            this.loadScript();
        }
    }

    loadScript = async () => {
        try {
            const result = await this.props.onGetScript(this.props.hookId);
            if (result.exists) {
                const scriptType = detectScriptType(result.content);
                this.setState({
                    scriptContent: result.content,
                    originalScriptContent: result.content,
                    hasScript: true,
                    scriptType: scriptType,
                    selectedTemplate: 'empty',
                    isEditMode: true,
                    scriptPath: result.path,
                });
            } else {
                this.setState({
                    scriptContent: '',
                    originalScriptContent: '',
                    hasScript: false,
                    scriptType: 'bash',
                    selectedTemplate: 'empty',
                    isEditMode: false,
                });
            }
        } catch (error) {
            this.props.snackManager.snack('加载脚本文件失败');
        }
    };

    handleClose = () => {
        this.setState({
            scriptContent: this.state.originalScriptContent,
            errors: [],
            selectedTemplate: 'empty',
        });
        this.props.onClose();
    };

    handleContentChange = (code: string) => {
        const scriptType = detectScriptType(code);
        this.setState({
            scriptContent: code,
            scriptType: scriptType,
            selectedTemplate: 'empty',
        });
    };

    handleTemplateChange = (event: SelectChangeEvent<string>) => {
        const templateKey = event.target.value as string;
        const templateContent = templates[templateKey as keyof typeof templates];
        this.setState({
            selectedTemplate: templateKey,
            scriptContent: templateContent,
            scriptType: detectScriptType(templateContent),
        });
    };

    handleSave = async () => {
        try {
            await this.props.onSaveScript(this.props.hookId, this.state.scriptContent);
            this.props.snackManager.snack('脚本文件保存成功');
            this.setState({
                originalScriptContent: this.state.scriptContent,
                hasScript: true,
                errors: [],
            });
            this.props.onClose();
        } catch (error: any) {
            if (error.response?.data?.errors) {
                this.setState({errors: error.response.data.errors});
            } else {
                this.props.snackManager.snack('保存脚本文件失败');
            }
        }
    };

    handleDelete = async () => {
        if (window.confirm('确定要删除脚本文件吗？')) {
            try {
                await this.props.onDeleteScript(this.props.hookId);
                this.props.snackManager.snack('脚本文件删除成功');
                this.setState({
                    scriptContent: '',
                    originalScriptContent: '',
                    hasScript: false,
                    errors: [],
                    scriptType: 'bash',
                    selectedTemplate: 'empty',
                    isEditMode: false,
                });
                this.props.onClose();
            } catch (error) {
                this.props.snackManager.snack('删除脚本文件失败');
            }
        }
    };

    render() {
        const {open, hookId} = this.props;
        const {scriptContent, hasScript, errors, scriptType, selectedTemplate, isEditMode} = this.state;

        const formatIndicator = scriptType.toUpperCase();
        const formatColor = {
            bash: '#4CAF50',
            javascript: '#FF9800',
            json: '#2196F3',
            yaml: '#9C27B0'
        }[scriptType];

        // 检测是否为深色主题
        const isDarkTheme = localStorage.getItem('gohook-theme') === 'dark';

        // 编辑器样式
        const editorStyles = {
            fontFamily: 'Consolas, Monaco, "Courier New", monospace',
            fontSize: 13,
            minHeight: 400,
            maxHeight: 400,
            outline: 0,
            background: 'transparent',
            whiteSpace: 'pre' as const,
            color: isDarkTheme ? '#ffffff' : '#000000',
        };

        const editorContainerStyle = {
            border: `1px solid ${isDarkTheme ? '#444' : '#e0e0e0'}`,
            borderRadius: 4,
            background: isDarkTheme ? '#2d2d2d' : isEditMode ? '#f8f8f8' : '#fafafa',
            maxHeight: 400,
            overflow: 'auto',
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
                        maxHeight: '85vh',
                        height: 'auto',
                        color: isDarkTheme ? '#ffffff' : '#000000',
                    },
                }}>
                <DialogTitle>
                    <Box display="flex" alignItems="center" justifyContent="space-between">
                        <span>
                            {isEditMode ? '编辑脚本文件' : '创建脚本文件'} - {hookId}
                        </span>
                        {(isEditMode || scriptContent) && (
                            <Chip
                                label={`格式: ${formatIndicator}`}
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
                                <InputLabel>选择模板</InputLabel>
                                <Select
                                    value={selectedTemplate}
                                    onChange={this.handleTemplateChange}
                                    label="选择模板">
                                    <MenuItem value="empty">空白</MenuItem>
                                    <MenuItem value="bash_simple">简单 Bash 脚本</MenuItem>
                                    <MenuItem value="bash_git_deploy">Git 部署脚本</MenuItem>
                                    <MenuItem value="javascript_simple">简单 JavaScript 脚本</MenuItem>
                                    <MenuItem value="javascript_webhook_handler">Webhook 处理脚本</MenuItem>
                                    <MenuItem value="json_config">JSON 配置</MenuItem>
                                    <MenuItem value="yaml_config">YAML 配置</MenuItem>
                                </Select>
                            </FormControl>
                            <Typography
                                variant="caption"
                                color="textSecondary"
                                style={{display: 'block', marginTop: '8px'}}>
                                选择模板将自动填充内容到编辑器中
                            </Typography>
                        </Box>
                    )}

                    {/* 语法高亮编辑器 */}
                    <Box
                        mb={2}
                        style={editorContainerStyle}
                        className={isDarkTheme ? 'prism-dark' : 'prism-light'}>
                        <Editor
                            value={scriptContent}
                            onValueChange={this.handleContentChange}
                            highlight={(code) => highlightScript(code, scriptType, isDarkTheme)}
                            padding={16}
                            style={editorStyles}
                            textareaId="script-editor"
                            placeholder={
                                !isEditMode
                                    ? `# ${formatIndicator} 脚本内容\n\n# 选择上方模板快速开始`
                                    : `# ${formatIndicator} 脚本文件`
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

                    {/* 提示信息 */}
                    <Box mt={2} mb={1}>
                        {(isEditMode || scriptContent) && (
                            <Typography variant="body2" color="textSecondary">
                                <strong>检测到格式：</strong>
                                {formatIndicator} 格式内容
                            </Typography>
                        )}
                        <Typography variant="body2" color="textSecondary">
                            脚本文件路径: <code>{this.state.scriptPath || '未知路径'}</code>
                        </Typography>
                        {!isEditMode && !scriptContent && (
                            <Typography variant="body2" color="primary" style={{marginTop: '8px'}}>
                                💡 提示：选择上方模板可快速开始配置脚本
                            </Typography>
                        )}
                    </Box>
                </DialogContent>
                <DialogActions style={{paddingLeft: 24, paddingRight: 24}}>
                    {hasScript && (
                        <Button onClick={this.handleDelete} variant="contained" color="error">
                            删除文件
                        </Button>
                    )}
                    <Box flexGrow={1} />
                    <Button onClick={this.handleClose} variant="contained" color="secondary">
                        取消
                    </Button>
                    <Button onClick={this.handleSave} color="primary" variant="contained">
                        保存
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
}

export default inject('snackManager')(ScriptEditDialog);
