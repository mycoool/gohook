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
    Card,
    CardContent,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    IconButton,
    FormControl,
    Select,
    MenuItem,
    FormControlLabel,
    Switch,
    Chip,
} from '@mui/material';
import {Add as AddIcon, Delete as DeleteIcon} from '@mui/icons-material';
import Grid from '@mui/material/Grid';
import {IHook} from '../types';
import useTranslation from '../i18n/useTranslation';

interface EditResponseDialogProps {
    open: boolean;
    onClose: () => void;
    hookId?: string;
    onSave: (
        hookId: string,
        responseData: {
            'http-methods': string[];
            'response-headers': {[key: string]: string};
            'include-command-output-in-response': boolean;
            'include-command-output-in-response-on-error': boolean;
        }
    ) => void;
    onGetHookDetails: (hookId: string) => Promise<IHook>;
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

export default function EditResponseDialog({
    open,
    onClose,
    hookId,
    onSave,
    onGetHookDetails,
}: EditResponseDialogProps) {
    const {t} = useTranslation();
    const [formData, setFormData] = useState({
        'http-methods': ['POST'] as string[],
        'response-headers': {} as {[key: string]: string},
        'include-command-output-in-response': false,
        'include-command-output-in-response-on-error': false,
    });
    const [loading, setLoading] = useState(false);
    const [newHeader, setNewHeader] = useState({name: '', value: ''});

    useEffect(() => {
        const loadHookData = async () => {
            if (hookId && open) {
                setLoading(true);
                try {
                    const hook = await onGetHookDetails(hookId);
                    setFormData({
                        'http-methods': hook['http-methods'] || ['POST'],
                        'response-headers': hook['response-headers'] || {},
                        'include-command-output-in-response':
                            hook['include-command-output-in-response'] || false,
                        'include-command-output-in-response-on-error':
                            hook['include-command-output-in-response-on-error'] || false,
                    });
                } catch (error) {
                    console.error('加载Hook数据失败:', error);
                } finally {
                    setLoading(false);
                }
            }
        };

        loadHookData();
    }, [hookId, open, onGetHookDetails]);

    const handleMethodToggle = (method: string) => {
        setFormData((prev) => {
            const methods = prev['http-methods'].includes(method)
                ? prev['http-methods'].filter((m) => m !== method)
                : [...prev['http-methods'], method];

            // 至少保留一个方法
            if (methods.length === 0) {
                return prev;
            }

            return {
                ...prev,
                'http-methods': methods,
            };
        });
    };

    const addResponseHeader = () => {
        if (!newHeader.name.trim() || !newHeader.value.trim()) {
            return;
        }

        setFormData((prev) => ({
            ...prev,
            'response-headers': {
                ...prev['response-headers'],
                [newHeader.name]: newHeader.value,
            },
        }));

        setNewHeader({name: '', value: ''});
    };

    const removeResponseHeader = (headerName: string) => {
        setFormData((prev) => {
            const headers = {...prev['response-headers']};
            delete headers[headerName];
            return {
                ...prev,
                'response-headers': headers,
            };
        });
    };

    const handleSave = () => {
        if (!hookId) return;

        onSave(hookId, formData);
        onClose();
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>{t('hook.editResponseTitle', {id: hookId || ''})}</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">{t('hook.response.description')}</Typography>
                    </Alert>

                    {/* HTTP方法配置 */}
                    <Card sx={{mb: 3}}>
                        <CardContent>
                            <Typography variant="h6" sx={{mb: 2}}>
                                {t('hook.response.supportedMethods')}
                            </Typography>
                            <Box sx={{display: 'flex', flexWrap: 'wrap', gap: 1}}>
                                {HTTP_METHODS.map((method) => (
                                    <Chip
                                        key={method}
                                        label={method}
                                        onClick={() => handleMethodToggle(method)}
                                        color={
                                            formData['http-methods'].includes(method)
                                                ? 'primary'
                                                : 'default'
                                        }
                                        variant={
                                            formData['http-methods'].includes(method)
                                                ? 'filled'
                                                : 'outlined'
                                        }
                                        clickable
                                    />
                                ))}
                            </Box>
                            <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                                {t('hook.response.methodsHint')}
                            </Typography>
                        </CardContent>
                    </Card>

                    {/* 响应头配置 */}
                    <Card sx={{mb: 3}}>
                        <CardContent>
                            <Typography variant="h6" sx={{mb: 2}}>
                                {t('hook.response.customHeaders')}
                            </Typography>

                            {/* 添加新响应头 */}
                            <Box sx={{display: 'flex', gap: 2, mb: 2}}>
                                <TextField
                                    size="small"
                                    label={t('hook.response.headerName')}
                                    value={newHeader.name}
                                    onChange={(e) =>
                                        setNewHeader((prev) => ({...prev, name: e.target.value}))
                                    }
                                    placeholder={t('hook.response.headerNamePlaceholder')}
                                />
                                <TextField
                                    size="small"
                                    label={t('hook.response.headerValue')}
                                    value={newHeader.value}
                                    onChange={(e) =>
                                        setNewHeader((prev) => ({...prev, value: e.target.value}))
                                    }
                                    placeholder={t('hook.response.headerValuePlaceholder')}
                                />
                                <Button
                                    variant="outlined"
                                    startIcon={<AddIcon />}
                                    onClick={addResponseHeader}
                                    disabled={!newHeader.name.trim() || !newHeader.value.trim()}>
                                    {t('common.add')}
                                </Button>
                            </Box>

                            {/* 响应头列表 */}
                            {Object.keys(formData['response-headers']).length === 0 ? (
                                <Typography color="textSecondary" sx={{textAlign: 'center', py: 2}}>
                                    {t('hook.response.noCustomHeaders')}
                                </Typography>
                            ) : (
                                <TableContainer component={Paper} variant="outlined">
                                    <Table size="small">
                                        <TableHead>
                                            <TableRow>
                                                <TableCell>
                                                    {t('hook.response.headerName')}
                                                </TableCell>
                                                <TableCell>
                                                    {t('hook.response.headerValue')}
                                                </TableCell>
                                                <TableCell width={100}>
                                                    {t('common.actions')}
                                                </TableCell>
                                            </TableRow>
                                        </TableHead>
                                        <TableBody>
                                            {Object.entries(formData['response-headers']).map(
                                                ([name, value]) => (
                                                    <TableRow key={name}>
                                                        <TableCell>{name}</TableCell>
                                                        <TableCell>{value}</TableCell>
                                                        <TableCell>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    removeResponseHeader(name)
                                                                }
                                                                color="error">
                                                                <DeleteIcon />
                                                            </IconButton>
                                                        </TableCell>
                                                    </TableRow>
                                                )
                                            )}
                                        </TableBody>
                                    </Table>
                                </TableContainer>
                            )}
                        </CardContent>
                    </Card>

                    {/* 输出控制 */}
                    <Card>
                        <CardContent>
                            <Typography variant="h6" sx={{mb: 2}}>
                                {t('hook.response.outputControl')}
                            </Typography>

                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={formData['include-command-output-in-response']}
                                        onChange={(e) =>
                                            setFormData((prev) => ({
                                                ...prev,
                                                'include-command-output-in-response':
                                                    e.target.checked,
                                            }))
                                        }
                                    />
                                }
                                label={t('hook.response.includeOutput')}
                            />

                            <Typography variant="body2" color="textSecondary" sx={{ml: 4, mb: 2}}>
                                {t('hook.response.includeOutputHint')}
                            </Typography>

                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={
                                            formData['include-command-output-in-response-on-error']
                                        }
                                        onChange={(e) =>
                                            setFormData((prev) => ({
                                                ...prev,
                                                'include-command-output-in-response-on-error':
                                                    e.target.checked,
                                            }))
                                        }
                                    />
                                }
                                label={t('hook.response.includeErrorOutput')}
                            />

                            <Typography variant="body2" color="textSecondary" sx={{ml: 4}}>
                                {t('hook.response.includeErrorOutputHint')}
                            </Typography>
                        </CardContent>
                    </Card>
                </Box>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t('common.cancel')}</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    {t('hook.response.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
