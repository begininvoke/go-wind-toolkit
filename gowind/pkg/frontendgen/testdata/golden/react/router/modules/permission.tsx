import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 权限管理路由配置
 */
export const permissionRoutes: AppRouteObject[] = [
  {
    name: 'permission',
    path: 'permission',
    meta: {
      title: 'routes:permission',
      icon: 'lucide:folder',
      order: 2000,
      keepAlive: true,
    },
    children: [
      {
        name: 'permission-role',
        path: 'role',
        element: createLazyRoute(() => import('@/pages/app/permission/role')),
        meta: {
          title: 'routes:permission-role',
          icon: 'lucide:shield-user',
          order: 1,
        },
      },
    ],
  },
];

export default permissionRoutes;
