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
        <a-space direction="vertical" size="middle" class="flowWeight-box">
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
              <a-input-number min="1" v-model:value="item.weight"></a-input-number>
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
                  <template v-if="column.key === 'label'">
                    <a-form-item :name="'rules[' + i + '].scope.key'">
                      <a-input v-model:value="item.scope.key"></a-input>
                    </a-form-item>
                  </template>
                  <template v-if="column.key === 'condition'">
                    <a-form-item :name="'rules[' + i + '].scope.condition'">
                      <a-input v-model:value="scopeConditionOfFlowWeight[i]"></a-input>
                    </a-form-item>
                  </template>
                  <template v-if="column.key === 'value'">
                    <a-form-item :name="'rules[' + i + '].scope.value'">
                      <a-input v-model:value="scopeValueOfFlowWeight[i]"></a-input>
                    </a-form-item>
                  </template>
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
                  <template v-if="column.key === 'label'">
                    <a-form-item :name="'rules[' + i + '].scope.key'">
                      <a-input v-model:value="item.scope.key"></a-input>
                    </a-form-item>
                  </template>
                  <template v-if="column.key === 'condition'">
                    <a-form-item :name="'rules[' + i + '].scope.condition'">
                      <a-input v-model:value="scopeConditionOfGrayIsolation[i]"></a-input>
                    </a-form-item>
                  </template>
                  <template v-if="column.key === 'value'">
                    <a-form-item :name="'rules[' + i + '].scope.value'">
                      <a-input v-model:value="scopeValueOfGrayIsolation[i]"></a-input>
                    </a-form-item>
                  </template>
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
import { onMounted, reactive, ref } from 'vue'
import ConfigPage from '@/components/ConfigPage.vue'
import { Icon } from '@iconify/vue'
import {
  getAppGrayIsolation,
  getAppLogSwitch,
  getAppTrafficWeight,
  updateAppGrayIsolation,
  updateAppLogSwitch,
  updateAppTrafficWeight
} from '@/api/service/app'
import { useRoute } from 'vue-router'

const route = useRoute()

let options: any = reactive({
  list: [
    {
      title: 'applicationDomain.operatorLog',
      key: 'log',
      form: {
        logFlag: false
      },
      submit: (form: any) => {
        return new Promise((resolve) => {
          resolve(updateLogFlag(form?.logFlag))
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
            weight: 10,
            scope: {
              key: 'version',
              value: {
                exact: 'v1'
              }
            }
          }
        ]
      },
      submit(form: {}) {
        return new Promise((resolve) => {
          resolve(updateFlowWeight())
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
            name: 'env-nam',
            scope: {
              key: 'env',
              value: {
                exact: 'gray'
              }
            }
          }
        ]
      },
      submit(form: {}) {
        return new Promise((resolve) => {
          resolve(updateGrayIsolation())
        })
      }
    }
  ],
  current: [0]
})

// Is execution log acquisition enabled?
const getLogFlag = async () => {
  const res = await getAppLogSwitch(<string>route.params?.pathId)
  console.log(res)
  if (res?.code == 200) {
    options.list.forEach((item: any) => {
      if (item.key === 'log') {
        item.form.logFlag = res.data.operatorLog
        return
      }
    })
  }
}

// Modify the execution log switch
const updateLogFlag = async (operatorLog: boolean) => {
  const res = await updateAppLogSwitch(<string>route.params?.pathId, operatorLog)
  console.log(res)
  if (res?.code == 200) {
    await getLogFlag()
  }
}

const scopeConditionOfFlowWeight: any = ref([])
const scopeValueOfFlowWeight: any = ref([])

// Obtain flow weight
const getFlowWeight = async () => {
  const res = await getAppTrafficWeight(<string>route.params?.pathId)
  if (res?.code == 200) {
    options.list.forEach((item: any) => {
      if (item.key === 'flow') {
        item.form.rules = JSON.parse(JSON.stringify(res.data.flowWeightSets))
        // 将后端发的流量权重数据拆出来
        item.form.rules.forEach((rule: any) => {
          if (Object.keys(rule.scope.value).length == 0) {
            scopeConditionOfFlowWeight.value.push('')
            scopeValueOfFlowWeight.value.push('')
          }
          for (const [key, value] of Object.entries(rule.scope.value)) {
            scopeConditionOfFlowWeight.value.push(key)
            scopeValueOfFlowWeight.value.push(value)
          }
          rule.scope.value = {}
        })
      }
    })
  }
}

// Modify flow weight
const updateFlowWeight = async () => {
  let flowWeightSets: any = []
  options.list.forEach((item: any) => {
    if (item.key === 'flow') {
      flowWeightSets = JSON.parse(JSON.stringify(item.form.rules))
      flowWeightSets.forEach((rule: any, index: number) => {
        if (scopeConditionOfFlowWeight.value[index] && scopeValueOfFlowWeight.value[index]) {
          rule.scope.value[scopeConditionOfFlowWeight.value[index]] =
            scopeValueOfFlowWeight.value[index]
        }
      })
    }
  })
  const res = await updateAppTrafficWeight(<string>route.params?.pathId, flowWeightSets)
  if (res.code === 200) {
    scopeConditionOfFlowWeight.value = []
    scopeValueOfFlowWeight.value = []
    await getFlowWeight()
  }
}

const scopeConditionOfGrayIsolation: any = ref([])
const scopeValueOfGrayIsolation: any = ref([])

// Obtain GrayIsolation
const getGrayIsolation = async () => {
  const res = await getAppGrayIsolation(<string>route.params?.pathId)
  if (res?.code == 200) {
    options.list.forEach((item: any) => {
      if (item.key === 'gray') {
        item.form.rules = JSON.parse(JSON.stringify(res.data.graySets))
        //Extract the traffic weight data sent by the backend.
        item.form.rules.forEach((rule: any) => {
          if (Object.keys(rule.scope.value).length == 0) {
            scopeConditionOfGrayIsolation.value.push('')
            scopeValueOfGrayIsolation.value.push('')
          }
          for (const [key, value] of Object.entries(rule.scope.value)) {
            scopeConditionOfGrayIsolation.value.push(key)
            scopeValueOfGrayIsolation.value.push(value)
          }
          rule.scope.value = {}
        })
      }
    })
  }
}

// Modify GrayIsolation
const updateGrayIsolation = async () => {
  let graySets: any = []
  options.list.forEach((item: any) => {
    if (item.key === 'gray') {
      graySets = JSON.parse(JSON.stringify(item.form.rules))
      graySets.forEach((rule: any, index: number) => {
        if (scopeConditionOfGrayIsolation.value[index] && scopeValueOfGrayIsolation.value[index]) {
          rule.scope.value[scopeConditionOfGrayIsolation.value[index]] =
            scopeValueOfGrayIsolation.value[index]
        }
      })
    }
  })
  const res = await updateAppGrayIsolation(<string>route.params?.pathId, graySets)
  if (res.code === 200) {
    scopeConditionOfGrayIsolation.value = []
    scopeValueOfGrayIsolation.value = []
    await getGrayIsolation()
  }
}

onMounted(() => {
  console.log(333)
  getLogFlag()
  getFlowWeight()
  getGrayIsolation()
})
</script>
<style lang="less" scoped>
.__container_app_config {
  .flowWeight-box {
    width: 100%;
    height: 100%;
    //overflow: scroll;
  }
}
</style>
