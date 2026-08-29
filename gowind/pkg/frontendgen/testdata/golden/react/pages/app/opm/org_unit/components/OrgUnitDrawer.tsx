import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_OrgUnit as OrgUnit } from '@/api/generated/admin/service/v1';
import { useCreateOrgUnit, useUpdateOrgUnit } from '@/api/hooks/org-unit';

interface OrgUnitDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: OrgUnit;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * OrgUnit 编辑/创建抽屉组件
 */
const OrgUnitDrawer: React.FC<OrgUnitDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('org-unit');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  const createMutation = useCreateOrgUnit({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listOrgUnits'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  const updateMutation = useUpdateOrgUnit({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listOrgUnits'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values });
      }
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('create') : t('edit')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      initialValues={
        mode === 'edit'
          ? { ...data }
          : {
        sortOrder: 1,
            }
      }
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        width: 480,
      }}
    >
      <ProFormText
        name="leader"
        label={t('leader')}
        placeholder={t('leaderPlaceholder')}
        rules={[{ required: true, message: t('requiredLeader') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[{ required: true, message: t('requiredName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="parentId"
        label={t('parentId')}
        placeholder={t('parentIdPlaceholder')}
        rules={[{ required: true, message: t('requiredParentId') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />
    </DrawerForm>
  );
};

export default OrgUnitDrawer;
