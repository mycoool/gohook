// GoHook UI 配色规范
// 黑白灰主题配色系统

export const colors = {
  // 主要配色
  primary: {
    // 黑色系 - 用于主要内容和强调
    black: '#000000',           // 纯黑
    darkGray: '#2c2c2c',       // 深灰 - 用于深色背景
    mediumGray: '#424242',     // 中灰 - 用于主要文字
    lightGray: '#616161',      // 浅灰 - 用于次要文字
  },

  // 背景配色
  background: {
    // 白色系 - 用于背景
    white: '#ffffff',          // 纯白
    lightGray: '#fafafa',      // 浅灰背景
    mediumGray: '#f5f5f5',     // 中浅灰背景
    overlay: '#f0f0f0',        // 覆盖层背景
  },

  // 边框配色
  border: {
    light: '#e0e0e0',          // 浅色边框
    medium: '#d0d0d0',         // 中等边框
    dark: '#bdbdbd',           // 深色边框
    contrast: '#555555',       // 高对比度边框（用于深色背景）
  },

  // 文字配色
  text: {
    primary: '#212121',        // 主要文字
    secondary: '#424242',      // 次要文字
    disabled: '#9e9e9e',       // 禁用文字
    onDark: '#e0e0e0',         // 深色背景上的文字
    onDarkSecondary: '#bdbdbd', // 深色背景上的次要文字
  },

  // 状态配色
  status: {
    // 信息提示
    info: {
      background: '#2c2c2c',   // 深色背景确保可见性
      border: '#555555',       // 中等对比度边框
      text: '#e0e0e0',         // 浅色文字确保可读性
    },
    
    // 警告
    warning: {
      background: '#3c3c3c',
      border: '#666666',
      text: '#f5f5f5',
    },
    
    // 错误
    error: {
      background: '#2c1f1f',
      border: '#5c2c2c',
      text: '#ffcdd2',
    },
    
    // 成功
    success: {
      background: '#1f2c1f',
      border: '#2c5c2c',
      text: '#c8e6c9',
    },
  },

  // 交互元素配色
  interactive: {
    // 按钮
    button: {
      command: '#616161',      // 命令模式按钮
      script: '#424242',       // 脚本模式按钮
      hover: '#757575',        // 悬停状态
      disabled: '#e0e0e0',     // 禁用状态
    },
    
    // 输入框
    input: {
      background: '#fafafa',
      border: '#d0d0d0',
      focus: '#9e9e9e',
      text: '#212121',
    },

    // 代码标签
    code: {
      background: '#2c2c2c',   // 深色背景确保可见性
      text: '#e0e0e0',         // 浅色文字
      padding: '2px 6px',      // 内边距
      borderRadius: 3,         // 圆角
      fontSize: '0.875rem',    // 字体大小
    },
  },
};

// 配色使用规范
export const colorUsageGuidelines = {
  // 可见性规则
  visibility: {
    // 高对比度组合 - 确保可读性
    highContrast: [
      { bg: colors.primary.darkGray, text: colors.text.onDark },
      { bg: colors.background.white, text: colors.text.primary },
      { bg: colors.primary.black, text: colors.background.white },
    ],
    
    // 避免使用的低对比度组合
    avoid: [
      { bg: colors.background.lightGray, text: colors.text.disabled },
      { bg: colors.background.overlay, text: colors.text.secondary },
    ],
  },

  // 提示框规范
  messageBox: {
    // 信息提示 - 使用深色背景确保可见性
    info: {
      backgroundColor: colors.status.info.background,
      border: `1px solid ${colors.status.info.border}`,
      color: colors.status.info.text,
      usage: '用于一般信息提示，确保在任何背景下都清晰可见',
    },
    
    // 示例代码框
    example: {
      backgroundColor: colors.background.mediumGray,
      border: `1px solid ${colors.border.medium}`,
      color: colors.text.secondary,
      usage: '用于代码示例展示',
    },
  },

  // 按钮规范
  buttons: {
    toggleActive: {
      backgroundColor: colors.interactive.button.command,
      color: colors.background.white,
      usage: '激活状态的切换按钮',
    },
    
    toggleInactive: {
      backgroundColor: 'transparent',
      border: `1px solid ${colors.border.medium}`,
      color: colors.text.secondary,
      usage: '非激活状态的切换按钮',
    },
  },
};

// 导出默认配色对象
export default colors; 