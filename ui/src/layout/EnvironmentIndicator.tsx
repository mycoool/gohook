import React, {Component} from 'react';
import {observer} from 'mobx-react';
import {Chip} from '@mui/material';
import {inject, Stores} from '../inject';
import useTranslation from '../i18n/useTranslation';

interface IState {
    initialized: boolean;
}

const EnvironmentIndicatorInner: React.FC<{mode: string}> = ({mode}) => {
    const {t} = useTranslation();

    const getEnvironmentConfig = (mode: string) => {
        switch (mode) {
            case 'dev':
                return {
                    label: t('environment.development') || '开发',
                    color: '#4CAF50', // 绿色
                    backgroundColor: '#E8F5E8',
                };
            case 'test':
                return {
                    label: t('environment.testing') || '测试',
                    color: '#FF9800', // 橙色
                    backgroundColor: '#FFF3E0',
                };
            case 'prod':
                return {
                    label: t('environment.production') || '生产',
                    color: '#F44336', // 红色
                    backgroundColor: '#FFE0E0',
                };
            default:
                return {
                    label: t('environment.unknown') || '未知',
                    color: '#9E9E9E', // 灰色
                    backgroundColor: '#F5F5F5',
                };
        }
    };

    const config = getEnvironmentConfig(mode);

    if (mode === 'unknown' || !mode) {
        return null; // 不显示未知环境
    }

    return (
        <Chip
            label={config.label}
            size="small"
            style={{
                backgroundColor: config.backgroundColor,
                color: config.color,
                fontWeight: 'bold',
                marginRight: 8,
                borderRadius: 5,
                lineHeight: '16px',
                border: `1px solid ${config.color}`,
                fontSize: '12px',
            }}
        />
    );
};

@observer
class EnvironmentIndicator extends Component<Stores<'appConfigStore' | 'currentUser'>, IState> {
    public state: IState = {
        initialized: false,
    };

    public componentDidMount() {
        this.initializeConfig();
    }

    public componentDidUpdate(prevProps: Stores<'appConfigStore' | 'currentUser'>) {
        // 当用户登录状态改变时，重新初始化配置
        if (prevProps.currentUser.loggedIn !== this.props.currentUser.loggedIn) {
            this.setState({initialized: false});
            this.initializeConfig();
        }
    }

    private async initializeConfig() {
        // 只在用户已登录时才进行 API 调用
        if (!this.state.initialized && this.props.currentUser.loggedIn) {
            await this.props.appConfigStore.fetchAppConfig();
            this.setState({initialized: true});
        }
    }

    public render() {
        const {appConfigStore} = this.props;
        const mode = appConfigStore.getEnvironmentMode();

        return <EnvironmentIndicatorInner mode={mode} />;
    }
}

export default inject('appConfigStore', 'currentUser')(EnvironmentIndicator);
