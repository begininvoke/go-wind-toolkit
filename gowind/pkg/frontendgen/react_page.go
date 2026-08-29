package frontendgen

import "strings"

// ==============================
// 字段筛选辅助（react）
// ==============================

var reactSearchSkipFields = map[string]bool{
	"id": true, "description": true, "remark": true, "sortOrder": true,
	"isDefault": true, "isEnabled": true,
}

func reactSearchFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if !reactSearchSkipFields[f.Name] && !f.IsArray && !f.IsDate {
			out = append(out, *f)
		}
	}
	return out
}

func reactTableFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if f.Name != "id" && !f.IsArray {
			out = append(out, *f)
		}
	}
	return out
}

// reactColumnsCode 生成 ProTable 列定义
func reactColumnsCode(service *ParsedService, modelPascal string) string {
	searchFields := reactSearchFields(service.Fields)
	isSearchField := func(name string) bool {
		for i := range searchFields {
			if searchFields[i].Name == name {
				return true
			}
		}
		return false
	}

	var lines []string

	// 序号列
	lines = append(lines, `    {
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
    },`)

	for i := range reactTableFields(service.Fields) {
		field := &reactTableFields(service.Fields)[i]
		lower := strings.ToLower(field.Name)

		switch {
		case field.IsBoolean:
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 100,
      hideInSearch: true,
      render: (_, record) => {
        const val = record.`+field.Name+` as boolean;
        return <Tag color={val ? 'success' : 'error'}>{val ? t('yes') : t('no')}</Tag>;
      },
    },`)
		case field.IsEnum && len(field.EnumValues) > 0 && strings.Contains(lower, "status"):
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: getStatusOptions(t),
      },
      render: (_, record) => {
        const statusMap = getStatusMap(t);
        const status = record.`+field.Name+` as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },`)
		case field.IsEnum && len(field.EnumValues) > 0:
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 120,
      hideInSearch: true,
    },`)
		case field.IsDate:
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },`)
		case field.IsInteger && strings.Contains(lower, "sort"):
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 100,
      hideInSearch: true,
    },`)
		case strings.Contains(lower, "description") || strings.Contains(lower, "remark"):
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      hideInSearch: true,
      ellipsis: true,
    },`)
		default:
			hideInSearch := ""
			if !isSearchField(field.Name) {
				hideInSearch = "\n      hideInSearch: true,"
			}
			lines = append(lines, `    {
      title: t('`+field.Name+`'),
      dataIndex: '`+field.Name+`',
      width: 150,`+hideInSearch+`
    },`)
		}
	}

	// 操作列
	lines = append(lines, `    {
      title: t('action'),
      valueType: 'option',
      width: 100,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setDrawerMode('edit');
            setSelected`+modelPascal+`(record);
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
    },`)

	return strings.Join(lines, "\n")
}

// ==============================
// 列表页 index.tsx
// ==============================

// reactPageCode 生成 ProTable 列表页（对应 TS 版 page-template.ts generatePageCode）
func reactPageCode(service *ParsedService, serviceName string) string {
	modelPascal := toPascalCase(service.ModelName)
	fileName := service.KebabName
	prefix := service.TypePrefix

	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	if !hasList {
		return "// " + service.TagName + " 没有 List 操作，无法生成列表页面"
	}

	hasStatusEnum := anyStatusField(service.Fields)
	columnsCode := reactColumnsCode(service, modelPascal)

	var sb strings.Builder
	sb.WriteString(`import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
`)

	_ = serviceName
	sb.WriteString("import type { " + prefix + "_" + modelPascal + " as " + modelPascal + " } from '@/api/generated/" + serviceName + "/service/v1';\n")
	sb.WriteString(`import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
`)

	sb.WriteString("import { fetchList" + modelPascal + "s, useDelete" + modelPascal + " } from '@/api/hooks/" + fileName + "';\n")

	statusImport := ""
	if hasStatusEnum {
		statusImport = "import { getStatusMap, getStatusOptions } from './constants';\n"
	}

	sb.WriteString(`import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
` + statusImport + `import ` + modelPascal + `Drawer from './components/` + modelPascal + `Drawer';

/**
 * ` + service.Description + `
 */
const ` + modelPascal + `Management = () => {
  const { t } = useTranslation('` + fileName + `');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected` + modelPascal + `, setSelected` + modelPascal + `] = useState<` + modelPascal + ` | undefined>();

  // 删除操作
  const deleteMutation = useDelete` + modelPascal + `({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['list` + modelPascal + `s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<` + modelPascal + `>[] = [
` + columnsCode + `
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ProTable<` + modelPascal + `>
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

                const response = await fetchList` + modelPascal + `s(query);

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
                  setSelected` + modelPascal + `(undefined);
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

      {/* ` + service.ModelName + ` 编辑/创建 Drawer */}
      <` + modelPascal + `Drawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected` + modelPascal + `}
        onClose={() => {
          setDrawerOpen(false);
          setSelected` + modelPascal + `(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default ` + modelPascal + `Management;
`)
	return sb.String()
}

