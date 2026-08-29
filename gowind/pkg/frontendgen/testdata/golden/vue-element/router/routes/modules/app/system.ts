import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const system: RouteRecordRaw[] = [
  {
    path: "/system",
    name: "SystemManagement",
    component: Layout,
    redirect: "/system/dict-type",
    meta: {
      order: 2000,
      icon: "lucide:folder",
      title: "routes.system.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "dict-type",
        name: "Dict-typeManagement",
        meta: {
          order: 1,
          icon: "lucide:library-big",
          title: "routes.system.dict-type",
        },
        component: () => import("@/pages/app/system/dict-type/index.vue"),
      },
    ],
  },
];

export default system;
