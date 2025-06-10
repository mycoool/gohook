import { useMemo } from 'react';
// import { useTranslation as useI18nTranslation } from 'react-i18next';

// 导入语言包
import zhTranslations from './locales/zh.json';
import enTranslations from './locales/en.json';

type TranslationKey = string;
type TranslationParams = Record<string, string | number>;

interface TranslationFunction {
  (key: TranslationKey, params?: TranslationParams): string;
}

interface UseTranslationReturn {
  t: TranslationFunction;
  i18n: {
    language: string;
    changeLanguage: (lng: string) => void;
  };
}

// 临时翻译实现，待依赖安装后替换为react-i18next
export const useTranslation = (): UseTranslationReturn => {
  // const { t: i18nT, i18n } = useI18nTranslation();
  
  const currentLanguage = useMemo(() => {
    // 检测浏览器语言
    const browserLang = navigator.language || 'zh';
    const storedLang = localStorage.getItem('i18nextLng');
    
    if (storedLang) {
      return storedLang;
    }
    
    // 根据浏览器语言确定默认语言
    if (browserLang.startsWith('zh')) {
      return 'zh';
    } else if (browserLang.startsWith('en')) {
      return 'en';
    }
    
    // 默认英文（按照用户需求）
    return 'en';
  }, []);

  const translations = useMemo(() => ({
    zh: zhTranslations,
    en: enTranslations
  }), []);

  const t: TranslationFunction = (key: TranslationKey, params?: TranslationParams) => {
    const getNestedValue = (obj: unknown, path: string): string | undefined => (
      path.split('.').reduce((current: unknown, prop: string) => {
        if (current && typeof current === 'object' && prop in current) {
          return (current as Record<string, unknown>)[prop];
        }
        return undefined;
      }, obj) as string | undefined
    );

    const interpolate = (text: string, params?: TranslationParams): string => {
      if (!params) return text;
      
      return Object.keys(params).reduce((result, key) => {
        const regex = new RegExp(`{{${key}}}`, 'g');
        return result.replace(regex, String(params[key]));
      }, text);
    };

    // 首先尝试当前语言
    let translation = getNestedValue(translations[currentLanguage as keyof typeof translations], key);
    
    // 如果当前语言找不到，根据用户需求回退到英文
    if (!translation) {
      translation = getNestedValue(translations.en, key);
    }
    
    // 如果还是找不到，返回key本身
    if (!translation) {
      console.warn(`Missing translation for key: ${key} in language: ${currentLanguage}`);
      return key;
    }

    return interpolate(translation, params);
  };

  const changeLanguage = (lng: string) => {
    localStorage.setItem('i18nextLng', lng);
    window.location.reload();
  };

  return {
    t,
    i18n: {
      language: currentLanguage,
      changeLanguage
    }
  };
};

export default useTranslation; 