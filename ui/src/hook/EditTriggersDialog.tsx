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
    FormControl,
    Select,
    MenuItem,
    FormControlLabel,
    Switch,
    Chip,
    Divider,
    IconButton,
    Tooltip,
    Paper,
    List,
    ListItem,
    ListItemText,
    ListItemSecondaryAction,
    Accordion,
    AccordionSummary,
    AccordionDetails,
    ToggleButton,
    ToggleButtonGroup,
    Tab,
    Tabs,
    InputLabel,
    FormHelperText,
} from '@mui/material';
import {
    Add as AddIcon,
    Delete as DeleteIcon,
    Edit as EditIcon,
    ExpandMore as ExpandMoreIcon,
    Code as CodeIcon,
    AccountTree as TreeIcon,
    Help as HelpIcon,
    Warning as WarningIcon,
    Info as InfoIcon,
} from '@mui/icons-material';
import Grid from '@mui/material/Grid';
import {IHook, ITriggerRule, IMatchRule, IParameter} from '../types';
import {useTheme} from '@mui/material/styles';

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

interface EditTriggersDialogProps {
    open: boolean;
    onClose: () => void;
    hookId?: string;
    onSave: (
        hookId: string,
        triggersData: {
            'trigger-rule': any;
            'trigger-rule-mismatch-http-response-code': number;
        }
    ) => void;
    onGetHookDetails: (hookId: string) => Promise<IHook>;
}

// 匹配类型定义
const MATCH_TYPES = [
    {value: 'value', label: '完全匹配', description: '参数值必须完全等于指定值'},
    {value: 'regex', label: '正则表达式', description: '参数值匹配正则表达式'},
    {value: 'payload-hmac-sha1', label: 'SHA1签名验证', description: '验证SHA1 HMAC签名'},
    {value: 'payload-hmac-sha256', label: 'SHA256签名验证', description: '验证SHA256 HMAC签名'},
    {value: 'payload-hmac-sha512', label: 'SHA512签名验证', description: '验证SHA512 HMAC签名'},
    {value: 'ip-whitelist', label: 'IP白名单', description: '检查请求IP是否在允许范围内'},
    {value: 'scalr-signature', label: 'Scalr签名', description: '验证Scalr平台签名'},
];

// 参数来源定义
const PARAMETER_SOURCES = [
    {value: 'payload', label: 'Payload数据', description: '从请求体中获取参数'},
    {value: 'header', label: 'HTTP头部', description: '从HTTP请求头中获取参数'},
    {value: 'query', label: '查询参数', description: '从URL查询字符串中获取参数'},
];

// 逻辑操作符定义
const LOGIC_OPERATORS = [
    {value: 'and', label: 'AND (且)', description: '所有子规则都必须满足'},
    {value: 'or', label: 'OR (或)', description: '任意一个子规则满足即可'},
    {value: 'not', label: 'NOT (非)', description: '子规则不满足时匹配'},
];

interface SimpleRule {
    id: string;
    type: string;
    parameter: IParameter;
    value?: string;
    regex?: string;
    secret?: string;
    'ip-range'?: string;
}

interface RuleGroup {
    id: string;
    operator: 'and' | 'or' | 'not';
    rules: (SimpleRule | RuleGroup)[];
}

