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
    Tooltip,
    FormControl,
    Select,
    MenuItem,
} from '@mui/material';
import {Add as AddIcon, Delete as DeleteIcon, Help as HelpIcon} from '@mui/icons-material';
import {IHook, IParameter, IEnvironmentVariable} from '../types';
import useTranslation from '../i18n/useTranslation';

interface EditParametersDialogProps {
    open: boolean;
    onClose: () => void;
    hookId?: string;
    onSave: (
        hookId: string,
        parametersData: {
            'pass-arguments-to-command': IParameter[];
            'pass-environment-to-command': IEnvironmentVariable[];
            'parse-parameters-as-json': string[];
        }
    ) => void;
    onGetHookDetails: (hookId: string) => Promise<IHook>;
}

// 参数来源选项
const buildParameterSources = (
    t: (key: string, options?: Record<string, string | number>) => string
) => [
    {
        value: 'payload',
        label: t('hook.parameters.source.payload.label'),
        description: t('hook.parameters.source.payload.description'),
    },
    {
        value: 'header',
        label: t('hook.parameters.source.header.label'),
        description: t('hook.parameters.source.header.description'),
    },
    {
        value: 'query',
        label: t('hook.parameters.source.query.label'),
        description: t('hook.parameters.source.query.description'),
    },
    {
        value: 'string',
        label: t('hook.parameters.source.string.label'),
        description: t('hook.parameters.source.string.description'),
    },
];

