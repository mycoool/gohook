import React, {Component} from 'react';
import {
    Box,
    Typography,
    Card,
    CardContent,
    Chip,
    IconButton,
    Tooltip,
    Paper,
} from '@mui/material';
import {
    Computer as ComputerIcon,
    Memory as MemoryIcon,
    Storage as StorageIcon,
    NetworkCheck as NetworkIcon,
    Refresh as RefreshIcon,
    Description as DescriptionIcon,
    Security as SecurityIcon,
    Timeline as TimelineIcon,
} from '@mui/icons-material';
import {observer} from 'mobx-react';
import useTranslation from '../i18n/useTranslation';

interface DashboardState {
    loading: boolean;
    lastUpdated: Date;
}

interface DashboardProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

// 圆形进度指示器组件
const CircularProgress: React.FC<{
    value: number;
    size?: number;
    thickness?: number;
    color?: string;
    backgroundColor?: string;
}> = ({value, size = 120, thickness = 8, color = '#4caf50', backgroundColor = '#e0e0e0'}) => {
    const radius = (size - thickness) / 2;
    const circumference = 2 * Math.PI * radius;
    const strokeDasharray = circumference;
    const strokeDashoffset = circumference - (value / 100) * circumference;

    return (
        <div style={{position: 'relative', width: size, height: size}}>
            <svg width={size} height={size} style={{transform: 'rotate(-90deg)'}}>
                {/* 背景圆 */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke={backgroundColor}
                    strokeWidth={thickness}
                    fill="none"
                />
                {/* 进度圆 */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke={color}
                    strokeWidth={thickness}
                    fill="none"
                    strokeDasharray={strokeDasharray}
                    strokeDashoffset={strokeDashoffset}
                    strokeLinecap="round"
                    style={{
                        transition: 'stroke-dashoffset 0.5s ease-in-out',
                    }}
                />
            </svg>
            {/* 中心文字 */}
            <div
                style={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)',
                    textAlign: 'center',
                }}>
                <Typography variant="h6" sx={{fontWeight: 'bold', color: 'text.primary'}}>
                    {value.toFixed(1)}%
                </Typography>
            </div>
        </div>
    );
};

@observer
class Dashboard extends Component<DashboardProps, DashboardState> {
    private refreshInterval?: NodeJS.Timeout;

    constructor(props: DashboardProps) {
        super(props);
        this.state = {
            loading: false,
            lastUpdated: new Date(),
        };
    }

    componentDidMount() {
        this.loadData();
        // 每30秒自动刷新
        this.refreshInterval = setInterval(() => {
            this.loadData();
        }, 30000);
    }

