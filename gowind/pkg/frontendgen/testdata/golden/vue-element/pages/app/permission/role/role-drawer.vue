<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    :size="DRAWER_WIDTH"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.role.code')" prop="code">
        <ElInput
          v-model="formData.code"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.description')" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
          
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.isProtected')" prop="isProtected">
        <ElSwitch v-model="formData.isProtected" />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.name')" prop="name">
        <ElInput
          v-model="formData.name"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.permissions')" prop="permissions">
        <ElInput
          v-model="formData.permissions"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="formData.sortOrder"
          :min="1"
          style="width: 100%"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton :value="'ON'">{{ $t("enum.status.ON") }}</ElRadioButton>
          <ElRadioButton :value="'OFF'">{{ $t("enum.status.OFF") }}</ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="$t('pages.role.tenantName')" prop="tenantName">
        <ElInput
          v-model="formData.tenantName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import {
  useCreateRole,
  useUpdateRole,
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();
const { mutateAsync: createRole } = useCreateRole();
const { mutateAsync: updateRole } = useUpdateRole();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  code: "",
  description: "",
  isProtected: false,
  name: "",
  permissions: 1,
  sortOrder: 1,
  status: "",
  tenantName: "",
});

// 表单验证规则
const formRules = {
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  permissions: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  tenantName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.role.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.role.moduleName") })
);

// 打开抽屉
function open(row?) {
  visible.value = true;

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentId.value = row.id;
    Object.assign(formData, row);
  } else {
    // 创建模式
    isCreate.value = true;
    currentId.value = undefined;
    resetForm();
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.code = "";
  formData.description = "";
  formData.isProtected = false;
  formData.name = "";
  formData.permissions = 1;
  formData.sortOrder = 1;
  formData.status = "";
  formData.tenantName = "";

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };

    if (isCreate.value) {
      await createRole(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateRole({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 不是验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法给父组件
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