export default function EditTriggersDialog({
    open,
    onClose,
    hookId,
    onSave,
    onGetHookDetails,
}: EditTriggersDialogProps) {
    const theme = useTheme();
    const [formData, setFormData] = useState({
        'trigger-rule': null as any,
        'trigger-rule-mismatch-http-response-code': 400,
    });
    const [loading, setLoading] = useState(false);

    // 简单模式状态
    const [ruleBuilder, setRuleBuilder] = useState<RuleGroup>({
        id: 'root',
        operator: 'and',
        rules: [],
    });

    useEffect(() => {
        const loadHookData = async () => {
            if (hookId && open) {
                setLoading(true);
                try {
                    const hook = await onGetHookDetails(hookId);
                    const triggerRule = hook['trigger-rule'];

                    setFormData({
                        'trigger-rule': triggerRule || null,
                        'trigger-rule-mismatch-http-response-code':
                            hook['trigger-rule-mismatch-http-response-code'] || 400,
                    });

                    // 解析规则到编辑器
                    if (triggerRule) {
                        parseToSimpleRules(triggerRule);
                    } else {
                        setRuleBuilder({id: 'root', operator: 'and', rules: []});
                    }
                } catch (error) {
                    console.error('加载Hook数据失败:', error);
                    setRuleBuilder({id: 'root', operator: 'and', rules: []});
                } finally {
                    setLoading(false);
                }
            }
        };

        loadHookData();
    }, [hookId, open, onGetHookDetails]);

    // 解析触发规则到简单规则构建器
    const parseToSimpleRules = (rule: any) => {
        if (!rule) {
            setRuleBuilder({id: 'root', operator: 'and', rules: []});
            return;
        }

        try {
            const parsedGroup = parseRule(rule, 'root');
            setRuleBuilder(parsedGroup);
        } catch (error) {
            console.error('解析触发规则失败:', error);
            setRuleBuilder({id: 'root', operator: 'and', rules: []});
        }
    };

    // 递归解析规则
    const parseRule = (rule: any, id: string): RuleGroup => {
        // 处理match规则
        if (rule.match) {
            const simpleRule: SimpleRule = {
                id: `rule_${Date.now()}_${Math.random()}`,
                type: rule.match.type || 'value',
                parameter: rule.match.parameter || {source: 'payload', name: ''},
            };

            // 根据类型设置相应的值
            switch (rule.match.type) {
                case 'value':
                    simpleRule.value = rule.match.value || '';
                    break;
                case 'regex':
                    simpleRule.regex = rule.match.regex || '';
                    break;
                case 'payload-hmac-sha1':
                case 'payload-hmac-sha256':
                case 'payload-hmac-sha512':
                case 'scalr-signature':
                    simpleRule.secret = rule.match.secret || '';
                    break;
                case 'ip-whitelist':
                    simpleRule['ip-range'] = rule.match['ip-range'] || '';
                    // IP白名单不需要parameter
                    simpleRule.parameter = {source: 'payload', name: ''};
                    break;
            }

            return {
                id,
                operator: 'and',
                rules: [simpleRule],
            };
        }

        // 处理逻辑操作符规则
        if (rule.and) {
            const rules = rule.and.map((childRule: any, index: number) =>
                parseRuleItem(childRule, `${id}_and_${index}`)
            );
            return {
                id,
                operator: 'and',
                rules,
            };
        }

        if (rule.or) {
            const rules = rule.or.map((childRule: any, index: number) =>
                parseRuleItem(childRule, `${id}_or_${index}`)
            );
            return {
                id,
                operator: 'or',
                rules,
            };
        }

        if (rule.not) {
            const childRule = parseRuleItem(rule.not, `${id}_not_0`);
            return {
                id,
                operator: 'not',
                rules: [childRule],
            };
        }

        // 默认返回空的and组
        return {
            id,
            operator: 'and',
            rules: [],
        };
    };

    // 解析单个规则项（可能是match规则或者嵌套的逻辑组）
    const parseRuleItem = (ruleItem: any, id: string): SimpleRule | RuleGroup => {
        if (ruleItem.match) {
            const simpleRule: SimpleRule = {
                id: `rule_${Date.now()}_${Math.random()}`,
                type: ruleItem.match.type || 'value',
                parameter: ruleItem.match.parameter || {source: 'payload', name: ''},
            };

            // 根据类型设置相应的值
            switch (ruleItem.match.type) {
                case 'value':
                    simpleRule.value = ruleItem.match.value || '';
                    break;
                case 'regex':
                    simpleRule.regex = ruleItem.match.regex || '';
                    break;
                case 'payload-hmac-sha1':
                case 'payload-hmac-sha256':
                case 'payload-hmac-sha512':
                case 'scalr-signature':
                    simpleRule.secret = ruleItem.match.secret || '';
                    break;
                case 'ip-whitelist':
                    simpleRule['ip-range'] = ruleItem.match['ip-range'] || '';
                    // IP白名单不需要parameter
                    simpleRule.parameter = {source: 'payload', name: ''};
                    break;
            }

            return simpleRule;
        }

        // 如果是嵌套的逻辑组，递归解析
        return parseRule(ruleItem, id);
    };

    // 从简单规则构建器生成触发规则
    const buildTriggerRule = (group: RuleGroup): any => {
        if (group.rules.length === 0) {
            return null;
        }

        // 如果只有一个规则
        if (group.rules.length === 1) {
            const rule = group.rules[0];
            if ('operator' in rule) {
                // 如果是规则组，递归构建
                const nestedRule = buildTriggerRule(rule);
                // 如果根级操作符是'and'且只有一个子规则组，可以直接返回子规则组
                if (group.id === 'root' && group.operator === 'and') {
                    return nestedRule;
                }
                // 否则保持当前组的操作符
                if (group.operator === 'not') {
                    return {not: nestedRule};
                }
                return {[group.operator]: [nestedRule]};
            } else {
                // 如果是简单规则
                const matchRule = buildMatchRule(rule);
                // 如果根级操作符是'and'且只有一个简单规则，直接返回match规则
                if (group.id === 'root' && group.operator === 'and') {
                    return matchRule;
                }
                // 否则包装在对应的操作符中
                if (group.operator === 'not') {
                    return {not: matchRule};
                }
                return {[group.operator]: [matchRule]};
            }
        }

        // 多个规则的情况
        const builtRules = group.rules
            .map((rule) => {
                if ('operator' in rule) {
                    // 嵌套的规则组
                    return buildTriggerRule(rule);
                } else {
                    // 简单规则，直接返回match对象
                    return buildMatchRule(rule);
                }
            })
            .filter(Boolean);

        // NOT操作符只能有一个子规则
        if (group.operator === 'not' && builtRules.length > 0) {
            return {not: builtRules[0]};
        }

        return {[group.operator]: builtRules};
    };

    // 构建匹配规则
    const buildMatchRule = (rule: SimpleRule): any => {
        const matchRule: any = {
            type: rule.type,
        };

        // 添加参数（IP白名单除外）
        if (rule.type !== 'ip-whitelist') {
            matchRule.parameter = rule.parameter;
        }

        // 根据类型添加特定字段
        switch (rule.type) {
            case 'value':
                matchRule.value = rule.value || '';
                break;
            case 'regex':
                matchRule.regex = rule.regex || '';
                break;
            case 'payload-hmac-sha1':
            case 'payload-hmac-sha256':
            case 'payload-hmac-sha512':
            case 'scalr-signature':
                matchRule.secret = rule.secret || '';
                break;
            case 'ip-whitelist':
                matchRule['ip-range'] = rule['ip-range'] || '';
                break;
        }

        return {match: matchRule};
    };

    // 添加简单规则
    const addSimpleRule = (targetGroupId?: string) => {
        const newRule: SimpleRule = {
            id: `rule_${Date.now()}`,
            type: 'value',
            parameter: {source: 'payload', name: ''},
            value: '',
        };

        if (!targetGroupId) {
            // 如果没有指定目标组，添加到根级别
            setRuleBuilder((prev) => ({
                ...prev,
                rules: [...prev.rules, newRule],
            }));
        } else {
            // 添加到指定的规则组
            const addToGroup = (group: RuleGroup): RuleGroup => {
                if (group.id === targetGroupId) {
                    return {
                        ...group,
                        rules: [...group.rules, newRule],
                    };
                }
                return {
                    ...group,
                    rules: group.rules.map((rule) =>
                        'operator' in rule ? addToGroup(rule) : rule
                    ),
                };
            };

            setRuleBuilder((prev) => addToGroup(prev));
        }
    };

    // 添加规则组
    const addRuleGroup = (targetGroupId?: string) => {
        const newGroup: RuleGroup = {
            id: `group_${Date.now()}`,
            operator: 'and',
            rules: [],
        };

        if (!targetGroupId) {
            // 如果没有指定目标组，添加到根级别
            setRuleBuilder((prev) => ({
                ...prev,
                rules: [...prev.rules, newGroup],
            }));
        } else {
            // 添加到指定的规则组
            const addToGroup = (group: RuleGroup): RuleGroup => {
                if (group.id === targetGroupId) {
                    return {
                        ...group,
                        rules: [...group.rules, newGroup],
                    };
                }
                return {
                    ...group,
                    rules: group.rules.map((rule) =>
                        'operator' in rule ? addToGroup(rule) : rule
                    ),
                };
            };

            setRuleBuilder((prev) => addToGroup(prev));
        }
    };

    // 删除规则
    const deleteRule = (ruleId: string) => {
        const deleteFromGroup = (group: RuleGroup): RuleGroup => ({
            ...group,
            rules: group.rules
                .filter((rule) => rule.id !== ruleId)
                .map((rule) => ('operator' in rule ? deleteFromGroup(rule) : rule)),
        });

        setRuleBuilder((prev) => deleteFromGroup(prev));
    };

    // 更新规则
    const updateRule = (ruleId: string, updates: Partial<SimpleRule | RuleGroup>) => {
        const updateInGroup = (group: RuleGroup): RuleGroup => ({
            ...group,
            rules: group.rules.map((rule) => {
                if (rule.id === ruleId) {
                    return {...rule, ...updates};
                } else if ('operator' in rule) {
                    return updateInGroup(rule);
                }
                return rule;
            }),
        });

        setRuleBuilder((prev) => updateInGroup(prev));
    };

    // 保存触发规则
    const handleSave = () => {
        if (!hookId) return;

        const triggerRule = buildTriggerRule(ruleBuilder);

        const saveData = {
            'trigger-rule': triggerRule,
            'trigger-rule-mismatch-http-response-code':
                formData['trigger-rule-mismatch-http-response-code'],
        };

        onSave(hookId, saveData);
        onClose();
    };

    // 渲染简单规则
    const renderSimpleRule = (rule: SimpleRule, depth: number = 0) => (
        <Card
            key={rule.id}
            sx={{
                ml: depth * 2,
                mb: 1,
                backgroundColor:
                    depth > 0
                        ? theme.custom.colors.primary.darkGray
                        : theme.custom.colors.background.overlay,
                border: `1px solid ${theme.custom.colors.border.light}`,
            }}>
            <CardContent sx={{py: 2}}>
                <Grid container spacing={2} alignItems="center">
                    <Grid size={4}>
                        <FormControl fullWidth size="small">
                            <Select
                                value={rule.type}
                                onChange={(e) => updateRule(rule.id, {type: e.target.value})}
                                sx={{
                                    backgroundColor: theme.custom.colors.primary.darkGray,
                                    color: theme.custom.colors.text.onDark,
                                    '& .MuiSelect-select': {
                                        color: theme.custom.colors.text.onDark,
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiSvgIcon-root': {
                                        color: theme.custom.colors.text.onDark,
                                    },
                                }}>
                                {MATCH_TYPES.map((type) => (
                                    <MenuItem key={type.value} value={type.value}>
                                        <Box>
                                            <Typography variant="body2">{type.label}</Typography>
                                            <Typography variant="caption" color="textSecondary">
                                                {type.description}
                                            </Typography>
                                        </Box>
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                    </Grid>

                    {rule.type !== 'ip-whitelist' && (
                        <>
                            <Grid size={2}>
                                <FormControl fullWidth size="small">
                                    <Select
                                        value={rule.parameter.source}
                                        onChange={(e) =>
                                            updateRule(rule.id, {
                                                parameter: {
                                                    ...rule.parameter,
                                                    source: e.target.value as any,
                                                },
                                            })
                                        }
                                        sx={{
                                            backgroundColor: theme.custom.colors.primary.darkGray,
                                            color: theme.custom.colors.text.onDark,
                                            '& .MuiSelect-select': {
                                                color: theme.custom.colors.text.onDark,
                                                backgroundColor:
                                                    theme.custom.colors.primary.darkGray,
                                            },
                                            '& .MuiOutlinedInput-notchedOutline': {
                                                borderColor: theme.custom.colors.border.medium,
                                            },
                                            '& .MuiSvgIcon-root': {
                                                color: theme.custom.colors.text.onDark,
                                            },
                                        }}>
                                        {PARAMETER_SOURCES.map((source) => (
                                            <MenuItem key={source.value} value={source.value}>
                                                {source.label}
                                            </MenuItem>
                                        ))}
                                    </Select>
                                </FormControl>
                            </Grid>

                            <Grid size={3}>
                                <TextField
                                    fullWidth
                                    size="small"
                                    placeholder="参数名"
                                    value={rule.parameter.name}
                                    onChange={(e) =>
                                        updateRule(rule.id, {
                                            parameter: {...rule.parameter, name: e.target.value},
                                        })
                                    }
                                    sx={{
                                        '& .MuiInputBase-root': {
                                            backgroundColor: theme.custom.colors.primary.darkGray,
                                            color: theme.custom.colors.text.onDark,
                                        },
                                        '& .MuiOutlinedInput-notchedOutline': {
                                            borderColor: theme.custom.colors.border.medium,
                                        },
                                        '& .MuiInputBase-input::placeholder': {
                                            color: theme.custom.colors.text.onDarkSecondary,
                                            opacity: 1,
                                        },
                                    }}
                                />
                            </Grid>
                        </>
                    )}

                    <Grid size={rule.type === 'ip-whitelist' ? 7 : 2}>
                        {rule.type === 'value' && (
                            <TextField
                                fullWidth
                                size="small"
                                placeholder="匹配值"
                                value={rule.value || ''}
                                onChange={(e) => updateRule(rule.id, {value: e.target.value})}
                                sx={{
                                    '& .MuiInputBase-root': {
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiInputBase-input::placeholder': {
                                        color: theme.custom.colors.text.onDarkSecondary,
                                        opacity: 1,
                                    },
                                }}
                            />
                        )}
                        {rule.type === 'regex' && (
                            <TextField
                                fullWidth
                                size="small"
                                placeholder="正则表达式"
                                value={rule.regex || ''}
                                onChange={(e) => updateRule(rule.id, {regex: e.target.value})}
                                sx={{
                                    '& .MuiInputBase-root': {
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiInputBase-input::placeholder': {
                                        color: theme.custom.colors.text.onDarkSecondary,
                                        opacity: 1,
                                    },
                                }}
                            />
                        )}
                        {(rule.type.includes('hmac') || rule.type === 'scalr-signature') && (
                            <TextField
                                fullWidth
                                size="small"
                                type="password"
                                placeholder="密钥"
                                value={rule.secret || ''}
                                onChange={(e) => updateRule(rule.id, {secret: e.target.value})}
                                sx={{
                                    '& .MuiInputBase-root': {
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiInputBase-input::placeholder': {
                                        color: theme.custom.colors.text.onDarkSecondary,
                                        opacity: 1,
                                    },
                                }}
                            />
                        )}
                        {rule.type === 'ip-whitelist' && (
                            <TextField
                                fullWidth
                                size="small"
                                placeholder="IP范围 (CIDR)"
                                value={rule['ip-range'] || ''}
                                onChange={(e) => updateRule(rule.id, {'ip-range': e.target.value})}
                                sx={{
                                    '& .MuiInputBase-root': {
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiInputBase-input::placeholder': {
                                        color: theme.custom.colors.text.onDarkSecondary,
                                        opacity: 1,
                                    },
                                }}
                            />
                        )}
                    </Grid>

                    <Grid size={1}>
                        <IconButton size="small" color="error" onClick={() => deleteRule(rule.id)}>
                            <DeleteIcon />
                        </IconButton>
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
    );

    // 渲染规则组
    const renderRuleGroup = (group: RuleGroup, depth: number = 0) => (
        <Card
            key={group.id}
            sx={{
                ml: depth * 2,
                mb: 2,
                backgroundColor:
                    depth > 0
                        ? theme.custom.colors.primary.darkGray
                        : theme.custom.colors.background.overlay,
                border: `2px solid ${theme.custom.colors.border.contrast}`,
            }}>
            <CardContent>
                <Box sx={{display: 'flex', alignItems: 'center', mb: 2}}>
                    <FormControl sx={{minWidth: 120, mr: 2}}>
                        <Select
                            size="small"
                            value={group.operator}
                            onChange={(e) =>
                                updateRule(group.id, {operator: e.target.value as any})
                            }
                            sx={{
                                backgroundColor: theme.custom.colors.primary.darkGray,
                                color: theme.custom.colors.text.onDark,
                                '& .MuiSelect-select': {
                                    color: theme.custom.colors.text.onDark,
                                    backgroundColor: theme.custom.colors.primary.darkGray,
                                },
                                '& .MuiOutlinedInput-notchedOutline': {
                                    borderColor: theme.custom.colors.border.medium,
                                },
                                '& .MuiSvgIcon-root': {
                                    color: theme.custom.colors.text.onDark,
                                },
                            }}>
                            {LOGIC_OPERATORS.map((op) => (
                                <MenuItem key={op.value} value={op.value}>
                                    <Box>
                                        <Typography variant="body2" fontWeight="bold">
                                            {op.label}
                                        </Typography>
                                        <Typography variant="caption" color="textSecondary">
                                            {op.description}
                                        </Typography>
                                    </Box>
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>

                    <Chip
                        label={`${group.rules.length} 个规则`}
                        size="small"
                        color={group.rules.length > 0 ? 'primary' : 'default'}
                    />

                    <Box sx={{ml: 'auto'}}>
                        <IconButton size="small" color="error" onClick={() => deleteRule(group.id)}>
                            <DeleteIcon />
                        </IconButton>
                    </Box>
                </Box>

                {group.rules.map((rule) =>
                    'operator' in rule
                        ? renderRuleGroup(rule, depth + 1)
                        : renderSimpleRule(rule, depth + 1)
                )}

                <Box sx={{mt: 2, display: 'flex', gap: 1}}>
                    <Button
                        size="small"
                        startIcon={<AddIcon />}
                        onClick={() => addSimpleRule(group.id)}>
                        添加规则
                    </Button>
                    <Button
                        size="small"
                        startIcon={<TreeIcon />}
                        onClick={() => addRuleGroup(group.id)}>
                        添加规则组
                    </Button>
                </Box>
            </CardContent>
        </Card>
    );

    return (
        <Dialog
            open={open}
            onClose={onClose}
            maxWidth="lg"
            fullWidth
            PaperProps={{
                sx: {
                    height: ruleBuilder.rules.length === 0 ? '50vh' : '70vh',
                    backgroundColor: theme.custom.colors.primary.darkGray,
                    color: theme.custom.colors.text.onDark,
                },
            }}>
            <DialogTitle>
                <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                    <Typography variant="h6" sx={{color: theme.custom.colors.text.onDark}}>
                        编辑触发规则 - {hookId}
                    </Typography>
                    <Tooltip title="查看触发规则文档">
                        <IconButton
                            size="small"
                            onClick={() => window.open('/docs/Hook-Rules.md', '_blank')}
                            sx={{color: theme.custom.colors.text.onDark}}>
                            <HelpIcon />
                        </IconButton>
                    </Tooltip>
                </Box>
            </DialogTitle>

            <DialogContent
                sx={{
                    p: 0,
                    backgroundColor: theme.custom.colors.primary.darkGray,
                    height:
                        ruleBuilder.rules.length === 0
                            ? 'calc(50vh - 120px)'
                            : 'calc(70vh - 120px)',
                    overflow: 'auto',
                    // 美化滚动条样式
                    '&::-webkit-scrollbar': {
                        width: '8px',
                    },
                    '&::-webkit-scrollbar-track': {
                        backgroundColor: theme.custom.colors.primary.darkGray,
                        borderRadius: '4px',
                    },
                    '&::-webkit-scrollbar-thumb': {
                        backgroundColor: theme.custom.colors.border.medium,
                        borderRadius: '4px',
                        '&:hover': {
                            backgroundColor: theme.custom.colors.border.dark,
                        },
                    },
                    // Firefox滚动条样式
                    scrollbarWidth: 'thin',
                    scrollbarColor: `${theme.custom.colors.border.medium} ${theme.custom.colors.primary.darkGray}`,
                }}>
                {loading && (
                    <Alert
                        severity="info"
                        sx={{m: 3, backgroundColor: theme.custom.colors.status.info.background}}>
                        <Typography
                            variant="body2"
                            sx={{color: theme.custom.colors.status.info.text}}>
                            正在加载Hook配置数据...
                        </Typography>
                    </Alert>
                )}

                <Box sx={{px: 3, pt: 2, pb: 2}}>
                    <Alert
                        severity="info"
                        sx={{mb: 3, backgroundColor: theme.custom.colors.status.info.background}}>
                        <Typography
                            variant="body2"
                            sx={{color: theme.custom.colors.status.info.text}}>
                            使用图形界面构建触发规则。支持多种匹配类型和逻辑组合（AND/OR/NOT）。
                            如果不添加任何规则，则所有请求都会执行命令。
                        </Typography>
                    </Alert>

                    <Box sx={{mb: 3}}>
                        {ruleBuilder.rules.length === 0 ? (
                            <Card
                                sx={{
                                    textAlign: 'center',
                                    py: 4,
                                    backgroundColor: theme.custom.colors.background.overlay,
                                    border: `1px solid ${theme.custom.colors.border.medium}`,
                                }}>
                                <Typography
                                    variant="h6"
                                    sx={{color: theme.custom.colors.text.secondary}}
                                    gutterBottom>
                                    还没有添加任何规则
                                </Typography>
                                <Typography
                                    variant="body2"
                                    sx={{color: theme.custom.colors.text.secondary, mb: 3}}>
                                    点击下面的按钮开始添加触发规则
                                </Typography>
                                <Box sx={{display: 'flex', gap: 2, justifyContent: 'center'}}>
                                    <Button
                                        variant="contained"
                                        startIcon={<AddIcon />}
                                        onClick={() => addSimpleRule()}>
                                        添加匹配规则
                                    </Button>
                                    <Button
                                        variant="outlined"
                                        startIcon={<TreeIcon />}
                                        onClick={() => addRuleGroup()}>
                                        添加规则组
                                    </Button>
                                </Box>
                            </Card>
                        ) : (
                            <>{renderRuleGroup(ruleBuilder)}</>
                        )}
                    </Box>

                    <Card sx={{backgroundColor: theme.custom.colors.background.overlay}}>
                        <CardContent>
                            <TextField
                                fullWidth
                                type="number"
                                label="触发规则不匹配时的HTTP响应码"
                                value={formData['trigger-rule-mismatch-http-response-code']}
                                onChange={(e) =>
                                    setFormData((prev) => ({
                                        ...prev,
                                        'trigger-rule-mismatch-http-response-code': parseInt(
                                            e.target.value
                                        ),
                                    }))
                                }
                                helperText="当请求不满足触发规则时返回的HTTP状态码（建议使用400-499范围）"
                                inputProps={{min: 200, max: 599}}
                                size="small"
                                sx={{
                                    '& .MuiInputBase-root': {
                                        backgroundColor: theme.custom.colors.primary.darkGray,
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiOutlinedInput-notchedOutline': {
                                        borderColor: theme.custom.colors.border.medium,
                                    },
                                    '& .MuiInputLabel-root': {
                                        color: theme.custom.colors.text.onDark,
                                    },
                                    '& .MuiFormHelperText-root': {
                                        color: theme.custom.colors.text.onDarkSecondary,
                                    },
                                }}
                            />
                        </CardContent>
                    </Card>
                </Box>
            </DialogContent>

            <DialogActions
                sx={{px: 3, py: 2, backgroundColor: theme.custom.colors.primary.darkGray}}>
                <Button onClick={onClose} sx={{color: theme.custom.colors.text.onDark}}>
                    取消
                </Button>
                <Button onClick={handleSave} variant="contained" color="primary">
                    保存触发规则
                </Button>
            </DialogActions>
        </Dialog>
    );
}
