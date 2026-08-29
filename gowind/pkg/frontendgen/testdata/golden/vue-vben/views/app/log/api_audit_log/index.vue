<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type auditservicev1_ApiAuditLog as ApiAuditLog } from '#/api';
import {
  fetchListApiAuditLogs,
  PaginationQuery,
  enableBoolToColor,
  enableBoolToName,
} from '#/api';
import { $t } from '#/locales';

import ApiAuditLogDrawer from './api-audit-log-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'duration',
      label: $t('page.apiAuditLog.duration'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'ipAddress',
      label: $t('page.apiAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'operatorName',
      label: $t('page.apiAuditLog.operatorName'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'requestMethod',
      label: $t('page.apiAuditLog.requestMethod'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'requestPath',
      label: $t('page.apiAuditLog.requestPath'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<ApiAuditLog> = {
  height: 'auto',
  stripe: false,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchListApiAuditLogs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    {
      title: $t('page.apiAuditLog.duration'),
      field: 'duration',
      minWidth: 120,
    },
    {
      title: $t('page.apiAuditLog.ipAddress'),
      field: 'ipAddress',
      minWidth: 120,
    },
    {
      title: $t('page.apiAuditLog.isSuccess'),
      field: 'isSuccess',
      slots: { default: 'isSuccess' },
      minWidth: 50,
    },
    {
      title: $t('ui.table.operatedAt'),
      field: 'operatedAt',
      formatter: 'formatDateTime',
      minWidth: 140,
    },
    {
      title: $t('page.apiAuditLog.operatorName'),
      field: 'operatorName',
      minWidth: 120,
    },
    {
      title: $t('page.apiAuditLog.requestMethod'),
      field: 'requestMethod',
      minWidth: 120,
    },
    {
      title: $t('page.apiAuditLog.requestPath'),
      field: 'requestPath',
      minWidth: 120,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 90,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: ApiAuditLogDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });
  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.api_audit_log.apiAuditLog')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.apiAuditLog.button.create') }}
        </a-button>
      </template>
      <template #isSuccess="{ row }">
        <a-tag :color="enableBoolToColor(row.isSuccess)">
          {{ enableBoolToName(row.isSuccess) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.apiAuditLog.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
