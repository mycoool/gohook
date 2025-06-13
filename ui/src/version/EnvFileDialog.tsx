import React, { useState, useEffect } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
    Typography,
    CircularProgress,
    Box,
    Chip
} from '@material-ui/core';
import { makeStyles } from '@material-ui/core/styles';
import { Save, Delete, FileCopy } from '@material-ui/icons';

const useStyles = makeStyles((theme) => ({
    content: {
        width: '600px',
        minHeight: '400px',
    },
    textArea: {
        fontFamily: 'Monaco, Menlo, "Ubuntu Mono", consolas, "source-code-pro", monospace',
        fontSize: '13px',
        lineHeight: '1.4',
    },
    pathInfo: {
        backgroundColor: theme.palette.grey[100],
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
        display: 'flex',
        alignItems: 'center',
        gap: theme.spacing(1),
    },
    errorAlert: {
        backgroundColor: theme.palette.error.light,
        color: theme.palette.error.contrastText,
        padding: theme.spacing(2),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
    },
    infoAlert: {
        backgroundColor: theme.palette.info.light,
        color: theme.palette.info.contrastText,
        padding: theme.spacing(2),
        borderRadius: theme.shape.borderRadius,
        marginBottom: theme.spacing(2),
    },
    errorDetail: {
        backgroundColor: 'rgba(0,0,0,0.1)',
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '12px',
        whiteSpace: 'pre-line',
        marginTop: theme.spacing(1),
    },
    exampleContent: {
        backgroundColor: theme.palette.grey[50],
        padding: theme.spacing(1),
        borderRadius: theme.shape.borderRadius,
        fontFamily: 'monospace',
        fontSize: '12px',
        marginTop: theme.spacing(1),
        border: `1px solid ${theme.palette.grey[300]}`,
        whiteSpace: 'pre-line',
        maxHeight: '200px',
        overflow: 'auto',
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
    
    const [loading, setLoading] = useState(false);
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

# 支持不同类型的值
PORT=3000
DEBUG=true

# 支持空值
EMPTY_VALUE=

# 支持带引号的值
DATABASE_URL="postgresql://user:pass@localhost/db"
SECRET_KEY='your-secret-key-here'

# 支持多行值（使用引号）
DESCRIPTION="这是一个
多行描述"

# 变量名规则：
# - 只能包含字母、数字和下划线
# - 必须以字母或下划线开头
# - 区分大小写`;

    useEffect(() => {
        if (open && projectName) {
            loadEnvFile();
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
            
            // 如果文件不存在，显示示例内容
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
        if (!exists) return;
        
        if (!window.confirm('确定要删除环境变量文件吗？此操作无法撤销。')) {
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

    const handleClose = () => {
        setContent('');
        setExists(false);
        setFilePath('');
        setError(null);
        setShowExample(false);
        onClose();
    };

    return (
        <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
            <DialogTitle>
                环境变量文件编辑 - {projectName}
            </DialogTitle>
            <DialogContent className={classes.content}>
                {loading ? (
                    <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
                        <CircularProgress />
                    </Box>
                ) : (
                    <>
                        <Box className={classes.pathInfo}>
                            <Typography variant="body2" color="textSecondary">
                                文件路径：
                            </Typography>
                            <Typography variant="body2" style={{ fontFamily: 'monospace' }}>
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
                                {error}
                                {error.includes('格式验证失败') && (
                                    <div className={classes.errorDetail}>
                                        {error.split('格式验证失败:\n')[1]}
                                    </div>
                                )}
                            </div>
                        )}

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
                                        style={{ marginTop: 8 }}>
                                        使用示例模板
                                    </Button>
                                </div>
                                
                                <Typography variant="subtitle2" style={{ marginTop: 16, marginBottom: 8 }}>
                                    示例内容：
                                </Typography>
                                <div className={classes.exampleContent}>
                                    {exampleEnvContent}
                                </div>
                            </Box>
                        )}

                        <TextField
                            fullWidth
                            multiline
                            minRows={15}
                            maxRows={25}
                            variant="outlined"
                            label="环境变量内容"
                            value={content}
                            onChange={(e) => setContent(e.target.value)}
                            placeholder={exists ? '请输入环境变量内容...' : '文件不存在，请创建新的环境变量文件'}
                            InputProps={{
                                className: classes.textArea,
                            }}
                            helperText="格式：变量名=变量值，每行一个。支持注释（#开头）和空行。"
                        />
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
                        color="secondary">
                        删除文件
                    </Button>
                )}
                <Button
                    onClick={handleSave}
                    disabled={saving || deleting || !content.trim()}
                    startIcon={saving ? <CircularProgress size={20} /> : <Save />}
                    color="primary"
                    variant="contained">
                    保存
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default EnvFileDialog; 