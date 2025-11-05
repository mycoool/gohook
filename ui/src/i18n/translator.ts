import zhTranslations from './locales/zh.json';
import enTranslations from './locales/en.json';

type TranslationKey = string;
type TranslationParams = Record<string, string | number>;

const translations = {
    zh: zhTranslations,
    en: enTranslations,
};

const getNestedValue = (obj: unknown, path: string): string | undefined =>
    path.split('.').reduce((current: unknown, prop: string) => {
        if (current && typeof current === 'object' && prop in current) {
            return (current as Record<string, unknown>)[prop];
        }
        return undefined;
    }, obj) as string | undefined;

const interpolate = (text: string, params?: TranslationParams): string => {
    if (!params) return text;

    return Object.keys(params).reduce((result, key) => {
        const regex = new RegExp(`{{${key}}}`, 'g');
        return result.replace(regex, String(params[key]));
    }, text);
};

const detectLanguage = (): keyof typeof translations => {
    try {
        const storedLang =
            typeof window !== 'undefined' ? localStorage.getItem('i18nextLng') : null;
        if (
            storedLang &&
            (storedLang === 'zh' || storedLang === 'en' || storedLang.startsWith('zh'))
        ) {
            return storedLang.startsWith('zh') ? 'zh' : 'en';
        }
        if (typeof navigator !== 'undefined') {
            const browserLang = navigator.language || navigator.languages?.[0] || 'en';
            if (browserLang.startsWith('zh')) {
                return 'zh';
            }
        }
    } catch (error) {
        console.warn('Failed to detect language, falling back to English.', error);
    }

    return 'en';
};

export const translate = (key: TranslationKey, params?: TranslationParams): string => {
    const currentLanguage = detectLanguage();

    let translation = getNestedValue(translations[currentLanguage], key);

    if (!translation) {
        translation = getNestedValue(translations.en, key);
    }

    if (!translation) {
        console.warn(`Missing translation for key: ${key} in language: ${currentLanguage}`);
        return key;
    }

    return interpolate(translation, params);
};

export default translate;
