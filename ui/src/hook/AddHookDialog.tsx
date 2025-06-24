import React, {useState} from 'react';
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

interface AddHookDialogProps {
    open: boolean;
    onClose: () => void;
    onSave: (hookData: {
        id: string;
        'execute-command': string;
        'command-working-directory': string;
        'response-message': string;
    }) => void;
}

export default function AddHookDialog({open, onClose, onSave}: AddHookDialogProps) {
    const [formData, setFormData] = useState({
        id: '',
        'execute-command': '',
        'command-working-directory': '',
        'response-message': '执行成功',
    });

    const [errors, setErrors] = useState<Record<string, string>>({});

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

        if (!formData.id.trim()) {
            newErrors.id = 'Hook ID不能为空';
        } else if (!/^[a-zA-Z0-9\-_]+$/.test(formData.id)) {
            newErrors.id = 'Hook ID只能包含字母、数字、连字符和下划线';
        }

        if (!formData['execute-command'].trim()) {
            newErrors['execute-command'] = '执行命令不能为空';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSave = () => {
        if (!validateForm()) {
            return;
        }

        onSave(formData);
        handleClose();
    };

    const handleClose = () => {
        setFormData({
            id: '',
            'execute-command': '',
            'command-working-directory': '',
            'response-message': '执行成功',
        });
        setErrors({});
        onClose();
    };

    return (
        <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
            <DialogTitle>添加新的Webhook</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">
                            创建webhook的基本信息。创建后，您可以在列表中进一步配置参数传递、触发规则和响应设置。
                        </Typography>
                    </Alert>

                    <Grid container spacing={3}>
                        <Grid size={{xs: 12, md: 6}}>
                            <TextField
                                fullWidth
                                label="Hook ID"
                                value={formData.id}
                                onChange={(e) => handleFieldChange('id', e.target.value)}
                                error={!!errors.id}
                                helperText={errors.id || 'webhook的唯一标识符，用于构建URL路径'}
                                placeholder="例如: github-deploy"
                                required
                            />
                        </Grid>

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
                <Button onClick={handleClose}>取消</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    创建Hook
                </Button>
            </DialogActions>
        </Dialog>
    );
}
