import React, {useState, useEffect} from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
    Box,
    Alert,
    Typography,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import {IHook} from '../types';

interface EditBasicDialogProps {
    open: boolean;
    onClose: () => void;
    hookId?: string;
    onSave: (
        hookId: string,
        basicData: {
            'execute-command': string;
            'command-working-directory': string;
            'response-message': string;
        }
    ) => void;
    onGetHookDetails: (hookId: string) => Promise<IHook>;
}

export default function EditBasicDialog({
    open,
    onClose,
    hookId,
    onSave,
    onGetHookDetails,
}: EditBasicDialogProps) {
    const [formData, setFormData] = useState({
        'execute-command': '',
        'command-working-directory': '',
        'response-message': '',
    });
    const [loading, setLoading] = useState(false);
    const [errors, setErrors] = useState<Record<string, string>>({});

    useEffect(() => {
        const loadHookData = async () => {
            if (hookId && open) {
                setLoading(true);
                try {
                    const hook = await onGetHookDetails(hookId);
                    setFormData({
                        'execute-command': hook['execute-command'] || '',
                        'command-working-directory': hook['command-working-directory'] || '',
                        'response-message': hook['response-message'] || '',
                    });
                } catch (error) {
                    console.error('加载Hook数据失败:', error);
                } finally {
                    setLoading(false);
                }
            }
            setErrors({});
        };

        loadHookData();
    }, [hookId, open, onGetHookDetails]);

    const handleFieldChange = (field: string, value: string) => {
        setFormData((prev) => ({
            ...prev,
            [field]: value,
        }));
        // 清除错误
        if (errors[field]) {
            setErrors((prev) => ({...prev, [field]: ''}));
        }
    };

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {};

        if (!formData['execute-command'].trim()) {
            newErrors['execute-command'] = '执行命令不能为空';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSave = () => {
        if (!validateForm() || !hookId) {
            return;
        }

        onSave(hookId, formData);
        onClose();
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>编辑基本信息 - {hookId}</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">
                            修改webhook的基本配置信息，包括执行命令、工作目录和响应消息。
                        </Typography>
                    </Alert>

                    {loading && (
                        <Alert severity="info" sx={{mb: 3}}>
                            <Typography variant="body2">正在加载Hook配置数据...</Typography>
                        </Alert>
                    )}

                    <Grid container spacing={3}>
                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label="执行命令"
                                value={formData['execute-command']}
                                onChange={(e) =>
                                    handleFieldChange('execute-command', e.target.value)
                                }
                                error={!!errors['execute-command']}
                                helperText={
                                    errors['execute-command'] ||
                                    '当webhook被触发时执行的命令或脚本路径'
                                }
                                placeholder="例如: /path/to/script.sh 或 node /path/to/handler.js"
                                required
                            />
                        </Grid>

                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label="工作目录"
                                value={formData['command-working-directory']}
                                onChange={(e) =>
                                    handleFieldChange('command-working-directory', e.target.value)
                                }
                                helperText="命令执行时的工作目录，留空则使用当前目录"
                                placeholder="例如: /var/www/project"
                            />
                        </Grid>

                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label="响应消息"
                                value={formData['response-message']}
                                onChange={(e) =>
                                    handleFieldChange('response-message', e.target.value)
                                }
                                helperText="webhook执行成功时返回的消息"
                                placeholder="例如: 部署完成"
                            />
                        </Grid>
                    </Grid>
                </Box>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>取消</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    保存
                </Button>
            </DialogActions>
        </Dialog>
    );
}
