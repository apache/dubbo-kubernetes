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
  <div class="__container_app_config">
    <config-page :options="options">
      <template v-slot:form_log="{ current }">
        <a-form-item :label="$t('applicationDomain.operatorLog')" name="logFlag">
          <a-switch v-model:checked="current.form.logFlag"></a-switch>
        </a-form-item>
      </template>
      <template v-slot:form_flow="{ current }">
        <a-space>
          <a-card v-for="(item, i) in current.form.rules">
            <template #title>
              {{ $t('applicationDomain.flowWeight') }} {{ i + 1 }}
              <div style="float: right">
                <a-space>
                  <a-button type="dashed" @click="">
                    <Icon
                      style="font-size: 20px"
                      icon="material-symbols-light:contract-edit"
                    ></Icon>
                  </a-button>
                  <a-button danger type="dashed" @click="">
                    <Icon style="font-size: 20px" icon="fluent:delete-12-filled"></Icon>
                  </a-button>
                </a-space>
              </div>
            </template>

            <a-form-item :name="'rules[' + i + '].weight'" label="权重">
              <a-input-number v-model:value="item.weight"></a-input-number>
            </a-form-item>
            <a-form-item label="作用范围">
              <a-table
                style="width: 40vw"
                :pagination="false"
                :columns="[
                  { key: 'label', title: 'label' },
                  { key: 'condition', title: 'condition' },
                  { key: 'value', title: 'value' }
                ]"
                :data-source="[item.scope]"
              >
                <template #bodyCell="{ column, record, index }">
                  <a-form-item
                    v-if="column.key === 'condition'"
                    :name="'rules[' + i + '].scope.condition'"
                    label=""
                  >
                    <a-select v-model:value="item.scope.condition">
                      <a-select-option value="=">=</a-select-option>
                      <a-select-option value="!=">!=</a-select-option>
                      <a-select-option value=">">></a-select-option>
                      <a-select-option value="<">{{ '<' }}</a-select-option>
                    </a-select>
                  </a-form-item>
                  <a-form-item
                    v-else
                    :name="'rules[' + i + '].scope.condition.' + column.key"
                    label=""
                  >
                    <a-input v-model:value="item.scope[column.key]"></a-input>
                  </a-form-item>
                </template>
              </a-table>
            </a-form-item>
          </a-card>
        </a-space>
      </template>
      <template v-slot:form_gray="{ current }">
        <a-space>
          <a-card v-for="(item, i) in current.form.rules">
            <template #title>
              {{ $t('applicationDomain.gray') }} {{ i + 1 }}
              <div style="float: right">
                <a-space>
                  <a-button type="dashed" @click="">
                    <Icon
                      style="font-size: 20px"
                      icon="material-symbols-light:contract-edit"
                    ></Icon>
                  </a-button>
                  <a-button danger type="dashed" @click="">
                    <Icon style="font-size: 20px" icon="fluent:delete-12-filled"></Icon>
                  </a-button>
                </a-space>
              </div>
            </template>

            <a-form-item :name="'rules[' + i + '].name'" label="环境名称">
              <a-input v-model:value="item.name"></a-input>
            </a-form-item>
            <a-form-item label="作用范围">
              <a-table
                style="width: 40vw"
                :pagination="false"
                :columns="[
                  { key: 'label', title: 'label' },
                  { key: 'condition', title: 'condition' },
                  { key: 'value', title: 'value' }
                ]"
                :data-source="[item.scope]"
              >
                <template #bodyCell="{ column, record, index }">
                  <a-form-item
                    v-if="column.key === 'condition'"
                    :name="'rules[' + i + '].scope.condition'"
                    label=""
                  >
                    <a-select v-model:value="item.scope.condition">
                      <a-select-option value="=">=</a-select-option>
                      <a-select-option value="!=">!=</a-select-option>
                      <a-select-option value=">">></a-select-option>
                      <a-select-option value="<">{{ '<' }}</a-select-option>
                    </a-select>
                  </a-form-item>
                  <a-form-item
                    v-else
                    :name="'rules[' + i + '].scope.condition.' + column.key"
                    label=""
                  >
                    <a-input v-model:value="item.scope[column.key]"></a-input>
                  </a-form-item>
                </template>
              </a-table>
            </a-form-item>
          </a-card>
        </a-space>
      </template>
    </config-page>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import ConfigPage from '@/components/ConfigPage.vue'
import { Icon } from '@iconify/vue'

let options: any = reactive({
  list: [
    {
      title: 'applicationDomain.operatorLog',
      key: 'log',
      form: {
        logFlag: false
      },
      submit: (form: {}) => {
        return new Promise((resolve) => {
          setTimeout(() => {
            resolve(1)
          }, 1000)
        })
      },
      reset(form: any) {
        form.logFlag = false
      }
    },
    {
      title: 'applicationDomain.flowWeight',
      key: 'flow',
      ext: {
        title: '添加权重配置',
        fun() {}
      },
      form: {
        rules: [
          {
            weight: '100',
            scope: {
              label: 'key1',
              condition: '=',
              value: 'value1'
            }
          }
        ]
      },
      submit(form: {}) {
        return new Promise((resolve) => {
          setTimeout(() => {
            resolve(1)
          }, 1000)
        })
      }
    },

    {
      title: 'applicationDomain.gray',
      key: 'gray',
      ext: {
        title: '添加灰度环境',
        fun() {}
      },
      form: {
        rules: [
          {
            name: '100',
            scope: {
              label: 'key1',
              condition: '=',
              value: 'value1'
            }
          }
        ]
      },
      submit(form: {}) {
        return new Promise((resolve) => {
          setTimeout(() => {
            resolve(1)
          }, 1000)
        })
      }
    }
  ],
  current: [0]
})
onMounted(() => {
  console.log(333)
})
</script>
<style lang="less" scoped>
.__container_app_config {
}
</style>
