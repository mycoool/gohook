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
import useTranslation from '../i18n/useTranslation';

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
    const {t} = useTranslation();
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
            newErrors['execute-command'] = t('hook.validation.commandRequired');
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
            <DialogTitle>{t('hook.editBasicTitle', {id: hookId || ''})}</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">{t('hook.editBasicDescription')}</Typography>
                    </Alert>

                    {loading && (
                        <Alert severity="info" sx={{mb: 3}}>
                            <Typography variant="body2">{t('hook.loadingConfig')}</Typography>
                        </Alert>
                    )}

                    <Grid container spacing={3}>
                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label={t('hook.fields.executeCommand')}
                                value={formData['execute-command']}
                                onChange={(e) =>
                                    handleFieldChange('execute-command', e.target.value)
                                }
                                error={!!errors['execute-command']}
                                helperText={
                                    errors['execute-command'] ||
                                    t('hook.fields.executeCommandHelper')
                                }
                                placeholder={t('hook.placeholders.executeCommand')}
                                required
                            />
                        </Grid>

                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label={t('hook.fields.workingDirectory')}
                                value={formData['command-working-directory']}
                                onChange={(e) =>
                                    handleFieldChange('command-working-directory', e.target.value)
                                }
                                helperText={t('hook.fields.workingDirectoryHelper')}
                                placeholder={t('hook.placeholders.workingDirectory')}
                            />
                        </Grid>

                        <Grid size={12}>
                            <TextField
                                fullWidth
                                label={t('hook.fields.responseMessage')}
                                value={formData['response-message']}
                                onChange={(e) =>
                                    handleFieldChange('response-message', e.target.value)
                                }
                                helperText={t('hook.fields.responseMessageHelper')}
                                placeholder={t('hook.placeholders.responseMessage')}
                            />
                        </Grid>
                    </Grid>
                </Box>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t('common.cancel')}</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    {t('common.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
