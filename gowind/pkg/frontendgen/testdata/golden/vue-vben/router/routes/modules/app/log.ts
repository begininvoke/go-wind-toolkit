import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const log: RouteRecordRaw[] = [
  {
    path: '/log',
    name: 'LogManagement',
    component: BasicLayout,
    redirect: '/log/api-audit-logs',
    meta: {
      order: 2001,
      icon: 'lucide:folder',
      title: $t('menu.log.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'api-audit-logs',
        name: 'ApiAuditLogManagement',
        meta: {
          order: 1,
          icon: 'lucide:route',
          title: $t('menu.log.apiAuditLog'),
        },
        component: () => import('#/views/app/log/api_audit_log/index.vue'),
      },
    ],
  },
];

export default log;
