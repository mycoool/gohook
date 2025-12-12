import React, {Component} from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
    Chip,
    Typography,
    Box,
    Grid,
} from '@mui/material';

import DefaultPage from '../common/DefaultPage';
import {inject, Stores} from '../inject';
import {withRouter, RouteComponentProps} from 'react-router-dom';
import {observer} from 'mobx-react';
import useTranslation from '../i18n/useTranslation';

type IProps = RouteComponentProps<{projectName: string}>;
type InjectedProps = IProps & Stores<'versionStore' | 'snackManager'>;

interface IPropsWithTranslation extends InjectedProps {
    t: (key: string, params?: Record<string, string | number>) => string;
}

interface IState {
    editingEnvFile: boolean;
    envFileContent: string;
    originalEnvFileContent: string;
    hasEnvFile: boolean;
    errors: string[];
    isTomlContent: boolean;
}

// detect if content uses TOML format (inside .env file)
function detectTomlFormat(content: string): boolean {
    const lines = content.split('\n');

    for (const line of lines) {
        const trimmedLine = line.trim();

        // skip empty lines and comments
        if (trimmedLine === '' || trimmedLine.startsWith('#')) {
            continue;
        }

        // check for TOML section headers [section]
        if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
            return true;
        }

        // check for TOML multiline strings
        if (trimmedLine.includes('"""')) {
            return true;
        }

        // check for TOML arrays
        if (/^\w+\s*=\s*\[.*\]/.test(trimmedLine)) {
            return true;
        }

        // check for TOML dotted keys
        if (/^\w+\.\w+\s*=/.test(trimmedLine)) {
            return true;
        }
    }

    return false;
}

// enhanced syntax highlighting for both .env and TOML content
function highlightEnvSyntax(content: string, isTomlContent: boolean, isDarkMode: boolean): string {
    const lines = content.split('\n');
    const highlightedLines = lines.map((line) => {
        const trimmedLine = line.trim();

        // empty line
        if (trimmedLine === '') {
            return '&nbsp;';
        }

        // comment line
        if (trimmedLine.startsWith('#')) {
            return `<span style="color: ${
                isDarkMode ? '#6A9955' : '#008000'
            }; font-style: italic;">${escapeHtml(line)}</span>`;
        }

        // TOML content highlighting
        if (isTomlContent) {
            // TOML section headers [section]
            if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
                return `<span style="color: ${
                    isDarkMode ? '#569CD6' : '#0000FF'
                }; font-weight: bold;">${escapeHtml(line)}</span>`;
            }
        }

        // key=value lines
        if (line.includes('=')) {
            const equalIndex = line.indexOf('=');
            const beforeEqual = line.substring(0, equalIndex);
            const afterEqual = line.substring(equalIndex);

            let keyColor = isDarkMode ? '#9CDCFE' : '#0451A5';
            let valueColor = isDarkMode ? '#CE9178' : '#A31515';

            // TOML content special coloring
            if (isTomlContent) {
                // dotted keys
                if (beforeEqual.trim().includes('.')) {
                    keyColor = isDarkMode ? '#4EC9B0' : '#008080';
                }

                const value = afterEqual.substring(1).trim();
                // boolean values
                if (value === 'true' || value === 'false') {
                    valueColor = isDarkMode ? '#569CD6' : '#0000FF';
                }
                // numeric values
                else if (/^\d+(\.\d+)?$/.test(value)) {
                    valueColor = isDarkMode ? '#B5CEA8' : '#098658';
                }
                // arrays
                else if (value.startsWith('[') && value.endsWith(']')) {
                    valueColor = isDarkMode ? '#D4D4D4' : '#000000';
                }
            }

            return `<span style="color: ${keyColor};">${escapeHtml(
                beforeEqual
            )}</span><span style="color: ${
                isDarkMode ? '#D4D4D4' : '#000000'
            };">=</span><span style="color: ${valueColor};">${escapeHtml(
                afterEqual.substring(1)
            )}</span>`;
        }

        // other lines
        return escapeHtml(line);
    });

    return highlightedLines.join('<br/>');
}

function escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

