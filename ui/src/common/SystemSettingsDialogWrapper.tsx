import React from 'react';
import SystemSettingsDialog from './SystemSettingsDialog';
import useTranslation from '../i18n/useTranslation';
import {inject, Stores} from '../inject';

interface SystemSettingsDialogWrapperProps {
    open: boolean;
    onClose: () => void;
    token: string;
}

const SystemSettingsDialogWrapperInner: React.FC<
    SystemSettingsDialogWrapperProps & Stores<'appConfigStore'>
> = (props) => {
    const {t} = useTranslation();

    const handleConfigSaved = () => {
        // 刷新 AppConfigStore 以更新环境指示器
        props.appConfigStore.fetchAppConfig();
    };

    return <SystemSettingsDialog {...props} t={t} onConfigSaved={handleConfigSaved} />;
};

export default inject('appConfigStore')(SystemSettingsDialogWrapperInner);
