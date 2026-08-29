import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const system: RouteRecordRaw[] = [
  {
    path: '/system',
    name: 'SystemManagement',
    component: BasicLayout,
    redirect: '/system/dict-types',
    meta: {
      order: 2001,
      icon: 'lucide:folder',
      title: $t('menu.system.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'dict-types',
        name: 'DictTypeManagement',
        meta: {
          order: 1,
          icon: 'lucide:library-big',
          title: $t('menu.system.dictType'),
        },
        component: () => import('#/views/app/system/dict/index.vue'),
      },
    ],
  },
];

export default system;
