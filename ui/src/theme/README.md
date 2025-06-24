# GoHook UI 配色规范使用指南

## 概述

本项目采用黑白灰主题配色系统，确保界面在各种环境下都有良好的可读性和一致性。所有颜色值都定义在 `colors.ts` 文件中，组件开发时必须使用这些预定义的颜色值。

## 配色原则

### 1. 可见性优先
- **高对比度**：深色背景配浅色文字，浅色背景配深色文字
- **避免低对比度组合**：如浅灰背景配灰色文字

### 2. 一致性保障
- 所有组件使用统一的颜色变量
- 相同功能的元素使用相同配色
- 避免硬编码颜色值

### 3. 主题适配
- 支持明暗主题切换
- 颜色值在不同主题下保持良好对比度

## 使用方法

### 导入配色
```typescript
import { colors } from '../theme/colors';
```

### 基础用法
```typescript
// ✅ 正确：使用预定义颜色
backgroundColor: colors.status.info.background,
color: colors.status.info.text,

// ❌ 错误：硬编码颜色
backgroundColor: '#2c2c2c',
color: '#e0e0e0',
```

## 核心配色分类

### 1. 主要配色 (primary)
- `black`: 纯黑 - 用于强调元素
- `darkGray`: 深灰 - 用于深色背景
- `mediumGray`: 中灰 - 用于主要文字
- `lightGray`: 浅灰 - 用于次要文字

### 2. 背景配色 (background)
- `white`: 纯白 - 主背景
- `lightGray`: 浅灰背景 - 输入框背景
- `mediumGray`: 中浅灰背景 - 示例代码框
- `overlay`: 覆盖层背景 - 模态框背景

### 3. 边框配色 (border)
- `light`: 浅色边框 - 一般分割线
- `medium`: 中等边框 - 输入框边框
- `dark`: 深色边框 - 强调边框
- `contrast`: 高对比度边框 - 深色背景上的边框

### 4. 文字配色 (text)
- `primary`: 主要文字 - 标题、重要内容
- `secondary`: 次要文字 - 说明文字
- `disabled`: 禁用文字 - 不可操作内容
- `onDark`: 深色背景上的文字 - 确保可读性
- `onDarkSecondary`: 深色背景上的次要文字

### 5. 状态配色 (status)
- `info`: 信息提示框 - 深色背景+浅色文字
- `warning`: 警告提示框
- `error`: 错误提示框
- `success`: 成功提示框

### 6. 交互元素配色 (interactive)
- `button`: 按钮相关配色
- `input`: 输入框相关配色
- `code`: 代码标签配色 - 深色背景+浅色文字确保可见性

## 常见使用场景

### 提示框/消息框
```typescript
// 信息提示框 - 确保高可见性
<Box style={{
  backgroundColor: colors.status.info.background,
  border: `1px solid ${colors.status.info.border}`,
  color: colors.status.info.text
}}>
```

### 示例代码框
```typescript
// 代码示例展示
<Box style={{
  backgroundColor: colors.background.mediumGray,
  border: `1px solid ${colors.border.medium}`,
  color: colors.text.secondary
}}>
```

### 按钮状态
```typescript
// 激活状态按钮
backgroundColor: colors.interactive.button.command,
color: colors.background.white,

// 非激活状态按钮
backgroundColor: 'transparent',
border: `1px solid ${colors.border.medium}`,
color: colors.text.secondary,
```

### 代码标签
```typescript
// 统一的代码标签样式 - 确保高可见性
const codeStyle = {
  backgroundColor: colors.interactive.code.background,
  color: colors.interactive.code.text,
  padding: colors.interactive.code.padding,
  borderRadius: colors.interactive.code.borderRadius,
  fontSize: colors.interactive.code.fontSize,
};

// 使用方式
<code style={codeStyle}>your code here</code>
```

## 避免的错误

### ❌ 错误示例
```typescript
// 1. 硬编码颜色值
backgroundColor: '#f0f0f0',
color: '#424242',

// 2. 低对比度组合
backgroundColor: '#f5f5f5',  // 浅灰背景
color: '#9e9e9e',           // 浅灰文字 - 不易阅读

// 3. 不一致的颜色使用
// 在不同组件中为相同功能使用不同颜色
```

### ✅ 正确示例
```typescript
// 1. 使用预定义颜色
backgroundColor: colors.status.info.background,
color: colors.status.info.text,

// 2. 高对比度组合
backgroundColor: colors.primary.darkGray,  // 深色背景
color: colors.text.onDark,                // 浅色文字 - 清晰可读

// 3. 一致性使用
// 所有信息提示框都使用 colors.status.info 配色
```

## 扩展配色

如需添加新颜色，请在 `colors.ts` 中添加，并更新此文档说明用途：

```typescript
// 在 colors.ts 中添加
export const colors = {
  // ... 现有配色
  newCategory: {
    newColor: '#xxxxxx',  // 说明用途
  }
};
```

## 检查清单

在组件开发完成后，请检查：

- [ ] 所有颜色都使用预定义变量
- [ ] 文字与背景有足够对比度
- [ ] 在明暗主题下都清晰可见
- [ ] 与其他组件保持视觉一致性
- [ ] 遵循本文档的配色规范

## 总结

遵循配色规范可以：
1. **提高可见性**：确保所有用户都能清晰阅读界面内容
2. **保持一致性**：统一的视觉体验
3. **便于维护**：集中管理颜色，易于主题切换
4. **避免错误**：防止出现低对比度等可用性问题

记住：**可见性比美观更重要**，确保用户能够清晰地看到和理解界面内容是第一要务。 