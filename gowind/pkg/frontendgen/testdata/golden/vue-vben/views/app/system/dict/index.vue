<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type dictservicev1_DictType as DictType } from '#/api';
import {
  fetchListDictTypes,
  PaginationQuery,
  useDeleteDictType,
  statusList,
  statusToColor,
  statusToName,
  enableBoolToColor,
  enableBoolToName,
} from '#/api';
import { $t } from '#/locales';

import DictTypeDrawer from './dict-type-drawer.vue';

const { mutateAsync: deleteDictType } = useDeleteDictType();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.dictType.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.dictType.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.dictType.status'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        options: statusList,
        showSearch: true,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
      },
    },
  ],
};

const gridOptions: VxeGridProps<DictType> = {
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
        return await fetchListDictTypes(
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
      title: $t('page.dictType.code'),
      field: 'code',
      minWidth: 120,
    },
    {
      title: $t('page.dictType.isEnabled'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      minWidth: 50,
    },
    {
      title: $t('page.dictType.name'),
      field: 'name',
      minWidth: 120,
    },
    {
      title: $t('ui.table.description'),
      field: 'remark',
      minWidth: 120,
    },
    {
      title: $t('ui.table.sortOrder'),
      field: 'sortOrder',
      minWidth: 100,
    },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 95,
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
  connectedComponent: DictTypeDrawer,

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

/* 删除 */
async function handleDelete(row: any) {
  try {
    await deleteDictType({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.dict.dictType')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.dictType.button.create') }}
        </a-button>
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="enableBoolToColor(row.isEnabled)">
          {{ enableBoolToName(row.isEnabled) }}
        </a-tag>
      </template>
      <template #status="{ row }">
        <a-tag :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
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
              moduleName: $t('page.dictType.moduleName'),
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
