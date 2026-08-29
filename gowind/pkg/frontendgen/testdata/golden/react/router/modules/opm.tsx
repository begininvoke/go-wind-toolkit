import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 组织人员路由配置
 */
export const opmRoutes: AppRouteObject[] = [
  {
    name: 'opm',
    path: 'opm',
    meta: {
      title: 'routes:opm',
      icon: 'lucide:folder',
      order: 2000,
      keepAlive: true,
    },
    children: [
      {
        name: 'opm-orgunit',
        path: 'org-unit',
        element: createLazyRoute(() => import('@/pages/app/opm/org-unit')),
        meta: {
          title: 'routes:opm-org-unit',
          icon: 'lucide:layers',
          order: 1,
        },
      },
    ],
  },
];

export default opmRoutes;
