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
import useTranslation from '../i18n/useTranslation';

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
    const {t} = useTranslation();
    const defaultResponseMessage = t('hook.defaultResponseMessage');
    const [formData, setFormData] = useState({
        id: '',
        'execute-command': '',
        'command-working-directory': '',
        'response-message': defaultResponseMessage,
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
            newErrors.id = t('hook.validation.idRequired');
        } else if (!/^[a-zA-Z0-9\-_]+$/.test(formData.id)) {
            newErrors.id = t('hook.validation.idPattern');
        }

        if (!formData['execute-command'].trim()) {
            newErrors['execute-command'] = t('hook.validation.commandRequired');
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
            'response-message': defaultResponseMessage,
        });
        setErrors({});
        onClose();
    };

    return (
        <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
            <DialogTitle>{t('hook.addDialogTitle')}</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">{t('hook.addDialogDescription')}</Typography>
                    </Alert>

                    <Grid container spacing={3}>
                        <Grid size={{xs: 12, md: 6}}>
                            <TextField
                                fullWidth
                                label={t('hook.fields.id')}
                                value={formData.id}
                                onChange={(e) => handleFieldChange('id', e.target.value)}
                                error={!!errors.id}
                                helperText={errors.id || t('hook.fields.idHelper')}
                                placeholder={t('hook.placeholders.id')}
                                required
                            />
                        </Grid>

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
                <Button onClick={handleClose}>{t('common.cancel')}</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    {t('hook.createHook')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
