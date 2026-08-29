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

      <ElFormItem :label="$t('pages.dictType.code')" prop="code">
        <ElInput
          v-model="formData.code"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dictType.isEnabled')" prop="isEnabled">
        <ElSwitch v-model="formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dictType.name')" prop="name">
        <ElInput
          v-model="formData.name"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dictType.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
          
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dictType.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="formData.sortOrder"
          :min="1"
          style="width: 100%"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dictType.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton :value="'ON'">{{ $t("enum.status.ON") }}</ElRadioButton>
          <ElRadioButton :value="'OFF'">{{ $t("enum.status.OFF") }}</ElRadioButton>
        </ElRadioGroup>
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
  useCreateDictType,
  useUpdateDictType,
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();
const { mutateAsync: createDictType } = useCreateDictType();
const { mutateAsync: updateDictType } = useUpdateDictType();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  code: "",
  isEnabled: true,
  name: "",
  remark: "",
  sortOrder: 1,
  status: "",
});

// 表单验证规则
const formRules = {
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.dictType.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.dictType.moduleName") })
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
  formData.isEnabled = true;
  formData.name = "";
  formData.remark = "";
  formData.sortOrder = 1;
  formData.status = "";

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
      await createDictType(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateDictType({ id: currentId.value!, values });
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
