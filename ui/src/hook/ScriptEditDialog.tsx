import React, {Component} from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Chip,
    Typography,
    Box,
    MenuItem,
    Select,
    FormControl,
    InputLabel,
    SelectChangeEvent,
    TextField,
    Paper,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
} from '@mui/material';
import {inject, Stores} from '../inject';
import {observer} from 'mobx-react';
import Editor from 'react-simple-code-editor';
import { createTheme } from '@mui/material/styles';
import Prism from 'prismjs';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-javascript';
import 'prismjs/components/prism-python';
import 'prismjs/themes/prism.css';
import '../version/EnvFileDialog.css';

// 扩展主题类型定义
declare module '@mui/material/styles' {
    interface Theme {
        custom: {
            colors: {
                primary: {
                    black: string;
                    darkGray: string;
                    mediumGray: string;
                    lightGray: string;
                };
                background: {
                    white: string;
                    lightGray: string;
                    mediumGray: string;
                    overlay: string;
                };
                border: {
                    light: string;
                    medium: string;
                    dark: string;
                    contrast: string;
                };
                text: {
                    primary: string;
                    secondary: string;
                    disabled: string;
                    onDark: string;
                    onDarkSecondary: string;
                };
                status: {
                    info: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    warning: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    error: {
                        background: string;
                        border: string;
                        text: string;
                    };
                    success: {
                        background: string;
                        border: string;
                        text: string;
                    };
                };
                interactive: {
                    button: {
                        command: string;
                        script: string;
                        hover: string;
                        disabled: string;
                    };
                    input: {
                        background: string;
                        border: string;
                        focus: string;
                        text: string;
                    };
                    code: {
                        background: string;
                        text: string;
                        padding: string;
                        borderRadius: number;
                        fontSize: string;
                    };
                };
            };
        };
    }
}

// 代码标签统一样式
const getCodeStyle = (theme: any) => ({
    backgroundColor: theme.custom.colors.interactive.code.background,
    color: theme.custom.colors.interactive.code.text,
    padding: theme.custom.colors.interactive.code.padding,
    borderRadius: theme.custom.colors.interactive.code.borderRadius,
    fontSize: theme.custom.colors.interactive.code.fontSize,
});

// 脚本类型定义
type ScriptType = 'bash' | 'javascript' | 'python';

