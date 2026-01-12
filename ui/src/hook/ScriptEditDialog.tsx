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
import {createTheme} from '@mui/material/styles';
import Prism from 'prismjs';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-javascript';
import 'prismjs/components/prism-python';
import 'prismjs/themes/prism.css';
import '../version/EnvFileDialog.css';
import translate from '../i18n/translator';

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
    if (
        trimmedContent.startsWith('#!/usr/bin/env python') ||
        trimmedContent.startsWith('#!/usr/bin/python') ||
        trimmedContent.includes('import ') ||
        trimmedContent.includes('def ') ||
        trimmedContent.includes('if __name__ == "__main__"') ||
        trimmedContent.includes('print(')
    ) {
        return 'python';
    }

    // 检测 JavaScript
    if (
        trimmedContent.includes('function') ||
        trimmedContent.includes('=>') ||
        trimmedContent.includes('const ') ||
        trimmedContent.includes('let ') ||
        trimmedContent.includes('var ') ||
        trimmedContent.includes('console.log')
    ) {
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
        "'": '&#039;',
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
const getTemplates = (t: (key: string, params?: Record<string, string | number>) => string) => ({
    empty: '',
    bash_simple: `#!/bin/bash
# ${t('hook.script.templateCodes.bashSimple.comment1')}

echo "${t('hook.script.templateCodes.bashSimple.line1')}"
echo "${t('hook.script.templateCodes.bashSimple.line2')}"
echo "${t('hook.script.templateCodes.bashSimple.line3')}"

# ${t('hook.script.templateCodes.bashSimple.comment2')}
echo "${t('hook.script.templateCodes.bashSimple.line4')}"`,

    bash_git_deploy: `#!/bin/bash
# ${t('hook.script.templateCodes.gitDeploy.comment1')}

set -e  # ${t('hook.script.templateCodes.gitDeploy.comment2')}

echo "${t('hook.script.templateCodes.gitDeploy.line1')}"

# ${t('hook.script.templateCodes.gitDeploy.comment3')}
cd /path/to/your/project

# ${t('hook.script.templateCodes.gitDeploy.comment4')}
git pull origin main

# ${t('hook.script.templateCodes.gitDeploy.comment5')}
# npm install
# yarn install
# composer install
# pip install -r requirements.txt

# ${t('hook.script.templateCodes.gitDeploy.comment6')}
# npm run build
# yarn build

# ${t('hook.script.templateCodes.gitDeploy.comment7')}
# systemctl restart your-service
# pm2 restart app
# docker-compose restart

echo "${t('hook.script.templateCodes.gitDeploy.line2')}"`,

    javascript_simple: `// ${t('hook.script.templateCodes.jsSimple.comment1')}
const hookId = process.env.HOOK_ID;
const method = process.env.HOOK_METHOD;
const remoteAddr = process.env.HOOK_REMOTE_ADDR;

console.log(\`${t('hook.script.templateCodes.jsSimple.line1')}\`);
console.log(\`${t('hook.script.templateCodes.jsSimple.line2')}\`);
console.log(\`${t('hook.script.templateCodes.jsSimple.line3')}\`);

// ${t('hook.script.templateCodes.jsSimple.comment2')}
console.log("${t('hook.script.templateCodes.jsSimple.line4')}");`,

    javascript_webhook_handler: `// ${t('hook.script.templateCodes.jsWebhook.comment1')}
const fs = require('fs');
const path = require('path');

// ${t('hook.script.templateCodes.jsWebhook.comment2')}
const hookId = process.env.HOOK_ID;
const payload = process.env.HOOK_PAYLOAD;

try {
    // ${t('hook.script.templateCodes.jsWebhook.comment3')}
    const data = payload ? JSON.parse(payload) : {};

    console.log('${t('hook.script.templateCodes.jsWebhook.line1')}', {
        hookId,
        event: data.event_type || 'unknown',
        repository: data.repository?.name || 'unknown'
    });

    // ${t('hook.script.templateCodes.jsWebhook.comment4')}
    const logEntry = {
        timestamp: new Date().toISOString(),
        hookId,
        data
    };

    fs.appendFileSync(
        path.join(__dirname, 'webhook.log'),
        JSON.stringify(logEntry) + '\\n'
    );

    console.log('${t('hook.script.templateCodes.jsWebhook.line2')}');
} catch (error) {
    console.error('${t('hook.script.templateCodes.jsWebhook.line3')}', error.message);
    process.exit(1);
}`,

    python_simple: `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# ${t('hook.script.templateCodes.pySimple.comment1')}

import os
import sys

def main():
    # ${t('hook.script.templateCodes.pySimple.comment2')}
    hook_id = os.environ.get('HOOK_ID', 'unknown')
    method = os.environ.get('HOOK_METHOD', 'unknown')
    remote_addr = os.environ.get('HOOK_REMOTE_ADDR', 'unknown')

    print(f"${t('hook.script.templateCodes.pySimple.line1')}")
    print(f"${t('hook.script.templateCodes.pySimple.line2')}")
    print(f"${t('hook.script.templateCodes.pySimple.line3')}")

    # ${t('hook.script.templateCodes.pySimple.comment3')}
    print("${t('hook.script.templateCodes.pySimple.line4')}")

if __name__ == "__main__":
    main()`,

    python_webhook_handler: `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# ${t('hook.script.templateCodes.pyWebhook.comment1')}

import os
import sys
import json
import logging
from datetime import datetime

# ${t('hook.script.templateCodes.pyWebhook.comment2')}
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

def process_webhook():
    """${t('hook.script.templateCodes.pyWebhook.comment3')}"""
    try:
        # ${t('hook.script.templateCodes.pyWebhook.comment4')}
        hook_id = os.environ.get('HOOK_ID')
        payload = os.environ.get('HOOK_PAYLOAD', '{}')
        method = os.environ.get('HOOK_METHOD', 'POST')
        remote_addr = os.environ.get('HOOK_REMOTE_ADDR', 'unknown')

        # ${t('hook.script.templateCodes.pyWebhook.comment5')}
        try:
            data = json.loads(payload)
        except json.JSONDecodeError:
            data = {}

        logging.info(f"${t('hook.script.templateCodes.pyWebhook.line1')}")
        logging.info(f"${t('hook.script.templateCodes.pyWebhook.line2')}")
        logging.info(f"${t('hook.script.templateCodes.pyWebhook.line3')}")

        # ${t('hook.script.templateCodes.pyWebhook.comment6')}
        event_type = data.get('event_type', 'unknown')
        repository = data.get('repository', {}).get('name', 'unknown')

        logging.info(f"${t('hook.script.templateCodes.pyWebhook.line4')}")
        logging.info(f"${t('hook.script.templateCodes.pyWebhook.line5')}")

        # ${t('hook.script.templateCodes.pyWebhook.comment7')}
        log_entry = {
            'timestamp': datetime.now().isoformat(),
            'hook_id': hook_id,
            'event_type': event_type,
            'repository': repository,
            'method': method,
            'remote_addr': remote_addr,
            'data': data
        }

        # ${t('hook.script.templateCodes.pyWebhook.comment8')}
        log_file = os.path.join(os.getcwd(), 'webhook.log')
        with open(log_file, 'a', encoding='utf-8') as f:
            f.write(json.dumps(log_entry, ensure_ascii=False) + '\\n')

        logging.info("${t('hook.script.templateCodes.pyWebhook.line6')}")
        return True

    except Exception as e:
        logging.error(f"${t('hook.script.templateCodes.pyWebhook.line7')}")
        return False

def main():
    """${t('hook.script.templateCodes.pyWebhook.comment9')}"""
    success = process_webhook()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()`,
});

interface IProps {
    open: boolean;
    hookId: string;
    onClose: () => void;
    onGetScript: (hookId: string) => Promise<{
        content: string;
        exists: boolean;
        path: string;
        isExecutable?: boolean;
        editable?: boolean;
        message?: string;
        suggestion?: string;
    }>;
    onSaveScript: (hookId: string, content: string, path?: string) => Promise<void>;
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
}

@observer
class ScriptEditDialog extends Component<IProps & Stores<'snackManager'>, IState> {
    private theme = createTheme({
        palette: {mode: 'dark'},
        custom: {
            colors: {
                primary: {darkGray: '#2c2c2c'},
                background: {white: '#424242'},
                border: {contrast: '#555555', light: '#616161'},
                text: {onDark: '#e0e0e0'},
                status: {info: {background: '#2c2c2c', border: '#555555', text: '#e0e0e0'}},
                interactive: {
                    button: {command: '#616161', script: '#424242'},
                    code: {
                        background: '#2c2c2c',
                        text: '#e0e0e0',
                        padding: '2px 6px',
                        borderRadius: 4,
                        fontSize: '0.875rem',
                    },
                },
            },
        },
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

    private getTemplates = () => {
        const t = translate;
        return getTemplates(t);
    };

    loadScript = async () => {
        try {
            const [scriptResult, hookDetails] = await Promise.all([
                this.props.onGetScript(this.props.hookId),
                this.props.onGetHookDetails(this.props.hookId),
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
            this.props.snackManager.snack(translate('hook.script.loadFailed'));
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
        const templates = this.getTemplates();
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
            const scriptPath =
                this.state.scriptCreationStage === 'editing' &&
                !this.state.isEditMode &&
                this.state.scriptPath
                    ? this.state.scriptPath
                    : undefined;

            await this.props.onSaveScript(this.props.hookId, this.state.scriptContent, scriptPath);

            // 如果是从设置阶段创建的脚本，需要同时更新Hook的execute-command
            if (
                this.state.scriptCreationStage === 'editing' &&
                !this.state.isEditMode &&
                this.state.scriptPath
            ) {
                try {
                    await this.props.onUpdateExecuteCommand(
                        this.props.hookId,
                        this.state.scriptPath
                    );
                } catch (error) {
                    // 如果更新execute-command失败，只显示警告，不阻止脚本保存
                    this.props.snackManager.snack(
                        translate('hook.script.saveWarningUpdateCommandFailed')
                    );
                    return;
                }
            }

            this.props.snackManager.snack(translate('hook.script.saveSuccess'));
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
                this.props.snackManager.snack(translate('hook.script.saveFailed'));
            }
        }
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
            this.props.snackManager.snack(translate('hook.script.updateCommandSuccess'));
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
            this.props.snackManager.snack(translate('hook.script.updateCommandFailed'));
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
            python: '.py',
        };
        return extensions[scriptType];
    };

    handleConfirmScriptSetup = () => {
        const {scriptName, scriptType, scriptWorkingDirectory, selectedTemplate} = this.state;

        if (!scriptName.trim()) {
            this.setState({
                errors: [translate('hook.script.validation.scriptNameRequired')],
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
        const templates = this.getTemplates();
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
            originalExecuteCommand,
        } = this.state;
        const t = translate;

        // 根据编辑模式和内容类型确定显示格式
        const getFormatDisplay = () => {
            if (editMode === 'executable') {
                return {
                    label: t('hook.script.modeCommand'),
                    color: this.theme.custom.colors.interactive.button.command,
                };
            } else if (hasScript || scriptContent) {
                return {
                    label: t('hook.script.modeScript'),
                    color: this.theme.custom.colors.interactive.button.script,
                };
            } else {
                return {
                    label: t('hook.script.modeScript'),
                    color: this.theme.custom.colors.interactive.button.script,
                };
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
                                {editMode === 'executable'
                                    ? t('hook.script.dialogTitleCommand')
                                    : isEditMode
                                    ? t('hook.script.dialogTitleEdit')
                                    : t('hook.script.dialogTitleCreate')}{' '}
                                - {hookId}
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
                                    {t('hook.script.commandSectionTitle')}
                                </Typography>
                                <Typography variant="body2" color="textSecondary" gutterBottom>
                                    {t('hook.script.commandSectionDescription')}
                                </Typography>

                                <TextField
                                    fullWidth
                                    label={t('hook.script.executeCommandLabel')}
                                    value={executeCommand}
                                    onChange={(e) =>
                                        this.handleExecuteCommandChange(e.target.value)
                                    }
                                    placeholder={t('hook.script.executeCommandPlaceholder')}
                                    variant="outlined"
                                    size="small"
                                    style={{marginBottom: 16}}
                                    multiline
                                    rows={2}
                                />

                                <Box
                                    mb={2}
                                    p={2}
                                    style={{
                                        backgroundColor: '#2c2c2c',
                                        border: '1px solid #555555',
                                        borderRadius: 4,
                                    }}>
                                    <Typography
                                        variant="subtitle2"
                                        style={{marginBottom: 8, color: '#e0e0e0'}}>
                                        {t('hook.script.commandExamplesTitle')}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        style={{
                                            marginBottom: 4,
                                            fontFamily: 'monospace',
                                            color: '#e0e0e0',
                                        }}>
                                        •{' '}
                                        <code style={getCodeStyle(this.theme)}>
                                            /bin/echo &quot;Hello World&quot;
                                        </code>{' '}
                                        - {t('hook.script.commandExampleOutput')}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        style={{
                                            marginBottom: 4,
                                            fontFamily: 'monospace',
                                            color: '#e0e0e0',
                                        }}>
                                        •{' '}
                                        <code style={getCodeStyle(this.theme)}>
                                            /usr/bin/curl -X POST https://api.example.com/webhook
                                        </code>{' '}
                                        - {t('hook.script.commandExampleHttp')}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        style={{
                                            marginBottom: 4,
                                            fontFamily: 'monospace',
                                            color: '#e0e0e0',
                                        }}>
                                        •{' '}
                                        <code style={getCodeStyle(this.theme)}>
                                            /usr/bin/python3 /path/to/your-script.py
                                        </code>{' '}
                                        - {t('hook.script.commandExamplePython')}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        style={{fontFamily: 'monospace', color: '#e0e0e0'}}>
                                        •{' '}
                                        <code style={getCodeStyle(this.theme)}>
                                            /bin/bash /path/to/your-script.sh
                                        </code>{' '}
                                        - {t('hook.script.commandExampleBash')}
                                    </Typography>
                                </Box>

                                {message && (
                                    <Box
                                        mb={2}
                                        p={2}
                                        style={{
                                            backgroundColor: '#2c2c2c',
                                            border: '1px solid #555555',
                                            borderRadius: 4,
                                            color: '#e0e0e0',
                                        }}>
                                        <Typography variant="body2">
                                            <strong>{t('hook.script.tipPrefix')}</strong> {message}
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
                                            {t('hook.script.setupTitle')}
                                        </Typography>
                                        <Typography
                                            variant="body2"
                                            color="textSecondary"
                                            gutterBottom>
                                            {t('hook.script.setupDescription')}
                                        </Typography>

                                        <Paper
                                            elevation={1}
                                            style={{padding: 16, marginBottom: 16}}>
                                            <Box mb={2}>
                                                <TextField
                                                    fullWidth
                                                    label={t('hook.script.scriptNameLabel')}
                                                    value={this.state.scriptName}
                                                    onChange={(e) =>
                                                        this.handleScriptNameChange(e.target.value)
                                                    }
                                                    placeholder={t(
                                                        'hook.script.scriptNamePlaceholder'
                                                    )}
                                                    variant="outlined"
                                                    size="small"
                                                    helperText={t('hook.script.scriptNameHelper')}
                                                />
                                            </Box>
                                            <Box mb={2}>
                                                <TextField
                                                    fullWidth
                                                    label={t('hook.script.scriptDirectoryLabel')}
                                                    value={this.state.scriptWorkingDirectory}
                                                    onChange={(e) =>
                                                        this.handleScriptWorkingDirectoryChange(
                                                            e.target.value
                                                        )
                                                    }
                                                    variant="outlined"
                                                    size="small"
                                                    helperText={t(
                                                        'hook.script.scriptDirectoryHelper'
                                                    )}
                                                />
                                            </Box>
                                            <Box mb={2}>
                                                <FormControl
                                                    fullWidth
                                                    variant="outlined"
                                                    size="small">
                                                    <InputLabel>
                                                        {t('hook.script.templateLabel')}
                                                    </InputLabel>
                                                    <Select
                                                        value={this.state.selectedTemplate}
                                                        onChange={this.handleTemplateChangeInSetup}
                                                        label={t('hook.script.templateLabel')}>
                                                        <MenuItem value="empty">
                                                            {t('hook.script.templates.emptyBash')}
                                                        </MenuItem>
                                                        <MenuItem value="bash_simple">
                                                            {t('hook.script.templates.simpleBash')}
                                                        </MenuItem>
                                                        <MenuItem value="bash_git_deploy">
                                                            {t('hook.script.templates.gitDeploy')}
                                                        </MenuItem>
                                                        <MenuItem value="javascript_simple">
                                                            {t(
                                                                'hook.script.templates.simpleJavascript'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="javascript_webhook_handler">
                                                            {t(
                                                                'hook.script.templates.webhookJavascript'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="python_simple">
                                                            {t(
                                                                'hook.script.templates.simplePython'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="python_webhook_handler">
                                                            {t(
                                                                'hook.script.templates.webhookPython'
                                                            )}
                                                        </MenuItem>
                                                    </Select>
                                                </FormControl>
                                            </Box>

                                            {/* 预览生成的路径 */}
                                            <Box
                                                mt={2}
                                                p={1}
                                                style={{
                                                    backgroundColor: '#2c2c2c',
                                                    borderRadius: 4,
                                                    border: '1px solid #555555',
                                                }}>
                                                <Typography
                                                    variant="body2"
                                                    style={{color: '#e0e0e0'}}>
                                                    <strong>
                                                        {t('hook.script.generatedPathLabel')}
                                                    </strong>
                                                </Typography>
                                                <Typography
                                                    variant="body2"
                                                    style={{
                                                        fontFamily: 'monospace',
                                                        marginTop: 4,
                                                        color: '#bdbdbd',
                                                    }}>
                                                    {this.state.scriptWorkingDirectory}
                                                    {this.state.scriptWorkingDirectory.endsWith('/')
                                                        ? ''
                                                        : '/'}
                                                    {this.state.scriptName}
                                                    {this.state.scriptName &&
                                                        this.getFileExtension(
                                                            this.state.scriptType
                                                        )}
                                                </Typography>
                                            </Box>
                                        </Paper>
                                    </Box>
                                ) : (
                                    // 脚本编辑阶段
                                    <Box>
                                        <Typography variant="h6" gutterBottom>
                                            {t('hook.script.editTitle')}
                                        </Typography>

                                        {/* 模板选择器 - 仅在创建模式显示 */}
                                        {!isEditMode && (
                                            <Box mb={2}>
                                                <FormControl
                                                    fullWidth
                                                    variant="outlined"
                                                    size="small">
                                                    <InputLabel>
                                                        {t('hook.script.templateSelectLabel')}
                                                    </InputLabel>
                                                    <Select
                                                        value={selectedTemplate}
                                                        onChange={this.handleTemplateChange}
                                                        label={t(
                                                            'hook.script.templateSelectLabel'
                                                        )}>
                                                        <MenuItem value="empty">
                                                            {t('hook.script.templates.empty')}
                                                        </MenuItem>
                                                        <MenuItem value="bash_simple">
                                                            {t('hook.script.templates.simpleBash')}
                                                        </MenuItem>
                                                        <MenuItem value="bash_git_deploy">
                                                            {t('hook.script.templates.gitDeploy')}
                                                        </MenuItem>
                                                        <MenuItem value="javascript_simple">
                                                            {t(
                                                                'hook.script.templates.simpleJavascript'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="javascript_webhook_handler">
                                                            {t(
                                                                'hook.script.templates.webhookJavascript'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="python_simple">
                                                            {t(
                                                                'hook.script.templates.simplePython'
                                                            )}
                                                        </MenuItem>
                                                        <MenuItem value="python_webhook_handler">
                                                            {t(
                                                                'hook.script.templates.webhookPython'
                                                            )}
                                                        </MenuItem>
                                                    </Select>
                                                </FormControl>
                                                <Typography
                                                    variant="caption"
                                                    color="textSecondary"
                                                    style={{display: 'block', marginTop: '8px'}}>
                                                    {t('hook.script.templateFillHint')}
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
                                                highlight={(code) =>
                                                    highlightScript(code, scriptType, isDarkTheme)
                                                }
                                                padding={16}
                                                style={editorStyles}
                                                textareaId="script-editor"
                                                placeholder={
                                                    !isEditMode
                                                        ? t('hook.script.editorPlaceholderNew', {
                                                              format: formatIndicator,
                                                          })
                                                        : t('hook.script.editorPlaceholderEdit', {
                                                              format: formatIndicator,
                                                          })
                                                }
                                            />
                                        </Box>

                                        {/* 脚本文件信息 */}
                                        <Box mt={2} mb={1}>
                                            <Typography variant="body2" color="textSecondary">
                                                {t('hook.script.scriptPathLabel')}{' '}
                                                <code
                                                    style={{
                                                        backgroundColor: '#2c2c2c',
                                                        color: '#e0e0e0',
                                                        padding: '2px 6px',
                                                        borderRadius: 4,
                                                        fontSize: '0.875rem',
                                                    }}>
                                                    {this.state.scriptPath ||
                                                        t('hook.script.unknownPath')}
                                                </code>
                                            </Typography>
                                            {!isEditMode && !scriptContent && (
                                                <Typography
                                                    variant="body2"
                                                    color="primary"
                                                    style={{marginTop: '8px'}}>
                                                    {t('hook.script.templateQuickHint')}
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
                                    {t('hook.script.validationTitle')}
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
                            }}>
                            <ToggleButton
                                value="executable"
                                style={{
                                    backgroundColor:
                                        editMode === 'executable' ? '#616161' : 'transparent',
                                    color: editMode === 'executable' ? '#ffffff' : '#e0e0e0',
                                    border: '1px solid #555555',
                                    minWidth: '60px',
                                }}>
                                {t('hook.script.modeCommand')}
                            </ToggleButton>
                            <ToggleButton
                                value="script"
                                style={{
                                    backgroundColor:
                                        editMode === 'script' ? '#424242' : 'transparent',
                                    color: editMode === 'script' ? '#ffffff' : '#e0e0e0',
                                    border: '1px solid #555555',
                                    minWidth: '60px',
                                }}>
                                {t('hook.script.modeScript')}
                            </ToggleButton>
                        </ToggleButtonGroup>

                        <Box flexGrow={1} />

                        {/* 右侧：操作按钮组 */}
                        <Box display="flex" gap={1}>
                            <Button onClick={this.handleClose} variant="outlined" color="secondary">
                                {t('common.close')}
                            </Button>
                        </Box>

                        {editMode === 'executable' ? (
                            // 可执行文件模式按钮
                            <Tooltip
                                title={
                                    executeCommand === originalExecuteCommand &&
                                    executeCommand.trim()
                                        ? t('hook.script.updateCommandNoChange')
                                        : !executeCommand.trim()
                                        ? t('hook.script.updateCommandMissing')
                                        : t('hook.script.updateCommandAction')
                                }
                                arrow>
                                <span>
                                    <Button
                                        onClick={this.handleUpdateExecuteCommand}
                                        color="primary"
                                        variant="contained"
                                        disabled={
                                            executeCommand === originalExecuteCommand ||
                                            !executeCommand.trim()
                                        }>
                                        {t('hook.script.updateCommandButton')}
                                    </Button>
                                </span>
                            </Tooltip>
                        ) : // 脚本文件模式按钮
                        this.state.scriptCreationStage === 'setup' ? (
                            <Button
                                onClick={this.handleConfirmScriptSetup}
                                color="primary"
                                variant="contained"
                                disabled={!this.state.scriptName.trim()}>
                                {t('hook.script.confirmCreateScript')}
                            </Button>
                        ) : (
                            <Button onClick={this.handleSave} color="primary" variant="contained">
                                {isEditMode
                                    ? t('hook.script.saveChanges')
                                    : t('hook.script.createScript')}
                            </Button>
                        )}
                    </DialogActions>
                </Dialog>
            </>
        );
    }
}

export default inject('snackManager')(ScriptEditDialog);