// ==============================
// 编辑抽屉 *Drawer.tsx
// ==============================

// reactFormFieldComponent 推断 ProForm 组件
func reactFormFieldComponent(field *ParsedField) (component, extraProps string) {
	lower := strings.ToLower(field.Name)
	if field.IsBoolean {
		return "ProFormSwitch", ""
	}
	if field.IsEnum && len(field.EnumValues) > 0 {
		if strings.Contains(lower, "status") {
			return "ProFormRadio.Group",
				"        options={getStatusOptions(t)}\n" +
					"        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}"
		}
		return "ProFormSelect", "        options={" + field.Name + "Options}"
	}
	if field.IsInteger && strings.Contains(lower, "sort") {
		return "ProFormDigit", "        fieldProps={{ precision: 0, min: 0 }}"
	}
	if field.IsDate {
		return "ProFormDateTimePicker", "        fieldProps={{ style: { width: '100%' } }}"
	}
	if strings.Contains(lower, "description") || strings.Contains(lower, "remark") {
		return "ProFormTextArea", "        fieldProps={{ allowClear: true, rows: 2 }}"
	}
	return "ProFormText", "        fieldProps={{ allowClear: true }}"
}

// reactDrawerCode 生成 DrawerForm 编辑抽屉
func reactDrawerCode(service *ParsedService, serviceName string) string {
	modelPascal := toPascalCase(service.ModelName)
	fileName := service.KebabName
	prefix := service.TypePrefix

	crudPaths := GetCrudPaths(service)
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil

	// 表单字段（排除系统字段和ID）
	var formFields []ParsedField
	for i := range service.Fields {
		f := &service.Fields[i]
		if f.Name != "id" && !f.IsDate {
			formFields = append(formFields, *f)
		}
	}

	hasStatusEnum := anyStatusField(service.Fields)

	// 收集所有需要 import 的 ProForm 组件（保持出现序去重）
	componentSet := map[string]bool{}
	var formComponents []string
	for i := range formFields {
		component, _ := reactFormFieldComponent(&formFields[i])
		if !componentSet[component] {
			componentSet[component] = true
			formComponents = append(formComponents, component)
		}
	}

	// 生成表单项代码
	var formItemCodes []string
	for i := range formFields {
		f := &formFields[i]
		component, extraProps := reactFormFieldComponent(f)
		isRequired := !f.IsBoolean && f.Name != "sortOrder" && f.Name != "description" && f.Name != "remark"

		var item strings.Builder
		item.WriteString("      <" + component + "\n")
		item.WriteString("        name=\"" + f.Name + "\"\n")
		item.WriteString("        label={t('" + f.Name + "')}\n")
		item.WriteString("        placeholder={t('" + f.Name + "Placeholder')}")
		if isRequired {
			item.WriteString("\n        rules={[{ required: true, message: t('required" + toPascalCase(f.Name) + "') }]}")
		}
		if extraProps != "" {
			item.WriteString("\n" + extraProps)
		}
		item.WriteString("\n      />")
		formItemCodes = append(formItemCodes, item.String())
	}

	// initialValues
	var defaultValues []string
	for i := range formFields {
		f := &formFields[i]
		lower := strings.ToLower(f.Name)
		switch {
		case f.IsBoolean:
			dv := "false"
			if f.Name == "isEnabled" || f.Name == "isDefault" {
				dv = "true"
			}
			defaultValues = append(defaultValues, "        "+f.Name+": "+dv+",")
		case f.IsInteger && strings.Contains(lower, "sort"):
			defaultValues = append(defaultValues, "        "+f.Name+": 1,")
		case f.IsEnum && strings.Contains(lower, "status"):
			defaultValues = append(defaultValues, "        "+f.Name+": 'ON',")
		}
	}

	drawerSize := 480
	if len(formFields) > 8 {
		drawerSize = 600
	}

	var componentImportLines []string
	for _, c := range formComponents {
		componentImportLines = append(componentImportLines, "  "+c+",")
	}

	statusImport := ""
	if hasStatusEnum {
		statusImport = "import { getStatusOptions } from '../constants';\n"
	}

	createBlock := ""
	if hasCreate {
		createBlock = `
  const createMutation = useCreate` + modelPascal + `({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['list` + modelPascal + `s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });
`
	}
	updateBlock := ""
	if hasUpdate {
		updateBlock = `
  const updateMutation = useUpdate` + modelPascal + `({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['list` + modelPascal + `s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });
`
	}

	var submitBranch string
	if hasCreate {
		submitBranch += "      if (mode === 'create') {\n" +
			"        await createMutation.mutateAsync({ data: values });\n" +
			"      }"
	}
	if hasUpdate {
		sep := ""
		if hasCreate {
			sep = " else"
		}
		submitBranch += sep + " if (data?.id) {\n" +
			"        await updateMutation.mutateAsync({ id: data.id, values });\n" +
			"      }"
	}

	loadingExtras := ""
	if hasCreate {
		loadingExtras += " || createMutation.isPending"
	}
	if hasUpdate {
		loadingExtras += " || updateMutation.isPending"
	}

	var sb strings.Builder
	sb.WriteString(`import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
` + strings.Join(componentImportLines, "\n") + `
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
`)

	_ = serviceName
	sb.WriteString("import type { " + prefix + "_" + modelPascal + " as " + modelPascal + " } from '@/api/generated/" + serviceName + "/service/v1';\n")
	sb.WriteString("import { useCreate" + modelPascal + ", useUpdate" + modelPascal + " } from '@/api/hooks/" + fileName + "';\n")
	sb.WriteString(statusImport)
	sb.WriteString("\n")
	sb.WriteString(`interface ` + modelPascal + `DrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: ` + modelPascal + `;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * ` + service.ModelName + ` 编辑/创建抽屉组件
 */
const ` + modelPascal + `Drawer: React.FC<` + modelPascal + `DrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('` + fileName + `');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
` + createBlock + updateBlock + `
  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
` + submitBranch + `
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
` + strings.Join(defaultValues, "\n") + `
            }
      }
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading` + loadingExtras + `,
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        width: ` + itoa(drawerSize) + `,
      }}
    >
` + strings.Join(formItemCodes, "\n\n") + `
    </DrawerForm>
  );
};

export default ` + modelPascal + `Drawer;
`)
	return sb.String()
}

// ==============================
// constants.ts
// ==============================

// reactConstantsCode 生成 constants.ts（包含 status 枚举时），否则返回空串
func reactConstantsCode(service *ParsedService) string {
	statusField := FindStatusField(service)
	if statusField == nil || len(statusField.EnumValues) == 0 {
		return ""
	}

	var statusMapEntries, statusOptions []string
	for _, v := range statusField.EnumValues {
		label := v
		if v == "ON" {
			label = "启用"
		} else if v == "OFF" {
			label = "禁用"
		}
		color := "default"
		if v == "ON" {
			color = "success"
		} else if v == "OFF" {
			color = "error"
		}
		statusMapEntries = append(statusMapEntries, "    "+v+": { text: t('"+label+"'), color: '"+color+"' },")
		statusOptions = append(statusOptions, "    { label: t('"+label+"'), value: '"+v+"' },")
	}

	return `/**
 * ` + service.ModelName + ` 模块常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

/** 状态映射 */
export function getStatusMap(t: TFn) {
  return {
` + strings.Join(statusMapEntries, "\n") + `
  };
}

/** 状态选项 */
export function getStatusOptions(t: TFn) {
  return [
` + strings.Join(statusOptions, "\n") + `
  ];
}
`
}