export default function EditParametersDialog({
    open,
    onClose,
    hookId,
    onSave,
    onGetHookDetails,
}: EditParametersDialogProps) {
    const {t} = useTranslation();
    const parameterSources = buildParameterSources(t);
    const [formData, setFormData] = useState({
        'pass-arguments-to-command': [] as IParameter[],
        'pass-environment-to-command': [] as IEnvironmentVariable[],
        'parse-parameters-as-json': [] as string[],
    });
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        const loadHookData = async () => {
            if (hookId && open) {
                setLoading(true);
                try {
                    const hook = await onGetHookDetails(hookId);
                    setFormData({
                        'pass-arguments-to-command': hook['pass-arguments-to-command'] || [],
                        'pass-environment-to-command': hook['pass-environment-to-command'] || [],
                        'parse-parameters-as-json': hook['parse-parameters-as-json'] || [],
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

    // 参数管理函数
    const addParameter = () => {
        const newParam: IParameter = {
            source: 'payload',
            name: '',
        };
        setFormData((prev) => ({
            ...prev,
            'pass-arguments-to-command': [...prev['pass-arguments-to-command'], newParam],
        }));
    };

    const updateParameter = (index: number, field: keyof IParameter, value: string) => {
        setFormData((prev) => {
            const updatedParams = [...prev['pass-arguments-to-command']];
            updatedParams[index] = {...updatedParams[index], [field]: value};
            return {
                ...prev,
                'pass-arguments-to-command': updatedParams,
            };
        });
    };

    const removeParameter = (index: number) => {
        setFormData((prev) => ({
            ...prev,
            'pass-arguments-to-command': prev['pass-arguments-to-command'].filter(
                (_, i) => i !== index
            ),
        }));
    };

    // 环境变量管理函数
    const addEnvironmentVariable = () => {
        const newEnv: IEnvironmentVariable = {
            name: '',
            source: 'payload',
        };
        setFormData((prev) => ({
            ...prev,
            'pass-environment-to-command': [...prev['pass-environment-to-command'], newEnv],
        }));
    };

    const updateEnvironmentVariable = (
        index: number,
        field: keyof IEnvironmentVariable,
        value: string
    ) => {
        setFormData((prev) => {
            const updatedEnvs = [...prev['pass-environment-to-command']];
            updatedEnvs[index] = {...updatedEnvs[index], [field]: value};
            return {
                ...prev,
                'pass-environment-to-command': updatedEnvs,
            };
        });
    };

    const removeEnvironmentVariable = (index: number) => {
        setFormData((prev) => ({
            ...prev,
            'pass-environment-to-command': prev['pass-environment-to-command'].filter(
                (_, i) => i !== index
            ),
        }));
    };

    const handleSave = () => {
        if (!hookId) return;

        onSave(hookId, formData);
        onClose();
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
            <DialogTitle>{t('hook.editParametersTitle', {id: hookId || ''})}</DialogTitle>

            <DialogContent>
                <Box sx={{pt: 2}}>
                    <Alert severity="info" sx={{mb: 3}}>
                        <Typography variant="body2">{t('hook.parameters.description')}</Typography>
                    </Alert>

                    {/* 命令参数配置 */}
                    <Card sx={{mb: 3}}>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', mb: 2}}>
                                <Typography variant="h6" sx={{flexGrow: 1}}>
                                    {t('hook.parameters.commandArguments')}
                                </Typography>
                                <Tooltip title={t('hook.parameters.commandArgumentsHelp')}>
                                    <IconButton size="small">
                                        <HelpIcon />
                                    </IconButton>
                                </Tooltip>
                                <Button
                                    startIcon={<AddIcon />}
                                    onClick={addParameter}
                                    variant="outlined"
                                    size="small">
                                    {t('hook.parameters.addArgument')}
                                </Button>
                            </Box>

                            {formData['pass-arguments-to-command'].length === 0 ? (
                                <Typography color="textSecondary" sx={{textAlign: 'center', py: 2}}>
                                    {t('hook.parameters.noArguments')}
                                </Typography>
                            ) : (
                                <TableContainer component={Paper} variant="outlined">
                                    <Table size="small">
                                        <TableHead>
                                            <TableRow>
                                                <TableCell>
                                                    {t('hook.parameters.sourceHeader')}
                                                </TableCell>
                                                <TableCell>
                                                    {t('hook.parameters.nameHeader')}
                                                </TableCell>
                                                <TableCell width={100}>
                                                    {t('common.actions')}
                                                </TableCell>
                                            </TableRow>
                                        </TableHead>
                                        <TableBody>
                                            {formData['pass-arguments-to-command'].map(
                                                (param, index) => (
                                                    <TableRow key={index}>
                                                        <TableCell>
                                                            <FormControl fullWidth size="small">
                                                                <Select
                                                                    value={param.source}
                                                                    onChange={(e) =>
                                                                        updateParameter(
                                                                            index,
                                                                            'source',
                                                                            e.target.value
                                                                        )
                                                                    }>
                                                                    {parameterSources.map(
                                                                        (source) => (
                                                                            <MenuItem
                                                                                key={source.value}
                                                                                value={
                                                                                    source.value
                                                                                }>
                                                                                <Box>
                                                                                    <Typography variant="body2">
                                                                                        {
                                                                                            source.label
                                                                                        }
                                                                                    </Typography>
                                                                                    <Typography
                                                                                        variant="caption"
                                                                                        color="textSecondary">
                                                                                        {
                                                                                            source.description
                                                                                        }
                                                                                    </Typography>
                                                                                </Box>
                                                                            </MenuItem>
                                                                        )
                                                                    )}
                                                                </Select>
                                                            </FormControl>
                                                        </TableCell>
                                                        <TableCell>
                                                            <TextField
                                                                fullWidth
                                                                size="small"
                                                                value={param.name}
                                                                onChange={(e) =>
                                                                    updateParameter(
                                                                        index,
                                                                        'name',
                                                                        e.target.value
                                                                    )
                                                                }
                                                                placeholder={
                                                                    param.source === 'payload'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.payload'
                                                                          )
                                                                        : param.source === 'header'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.header'
                                                                          )
                                                                        : param.source === 'query'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.query'
                                                                          )
                                                                        : t(
                                                                              'hook.parameters.placeholders.string'
                                                                          )
                                                                }
                                                            />
                                                        </TableCell>
                                                        <TableCell>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    removeParameter(index)
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

                    {/* 环境变量配置 */}
                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', mb: 2}}>
                                <Typography variant="h6" sx={{flexGrow: 1}}>
                                    {t('hook.parameters.environmentVariables')}
                                </Typography>
                                <Tooltip title={t('hook.parameters.environmentVariablesHelp')}>
                                    <IconButton size="small">
                                        <HelpIcon />
                                    </IconButton>
                                </Tooltip>
                                <Button
                                    startIcon={<AddIcon />}
                                    onClick={addEnvironmentVariable}
                                    variant="outlined"
                                    size="small">
                                    {t('hook.parameters.addEnvironmentVariable')}
                                </Button>
                            </Box>

                            {formData['pass-environment-to-command'].length === 0 ? (
                                <Typography color="textSecondary" sx={{textAlign: 'center', py: 2}}>
                                    {t('hook.parameters.noEnvironmentVariables')}
                                </Typography>
                            ) : (
                                <TableContainer component={Paper} variant="outlined">
                                    <Table size="small">
                                        <TableHead>
                                            <TableRow>
                                                <TableCell>
                                                    {t('hook.parameters.envNameHeader')}
                                                </TableCell>
                                                <TableCell>
                                                    {t('hook.parameters.envSourceHeader')}
                                                </TableCell>
                                                <TableCell>
                                                    {t('hook.parameters.envPathHeader')}
                                                </TableCell>
                                                <TableCell width={100}>
                                                    {t('common.actions')}
                                                </TableCell>
                                            </TableRow>
                                        </TableHead>
                                        <TableBody>
                                            {formData['pass-environment-to-command'].map(
                                                (env, index) => (
                                                    <TableRow key={index}>
                                                        <TableCell>
                                                            <TextField
                                                                fullWidth
                                                                size="small"
                                                                value={env.name}
                                                                onChange={(e) =>
                                                                    updateEnvironmentVariable(
                                                                        index,
                                                                        'name',
                                                                        e.target.value
                                                                    )
                                                                }
                                                                placeholder={t(
                                                                    'hook.parameters.placeholders.envName'
                                                                )}
                                                            />
                                                        </TableCell>
                                                        <TableCell>
                                                            <FormControl fullWidth size="small">
                                                                <Select
                                                                    value={env.source}
                                                                    onChange={(e) =>
                                                                        updateEnvironmentVariable(
                                                                            index,
                                                                            'source',
                                                                            e.target.value
                                                                        )
                                                                    }>
                                                                    {parameterSources.map(
                                                                        (source) => (
                                                                            <MenuItem
                                                                                key={source.value}
                                                                                value={
                                                                                    source.value
                                                                                }>
                                                                                <Box>
                                                                                    <Typography variant="body2">
                                                                                        {
                                                                                            source.label
                                                                                        }
                                                                                    </Typography>
                                                                                    <Typography
                                                                                        variant="caption"
                                                                                        color="textSecondary">
                                                                                        {
                                                                                            source.description
                                                                                        }
                                                                                    </Typography>
                                                                                </Box>
                                                                            </MenuItem>
                                                                        )
                                                                    )}
                                                                </Select>
                                                            </FormControl>
                                                        </TableCell>
                                                        <TableCell>
                                                            <TextField
                                                                fullWidth
                                                                size="small"
                                                                value={env.name}
                                                                onChange={(e) =>
                                                                    updateEnvironmentVariable(
                                                                        index,
                                                                        'name',
                                                                        e.target.value
                                                                    )
                                                                }
                                                                placeholder={
                                                                    env.source === 'payload'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.envPayload'
                                                                          )
                                                                        : env.source === 'header'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.envHeader'
                                                                          )
                                                                        : env.source === 'query'
                                                                        ? t(
                                                                              'hook.parameters.placeholders.envQuery'
                                                                          )
                                                                        : t(
                                                                              'hook.parameters.placeholders.string'
                                                                          )
                                                                }
                                                            />
                                                        </TableCell>
                                                        <TableCell>
                                                            <IconButton
                                                                size="small"
                                                                onClick={() =>
                                                                    removeEnvironmentVariable(index)
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
                </Box>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t('common.cancel')}</Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    {t('hook.parameters.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