@observer
class EnvFileDialog extends Component<IPropsWithTranslation, IState> {
    state: IState = {
        editingEnvFile: false,
        envFileContent: '',
        originalEnvFileContent: '',
        hasEnvFile: false,
        errors: [],
        isTomlContent: false,
    };

    componentDidMount() {
        this.loadEnvFile();
    }

    get projectName(): string {
        return this.props.match.params.projectName;
    }

    loadEnvFile = async () => {
        try {
            const result = await this.props.versionStore.getEnvFile(this.projectName);
            if (result.exists) {
                const isToml = detectTomlFormat(result.content);
                this.setState({
                    envFileContent: result.content,
                    originalEnvFileContent: result.content,
                    hasEnvFile: true,
                    isTomlContent: isToml,
                });
            } else {
                this.setState({
                    hasEnvFile: false,
                    isTomlContent: false,
                });
            }
        } catch (error) {
            this.props.snackManager.snack(this.props.t('version.env.loadError'));
        }
    };

    openEnvFileEditor = () => {
        this.setState({editingEnvFile: true});
    };

    closeEnvFileEditor = () => {
        this.setState({
            editingEnvFile: false,
            envFileContent: this.state.originalEnvFileContent,
            errors: [],
        });
    };

    handleEnvFileContentChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const content = event.target.value;
        const isToml = detectTomlFormat(content);
        this.setState({
            envFileContent: content,
            isTomlContent: isToml,
        });
    };

    validateAndSaveEnvFile = async () => {
        try {
            await this.props.versionStore.saveEnvFile(this.projectName, this.state.envFileContent);
            this.props.snackManager.snack(this.props.t('version.env.saveSuccess'));
            this.setState({
                editingEnvFile: false,
                originalEnvFileContent: this.state.envFileContent,
                hasEnvFile: true,
                errors: [],
            });
        } catch (error: any) {
            if (error.response?.data?.errors) {
                this.setState({errors: error.response.data.errors});
            } else {
                this.props.snackManager.snack(this.props.t('version.env.saveError'));
            }
        }
    };

    deleteEnvFile = async () => {
        try {
            await this.props.versionStore.deleteEnvFile(this.projectName);
            this.props.snackManager.snack(this.props.t('version.env.deleteSuccess'));
            this.setState({
                envFileContent: '',
                originalEnvFileContent: '',
                hasEnvFile: false,
                editingEnvFile: false,
                errors: [],
                isTomlContent: false,
            });
        } catch (error) {
            this.props.snackManager.snack(this.props.t('version.env.deleteError'));
        }
    };

    applyTemplate = (template: string) => {
        const isToml = detectTomlFormat(template);
        this.setState({
            envFileContent: template,
            isTomlContent: isToml,
        });
    };

    render() {
        const {editingEnvFile, envFileContent, hasEnvFile, errors, isTomlContent} = this.state;
        // 获取当前主题模式
        const isDarkMode = localStorage.getItem('gohook-theme') === 'dark';

        // format indicator
        const formatIndicator = isTomlContent ? 'TOML' : 'ENV';
        const formatColor = isTomlContent ? '#4CAF50' : '#2196F3';

        // templates
        const envTemplate = `# Basic .env format
APP_NAME=MyApplication
APP_VERSION=1.0.0
APP_DEBUG=true

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=username
DB_PASSWORD=password

# External Services
API_KEY=your-api-key-here
SECRET_KEY=your-secret-key-here`;

        const tomlTemplate = `# TOML format content in .env file
# Application settings
[app]
name = "MyApplication"
version = "1.0.0"
debug = true
environment = "development"

# Database configuration
[database]
host = "localhost"
port = 5432
name = "myapp"
username = "user"
password = "password"
max_connections = 100

# API configuration
[api]
endpoints = ["https://api1.example.com", "https://api2.example.com"]
timeout = 30
retry_count = 3

# Feature flags
[features]
enable_cache = true
enable_logging = true
enable_metrics = false

# Nested configuration
app.cache.ttl = 3600
app.cache.size = 1024`;

        return (
            <DefaultPage title={this.props.t('version.env.manage')} maxWidth={900}>
                <Box mb={3}>
                    <Typography variant="h6" gutterBottom>
                        {this.props.t('version.env.manage')} (.env)
                    </Typography>
                    <Typography variant="body2" color="textSecondary" gutterBottom>
                        {this.props.t('version.env.saveInfoPrefix')} <code>.env</code>{' '}
                        {this.props.t('version.env.saveInfoSuffix')}
                    </Typography>
                </Box>
                {hasEnvFile ? (
                    <Box mb={2}>
                        <Box
                            display="flex"
                            alignItems="center"
                            mb={2}
                            flexWrap="wrap"
                            style={{gap: '16px'}}>
                            <Chip
                                label={`Format: ${formatIndicator}`}
                                style={{backgroundColor: formatColor, color: 'white'}}
                                size="small"
                            />
                            <Button
                                variant="outlined"
                                color="primary"
                                onClick={this.openEnvFileEditor}>
                                Edit .env File
                            </Button>
                            <Button
                                variant="outlined"
                                color="secondary"
                                onClick={this.deleteEnvFile}>
                                Delete .env File
                            </Button>
                        </Box>
                        <Box
                            p={2}
                            border={1}
                            borderColor={isDarkMode ? '#30363d' : '#d0d7de'}
                            borderRadius="6px"
                            bgcolor={isDarkMode ? '#0d1117' : '#f6f8fa'}
                            maxHeight="400px"
                            overflow="auto"
                            sx={{
                                '&::-webkit-scrollbar': {
                                    width: '8px',
                                },
                                '&::-webkit-scrollbar-track': {
                                    backgroundColor: isDarkMode ? '#2d2d2d' : '#f1f3f4',
                                },
                                '&::-webkit-scrollbar-thumb': {
                                    backgroundColor: isDarkMode ? '#30363d' : '#c1c8cd',
                                    borderRadius: '4px',
                                },
                                '&::-webkit-scrollbar-thumb:hover': {
                                    backgroundColor: isDarkMode ? '#484f58' : '#a8b3ba',
                                },
                            }}>
                            <Typography
                                variant="body2"
                                component="div"
                                style={{
                                    fontFamily: 'monospace',
                                    whiteSpace: 'pre-wrap',
                                    fontSize: '13px',
                                    lineHeight: '1.4',
                                }}
                                dangerouslySetInnerHTML={{
                                    __html: highlightEnvSyntax(
                                        envFileContent,
                                        isTomlContent,
                                        isDarkMode
                                    ),
                                }}
                            />
                        </Box>
                    </Box>
                ) : (
                    <Box mb={2}>
                        <Typography variant="body1" color="textSecondary" gutterBottom>
                            No .env file found. You can create one using different formats:
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid size={{xs: 12, sm: 6}}>
                                <Box
                                    p={2}
                                    border={1}
                                    borderColor={isDarkMode ? '#30363d' : '#d0d7de'}
                                    borderRadius="6px"
                                    bgcolor={isDarkMode ? '#2d2d2d' : '#ffffff'}>
                                    <Typography variant="h6" gutterBottom>
                                        Standard ENV Format
                                    </Typography>
                                    <Typography variant="body2" color="textSecondary" gutterBottom>
                                        Simple key=value pairs
                                    </Typography>
                                    <Button
                                        variant="outlined"
                                        size="small"
                                        onClick={() => this.applyTemplate(envTemplate)}>
                                        Use ENV Template
                                    </Button>
                                </Box>
                            </Grid>
                            <Grid size={{xs: 12, sm: 6}}>
                                <Box
                                    p={2}
                                    border={1}
                                    borderColor={isDarkMode ? '#30363d' : '#d0d7de'}
                                    borderRadius="6px"
                                    bgcolor={isDarkMode ? '#2d2d2d' : '#ffffff'}>
                                    <Typography variant="h6" gutterBottom>
                                        TOML Format
                                    </Typography>
                                    <Typography variant="body2" color="textSecondary" gutterBottom>
                                        Structured configuration with sections
                                    </Typography>
                                    <Button
                                        variant="outlined"
                                        size="small"
                                        onClick={() => this.applyTemplate(tomlTemplate)}>
                                        Use TOML Template
                                    </Button>
                                </Box>
                            </Grid>
                        </Grid>
                        <Box mt={2}>
                            <Button
                                variant="contained"
                                color="primary"
                                onClick={this.openEnvFileEditor}>
                                Create .env File
                            </Button>
                        </Box>
                    </Box>
                )}
                <Dialog
                    open={editingEnvFile}
                    onClose={this.closeEnvFileEditor}
                    maxWidth="md"
                    fullWidth
                    PaperProps={{
                        sx: {
                            // 移除深色灰色背景，使用默认主题背景以提高文字可读性
                            color: isDarkMode ? '#e6edf3' : '#000000',
                        },
                    }}>
                    <DialogTitle>
                        <Box display="flex" alignItems="center" justifyContent="space-between">
                            <span>Edit .env File</span>
                            <Chip
                                label={`Format: ${formatIndicator}`}
                                style={{backgroundColor: formatColor, color: 'white'}}
                                size="small"
                            />
                        </Box>
                    </DialogTitle>
                    <DialogContent>
                        <TextField
                            autoFocus
                            multiline
                            fullWidth
                            rows={20}
                            variant="outlined"
                            value={envFileContent}
                            onChange={this.handleEnvFileContentChange}
                            placeholder={
                                isTomlContent
                                    ? '# TOML format content\n[section]\nkey = "value"'
                                    : '# Standard ENV format\nKEY=value\nANOTHER_KEY=another_value'
                            }
                            InputProps={{
                                style: {
                                    fontFamily:
                                        'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                                    fontSize: '13px',
                                    backgroundColor: isDarkMode ? '#0d1117' : '#ffffff',
                                    color: isDarkMode ? '#e6edf3' : '#000000',
                                },
                            }}
                            sx={{
                                '& .MuiOutlinedInput-root': {
                                    backgroundColor: isDarkMode ? '#0d1117' : '#ffffff',
                                    '& fieldset': {
                                        borderColor: isDarkMode ? '#30363d' : '#d0d7de',
                                    },
                                    '&:hover fieldset': {
                                        borderColor: isDarkMode ? '#58a6ff' : '#0969da',
                                    },
                                    '&.Mui-focused fieldset': {
                                        borderColor: isDarkMode ? '#58a6ff' : '#0969da',
                                    },
                                },
                                '& .MuiInputBase-input': {
                                    color: isDarkMode ? '#e6edf3' : '#000000',
                                },
                                '& .MuiInputBase-input::placeholder': {
                                    color: isDarkMode ? '#8b949e' : '#656d76',
                                    opacity: 1,
                                },
                            }}
                        />
                        {errors.length > 0 && (
                            <Box mt={2}>
                                <Typography variant="subtitle2" color="error">
                                    Validation Errors:
                                </Typography>
                                {errors.map((error, index) => (
                                    <Typography key={index} variant="body2" color="error">
                                        • {error}
                                    </Typography>
                                ))}
                            </Box>
                        )}
                        <Box mt={2}>
                            <Typography variant="body2" color="textSecondary">
                                <strong>Format detected:</strong> {formatIndicator} format content
                                in .env file
                            </Typography>
                            <Typography variant="body2" color="textSecondary">
                                The file will always be saved as <code>.env</code> regardless of
                                content format.
                            </Typography>
                        </Box>
                    </DialogContent>
                    <DialogActions>
                        <Button
                            onClick={this.closeEnvFileEditor}
                            variant="contained"
                            color="secondary">
                            Cancel
                        </Button>
                        <Button
                            onClick={this.validateAndSaveEnvFile}
                            color="primary"
                            variant="contained">
                            Validate & Save
                        </Button>
                    </DialogActions>
                </Dialog>
            </DefaultPage>
        );
    }
}

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
const EnvFileDialogWithTranslation: React.FC<InjectedProps> = (props) => {
    const {t} = useTranslation();
    return <EnvFileDialog {...props} t={t} />;
};

const InjectedEnvFileDialog = inject('versionStore', 'snackManager')(EnvFileDialogWithTranslation);

export default withRouter(InjectedEnvFileDialog as unknown as React.ComponentType<IProps>);
