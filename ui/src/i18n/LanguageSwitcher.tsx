import React from 'react';
import { IconButton, Menu, MenuItem, Tooltip } from '@material-ui/core';
import { GTranslate as LanguageIcon } from '@material-ui/icons';
// import { useTranslation } from 'react-i18next';

interface LanguageOption {
  code: string;
  name: string;
  flag: string;
}

const languages: LanguageOption[] = [
  { code: 'zh', name: '中文', flag: '🇨🇳' },
  { code: 'en', name: 'English', flag: '🇺🇸' }
];

export const LanguageSwitcher: React.FC = () => {
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
  // const { i18n } = useTranslation();
  
  // 临时使用localStorage直接获取语言
  const getCurrentLanguage = () => (
    localStorage.getItem('i18nextLng') ?? 'zh'
  );

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleLanguageChange = (languageCode: string) => {
    // 临时直接设置localStorage，待i18n配置完成后改为i18n.changeLanguage
    localStorage.setItem('i18nextLng', languageCode);
    // i18n.changeLanguage(languageCode);
    handleClose();
    // 刷新页面以应用语言更改
    window.location.reload();
  };

  const currentLang = getCurrentLanguage();

  return (
    <>
      <Tooltip title="切换语言 / Switch Language">
        <IconButton color="inherit" onClick={handleClick}>
          <LanguageIcon />
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
        getContentAnchorEl={null}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'center',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'center',
        }}
      >
        {languages.map((language) => (
          <MenuItem
            key={language.code}
            onClick={() => handleLanguageChange(language.code)}
            selected={language.code === currentLang}
          >
            <span style={{ marginRight: 8 }}>{language.flag}</span>
            {language.name}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default LanguageSwitcher; 