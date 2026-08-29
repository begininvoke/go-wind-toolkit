import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const opm: RouteRecordRaw[] = [
  {
    path: '/opm',
    name: 'OpmManagement',
    component: BasicLayout,
    redirect: '/opm/org-units',
    meta: {
      order: 2001,
      icon: 'lucide:folder',
      title: $t('menu.opm.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'org-units',
        name: 'OrgUnitManagement',
        meta: {
          order: 1,
          icon: 'lucide:building',
          title: $t('menu.opm.orgUnit'),
        },
        component: () => import('#/views/app/opm/org_unit/index.vue'),
      },
    ],
  },
];

export default opm;
