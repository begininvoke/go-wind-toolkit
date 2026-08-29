import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_ApiAuditLog as ApiAuditLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListApiAuditLogs, useDeleteApiAuditLog } from '@/api/hooks/api-audit-log';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import ApiAuditLogDrawer from './components/ApiAuditLogDrawer';

/**
 * API审计日志查询服务
 */
const ApiAuditLogManagement = () => {
  const { t } = useTranslation('api-audit-log');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedApiAuditLog, setSelectedApiAuditLog] = useState<ApiAuditLog | undefined>();

  // 删除操作
  const deleteMutation = useDeleteApiAuditLog({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listApiAuditLogs'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<ApiAuditLog>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || TABLE.DEFAULT_PAGE_SIZE;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('duration'),
      dataIndex: 'duration',
      width: 150,
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 150,
    },
    {
      title: t('isSuccess'),
      dataIndex: 'isSuccess',
      width: 100,
      hideInSearch: true,
      render: (_, record) => {
        const val = record.isSuccess as boolean;
        return <Tag color={val ? 'success' : 'error'}>{val ? t('yes') : t('no')}</Tag>;
      },
    },
    {
      title: t('operatedAt'),
      dataIndex: 'operatedAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('operatorName'),
      dataIndex: 'operatorName',
      width: 150,
    },
    {
      title: t('requestMethod'),
      dataIndex: 'requestMethod',
      width: 120,
      hideInSearch: true,
    },
    {
      title: t('requestPath'),
      dataIndex: 'requestPath',
      width: 150,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 100,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setDrawerMode('edit');
            setSelectedApiAuditLog(record);
            setDrawerOpen(true);
          }}
        >
          <EditOutlined />
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { moduleName: t('moduleName') })}
          onConfirm={() => record.id && deleteMutation.mutate({ id: record.id })}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }}><DeleteOutlined /></a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ProTable<ApiAuditLog>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              try {
                const query = new PaginationQuery({
                  paging: {
                    page: params.current || 1,
                    pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                  },
                  formValues: Object.fromEntries(
                    Object.entries(params).filter(
                      ([key]) => !['current', 'pageSize'].includes(key),
                    ),
                  ),
                });

                const response = await fetchListApiAuditLogs(query);

                return {
                  data: response.items || [],
                  total: response.total || 0,
                  success: true,
                };
              } catch (error: any) {
                message.error(error.message || t('fetchFailed'));
                return { data: [], total: 0, success: false };
              }
            }}
            rowKey="id"
            search={{
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedApiAuditLog(undefined);
                  setDrawerOpen(true);
                }}
              >
                {t('create')}
              </Button>,
            ]}
            options={{
              density: true,
              fullScreen: true,
              setting: true,
              reload: true,
            }}
            size="middle"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY, x: 1000 }}
          />
        </div>
      </ContentContainer>

      {/* ApiAuditLog 编辑/创建 Drawer */}
      <ApiAuditLogDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedApiAuditLog}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedApiAuditLog(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default ApiAuditLogManagement;
