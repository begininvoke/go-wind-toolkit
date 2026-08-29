import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 系统管理路由配置
 */
export const systemRoutes: AppRouteObject[] = [
  {
    name: 'system',
    path: 'system',
    meta: {
      title: 'routes:system',
      icon: 'lucide:folder',
      order: 2000,
      keepAlive: true,
    },
    children: [
      {
        name: 'system-dicttype',
        path: 'dict-type',
        element: createLazyRoute(() => import('@/pages/app/system/dict-type')),
        meta: {
          title: 'routes:system-dict-type',
          icon: 'lucide:library-big',
          order: 1,
        },
      },
    ],
  },
];

export default systemRoutes;
