import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const permission: RouteRecordRaw[] = [
  {
    path: '/permission',
    name: 'PermissionManagement',
    component: BasicLayout,
    redirect: '/permission/roles',
    meta: {
      order: 2001,
      icon: 'lucide:folder',
      title: $t('menu.permission.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'roles',
        name: 'RoleManagement',
        meta: {
          order: 1,
          icon: 'lucide:shield-user',
          title: $t('menu.permission.role'),
        },
        component: () => import('#/views/app/permission/role/index.vue'),
      },
    ],
  },
];

export default permission;
