import React, { useState, useEffect } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    Typography,
    CircularProgress,
    Box,
    Chip,
    Paper
} from '@material-ui/core';
import { makeStyles, useTheme } from '@material-ui/core/styles';
import { Save, Delete, FileCopy } from '@material-ui/icons';
import MonacoEditor from 'react-monaco-editor';

const useStyles = makeStyles((theme) => ({
    content: {
        width: '100%',
        height: '70vh',
        display: 'flex',
        flexDirection: 'column',
        paddingTop: theme.spacing(1),
        paddingBottom: 0,
    },
    editorContainer: {
        flexGrow: 1,
        border: `1px solid ${theme.palette.divider}`,
        borderRadius: theme.shape.borderRadius,
        overflow: 'hidden',
        backgroundColor: theme.palette.type === 'dark' ? '#1e1e1e' : '#ffffff',
    },
    pathInfo: {
        backgroundColor: theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.05)' : theme.palette.grey[100],
        padding: theme.spacing(1.5),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
        display: 'flex',
        alignItems: 'center',
        gap: theme.spacing(1),
        border: `1px solid ${theme.palette.divider}`,
    },
    pathText: {
        fontFamily: 'monospace',
        color: theme.palette.type === 'dark' ? '#ffffff' : theme.palette.text.primary,
        fontWeight: 500,
    },
    errorAlert: {
        backgroundColor: theme.palette.error.dark,
        color: theme.palette.error.contrastText,
        padding: theme.spacing(2),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
        border: `1px solid ${theme.palette.error.main}`,
    },
    infoAlert: {
        backgroundColor: theme.palette.type === 'dark' ? 'rgba(33, 150, 243, 0.1)' : theme.palette.info.light,
        color: theme.palette.type === 'dark' ? '#90caf9' : theme.palette.info.contrastText,
        padding: theme.spacing(2),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
        border: `1px solid ${theme.palette.type === 'dark' ? 'rgba(33, 150, 243, 0.3)' : theme.palette.info.main}`,
    },
    errorDetail: {
        backgroundColor: 'rgba(0,0,0,0.2)',
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '12px',
        whiteSpace: 'pre-line',
        marginTop: theme.spacing(1),
    },
    exampleContent: {
        backgroundColor: theme.palette.type === 'dark' ? 'rgba(255, 255, 255, 0.03)' : theme.palette.grey[50],
        padding: theme.spacing(1.5),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '12px',
        marginTop: theme.spacing(1),
        border: `1px solid ${theme.palette.divider}`,
        whiteSpace: 'pre-line',
        maxHeight: '200px',
        overflow: 'auto',
        color: theme.palette.text.secondary,
    },
    dialogPaper: {
        maxWidth: '90vw',
        width: '1000px',
        maxHeight: '90vh',
    },
    exampleButton: {
        marginTop: theme.spacing(1),
        color: theme.palette.type === 'dark' ? '#90caf9' : theme.palette.info.main,
        borderColor: theme.palette.type === 'dark' ? '#90caf9' : theme.palette.info.main,
        '&:hover': {
            backgroundColor: theme.palette.type === 'dark' ? 'rgba(144, 202, 249, 0.1)' : 'rgba(33, 150, 243, 0.1)',
        }
    }
}));

interface EnvFileDialogProps {
    open: boolean;
    projectName: string;
    onClose: () => void;
    onGetEnvFile: (name: string) => Promise<{ content: string; exists: boolean; path: string }>;
    onSaveEnvFile: (name: string, content: string) => Promise<void>;
    onDeleteEnvFile: (name: string) => Promise<void>;
}

