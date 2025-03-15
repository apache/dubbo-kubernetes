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
  <div class="__container_services_tabs_scene_config">
    <a-tabs v-model:activeKey="activeKey" tab-position="left" animated>
      <a-tab-pane key="timeout" tab="超时时间">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="超时时间">
            <a-flex v-if="!editForm.timeout.isEdit">
              <span class="item-content">{{ timeout }}ms</span>
              <EditOutlined @click="showEdit('timeout')" class="item-icon" />
            </a-flex>
            <a-flex v-else align="center">
              <a-input-number min="0" v-model:value="editForm.timeout.value" class="item-input" />
              <span style="margin-left: 5px">ms</span>
              <CheckOutlined @click="saveEdit('timeout')" class="item-icon" />
              <CloseOutlined @click="hideEdit('timeout')" class="item-icon" />
            </a-flex>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="retryNum" tab="重试次数">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="重试次数">
            <a-flex v-if="!editForm.retryNum.isEdit">
              <span class="item-content">{{ retryNum }}次</span>
              <EditOutlined @click="showEdit('retryNum')" class="item-icon" />
            </a-flex>
            <a-flex v-else align="center">
              <a-input-number min="0" v-model:value="editForm.retryNum.value" class="item-input" />
              <span style="margin-left: 5px">次</span>
              <CheckOutlined @click="saveEdit('retryNum')" class="item-icon" />
              <CloseOutlined @click="hideEdit('retryNum')" class="item-icon" />
            </a-flex>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="sameArea" tab="同区域优先">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="同区域优先">
            <a-radio-group
              @change="changeServiceIntraRegionPriority"
              v-bind:value="editForm.sameArea.value"
              button-style="solid"
            >
              <a-radio-button :value="false">关闭</a-radio-button>
              <a-radio-button :value="true">开启</a-radio-button>
            </a-radio-group>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="paramRoute" tab="参数路由">
        <ParamRoute
          v-for="(item, index) in paramRouteForms"
          class="param-route"
          :key="index"
          :paramRouteForm="item"
          :index="index"
          @update="updateParamRouteFormsItem"
          @deleteParamRoute="deleteParamRoute"
        />
        <a-button type="primary" style="margin-top: 20px" @click="addParamRoute">增加路由</a-button>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons-vue'
import ParamRoute from './paramRoute.vue'
import {
  getParamRouteAPI,
  getServiceIntraRegionPriorityAPI,
  getServiceRetryAPI,
  getServiceTimeoutAPI,
  updateParamRouteAPI,
  updateServiceIntraRegionPriorityAPI,
  updateServiceRetryAPI,
  updateServiceTimeoutAPI
} from '@/api/service/service'
import { useRoute } from 'vue-router'

const route = useRoute()
const timeout = ref(1000)
const retryNum = ref(1000)
const editForm: any = reactive({
  timeout: {
    isEdit: false,
    value: ''
  },
  retryNum: {
    isEdit: false,
    value: ''
  },
  sameArea: {
    value: false
  }
})

const activeKey = ref('timeout')
const showEdit = (param: string) => {
  editForm[param].isEdit = true
  switch (param) {
    case 'timeout':
      editForm[param].value = timeout.value
      break
    case 'retryNum':
      editForm[param].value = retryNum.value
      break
    default:
      break
  }
}

const saveEdit = async (param: string) => {
  switch (param) {
    case 'timeout': {
      await updateServiceTimeout()
      await getServiceTimeout()
      break
    }
    case 'retryNum': {
      await updateServiceRetry()
      await getServiceRetry()
      break
    }
    default:
      break
  }
  editForm[param].isEdit = false
}
const hideEdit = (param: string) => {
  editForm[param].isEdit = false
  switch (param) {
    case 'timeout':
      timeout.value = editForm[param].value
      break
  }
}

