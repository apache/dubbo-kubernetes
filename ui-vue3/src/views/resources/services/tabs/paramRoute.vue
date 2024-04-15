<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->
<template>
  <div class="__container_services_tabs_param_route">
    <a-card :bordered="false" style="width: 1000px">
      <template #title>
        <a-flex justify="space-between">
          <span>路由</span>
          <a-flex class="handle-form" v-if="!isEdit">
            <EditOutlined @click="changeEditState" class="edit-icon" />
            <DeleteOutlined @click="emit('deleteParamRoute', props.index)" class="edit-icon" />
          </a-flex>
          <a-flex class="handle-form" v-else>
            <CheckOutlined @click="update"  class="edit-icon" />
            <CloseOutlined @click="reset" class="edit-icon" />
          </a-flex>
        </a-flex>
      </template>
      <a-form :labelCol="{ span: 3 }" :disabled="!isEdit">
        <a-form-item label="选择方法">
          <a-select v-model:value="editValue.method.value" style="width: 120px">
            <a-select-option v-for="(item, index) in editValue.method.selectArr" :value="item" :key="index">
              {{ item }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="指定方法参数">
          <a-table
            :columns="functionParamsColumn"
            :data-source="editValue.functionParams"
            :pagination="false"
          >
            <template #bodyCell="{ column, index: idx }">
              <template v-if="column.dataIndex === 'param'">
                <a-input v-model:value="editValue.functionParams[idx].param" />
              </template>
              <template v-if="column.dataIndex === 'relation'">
                <a-input v-model:value="editValue.functionParams[idx].relation" />
              </template>
              <template v-if="column.dataIndex === 'value'">
                <a-input v-model:value="editValue.functionParams[idx].value" />
              </template>
              <template v-if="column.dataIndex === 'handle'">
                <a-flex justify="space-between">
                  <PlusOutlined
                    class="edit-icon"
                    :class="{'disabled-icon': !isEdit}"
                    @click="isEdit && addFunctionParams(props.index, idx)"
                  />
                  <MinusOutlined
                    class="edit-icon"
                    :class="{'disabled-icon': !isEdit || editValue.functionParams.length === 1}"
                    @click="isEdit && editValue.functionParams.length !== 1 && deleteFunctionParams(props.index, idx)"
                  />
                </a-flex>
              </template>
            </template>
          </a-table>
        </a-form-item>
        <a-form-item label="路由目的地">
          <a-table
            :columns="destinationColumn"
            :data-source="editValue.destination"
            :pagination="false"
          >
            <template #bodyCell="{ column, index: idx }">
              <template v-if="column.dataIndex === 'label'">
                <a-input v-model:value="editValue.destination[idx].label" />
              </template>
              <template v-if="column.dataIndex === 'relation'">
                <a-input v-model:value="editValue.destination[idx].relation" />
              </template>
              <template v-if="column.dataIndex === 'value'">
                <a-input v-model:value="editValue.destination[idx].value" />
              </template>
              <template v-if="column.dataIndex === 'weight'">
                <a-input v-model:value="editValue.destination[idx].weight" />
              </template>
              <template v-if="column.dataIndex === 'handle'">
                <a-flex justify="space-between">
                  <PlusOutlined
                    class="edit-icon"
                    :class="{'disabled-icon': !isEdit}"
                    @click="isEdit && addDestination(idx)"
                  />
                  <MinusOutlined
                    class="edit-icon"
                    :class="{'disabled-icon': !isEdit || editValue.functionParams.length === 1}"
                    @click="isEdit && editValue.functionParams.length !== 1 && deleteDestination(idx)"
                  />
                </a-flex>
              </template>
            </template>
          </a-table>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { CheckOutlined, CloseOutlined, EditOutlined, DeleteOutlined, PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import { ref } from 'vue'

const isEdit = ref(false);

const changeEditState = () => {
  isEdit.value = true;
}

const emit = defineEmits(['deleteParamRoute', 'update'])
const props = defineProps({
  paramRouteForm: {
    type: Object,
    default: () => ({})
  },
  index: {
    type: Number
  }
})

const editValue = ref(
  {
    method: {
      value: undefined,
      selectArr: []
    },
    functionParams: [],
    destination: []
  }
)

const reset = () => {
  isEdit.value = false;
  editValue.value = JSON.parse(JSON.stringify(props.paramRouteForm))
}

reset();

const update = () => {
  isEdit.value = false;
  emit('update', props.index, editValue)
}

const functionParamsColumn = [
  {
    title: '参数索引',
    key: 'param',
    dataIndex: 'param',
    width: '30%'
  },
  {
    title: '关系',
    key: 'relation',
    dataIndex: 'relation',
    width: '30%'
  },
  {
    title: '值',
    key: 'value',
    dataIndex: 'value',
    width: '30%'
  },
  {
    title: '操作',
    key: 'handle',
    dataIndex: 'handle',
    width: '10%'
  }
]

const destinationColumn = [
  {
    title: '标签',
    key: 'label',
    dataIndex: 'label',
    width: '25%'
  },
  {
    title: '关系',
    key: 'relation',
    dataIndex: 'relation',
    width: '25%'
  },
  {
    title: '值',
    key: 'value',
    dataIndex: 'value',
    width: '25%'
  },
  {
    title: '权重',
    key: 'weight',
    dataIndex: 'weight',
    width: '15%'
  },
  {
    title: '操作',
    key: 'handle',
    dataIndex: 'handle',
    width: '10%'
  }
]

const addFunctionParams = (idx: number) => {
  editValue.value.functionParams.splice(idx + 1, 0, {
    param: '',
    relation: '',
    value: ''
  })
}

const deleteFunctionParams = (idx: number) => {
  editValue.value.functionParams.splice(idx, 1)
}

const addDestination = (idx: number) => {
  editValue.value.destination.splice(idx + 1, 0, {
    label: '',
    relation: '',
    value: '',
    weight: ''
  })
}

const deleteDestination = (idx: number) => {
  editValue.value.destination.splice(idx, 1)
}
</script>

<style lang="less" scoped>
.__container_services_tabs_param_route {
  .handle-form {
    width: 50px;
    justify-content: space-between;
  }
  .edit-icon {
    font-size: 18px;
  }
  .disabled-icon {
    color: #71777d;
  }
}
</style>
