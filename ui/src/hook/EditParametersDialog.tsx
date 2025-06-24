import React, { useState, useEffect } from 'react';
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
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Help as HelpIcon,
} from '@mui/icons-material';
import { IHook, IParameter, IEnvironmentVariable } from '../types';

interface EditParametersDialogProps {
  open: boolean;
  onClose: () => void;
  hookId?: string;
  onSave: (hookId: string, parametersData: {
    'pass-arguments-to-command': IParameter[];
    'pass-environment-to-command': IEnvironmentVariable[];
    'parse-parameters-as-json': string[];
  }) => void;
  onGetHookDetails: (hookId: string) => Promise<IHook>;
}

// 参数来源选项
const PARAMETER_SOURCES = [
  { value: 'payload', label: 'Payload', description: '从请求体JSON中获取' },
  { value: 'header', label: 'Header', description: '从HTTP请求头中获取' },
  { value: 'query', label: 'Query', description: '从URL查询参数中获取' },
  { value: 'string', label: 'String', description: '固定字符串值' },
];

export default function EditParametersDialog({ open, onClose, hookId, onSave, onGetHookDetails }: EditParametersDialogProps) {
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
    setFormData(prev => ({
      ...prev,
      'pass-arguments-to-command': [...prev['pass-arguments-to-command'], newParam],
    }));
  };

  const updateParameter = (index: number, field: keyof IParameter, value: string) => {
    setFormData(prev => {
      const updatedParams = [...prev['pass-arguments-to-command']];
      updatedParams[index] = { ...updatedParams[index], [field]: value };
      return {
        ...prev,
        'pass-arguments-to-command': updatedParams,
      };
    });
  };

  const removeParameter = (index: number) => {
    setFormData(prev => ({
      ...prev,
      'pass-arguments-to-command': prev['pass-arguments-to-command'].filter((_, i) => i !== index),
    }));
  };

  // 环境变量管理函数
  const addEnvironmentVariable = () => {
    const newEnv: IEnvironmentVariable = {
      name: '',
      source: 'payload',
    };
    setFormData(prev => ({
      ...prev,
      'pass-environment-to-command': [...prev['pass-environment-to-command'], newEnv],
    }));
  };

  const updateEnvironmentVariable = (index: number, field: keyof IEnvironmentVariable, value: string) => {
    setFormData(prev => {
      const updatedEnvs = [...prev['pass-environment-to-command']];
      updatedEnvs[index] = { ...updatedEnvs[index], [field]: value };
      return {
        ...prev,
        'pass-environment-to-command': updatedEnvs,
      };
    });
  };

  const removeEnvironmentVariable = (index: number) => {
    setFormData(prev => ({
      ...prev,
      'pass-environment-to-command': prev['pass-environment-to-command'].filter((_, i) => i !== index),
    }));
  };

  const handleSave = () => {
    if (!hookId) return;

    onSave(hookId, formData);
    onClose();
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="lg"
      fullWidth
    >
      <DialogTitle>编辑参数传递 - {hookId}</DialogTitle>
      
      <DialogContent>
        <Box sx={{ pt: 2 }}>
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

              {formData['pass-arguments-to-command'].length === 0 ? (
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
                      {formData['pass-arguments-to-command'].map((param, index) => (
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

              {formData['pass-environment-to-command'].length === 0 ? (
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
                      {formData['pass-environment-to-command'].map((env, index) => (
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
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>
          取消
        </Button>
        <Button 
          onClick={handleSave} 
          variant="contained" 
          color="primary"
        >
          保存参数配置
        </Button>
      </DialogActions>
    </Dialog>
  );
} 