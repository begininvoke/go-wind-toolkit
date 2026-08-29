/**
 * DictType 模块常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

/** 状态映射 */
export function getStatusMap(t: TFn) {
  return {
    ON: { text: t('启用'), color: 'success' },
    OFF: { text: t('禁用'), color: 'error' },
  };
}

/** 状态选项 */
export function getStatusOptions(t: TFn) {
  return [
    { label: t('启用'), value: 'ON' },
    { label: t('禁用'), value: 'OFF' },
  ];
}
