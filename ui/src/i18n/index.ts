// 临时注释依赖，待npm安装完成后恢复
/*
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import zh from './locales/zh.json';
import en from './locales/en.json';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      zh: { 
        translation: zh 
      },
      'zh-CN': { 
        translation: zh 
      },
      'zh-TW': { 
        translation: zh 
      },
      en: { 
        translation: en 
      },
      'en-US': { 
        translation: en 
      },
      'en-GB': { 
        translation: en 
      }
    },
    // 设置语言回退规则：优先中文，缺失时使用英文
    fallbackLng: {
      'zh': ['zh', 'en'],
      'zh-CN': ['zh', 'en'], 
      'zh-TW': ['zh', 'en'],
      'default': ['en']
    },
    // 默认语言设为中文
    lng: 'zh',
    
    debug: process.env.NODE_ENV === 'development',
    
    interpolation: {
      escapeValue: false, // React已经处理了XSS
    },
    
    detection: {
      // 检测顺序：localStorage > 浏览器语言 > 默认语言
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupLocalStorage: 'i18nextLng',
    },
    
    // 当翻译缺失时的处理
    saveMissing: process.env.NODE_ENV === 'development',
    missingKeyHandler: (lng: any, ns: any, key: any, fallbackValue: any) => {
      if (process.env.NODE_ENV === 'development') {
        console.warn(`Missing translation: ${lng}.${key}`);
      }
    }
  });

export default i18n;
*/

// 临时导出空对象，使用自定义的useTranslation Hook
export default {};
