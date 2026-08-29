import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const permission: RouteRecordRaw[] = [
  {
    path: "/permission",
    name: "PermissionManagement",
    component: Layout,
    redirect: "/permission/role",
    meta: {
      order: 2000,
      icon: "lucide:folder",
      title: "routes.permission.moduleName",
      keepAlive: true,
    },
    children: [
      {
        path: "role",
        name: "RoleManagement",
        meta: {
          order: 1,
          icon: "lucide:shield-user",
          title: "routes.permission.role",
        },
        component: () => import("@/pages/app/permission/role/index.vue"),
      },
    ],
  },
];

export default permission;
