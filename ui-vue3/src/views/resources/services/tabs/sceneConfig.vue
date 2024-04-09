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
    <a-tabs v-model:activeKey="activeKey" :tab-position="'left'" animated>
      <a-tab-pane key="timeout" tab="超时时间">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="超时时间">
            <a-flex v-if="!editForm.timeout.isEdit">
              <span class="item-content">1000ms</span>
              <EditOutlined @click="showEdit('timeout')" style="font-size: 18px" />
            </a-flex>
            <a-flex v-else align="center">
              <a-input v-model:value="editForm.timeout.value" style="width: 200px" />
              <span style="margin-left: 5px">ms</span>
              <CheckOutlined @click="hideEdit('timeout')" style="margin-left: 15px; font-size: 18px;" />
              <CloseOutlined @click="hideEdit('timeout')" style="margin-left: 15px; font-size: 18px;" />
            </a-flex>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="retryNum" tab="重试次数">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="重试次数">
            <a-flex v-if="!editForm.retryNum.isEdit">
              <span class="item-content">1000次</span>
              <EditOutlined @click="showEdit('retryNum')" style="font-size: 18px" />
            </a-flex>
            <a-flex v-else align="center">
              <a-input v-model:value="editForm.retryNum.value" style="width: 200px" />
              <span style="margin-left: 5px">次</span>
              <CheckOutlined @click="hideEdit('retryNum')" style="margin-left: 15px; font-size: 18px;" />
              <CloseOutlined @click="hideEdit('retryNum')" style="margin-left: 15px; font-size: 18px;" />
            </a-flex>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="sameArea" tab="同区域优先">
        <a-descriptions layout="vertical">
          <a-descriptions-item label="同区域优先">
            <a-radio-group v-model:value="editForm.sameArea.value" button-style="solid">
              <a-radio-button value="close">关闭</a-radio-button>
              <a-radio-button value="open">开启</a-radio-button>
            </a-radio-group>
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
      <a-tab-pane key="paramRoute" tab="参数路由">
        <paramRoute v-for="item, index in paramRouteForms" :key="index" :paramRouteForm="item" />
        <a-button type="primary" style="margin-top: 20px">增加路由</a-button>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { EditOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons-vue';
import paramRoute from './paramRoute.vue'

const editForm = reactive({
  timeout: {
    isEdit: false,
    value: '',
  },
  retryNum: {
    isEdit: false,
    value: '',
  },
  sameArea: {
    value: 'close',
  },
  paramRoute: {
    isEdit: false,
    value: {},
  }
})

const activeKey = ref('timeout');
const showEdit = (param: string) => {
  editForm[param].isEdit = true;
}
const hideEdit = (param: string) => {
  editForm[param].isEdit = false;
}

const paramRouteForms = [
  {
    method: {
      value: 'getUserInfo',
      selectArr: [
        'getUserInfo',
        'register',
        'login'
      ]
    },
    functionParams: [{
      param: '',
      relation: '',
      value: '',
    }],
    destination: [{
      label: '',
      relation: '',
      value: '',
      weight: '',
    }]
  }
]
</script>

<style lang="less" scoped>
.__container_services_tabs_scene_config {
  .item-content {
    margin-left: 20px;
    margin-right: 20px;
  }
  .handle-form {
    width: 50px;
    justify-content: space-between;
  }
}
</style>
