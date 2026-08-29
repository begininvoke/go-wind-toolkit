import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const opm: RouteRecordRaw[] = [
  {
    path: "/opm",
    name: "OpmManagement",
    component: Layout,
    redirect: "/opm/org-unit",
    meta: {
      order: 2000,
      icon: "lucide:folder",
      title: "routes.opm.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "org-unit",
        name: "Org-unitManagement",
        meta: {
          order: 1,
          icon: "lucide:layers",
          title: "routes.opm.org-unit",
        },
        component: () => import("@/pages/app/opm/org-unit/index.vue"),
      },
    ],
  },
];

export default opm;
