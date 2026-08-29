import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 日志审计路由配置
 */
export const logRoutes: AppRouteObject[] = [
  {
    name: 'log',
    path: 'log',
    meta: {
      title: 'routes:log',
      icon: 'lucide:folder',
      order: 2000,
      keepAlive: true,
    },
    children: [
      {
        name: 'log-apiauditlog',
        path: 'api-audit-log',
        element: createLazyRoute(() => import('@/pages/app/log/api-audit-log')),
        meta: {
          title: 'routes:log-api-audit-log',
          icon: 'lucide:file-text',
          order: 1,
        },
      },
    ],
  },
];

export default logRoutes;