const paramRouteForms: any = ref([
  {
    method: 'string',
    conditions: [
      {
        index: 'string',
        relation: 'string',
        value: 'string'
      }
    ],
    destinations: [
      {
        conditions: [
          {
            tag: 'string',
            relation: 'string',
            value: 'string'
          }
        ],
        weight: 0
      }
    ]
  }
])

const addParamRoute = () => {
  paramRouteForms.value.push({
    method: 'string',
    conditions: [
      {
        index: 'string',
        relation: 'string',
        value: 'string'
      }
    ],
    destinations: [
      {
        conditions: [
          {
            tag: 'string',
            relation: 'string',
            value: 'string'
          }
        ],
        weight: 0
      }
    ]
  })
  updateParamRoute()
  getParamRoute()
}

const updateParamRouteFormsItem = (index: number, value: any) => {
  const route = {
    conditions: value.functionParams,
    destinations: value.destination,
    method: value.method.value
  }
  paramRouteForms.value[index] = route
  console.log(paramRouteForms.value)
  updateParamRoute()
  getParamRoute()
}

const updateParamRoute = async () => {
  const { pathId: serviceName, group, version } = route.params
  await updateParamRouteAPI({
    serviceName: serviceName,
    group: group || '',
    version: version || '',
    routes: paramRouteForms.value
  })
}
const deleteParamRoute = (index: number) => {
  paramRouteForms.value.splice(index, 1)
}

const getParamRoute = async () => {
  const { pathId: serviceName, group, version } = route.params
  const params = {
    serviceName,
    group,
    version
  }
  const res = await getParamRouteAPI(params)
  if (res.code === 200) {
    paramRouteForms.value = res.data?.routes
  }
}

// get timeout
const getServiceTimeout = async () => {
  const { pathId: serviceName, group, version } = route.params
  const params = {
    serviceName,
    group: group || '',
    version: version || ''
  }
  const res = await getServiceTimeoutAPI(params)
  timeout.value = res.data?.timeout
}

// update timeout
const updateServiceTimeout = async () => {
  const { pathId: serviceName, group, version } = route.params
  const data = {
    serviceName,
    group: group || '',
    version: version || '',
    timeout: parseInt(editForm.timeout.value)
  }
  await updateServiceTimeoutAPI(data)
}

// get service retry
const getServiceRetry = async () => {
  const { pathId: serviceName, group, version } = route.params
  const params = {
    serviceName,
    group: group || '',
    version: version || ''
  }
  const res = await getServiceRetryAPI(params)
  retryNum.value = res.data?.retryTimes
}

// update service retry
const updateServiceRetry = async () => {
  const { pathId: serviceName, group, version } = route.params
  const data = {
    serviceName,
    group: group || '',
    version: version || '',
    retryTimes: parseInt(editForm.retryNum.value)
  }
  await updateServiceRetryAPI(data)
}

const changeServiceIntraRegionPriority = async (e) => {
  editForm.sameArea.value = e.target.value
  await updateServiceIntraRegionPriority()
  await getServiceIntraRegionPriority()
}

const getServiceIntraRegionPriority = async () => {
  const { pathId: serviceName, group, version } = route.params
  const params = {
    serviceName,
    group: group || '',
    version: version || ''
  }
  const res: any = await getServiceIntraRegionPriorityAPI(params)
  editForm.sameArea.value = res.data?.enabled
}

const updateServiceIntraRegionPriority = async () => {
  const { pathId: serviceName, group, version } = route.params
  const data = {
    serviceName,
    group: group || '',
    version: version || '',
    enabled: editForm.sameArea.value
  }
  await updateServiceIntraRegionPriorityAPI(data)
}

onMounted(async () => {
  await getServiceTimeout()
  await getServiceRetry()
  await getServiceIntraRegionPriority()
  await getParamRoute()
})
</script>

<style lang="less" scoped>
.__container_services_tabs_scene_config {
  .item-content {
    margin-right: 20px;
  }

  .item-input {
    width: 200px;
  }

  .item-icon {
    margin-left: 15px;
    font-size: 18px;
  }

  .param-route {
    margin-bottom: 20px;
  }
}
</style>