// 检测脚本类型
const detectScriptType = (content: string): ScriptType => {
    const trimmedContent = content.trim();
    
    // 检测 Python
    if (trimmedContent.startsWith('#!/usr/bin/env python') || 
        trimmedContent.startsWith('#!/usr/bin/python') ||
        trimmedContent.includes('import ') || 
        trimmedContent.includes('def ') || 
        trimmedContent.includes('if __name__ == "__main__"') ||
        trimmedContent.includes('print(')) {
        return 'python';
    }
    
    // 检测 JavaScript
    if (trimmedContent.includes('function') || 
        trimmedContent.includes('=>') || 
        trimmedContent.includes('const ') || 
        trimmedContent.includes('let ') ||
        trimmedContent.includes('var ') ||
        trimmedContent.includes('console.log')) {
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
            case 'python':
                return Prism.highlight(code, Prism.languages.python, 'python');
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

    python_simple: `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# Python 脚本示例

import os
import sys

def main():
    # 从环境变量获取 webhook 信息
    hook_id = os.environ.get('HOOK_ID', 'unknown')
    method = os.environ.get('HOOK_METHOD', 'unknown')
    remote_addr = os.environ.get('HOOK_REMOTE_ADDR', 'unknown')
    
    print(f"Hook 被触发: {hook_id}")
    print(f"请求方法: {method}")
    print(f"远程地址: {remote_addr}")
    
    # 执行你的逻辑
    print("执行完成")

if __name__ == "__main__":
    main()`,

    python_webhook_handler: `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# Python Webhook 处理脚本示例

import os
import sys
import json
import logging
from datetime import datetime

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

def process_webhook():
    """处理 webhook 请求"""
    try:
        # 从环境变量获取数据
        hook_id = os.environ.get('HOOK_ID')
        payload = os.environ.get('HOOK_PAYLOAD', '{}')
        method = os.environ.get('HOOK_METHOD', 'POST')
        remote_addr = os.environ.get('HOOK_REMOTE_ADDR', 'unknown')
        
        # 解析 JSON payload
        try:
            data = json.loads(payload)
        except json.JSONDecodeError:
            data = {}
        
        logging.info(f"收到 Webhook: {hook_id}")
        logging.info(f"请求方法: {method}")
        logging.info(f"远程地址: {remote_addr}")
        
        # 处理不同类型的事件
        event_type = data.get('event_type', 'unknown')
        repository = data.get('repository', {}).get('name', 'unknown')
        
        logging.info(f"事件类型: {event_type}")
        logging.info(f"仓库名称: {repository}")
        
        # 记录到文件
        log_entry = {
            'timestamp': datetime.now().isoformat(),
            'hook_id': hook_id,
            'event_type': event_type,
            'repository': repository,
            'method': method,
            'remote_addr': remote_addr,
            'data': data
        }
        
        # 写入日志文件
        log_file = os.path.join(os.getcwd(), 'webhook.log')
        with open(log_file, 'a', encoding='utf-8') as f:
            f.write(json.dumps(log_entry, ensure_ascii=False) + '\\n')
        
        logging.info("处理完成")
        return True
        
    except Exception as e:
        logging.error(f"处理失败: {str(e)}")
        return False

def main():
    """主函数"""
    success = process_webhook()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()`
};

interface IProps {
    open: boolean;
    hookId: string;
    onClose: () => void;
    onGetScript: (hookId: string) => Promise<{content: string; exists: boolean; path: string; isExecutable?: boolean; editable?: boolean; message?: string; suggestion?: string}>;
    onSaveScript: (hookId: string, content: string, path?: string) => Promise<void>;
    onDeleteScript: (hookId: string) => Promise<void>;
    onUpdateExecuteCommand: (hookId: string, executeCommand: string) => Promise<void>;
    onGetHookDetails: (hookId: string) => Promise<any>;
}

// 编辑模式类型
type EditMode = 'executable' | 'script';

// 脚本创建阶段
type ScriptCreationStage = 'setup' | 'editing';

interface IState {
    scriptContent: string;
    originalScriptContent: string;
    hasScript: boolean;
    scriptType: ScriptType;
    selectedTemplate: string;
    isEditMode: boolean;
    errors: string[];
    scriptPath?: string;
    isExecutable: boolean;
    editable: boolean;
    message?: string;
    suggestion?: string;
    editMode: EditMode;
    executeCommand: string;
    originalExecuteCommand: string;
    scriptCreationStage: ScriptCreationStage;
    scriptName: string;
    scriptWorkingDirectory: string;
    showDeleteConfirm: boolean;
}

@observer
class ScriptEditDialog extends Component<IProps & Stores<'snackManager'>, IState> {
    private theme = createTheme({
        palette: { mode: 'dark' },
        custom: {
            colors: {
                primary: { darkGray: '#2c2c2c' },
                background: { white: '#424242' },
                border: { contrast: '#555555', light: '#616161' },
                text: { onDark: '#e0e0e0' },
                status: { info: { background: '#2c2c2c', border: '#555555', text: '#e0e0e0' } },
                interactive: { 
                    button: { command: '#616161', script: '#424242' },
                    code: { background: '#2c2c2c', text: '#e0e0e0', padding: '2px 6px', borderRadius: 4, fontSize: '0.875rem' }
                }
            }
        }
    } as any);
    state: IState = {
        scriptContent: '',
        originalScriptContent: '',
        hasScript: false,
        scriptType: 'bash',
        selectedTemplate: 'empty',
        isEditMode: false,
        errors: [],
        scriptPath: undefined,
        isExecutable: false,
        editable: true,
        message: undefined,
        suggestion: undefined,
        editMode: 'script',
        executeCommand: '',
        originalExecuteCommand: '',
        scriptCreationStage: 'editing',
        scriptName: '',
        scriptWorkingDirectory: '',
        showDeleteConfirm: false,
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
            const [scriptResult, hookDetails] = await Promise.all([
                this.props.onGetScript(this.props.hookId),
                this.props.onGetHookDetails(this.props.hookId)
            ]);
            
            // 设置默认的脚本名称和工作目录
            const defaultScriptName = this.props.hookId;
            const defaultWorkingDirectory = hookDetails['command-working-directory'] || '/tmp';
            
            // 检查是否不可编辑（可执行文件）
            if (scriptResult.editable === false || scriptResult.isExecutable === true) {
                this.setState({
                    scriptContent: '',
                    originalScriptContent: '',
                    hasScript: scriptResult.exists,
                    scriptType: 'bash',
                    selectedTemplate: 'empty',
                    isEditMode: false,
                    scriptPath: scriptResult.path,
                    isExecutable: scriptResult.isExecutable || false,
                    editable: scriptResult.editable || false,
                    message: scriptResult.message,
                    suggestion: scriptResult.suggestion,
                    editMode: 'executable', // 可执行文件模式
                    executeCommand: scriptResult.path,
                    originalExecuteCommand: scriptResult.path,
                    scriptCreationStage: 'editing',
                    scriptName: defaultScriptName,
                    scriptWorkingDirectory: defaultWorkingDirectory,
                });
                return;
            }
            
            if (scriptResult.exists && scriptResult.content) {
                const scriptType = detectScriptType(scriptResult.content);
                this.setState({
                    scriptContent: scriptResult.content,
                    originalScriptContent: scriptResult.content,
                    hasScript: true,
                    scriptType: scriptType,
                    selectedTemplate: 'empty',
                    isEditMode: true,
                    scriptPath: scriptResult.path,
                    isExecutable: false,
                    editable: true,
                    message: undefined,
                    suggestion: undefined,
                    editMode: 'script', // 脚本文件模式
                    executeCommand: scriptResult.path,
                    originalExecuteCommand: scriptResult.path,
                    scriptCreationStage: 'editing',
                    scriptName: defaultScriptName,
                    scriptWorkingDirectory: defaultWorkingDirectory,
                });
            } else {
                this.setState({
                    scriptContent: '',
                    originalScriptContent: '',
                    hasScript: false,
                    scriptType: 'bash',
                    selectedTemplate: 'empty',
                    isEditMode: false,
                    scriptPath: scriptResult.path,
                    isExecutable: false,
                    editable: true,
                    message: undefined,
                    suggestion: undefined,
                    editMode: 'script', // 默认脚本模式
                    executeCommand: scriptResult.path,
                    originalExecuteCommand: scriptResult.path,
                    scriptCreationStage: 'editing',
                    scriptName: defaultScriptName,
                    scriptWorkingDirectory: defaultWorkingDirectory,
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
            // 如果是创建新脚本模式，传递脚本路径；否则不传递路径
            const scriptPath = (this.state.scriptCreationStage === 'editing' && !this.state.isEditMode && this.state.scriptPath) 
                ? this.state.scriptPath 
                : undefined;
            
            await this.props.onSaveScript(this.props.hookId, this.state.scriptContent, scriptPath);
            
            // 如果是从设置阶段创建的脚本，需要同时更新Hook的execute-command
            if (this.state.scriptCreationStage === 'editing' && !this.state.isEditMode && this.state.scriptPath) {
                try {
                    await this.props.onUpdateExecuteCommand(this.props.hookId, this.state.scriptPath);
                } catch (error) {
                    // 如果更新execute-command失败，只显示警告，不阻止脚本保存
                    this.props.snackManager.snack('脚本保存成功，但更新执行命令失败，请手动配置');
                    return;
                }
            }
            
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

    handleDelete = () => {
        this.setState({ showDeleteConfirm: true });
    };

    handleDeleteConfirm = async () => {
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
                showDeleteConfirm: false,
                });
                this.props.onClose();
            } catch (error) {
                this.props.snackManager.snack('删除脚本文件失败');
            this.setState({ showDeleteConfirm: false });
        }
    };

    handleDeleteCancel = () => {
        this.setState({ showDeleteConfirm: false });
    };

    handleExecuteCommandChange = (value: string) => {
        this.setState({
            executeCommand: value,
        });
    };

    handleModeSwitch = (newMode: EditMode) => {
        if (newMode === 'script' && this.state.editMode === 'executable') {
            // 从可执行文件模式切换到脚本模式，进入设置阶段
            this.setState({
                editMode: newMode,
                errors: [],
                scriptCreationStage: 'setup',
                scriptContent: '',
                scriptType: 'bash',
                selectedTemplate: 'empty',
            });
        } else {
            this.setState({
                editMode: newMode,
                errors: [],
            });
        }
    };

    handleUpdateExecuteCommand = async () => {
        try {
            await this.props.onUpdateExecuteCommand(this.props.hookId, this.state.executeCommand);
            this.props.snackManager.snack('执行命令更新成功');
            this.setState({
                originalExecuteCommand: this.state.executeCommand,
                errors: [],
            });
            
            // 在命令模式下，不需要重新加载脚本状态，避免界面跳转
            // 只有在脚本模式下才需要重新加载脚本内容
            if (this.state.editMode === 'script') {
                await this.loadScript();
            }
        } catch (error: any) {
            this.props.snackManager.snack('更新执行命令失败');
        }
    };

    handleScriptNameChange = (name: string) => {
        this.setState({
            scriptName: name,
        });
    };

    handleScriptWorkingDirectoryChange = (directory: string) => {
        this.setState({
            scriptWorkingDirectory: directory,
        });
    };

    handleTemplateChangeInSetup = (event: SelectChangeEvent<string>) => {
        const template = event.target.value;
        const scriptType = this.getScriptTypeFromTemplate(template);
        this.setState({
            selectedTemplate: template,
            scriptType: scriptType,
        });
    };

    getScriptTypeFromTemplate = (template: string): ScriptType => {
        if (template.includes('bash')) return 'bash';
        if (template.includes('javascript')) return 'javascript';
        if (template.includes('python')) return 'python';
        return 'bash';
    };

    getFileExtension = (scriptType: ScriptType): string => {
        const extensions = {
            bash: '.sh',
            javascript: '.js',
            python: '.py'
        };
        return extensions[scriptType];
    };

    handleConfirmScriptSetup = () => {
        const {scriptName, scriptType, scriptWorkingDirectory, selectedTemplate} = this.state;
        
        if (!scriptName.trim()) {
            this.setState({
                errors: ['请输入脚本名称']
            });
            return;
        }

        // 生成完整的脚本路径
        const extension = this.getFileExtension(scriptType);
        const fileName = scriptName.endsWith(extension) ? scriptName : scriptName + extension;
        const fullPath = scriptWorkingDirectory.endsWith('/') 
            ? scriptWorkingDirectory + fileName 
            : scriptWorkingDirectory + '/' + fileName;

        // 根据模板生成初始内容
        const initialContent = templates[selectedTemplate as keyof typeof templates];

        this.setState({
            scriptCreationStage: 'editing',
            scriptPath: fullPath,
            scriptContent: initialContent,
            originalScriptContent: '',
            isEditMode: false,
            errors: [],
        });
    };

    render() {
        const {open, hookId} = this.props;
        const {
            scriptContent, 
            hasScript, 
            errors, 
            scriptType, 
            selectedTemplate, 
            isEditMode, 
            editable, 
            message, 
            suggestion, 
            isExecutable,
            editMode,
            executeCommand,
            originalExecuteCommand
        } = this.state;

        // 根据编辑模式和内容类型确定显示格式
        const getFormatDisplay = () => {
            if (editMode === 'executable') {
                return { label: '命令', color: this.theme.custom.colors.interactive.button.command };
            } else if (hasScript || scriptContent) {
                return { label: '脚本', color: this.theme.custom.colors.interactive.button.script };
            } else {
                return { label: '脚本', color: this.theme.custom.colors.interactive.button.script };
            }
        };
        
        const formatDisplay = getFormatDisplay();
        const formatIndicator = scriptType.toUpperCase();

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
            <>
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
                                {editMode === 'executable' ? '执行命令配置' : (isEditMode ? '编辑脚本文件' : '创建脚本文件')} - {hookId}
                        </span>
                            <Chip
                                label={formatDisplay.label}
                                style={{backgroundColor: formatDisplay.color, color: 'white'}}
                                size="small"
                            />
                    </Box>
                </DialogTitle>
                <DialogContent style={{paddingBottom: 0, overflow: 'visible'}}>

                    {editMode === 'executable' ? (
                        // 可执行文件模式
                        <Box>
                            <Typography variant="h6" gutterBottom>
                                ⚙️ 执行命令配置
                            </Typography>
                            <Typography variant="body2" color="textSecondary" gutterBottom>
                                配置要执行的命令或可执行文件路径，支持添加参数和选项
                            </Typography>
                            
                            <TextField
                                fullWidth
                                label="执行命令"
                                value={executeCommand}
                                onChange={(e) => this.handleExecuteCommandChange(e.target.value)}
                                placeholder="例如: /bin/echo hello 或 /usr/bin/python3 /path/to/script.py"
                                variant="outlined"
                                size="small"
                                style={{ marginBottom: 16 }}
                                multiline
                                rows={2}
                            />

                            <Box mb={2} p={2} style={{
                                backgroundColor: '#2c2c2c',
                                border: '1px solid #555555',
                                borderRadius: 4,
                            }}>
                                <Typography variant="subtitle2" style={{marginBottom: 8, color: '#e0e0e0'}}>
                                    💡 使用示例：
                                </Typography>
                                <Typography variant="body2" style={{marginBottom: 4, fontFamily: 'monospace', color: '#e0e0e0'}}>
                                    • <code style={getCodeStyle(this.theme)}>/bin/echo &quot;Hello World&quot;</code> - 输出文本
                                </Typography>
                                <Typography variant="body2" style={{marginBottom: 4, fontFamily: 'monospace', color: '#e0e0e0'}}>
                                    • <code style={getCodeStyle(this.theme)}>/usr/bin/curl -X POST https://api.example.com/webhook</code> - 发送HTTP请求
                                </Typography>
                                <Typography variant="body2" style={{marginBottom: 4, fontFamily: 'monospace', color: '#e0e0e0'}}>
                                    • <code style={getCodeStyle(this.theme)}>/usr/bin/python3 /path/to/your-script.py</code> - 执行Python脚本
                                </Typography>
                                <Typography variant="body2" style={{fontFamily: 'monospace', color: '#e0e0e0'}}>
                                    • <code style={getCodeStyle(this.theme)}>/bin/bash /path/to/your-script.sh</code> - 执行Bash脚本
                                </Typography>
                            </Box>

                            {message && (
                                <Box mb={2} p={2} style={{
                                    backgroundColor: '#2c2c2c',
                                    border: '1px solid #555555',
                                    borderRadius: 4,
                                    color: '#e0e0e0'
                                }}>
                                    <Typography variant="body2">
                                        <strong>💡 提示:</strong> {message}
                                    </Typography>
                                </Box>
                            )}
                        </Box>
                    ) : (
                        // 脚本文件模式
                        <Box>
                            {this.state.scriptCreationStage === 'setup' ? (
                                // 脚本设置阶段
                                <Box>
                                    <Typography variant="h6" gutterBottom>
                                        📝 创建新脚本文件
                                    </Typography>
                                    <Typography variant="body2" color="textSecondary" gutterBottom>
                                        请配置脚本文件的基本信息
                                    </Typography>

                                    <Paper elevation={1} style={{ padding: 16, marginBottom: 16 }}>
                                        <Box mb={2}>
                                            <TextField
                                                fullWidth
                                                label="脚本名称"
                                                value={this.state.scriptName}
                                                onChange={(e) => this.handleScriptNameChange(e.target.value)}
                                                placeholder="例如: webhook-handler"
                                                variant="outlined"
                                                size="small"
                                                helperText="不需要包含文件扩展名，会根据选择的类型自动添加"
                                            />
                                        </Box>
                                        <Box mb={2}>
                                            <TextField
                                                fullWidth
                                                label="保存目录"
                                                value={this.state.scriptWorkingDirectory}
                                                onChange={(e) => this.handleScriptWorkingDirectoryChange(e.target.value)}
                                                variant="outlined"
                                                size="small"
                                                helperText="脚本文件将保存到此目录"
                                            />
                                        </Box>
                                        <Box mb={2}>
                                            <FormControl fullWidth variant="outlined" size="small">
                                                <InputLabel>脚本类型和模板</InputLabel>
                                                <Select
                                                    value={this.state.selectedTemplate}
                                                    onChange={this.handleTemplateChangeInSetup}
                                                    label="脚本类型和模板">
                                                    <MenuItem value="empty">空白 Bash 脚本 (.sh)</MenuItem>
                                                    <MenuItem value="bash_simple">简单 Bash 脚本 (.sh)</MenuItem>
                                                    <MenuItem value="bash_git_deploy">Git 部署脚本 (.sh)</MenuItem>
                                                    <MenuItem value="javascript_simple">简单 JavaScript 脚本 (.js)</MenuItem>
                                                    <MenuItem value="javascript_webhook_handler">Webhook 处理脚本 (.js)</MenuItem>
                                                    <MenuItem value="python_simple">简单 Python 脚本 (.py)</MenuItem>
                                                    <MenuItem value="python_webhook_handler">Python Webhook 处理脚本 (.py)</MenuItem>
                                                </Select>
                                            </FormControl>
                                        </Box>

                                        {/* 预览生成的路径 */}
                                        <Box mt={2} p={1} style={{
                                            backgroundColor: '#2c2c2c',
                                            borderRadius: 4,
                                            border: '1px solid #555555'
                                        }}>
                                            <Typography variant="body2" style={{ color: '#e0e0e0' }}>
                                                <strong>生成的文件路径:</strong>
                                            </Typography>
                                            <Typography variant="body2" style={{ 
                                                fontFamily: 'monospace', 
                                                marginTop: 4,
                                                color: '#bdbdbd'
                                            }}>
                                                {this.state.scriptWorkingDirectory}
                                                {this.state.scriptWorkingDirectory.endsWith('/') ? '' : '/'}
                                                {this.state.scriptName}
                                                {this.state.scriptName && this.getFileExtension(this.state.scriptType)}
                                            </Typography>
                                        </Box>
                                    </Paper>
                                </Box>
                            ) : (
                                // 脚本编辑阶段
                                <Box>
                                    <Typography variant="h6" gutterBottom>
                                        📄 脚本文件编辑
                                    </Typography>
                                    
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
                                                    <MenuItem value="python_simple">简单 Python 脚本</MenuItem>
                                                    <MenuItem value="python_webhook_handler">Python Webhook 处理脚本</MenuItem>
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

                                    {/* 脚本文件信息 */}
                                    <Box mt={2} mb={1}>
                                        <Typography variant="body2" color="textSecondary">
                                            脚本文件路径: <code style={{backgroundColor: '#2c2c2c', color: '#e0e0e0', padding: '2px 6px', borderRadius: 4, fontSize: '0.875rem'}}>{this.state.scriptPath || '未知路径'}</code>
                                        </Typography>
                                        {!isEditMode && !scriptContent && (
                                            <Typography variant="body2" color="primary" style={{marginTop: '8px'}}>
                                                💡 提示：选择上方模板可快速开始配置脚本
                                            </Typography>
                                        )}
                                    </Box>
                                </Box>
                            )}
                        </Box>
                    )}

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
                </DialogContent>
                <DialogActions style={{paddingLeft: 24, paddingRight: 24}}>
                    {/* 左侧：模式切换 */}
                    <ToggleButtonGroup
                        value={editMode}
                        exclusive
                        onChange={(e, newMode) => newMode && this.handleModeSwitch(newMode)}
                        size="small"
                        style={{
                            backgroundColor: '#2c2c2c',
                            borderRadius: 4,
                        }}
                    >
                        <ToggleButton 
                            value="executable"
                            style={{
                                backgroundColor: editMode === 'executable' ? '#616161' : 'transparent',
                                color: editMode === 'executable' ? '#ffffff' : '#e0e0e0',
                                border: '1px solid #555555',
                                minWidth: '60px'
                            }}>
                            命令
                        </ToggleButton>
                        <ToggleButton 
                            value="script"
                            style={{
                                backgroundColor: editMode === 'script' ? '#424242' : 'transparent',
                                color: editMode === 'script' ? '#ffffff' : '#e0e0e0',
                                border: '1px solid #555555',
                                minWidth: '60px'
                            }}>
                            脚本
                        </ToggleButton>
                    </ToggleButtonGroup>
                    
                    <Box flexGrow={1} />
                    
                    {/* 右侧：操作按钮组 */}
                    <Box display="flex" gap={1}>
                        {/* 删除按钮 */}
                        {editMode === 'script' && hasScript && (
                            <Button onClick={this.handleDelete} variant="outlined" color="error">
                                删除
                            </Button>
                        )}
                        
                                                 {/* 关闭按钮 */}
                         <Button onClick={this.handleClose} variant="outlined" color="secondary">
                             关闭
                         </Button>
                     </Box>
                    
                    {editMode === 'executable' ? (
                        // 可执行文件模式按钮
                        <Tooltip
                            title={
                                executeCommand === originalExecuteCommand && executeCommand.trim()
                                    ? "命令未改变，无需更新"
                                    : !executeCommand.trim()
                                    ? "请输入执行命令"
                                    : "更新执行命令"
                            }
                            arrow
                        >
                            <span>
                                <Button 
                                    onClick={this.handleUpdateExecuteCommand} 
                                    color="primary" 
                                    variant="contained"
                                    disabled={executeCommand === originalExecuteCommand || !executeCommand.trim()}>
                                    更新执行命令
                                </Button>
                            </span>
                        </Tooltip>
                    ) : (
                        // 脚本文件模式按钮
                        this.state.scriptCreationStage === 'setup' ? (
                            <Button 
                                onClick={this.handleConfirmScriptSetup} 
                                color="primary" 
                                variant="contained"
                                disabled={!this.state.scriptName.trim()}>
                                确认并创建脚本
                            </Button>
                        ) : (
                            <Button onClick={this.handleSave} color="primary" variant="contained">
                                {isEditMode ? '保存修改' : '创建脚本'}
                        </Button>
                        )
                    )}
                </DialogActions>
            </Dialog>
            
            {/* 删除确认对话框 */}
            <Dialog
                open={this.state.showDeleteConfirm}
                onClose={this.handleDeleteCancel}
                aria-labelledby="delete-confirm-dialog-title"
                aria-describedby="delete-confirm-dialog-description"
            >
                <DialogTitle id="delete-confirm-dialog-title">
                    确认删除脚本文件
                </DialogTitle>
                <DialogContent>
                    <DialogContentText id="delete-confirm-dialog-description">
                        您确定要删除当前的脚本文件吗？此操作无法撤销。
                        <br />
                        <br />
                        脚本路径：<code style={{backgroundColor: '#2c2c2c', color: '#e0e0e0', padding: '2px 6px', borderRadius: 4, fontSize: '0.875rem'}}>{this.state.scriptPath}</code>
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.handleDeleteCancel} color="primary" variant="outlined">
                        取消
                    </Button>
                    <Button onClick={this.handleDeleteConfirm} color="error" variant="contained" autoFocus>
                        确认删除
                    </Button>
                </DialogActions>
            </Dialog>
            </>
        );
    }
}

export default inject('snackManager')(ScriptEditDialog);