const EnvFileDialog: React.FC<EnvFileDialogProps> = ({
    open,
    projectName,
    onClose,
    onGetEnvFile,
    onSaveEnvFile,
    onDeleteEnvFile
}) => {
    const classes = useStyles();
    const theme = useTheme();
    
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [content, setContent] = useState('');
    const [exists, setExists] = useState(false);
    const [filePath, setFilePath] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [showExample, setShowExample] = useState(false);

    const exampleEnvContent = `# 环境变量配置文件示例
# 这是注释行，以#开头

# 基本格式：变量名=变量值
APP_NAME=MyApplication
APP_ENV=production

# 支持空值
EMPTY_VALUE=

# 支持带引号的值
DATABASE_URL="postgresql://user:pass@localhost/db"
SECRET_KEY='your-secret-key-here'

# 数字类型的值
PORT=3000
DEBUG=true`;

    useEffect(() => {
        if (open && projectName) {
            loadEnvFile();
        } else if (!open) {
            // Reset state when dialog is closed
            setLoading(true);
            setContent('');
            setError(null);
            setShowExample(false);
            setFilePath('');
            setExists(false);
        }
    }, [open, projectName]);

    const loadEnvFile = async () => {
        setLoading(true);
        setError(null);
        try {
            const result = await onGetEnvFile(projectName);
            setContent(result.content);
            setExists(result.exists);
            setFilePath(result.path);
            
            if (!result.exists) {
                setShowExample(true);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : '加载环境变量文件失败');
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        try {
            await onSaveEnvFile(projectName, content);
            setExists(true);
            setShowExample(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : '保存环境变量文件失败');
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async () => {
        if (!exists || !window.confirm('确定要删除环境变量文件吗？此操作无法撤销。')) {
            return;
        }

        setDeleting(true);
        setError(null);
        try {
            await onDeleteEnvFile(projectName);
            setContent('');
            setExists(false);
            setShowExample(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : '删除环境变量文件失败');
        } finally {
            setDeleting(false);
        }
    };

    const handleUseExample = () => {
        setContent(exampleEnvContent);
        setShowExample(false);
    };

    const handleEditorChange = (newValue: string) => {
        setContent(newValue);
    };

    const handleClose = () => {
        onClose();
    };

    // 根据主题选择编辑器主题
    const editorTheme = theme.palette.type === 'dark' ? 'vs-dark' : 'vs-light';
    
    const editorOptions = {
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 14,
        wordWrap: 'on' as const,
        lineNumbers: 'on' as const,
        renderLineHighlight: 'line' as const,
        selectOnLineNumbers: true,
        automaticLayout: true,
        tabSize: 2,
    };

    return (
        <Dialog 
            open={open} 
            onClose={handleClose} 
            maxWidth={false}
            fullWidth
            PaperProps={{
                className: classes.dialogPaper
            }}
        >
            <DialogTitle>
                环境变量文件编辑 - {projectName}
            </DialogTitle>
            <DialogContent className={classes.content}>
                <Box className={classes.pathInfo}>
                    <Typography variant="body2" color="textSecondary">
                        文件路径：
                    </Typography>
                    <Typography variant="body2" className={classes.pathText}>
                        {filePath}
                    </Typography>
                    {exists ? (
                        <Chip label="存在" size="small" color="primary" />
                    ) : (
                        <Chip label="不存在" size="small" color="default" />
                    )}
                </Box>

                {error && (
                    <div className={classes.errorAlert}>
                        <Typography variant="body2" style={{ fontWeight: 'bold', marginBottom: 8 }}>
                            错误信息
                        </Typography>
                        {error}
                        {error.includes('格式验证失败') && (
                            <div className={classes.errorDetail}>
                                {error.split('格式验证失败:\n')[1]}
                            </div>
                        )}
                    </div>
                )}

                {loading ? (
                    <Box display="flex" justifyContent="center" alignItems="center" flexGrow={1}>
                        <CircularProgress />
                        <Typography variant="body2" style={{ marginLeft: 16 }}>
                            正在加载环境变量文件...
                        </Typography>
                    </Box>
                ) : (
                    <>
                        {!exists && showExample && (
                            <Box marginBottom={2}>
                                <div className={classes.infoAlert}>
                                    <Typography variant="body2" style={{ marginBottom: 8 }}>
                                        当前项目没有环境变量文件，您可以创建一个新的。
                                    </Typography>
                                    <Button 
                                        size="small" 
                                        onClick={handleUseExample} 
                                        startIcon={<FileCopy />}
                                        variant="outlined"
                                        className={classes.exampleButton}
                                    >
                                        使用示例模板
                                    </Button>
                                </div>
                                <Typography variant="subtitle2" style={{ marginTop: 16, marginBottom: 8 }}>
                                    示例内容预览：
                                </Typography>
                                <div className={classes.exampleContent}>
                                    {exampleEnvContent}
                                </div>
                            </Box>
                        )}
                        
                        <Paper className={classes.editorContainer} variant="outlined">
                            <MonacoEditor
                                width="100%"
                                height="100%"
                                language="properties"
                                theme={editorTheme}
                                value={content}
                                options={editorOptions}
                                onChange={handleEditorChange}
                            />
                        </Paper>
                    </>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose} disabled={saving || deleting}>
                    取消
                </Button>
                {exists && (
                    <Button 
                        onClick={handleDelete} 
                        disabled={saving || deleting} 
                        startIcon={deleting ? <CircularProgress size={20} /> : <Delete />} 
                        color="secondary"
                    >
                        删除文件
                    </Button>
                )}
                <Button 
                    onClick={handleSave} 
                    disabled={saving || deleting || loading} 
                    startIcon={saving ? <CircularProgress size={20} /> : <Save />} 
                    color="primary" 
                    variant="contained"
                >
                    保存
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default EnvFileDialog; 