    componentWillUnmount() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }

    loadData = async () => {
        this.setState({loading: true});
        try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 500));
            this.setState({
                lastUpdated: new Date(),
            });
        } catch (error) {
            console.error('加载数据失败:', error);
        } finally {
            this.setState({loading: false});
        }
    };

    render() {
        const {loading, lastUpdated} = this.state;

        return (
            <Box sx={{p: 3}}>
                {/* 页头 */}
                <Box sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3}}>
                    <Typography variant="h4" component="h1" sx={{fontWeight: 'bold'}}>
                        概览
                    </Typography>
                    <Box sx={{display: 'flex', alignItems: 'center', gap: 2}}>
                        <Typography variant="body2" color="textSecondary">
                            最后更新: {lastUpdated.toLocaleTimeString()}
                        </Typography>
                        <Tooltip title="刷新数据">
                            <IconButton onClick={this.loadData} disabled={loading}>
                                <RefreshIcon />
                            </IconButton>
                        </Tooltip>
                    </Box>
                </Box>

                {/* 系统监控卡片 */}
                <Box sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                        xs: '1fr',
                        sm: 'repeat(2, 1fr)',
                        md: 'repeat(4, 1fr)'
                    },
                    gap: 3,
                    mb: 3
                }}>
                    <Card>
                        <CardContent sx={{textAlign: 'center', p: 2}}>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2}}>
                                <ComputerIcon color="action" />
                                <Typography variant="h6">CPU</Typography>
                            </Box>
                            <CircularProgress 
                                value={41.6} 
                                color="#2196f3"
                                size={100}
                            />
                            <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                                9.2%
                            </Typography>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent sx={{textAlign: 'center', p: 2}}>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2}}>
                                <MemoryIcon color="action" />
                                <Typography variant="h6">内存</Typography>
                            </Box>
                            <CircularProgress 
                                value={75.88} 
                                color="#4caf50"
                                size={100}
                            />
                            <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                                5%
                            </Typography>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent sx={{textAlign: 'center', p: 2}}>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2}}>
                                <StorageIcon color="action" />
                                <Typography variant="h6">磁盘</Typography>
                            </Box>
                            <CircularProgress 
                                value={75.88} 
                                color="#ff9800"
                                size={100}
                            />
                            <Typography variant="body2" color="textSecondary" sx={{mt: 1}}>
                                8.58%/16GB
                            </Typography>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent sx={{textAlign: 'center', p: 2}}>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2}}>
                                <NetworkIcon color="action" />
                                <Typography variant="h6">网络</Typography>
                            </Box>
                            <Box sx={{mt: 2}}>
                                <Box sx={{display: 'flex', justifyContent: 'space-between', mb: 1}}>
                                    <Typography variant="body2">↓下载</Typography>
                                    <Typography variant="body2">1.2 KB/s</Typography>
                                </Box>
                                <Box sx={{display: 'flex', justifyContent: 'space-between'}}>
                                    <Typography variant="body2">↑上传</Typography>
                                    <Typography variant="body2">0.8 KB/s</Typography>
                                </Box>
                            </Box>
                        </CardContent>
                    </Card>
                </Box>

                {/* 统计信息卡片 */}
                <Box sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                        xs: '1fr',
                        sm: 'repeat(2, 1fr)',
                        md: 'repeat(4, 1fr)'
                    },
                    gap: 3,
                    mb: 3
                }}>
                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                                <Box>
                                    <Typography color="textSecondary" gutterBottom>
                                        网站总数
                                    </Typography>
                                    <Typography variant="h4">
                                        17
                                    </Typography>
                                </Box>
                                <ComputerIcon sx={{fontSize: 40, color: 'action.disabled'}} />
                            </Box>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                                <Box>
                                    <Typography color="textSecondary" gutterBottom>
                                        数据库总数
                                    </Typography>
                                    <Typography variant="h4">
                                        3
                                    </Typography>
                                </Box>
                                <DescriptionIcon sx={{fontSize: 40, color: 'action.disabled'}} />
                            </Box>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                                <Box>
                                    <Typography color="textSecondary" gutterBottom>
                                        安全风险
                                    </Typography>
                                    <Typography variant="h4" color="error">
                                        6
                                    </Typography>
                                </Box>
                                <SecurityIcon sx={{fontSize: 40, color: 'error.main'}} />
                            </Box>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                                <Box>
                                    <Typography color="textSecondary" gutterBottom>
                                        备份任务
                                    </Typography>
                                    <Typography variant="body2" color="textSecondary">
                                        当前版本自带一键配
                                    </Typography>
                                </Box>
                                <TimelineIcon sx={{fontSize: 40, color: 'action.disabled'}} />
                            </Box>
                        </CardContent>
                    </Card>
                </Box>

                {/* 软件信息和流量统计 */}
                <Box sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                        xs: '1fr',
                        md: 'repeat(2, 1fr)'
                    },
                    gap: 3
                }}>
                    {/* 软件信息 */}
                    <Card>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>
                                软件
                            </Typography>
                            <Box sx={{
                                display: 'grid',
                                gridTemplateColumns: {
                                    xs: 'repeat(2, 1fr)',
                                    sm: 'repeat(4, 1fr)'
                                },
                                gap: 2
                            }}>
                                {[
                                    {version: '1.0', name: '宝塔面板'},
                                    {version: '2.4', name: 'Linux工具箱'},
                                    {version: '7.4 ▶', name: 'Redis'},
                                    {version: '2.4(3.10) ▶', name: 'Nginx+Tengine'},
                                    {version: '1.7', name: '插件安装管理器'},
                                    {version: '2.5', name: '宝塔WebHook'},
                                    {version: '8.1.31 ▶', name: 'PHP'},
                                    {version: '3.0.5', name: '静态手册管理器'},
                                ].map((software, index) => (
                                    <Paper key={index} sx={{p: 1.5, textAlign: 'center'}}>
                                        <Typography variant="body1" color="primary" sx={{fontWeight: 'bold'}}>
                                            {software.version}
                                        </Typography>
                                        <Typography variant="body2" color="textSecondary">
                                            {software.name}
                                        </Typography>
                                    </Paper>
                                ))}
                            </Box>
                        </CardContent>
                    </Card>

                    {/* 流量统计图表区域 */}
                    <Card>
                        <CardContent>
                            <Box sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2}}>
                                <Typography variant="h6">流量</Typography>
                                <Box sx={{display: 'flex', gap: 1}}>
                                    <Chip label="昨天" size="small" />
                                    <Chip label="今天" size="small" color="primary" />
                                </Box>
                            </Box>
                            <Box sx={{display: 'flex', justifyContent: 'space-around', mb: 2}}>
                                <Box sx={{textAlign: 'center'}}>
                                    <Typography variant="h6" color="warning.main">98.21 KB</Typography>
                                    <Typography variant="body2" color="textSecondary">入网</Typography>
                                </Box>
                                <Box sx={{textAlign: 'center'}}>
                                    <Typography variant="h6" color="info.main">96.76 KB</Typography>
                                    <Typography variant="body2" color="textSecondary">出网</Typography>
                                </Box>
                                <Box sx={{textAlign: 'center'}}>
                                    <Typography variant="h6">227.36 GB</Typography>
                                    <Typography variant="body2" color="textSecondary">总流量</Typography>
                                </Box>
                                <Box sx={{textAlign: 'center'}}>
                                    <Typography variant="h6">273.72 GB</Typography>
                                    <Typography variant="body2" color="textSecondary">总流量</Typography>
                                </Box>
                            </Box>
                            {/* 模拟图表区域 */}
                            <Box sx={{
                                height: 200,
                                background: 'linear-gradient(45deg, #2196F3 30%, #21CBF3 90%)',
                                borderRadius: 1,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: 'white'
                            }}>
                                <Typography>流量统计图表</Typography>
                            </Box>
                        </CardContent>
                    </Card>
                </Box>
            </Box>
        );
    }
}

// 使用翻译Hook的容器组件
const DashboardWithTranslation: React.FC = () => {
    const {t} = useTranslation();
    return <Dashboard t={t} />;
};

export default DashboardWithTranslation;