import React from 'react';
import SystemSettingsDialog from './SystemSettingsDialog';
import useTranslation from '../i18n/useTranslation';

interface SystemSettingsDialogWrapperProps {
    open: boolean;
    onClose: () => void;
    token: string;
}

const SystemSettingsDialogWrapper: React.FC<SystemSettingsDialogWrapperProps> = (props) => {
    const {t} = useTranslation();
    
    return <SystemSettingsDialog {...props} t={t} />;
};

export default SystemSettingsDialogWrapper; 