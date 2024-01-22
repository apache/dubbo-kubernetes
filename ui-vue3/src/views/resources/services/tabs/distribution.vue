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
  <div class="__container_services_tabs_distribution">
    <a-flex vertical>
      <a-flex class="service-filter">
        <a-flex>
          <div>
            <span>版本&分组:</span>
            <a-select
              v-model:value="versionAndGroup"
              :options="versionAndGroupOptions"
              class="service-filter-select"
            ></a-select>
          </div>
          <a-input-search
            v-model:value="searchValue"
            placeholder="搜索应用，ip，支持前缀搜索"
            class="service-filter-input"
            @search="() => {}"
            enter-button
          />
        </a-flex>
        <div>
          <a-radio-group
            v-model:value="type"
            button-style="solid"
          >
            <a-radio-button value="producer">生产者</a-radio-button>
            <a-radio-button value="consumer">消费者</a-radio-button>
          </a-radio-group>
        </div>
      </a-flex>
      <a-table :columns="tableColumns" :data-source="tableData">
        <template #bodyCell="{ column, text }">
          <template v-if="column.dataIndex === 'applicationName'">
            <a-button type="link">{{ text }}</a-button>
          </template>
          <template v-if="column.dataIndex === 'instanceIP'">
            <a-flex justify="">
              <a-button v-for="ip in text.slice(0, 3)" :key="ip" type="link">{{ ip }}</a-button>
              <a-button v-if="text.length > 3" type="link">更多</a-button>
            </a-flex>
          </template>
        </template>
      </a-table>
    </a-flex>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'

const searchValue = ref('')
const versionAndGroupOptions = reactive([
  {
    label: '不指定',
    value: ''
  },
  {
    label: 'version=1.0.0',
    value: 'version=1.0.0'
  },
  {
    label: 'group=group1',
    value: 'group=group1'
  },
  {
    label: 'version=1.0.0,group=group1',
    value: 'version=1.0.0,group=group1'
  }
])
const versionAndGroup = ref(versionAndGroupOptions[0].value)
const type = ref('producer')

const tableColumns = [
  {
    title: '应用名',
    dataIndex: 'applicationName',
    width: '25%',
    sorter: true
  },
  {
    title: '实例数',
    dataIndex: 'instanceNum',
    width: '25%',
    sorter: true
  },
  {
    title: '实例ip',
    dataIndex: 'instanceIP',
    width: '50%'
  }
]

const tableData = reactive([
  {
    applicationName: 'shop-order',
    instanceNum: 15,
    instanceIP: [
      '192.168.32.28:8697',
      '192.168.32.26:20880',
      '192.168.32.24:28080',
      '192.168.32.22:20880'
    ]
  },
  {
    applicationName: 'shop-order',
    instanceNum: 15,
    instanceIP: ['192.168.32.28:8697', '192.168.32.26:20880', '192.168.32.24:28080']
  },
  {
    applicationName: 'shop-user',
    instanceNum: 12,
    instanceIP: ['192.168.32.28:8697', '192.168.32.24:28080']
  }
])
</script>
<style lang="less" scoped>
.__container_services_tabs_distribution {
  .service-filter {
    justify-content: space-between;
    margin-bottom: 20px;
    .service-filter-select {
      margin-left: 10px;
      width: 250px;
    }
    .service-filter-input {
      margin-left: 30px;
      width: 300px;
    }
  }
}
</style>
