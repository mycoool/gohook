import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Typography,
  Tabs,
  Tab,
  Card,
  CardContent,
  IconButton,
  Tooltip,
  Alert,
  Chip,
  Switch,
  FormControlLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Divider,
  FormHelperText,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  Info as InfoIcon,
  Help as HelpIcon,
} from '@mui/icons-material';
import { IHook, IParameter, IEnvironmentVariable, ITriggerRule } from '../types';

interface HookConfigDialogProps {
  open: boolean;
  onClose: () => void;
  hook?: IHook;
  onSave: (hook: IHook) => void;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`hook-tabpanel-${index}`}
      aria-labelledby={`hook-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ py: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

function a11yProps(index: number) {
  return {
    id: `hook-tab-${index}`,
    'aria-controls': `hook-tabpanel-${index}`,
  };
}

// 匹配类型选项
const MATCH_TYPES = [
  { value: 'value', label: '值匹配', description: '精确匹配指定的值' },
  { value: 'regex', label: '正则表达式', description: '使用正则表达式匹配' },
  { value: 'payload-hmac-sha1', label: 'HMAC-SHA1签名', description: 'GitHub webhook签名验证' },
  { value: 'payload-hmac-sha256', label: 'HMAC-SHA256签名', description: 'GitHub/GitLab webhook签名验证' },
  { value: 'payload-hmac-sha512', label: 'HMAC-SHA512签名', description: '高强度签名验证' },
  { value: 'ip-whitelist', label: 'IP白名单', description: '限制访问的IP地址范围' },
  { value: 'scalr-signature', label: 'Scalr签名', description: 'Scalr平台签名验证' },
];

// 参数来源选项
const PARAMETER_SOURCES = [
  { value: 'payload', label: 'Payload', description: '从请求体JSON中获取' },
  { value: 'header', label: 'Header', description: '从HTTP请求头中获取' },
  { value: 'query', label: 'Query', description: '从URL查询参数中获取' },
  { value: 'string', label: 'String', description: '固定字符串值' },
];

export default function HookConfigDialog({ open, onClose, hook, onSave }: HookConfigDialogProps) {
  const [tabValue, setTabValue] = useState(0);
  const [formData, setFormData] = useState<IHook>({
    id: '',
    'execute-command': '',
    'command-working-directory': '',
    'response-message': '执行成功',
    'http-methods': ['POST'],
    'response-headers': {},
    'pass-arguments-to-command': [],
    'pass-environment-to-command': [],
    'trigger-rule': {},
    'include-command-output-in-response': false,
    'include-command-output-in-response-on-error': false,
    'parse-parameters-as-json': [],
    'trigger-rule-mismatch-http-response-code': 400,
    success: true,
    'last-execution': new Date().toISOString(),
    argumentsCount: 0,
    environmentCount: 0,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const isEditMode = useMemo(() => !!hook, [hook]);

  useEffect(() => {
    if (hook) {
      // 确保从后端获取的hook数据有正确的默认值
      setFormData({
        id: hook.id || '',
        'execute-command': hook['execute-command'] || '',
        'command-working-directory': hook['command-working-directory'] || '',
        'response-message': hook['response-message'] || '执行成功',
        'http-methods': hook['http-methods'] || ['POST'],
        'response-headers': hook['response-headers'] || {},
        'pass-arguments-to-command': hook['pass-arguments-to-command'] || [],
        'pass-environment-to-command': hook['pass-environment-to-command'] || [],
        'trigger-rule': hook['trigger-rule'] || {},
        'include-command-output-in-response': hook['include-command-output-in-response'] || false,
        'include-command-output-in-response-on-error': hook['include-command-output-in-response-on-error'] || false,
        'parse-parameters-as-json': hook['parse-parameters-as-json'] || [],
        'trigger-rule-mismatch-http-response-code': hook['trigger-rule-mismatch-http-response-code'] || 400,
        success: hook.success !== undefined ? hook.success : true,
        'last-execution': hook['last-execution'] || new Date().toISOString(),
        argumentsCount: hook.argumentsCount || 0,
        environmentCount: hook.environmentCount || 0,
      });
    } else {
      // 重置为默认值
      setFormData({
        id: '',
        'execute-command': '',
        'command-working-directory': '',
        'response-message': '执行成功',
        'http-methods': ['POST'],
        'response-headers': {},
        'pass-arguments-to-command': [],
        'pass-environment-to-command': [],
        'trigger-rule': {},
        'include-command-output-in-response': false,
        'include-command-output-in-response-on-error': false,
        'parse-parameters-as-json': [],
        'trigger-rule-mismatch-http-response-code': 400,
        success: true,
        'last-execution': new Date().toISOString(),
        argumentsCount: 0,
        environmentCount: 0,
      });
    }
    setTabValue(0);
    setErrors({});
  }, [hook, open]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleFieldChange = (field: keyof IHook, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value,
    }));
    // 清除错误
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  // 验证表单
  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.id.trim()) {
      newErrors.id = 'Hook ID不能为空';
    }
    if (!formData['execute-command'].trim()) {
      newErrors['execute-command'] = '执行命令不能为空';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSave = () => {
    if (!validateForm()) {
      // 跳转到有错误的tab
      if (errors.id || errors['execute-command']) {
        setTabValue(0); // 基本信息
      }
      return;
    }

    // 计算参数数量
    const updatedFormData = {
      ...formData,
      argumentsCount: (formData['pass-arguments-to-command'] || []).length,
      environmentCount: (formData['pass-environment-to-command'] || []).length,
    };

    onSave(updatedFormData);
  };

  // 参数管理函数
  const addParameter = () => {
    const newParam: IParameter = {
      source: 'payload',
      name: '',
    };
    handleFieldChange('pass-arguments-to-command', [
      ...(formData['pass-arguments-to-command'] || []),
      newParam,
    ]);
  };

  const updateParameter = (index: number, field: keyof IParameter, value: string) => {
    const updatedParams = [...(formData['pass-arguments-to-command'] || [])];
    updatedParams[index] = { ...updatedParams[index], [field]: value };
    handleFieldChange('pass-arguments-to-command', updatedParams);
  };

  const removeParameter = (index: number) => {
    const updatedParams = (formData['pass-arguments-to-command'] || []).filter((_, i) => i !== index);
    handleFieldChange('pass-arguments-to-command', updatedParams);
  };

  // 环境变量管理函数
  const addEnvironmentVariable = () => {
    const newEnv: IEnvironmentVariable = {
      name: '',
      source: 'payload',
    };
    handleFieldChange('pass-environment-to-command', [
      ...(formData['pass-environment-to-command'] || []),
      newEnv,
    ]);
  };

  const updateEnvironmentVariable = (index: number, field: keyof IEnvironmentVariable, value: string) => {
    const updatedEnvs = [...(formData['pass-environment-to-command'] || [])];
    updatedEnvs[index] = { ...updatedEnvs[index], [field]: value };
    handleFieldChange('pass-environment-to-command', updatedEnvs);
  };

  const removeEnvironmentVariable = (index: number) => {
    const updatedEnvs = (formData['pass-environment-to-command'] || []).filter((_, i) => i !== index);
    handleFieldChange('pass-environment-to-command', updatedEnvs);
  };

  // 触发规则管理
  const [ruleType, setRuleType] = useState<'simple' | 'advanced'>('simple');
  const [simpleRule, setSimpleRule] = useState({
    type: 'value',
    value: '',
    parameter: { source: 'payload', name: '' },
    secret: '',
    regex: '',
    'ip-range': '',
  });

  // 将简单规则转换为完整规则
  const applySimpleRule = () => {
    const rule: any = {
      match: {
        type: simpleRule.type,
        parameter: simpleRule.parameter,
      }
    };

    switch (simpleRule.type) {
      case 'value':
        rule.match.value = simpleRule.value;
        break;
      case 'regex':
        rule.match.regex = simpleRule.regex;
        break;
      case 'payload-hmac-sha1':
      case 'payload-hmac-sha256':
      case 'payload-hmac-sha512':
      case 'scalr-signature':
        rule.match.secret = simpleRule.secret;
        break;
      case 'ip-whitelist':
        rule.match['ip-range'] = simpleRule['ip-range'];
        delete rule.match.parameter; // IP白名单不需要参数
        break;
    }

    handleFieldChange('trigger-rule', rule);
  };

  // 渲染基本信息Tab
  const renderBasicTab = () => (
    <Grid container spacing={3}>
      <Grid size={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          <Typography variant="body2">
            配置webhook的基本信息，包括ID、执行命令和工作目录。
          </Typography>
        </Alert>
      </Grid>
      
      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          label="Hook ID"
          value={formData.id}
          onChange={(e) => handleFieldChange('id', e.target.value)}
          error={!!errors.id}
          helperText={errors.id || 'webhook的唯一标识符，用于构建URL路径'}
          disabled={isEditMode}
          required
        />
      </Grid>

      <Grid size={12}>
        <TextField
          fullWidth
          label="执行命令"
          value={formData['execute-command']}
          onChange={(e) => handleFieldChange('execute-command', e.target.value)}
          error={!!errors['execute-command']}
          helperText={errors['execute-command'] || '当webhook被触发时执行的命令或脚本路径'}
          placeholder="例如: /path/to/script.sh 或 node /path/to/handler.js"
          required
        />
      </Grid>

      <Grid size={12}>
        <TextField
          fullWidth
          label="工作目录"
          value={formData['command-working-directory'] || ''}
          onChange={(e) => handleFieldChange('command-working-directory', e.target.value)}
          helperText="命令执行时的工作目录，留空则使用当前目录"
          placeholder="例如: /var/www/project"
        />
      </Grid>

      <Grid size={12}>
        <TextField
          fullWidth
          label="响应消息"
          value={formData['response-message']}
          onChange={(e) => handleFieldChange('response-message', e.target.value)}
          helperText="webhook执行成功时返回的消息"
        />
      </Grid>
    </Grid>
  );

  // 渲染参数传递Tab
  const renderParametersTab = () => (
    <Box>
      <Alert severity="info" sx={{ mb: 3 }}>
        <Typography variant="body2">
          配置如何将webhook请求中的数据传递给执行命令。参数会按顺序传递给命令，环境变量会设置到命令的执行环境中。
        </Typography>
      </Alert>

      {/* 命令参数配置 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6" sx={{ flexGrow: 1 }}>
              命令参数
            </Typography>
            <Tooltip title="参数会按顺序传递给执行命令">
              <IconButton size="small">
                <HelpIcon />
              </IconButton>
            </Tooltip>
            <Button
              startIcon={<AddIcon />}
              onClick={addParameter}
              variant="outlined"
              size="small"
            >
              添加参数
            </Button>
          </Box>

          {(formData['pass-arguments-to-command'] || []).length === 0 ? (
            <Typography color="textSecondary" sx={{ textAlign: 'center', py: 2 }}>
              暂无命令参数配置
            </Typography>
          ) : (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>参数来源</TableCell>
                    <TableCell>参数名称</TableCell>
                    <TableCell width={100}>操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(formData['pass-arguments-to-command'] || []).map((param, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <FormControl fullWidth size="small">
                          <Select
                            value={param.source}
                            onChange={(e) => updateParameter(index, 'source', e.target.value)}
                          >
                            {PARAMETER_SOURCES.map((source) => (
                              <MenuItem key={source.value} value={source.value}>
                                <Box>
                                  <Typography variant="body2">{source.label}</Typography>
                                  <Typography variant="caption" color="textSecondary">
                                    {source.description}
                                  </Typography>
                                </Box>
                              </MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      </TableCell>
                      <TableCell>
                        <TextField
                          fullWidth
                          size="small"
                          value={param.name}
                          onChange={(e) => updateParameter(index, 'name', e.target.value)}
                          placeholder={
                            param.source === 'payload' ? '例如: repository.name' :
                            param.source === 'header' ? '例如: X-GitHub-Event' :
                            param.source === 'query' ? '例如: token' :
                            '固定值'
                          }
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton
                          size="small"
                          onClick={() => removeParameter(index)}
                          color="error"
                        >
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      {/* 环境变量配置 */}
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6" sx={{ flexGrow: 1 }}>
              环境变量
            </Typography>
            <Tooltip title="环境变量会设置到命令的执行环境中">
              <IconButton size="small">
                <HelpIcon />
              </IconButton>
            </Tooltip>
            <Button
              startIcon={<AddIcon />}
              onClick={addEnvironmentVariable}
              variant="outlined"
              size="small"
            >
              添加环境变量
            </Button>
          </Box>

          {(formData['pass-environment-to-command'] || []).length === 0 ? (
            <Typography color="textSecondary" sx={{ textAlign: 'center', py: 2 }}>
              暂无环境变量配置
            </Typography>
          ) : (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>变量名</TableCell>
                    <TableCell>数据来源</TableCell>
                    <TableCell>数据路径</TableCell>
                    <TableCell width={100}>操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(formData['pass-environment-to-command'] || []).map((env, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <TextField
                          fullWidth
                          size="small"
                          value={env.name}
                          onChange={(e) => updateEnvironmentVariable(index, 'name', e.target.value)}
                          placeholder="环境变量名"
                        />
                      </TableCell>
                      <TableCell>
                        <FormControl fullWidth size="small">
                          <Select
                            value={env.source}
                            onChange={(e) => updateEnvironmentVariable(index, 'source', e.target.value)}
                          >
                            {PARAMETER_SOURCES.map((source) => (
                              <MenuItem key={source.value} value={source.value}>
                                <Box>
                                  <Typography variant="body2">{source.label}</Typography>
                                  <Typography variant="caption" color="textSecondary">
                                    {source.description}
                                  </Typography>
                                </Box>
                              </MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      </TableCell>
                      <TableCell>
                        <TextField
                          fullWidth
                          size="small"
                          value={env.name}
                          onChange={(e) => updateEnvironmentVariable(index, 'name', e.target.value)}
                          placeholder={
                            env.source === 'payload' ? '例如: pusher.name' :
                            env.source === 'header' ? '例如: User-Agent' :
                            env.source === 'query' ? '例如: branch' :
                            '固定值'
                          }
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton
                          size="small"
                          onClick={() => removeEnvironmentVariable(index)}
                          color="error"
                        >
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>
    </Box>
  );

  // 渲染触发规则Tab
  const renderTriggersTab = () => (
    <Box>
      <Alert severity="info" sx={{ mb: 3 }}>
        <Typography variant="body2">
          配置webhook的触发条件。只有满足触发规则的请求才会执行命令。支持值匹配、正则表达式、签名验证等多种方式，
          以及复杂的and/or/not逻辑组合。
        </Typography>
      </Alert>

      <Card>
        <CardContent>
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" gutterBottom>
              规则配置模式
            </Typography>
            <FormControl component="fieldset">
              <Box sx={{ display: 'flex', gap: 2 }}>
                <Button
                  variant={ruleType === 'simple' ? 'contained' : 'outlined'}
                  onClick={() => setRuleType('simple')}
                >
                  简单模式
                </Button>
                <Button
                  variant={ruleType === 'advanced' ? 'contained' : 'outlined'}
                  onClick={() => setRuleType('advanced')}
                >
                  高级模式
                </Button>
              </Box>
            </FormControl>
          </Box>

          {ruleType === 'simple' ? (
            <Box>
              <Typography variant="subtitle1" gutterBottom>
                简单规则配置
              </Typography>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 4 }}>
                  <FormControl fullWidth>
                    <InputLabel>匹配类型</InputLabel>
                    <Select
                      value={simpleRule.type}
                      onChange={(e) => setSimpleRule(prev => ({ ...prev, type: e.target.value }))}
                    >
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

                {simpleRule.type !== 'ip-whitelist' && (
                  <>
                    <Grid size={{ xs: 12, md: 4 }}>
                      <FormControl fullWidth>
                        <InputLabel>参数来源</InputLabel>
                        <Select
                          value={simpleRule.parameter.source}
                          onChange={(e) => setSimpleRule(prev => ({
                            ...prev,
                            parameter: { ...prev.parameter, source: e.target.value }
                          }))}
                        >
                          {PARAMETER_SOURCES.map((source) => (
                            <MenuItem key={source.value} value={source.value}>
                              {source.label}
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    </Grid>

                    <Grid size={{ xs: 12, md: 4 }}>
                      <TextField
                        fullWidth
                        label="参数名称"
                        value={simpleRule.parameter.name}
                        onChange={(e) => setSimpleRule(prev => ({
                          ...prev,
                          parameter: { ...prev.parameter, name: e.target.value }
                        }))}
                        placeholder="例如: ref, X-GitHub-Event"
                      />
                    </Grid>
                  </>
                )}

                {simpleRule.type === 'value' && (
                  <Grid size={12}>
                    <TextField
                      fullWidth
                      label="匹配值"
                      value={simpleRule.value}
                      onChange={(e) => setSimpleRule(prev => ({ ...prev, value: e.target.value }))}
                      placeholder="例如: refs/heads/master"
                    />
                  </Grid>
                )}

                {simpleRule.type === 'regex' && (
                  <Grid size={12}>
                    <TextField
                      fullWidth
                      label="正则表达式"
                      value={simpleRule.regex}
                      onChange={(e) => setSimpleRule(prev => ({ ...prev, regex: e.target.value }))}
                      placeholder="例如: refs/heads/(master|main)"
                      helperText="使用Go语言正则表达式语法"
                    />
                  </Grid>
                )}

                {(simpleRule.type.includes('hmac') || simpleRule.type === 'scalr-signature') && (
                  <Grid size={12}>
                    <TextField
                      fullWidth
                      label="密钥"
                      type="password"
                      value={simpleRule.secret}
                      onChange={(e) => setSimpleRule(prev => ({ ...prev, secret: e.target.value }))}
                      placeholder="用于签名验证的密钥"
                    />
                  </Grid>
                )}

                {simpleRule.type === 'ip-whitelist' && (
                  <Grid size={12}>
                    <TextField
                      fullWidth
                      label="IP地址范围"
                      value={simpleRule['ip-range']}
                      onChange={(e) => setSimpleRule(prev => ({ ...prev, 'ip-range': e.target.value }))}
                      placeholder="例如: 192.168.1.0/24 或 192.168.1.100/32"
                      helperText="使用CIDR格式，单个IP地址请使用/32"
                    />
                  </Grid>
                )}

                <Grid size={12}>
                  <Button 
                    variant="contained" 
                    onClick={applySimpleRule}
                    startIcon={<InfoIcon />}
                  >
                    应用简单规则
                  </Button>
                </Grid>
              </Grid>
            </Box>
          ) : (
            <Box>
              <Typography variant="subtitle1" gutterBottom>
                高级规则配置 (JSON格式)
              </Typography>
              <Alert severity="warning" sx={{ mb: 2 }}>
                <Typography variant="body2">
                  高级模式支持复杂的and/or/not逻辑组合。请参考文档编写JSON格式的规则。
                  支持嵌套组合，例如：and条件中包含多个or条件等。
                </Typography>
              </Alert>
              <TextField
                fullWidth
                multiline
                rows={12}
                value={JSON.stringify(formData['trigger-rule'], null, 2)}
                onChange={(e) => {
                  try {
                    const rule = JSON.parse(e.target.value);
                    handleFieldChange('trigger-rule', rule);
                  } catch (err) {
                    // 忽略JSON解析错误，让用户继续编辑
                  }
                }}
                placeholder={`{
  "and": [
    {
      "match": {
        "type": "value",
        "value": "refs/heads/master",
        "parameter": {
          "source": "payload",
          "name": "ref"
        }
      }
    },
    {
      "or": [
        {
          "match": {
            "type": "value",
            "value": "push",
            "parameter": {
              "source": "header",
              "name": "X-GitHub-Event"
            }
          }
        },
        {
          "match": {
            "type": "value",
            "value": "ping",
            "parameter": {
              "source": "header",
              "name": "X-GitHub-Event"
            }
          }
        }
      ]
    }
  ]
}`}
                sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}
              />
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  );

  // 渲染响应配置Tab
  const renderResponseTab = () => (
    <Grid container spacing={3}>
      <Grid size={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          <Typography variant="body2">
            配置webhook的HTTP响应行为，包括允许的HTTP方法、响应状态码和输出选项。
          </Typography>
        </Alert>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <FormControl fullWidth>
          <InputLabel>允许的HTTP方法</InputLabel>
          <Select
            multiple
            value={formData['http-methods']}
            onChange={(e) => handleFieldChange('http-methods', e.target.value)}
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {(selected as string[]).map((value) => (
                  <Chip key={value} label={value} size="small" />
                ))}
              </Box>
            )}
          >
            {['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((method) => (
              <MenuItem key={method} value={method}>
                {method}
              </MenuItem>
            ))}
          </Select>
          <FormHelperText>选择webhook接受的HTTP方法</FormHelperText>
        </FormControl>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          label="规则不匹配时的状态码"
          type="number"
          value={formData['trigger-rule-mismatch-http-response-code']}
          onChange={(e) => handleFieldChange('trigger-rule-mismatch-http-response-code', parseInt(e.target.value))}
          helperText="当触发规则不匹配时返回的HTTP状态码"
        />
      </Grid>

      <Grid size={12}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              命令输出选项
            </Typography>
            
            <FormControlLabel
              control={
                <Switch
                  checked={formData['include-command-output-in-response']}
                  onChange={(e) => handleFieldChange('include-command-output-in-response', e.target.checked)}
                />
              }
              label="在响应中包含命令输出"
            />
            <Typography variant="body2" color="textSecondary" sx={{ mt: 1, mb: 2 }}>
              启用后，webhook响应会包含执行命令的标准输出
            </Typography>

            <FormControlLabel
              control={
                <Switch
                  checked={formData['include-command-output-in-response-on-error']}
                  onChange={(e) => handleFieldChange('include-command-output-in-response-on-error', e.target.checked)}
                />
              }
              label="错误时在响应中包含命令输出"
            />
            <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
              启用后，当命令执行失败时，响应会包含错误输出以便调试
            </Typography>
          </CardContent>
        </Card>
      </Grid>

      <Grid size={12}>
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="h6">高级响应配置</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Grid container spacing={2}>
              <Grid size={12}>
                <Typography variant="subtitle2" gutterBottom>
                  JSON参数解析
                </Typography>
                <Typography variant="body2" color="textSecondary" gutterBottom>
                  指定哪些参数应该被解析为JSON对象
                </Typography>
                <TextField
                  fullWidth
                  label="JSON参数列表"
                  value={formData['parse-parameters-as-json'].join(', ')}
                  onChange={(e) => handleFieldChange('parse-parameters-as-json', 
                    e.target.value.split(',').map(s => s.trim()).filter(s => s)
                  )}
                  placeholder="例如: payload, config"
                  helperText="多个参数用逗号分隔"
                />
              </Grid>

              <Grid size={12}>
                <Typography variant="subtitle2" gutterBottom>
                  自定义响应头
                </Typography>
                <Typography variant="body2" color="textSecondary" gutterBottom>
                  添加自定义的HTTP响应头（JSON格式）
                </Typography>
                <TextField
                  fullWidth
                  multiline
                  rows={3}
                  value={JSON.stringify(formData['response-headers'], null, 2)}
                  onChange={(e) => {
                    try {
                      const headers = JSON.parse(e.target.value);
                      handleFieldChange('response-headers', headers);
                    } catch (err) {
                      // 忽略JSON解析错误
                    }
                  }}
                  placeholder={`{
  "Content-Type": "application/json",
  "X-Custom-Header": "value"
}`}
                  sx={{ fontFamily: 'monospace' }}
                />
              </Grid>
            </Grid>
          </AccordionDetails>
        </Accordion>
      </Grid>
    </Grid>
  );

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: { height: '90vh' }
      }}
    >
      <DialogTitle>
        {isEditMode ? '编辑Webhook配置' : '添加Webhook配置'}
      </DialogTitle>
      
      <DialogContent sx={{ p: 0 }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={tabValue} onChange={handleTabChange} aria-label="hook configuration tabs">
            <Tab label="基本信息" {...a11yProps(0)} />
            <Tab label="参数传递" {...a11yProps(1)} />
            <Tab label="触发规则" {...a11yProps(2)} />
            <Tab label="响应配置" {...a11yProps(3)} />
          </Tabs>
        </Box>

        <Box sx={{ px: 3, maxHeight: 'calc(90vh - 160px)', overflow: 'auto' }}>
          <TabPanel value={tabValue} index={0}>
            {renderBasicTab()}
          </TabPanel>
          <TabPanel value={tabValue} index={1}>
            {renderParametersTab()}
          </TabPanel>
          <TabPanel value={tabValue} index={2}>
            {renderTriggersTab()}
          </TabPanel>
          <TabPanel value={tabValue} index={3}>
            {renderResponseTab()}
          </TabPanel>
        </Box>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose}>
          取消
        </Button>
        <Button onClick={handleSave} variant="contained">
          {isEditMode ? '更新' : '创建'}
        </Button>
      </DialogActions>
    </Dialog>
  );
} 