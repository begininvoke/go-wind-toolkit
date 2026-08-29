import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const log: RouteRecordRaw[] = [
  {
    path: "/log",
    name: "LogManagement",
    component: Layout,
    redirect: "/log/api-audit-log",
    meta: {
      order: 2000,
      icon: "lucide:folder",
      title: "routes.log.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "api-audit-log",
        name: "Api-audit-logManagement",
        meta: {
          order: 1,
          icon: "lucide:file-text",
          title: "routes.log.api-audit-log",
        },
        component: () => import("@/pages/app/log/api-audit-log/index.vue"),
      },
    ],
  },
];

export default log;
