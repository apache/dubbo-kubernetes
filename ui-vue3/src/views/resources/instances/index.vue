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
  <div class="__container_resources_application_index">
    <search-table :search-domain="searchDomain">
      <template #bodyCell="{ text, record, index, column }">
        <template v-if="column.dataIndex === 'deployState'">
          <a-typography-text type="success">{{ text.label }}</a-typography-text>
        </template>

        <template v-else-if="column.dataIndex === 'deployCluster'">
          <a-tag color="success">
            {{ text }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'registerStates'">
          <a-typography-text type="success" v-for="t in text">{{ t.label }}</a-typography-text>
        </template>

        <template v-if="column.dataIndex === 'registerClusters'">
          <a-tag v-for="t in text" color="warning">
            {{ t }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'labels'">
          <a-tag v-for="t in text" color="warning">
            {{ t }}
          </a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'name'">
          <router-link :to="`detail/${record[column.key]}`">{{ text }}</router-link>
        </template>
        <template v-else-if="column.dataIndex === 'ip'">
          <router-link :to="`detail/${record[column.key]}`">{{ text }}</router-link>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, provide, reactive } from 'vue'
import { searchInstances } from '@/api/service/instance'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain, sortString } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'

let columns = [
  {
    title: 'instanceIP',
    key: 'ip',
    dataIndex: 'ip',
    sorter: (a: any, b: any) => sortString(a.instanceIP, b.instanceIP),
    width: 140
  },
  {
    title: 'instanceName',
    key: 'name',
    dataIndex: 'name',
    sorter: (a: any, b: any) => sortString(a.instanceName, b.instanceName),
    width: 140
  },
  {
    title: 'deployState',
    key: 'deployState',
    dataIndex: 'deployState',
    width: 120,
    sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },

  {
    title: 'deployCluster',
    key: 'deployCluster',
    dataIndex: 'deployCluster',
    sorter: (a: any, b: any) => sortString(a.deployCluster, b.deployCluster),
    width: 120
  },
  {
    title: 'registerStates',
    key: 'registerStates',
    dataIndex: 'registerStates',
    sorter: (a: any, b: any) => sortString(a.registerStates, b.registerStates),
    width: 120
  },
  {
    title: 'registerCluster',
    key: 'registerClusters',
    dataIndex: 'registerClusters',
    sorter: (a: any, b: any) => sortString(a.registerCluster, b.registerCluster),
    width: 140
  },
  {
    title: 'CPU',
    key: 'cpu',
    dataIndex: 'cpu',
    sorter: (a: any, b: any) => sortString(a.CPU, b.CPU),
    width: 140
  },
  {
    title: 'memory',
    key: 'memory',
    dataIndex: 'memory',
    sorter: (a: any, b: any) => sortString(a.memory, b.memory),
    width: 80
  },
  {
    title: 'startTime_k8s',
    key: 'startTime_k8s',
    dataIndex: 'startTime',
    sorter: (a: any, b: any) => sortString(a.startTime_k8s, b.startTime_k8s),
    width: 200
  },
  {
    title: 'registerTime',
    key: 'registerTime',
    dataIndex: 'registerTime',
    sorter: (a: any, b: any) => sortString(a.registerTime, b.registerTime),
    width: 200
  },
  {
    title: 'labels',
    key: 'labels',
    dataIndex: 'labels',
    width: 140
  }
]

// search
const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'appName',
        param: 'appName',
        placeholder: 'typeAppName',
        style: {
          width: '200px'
        }
      }
    ],
    searchInstances,
    columns
  )
)

onMounted(() => {
  searchDomain.onSearch()
  console.log(searchDomain.result)
})

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped>
.search-table-container {
  min-height: 60vh;
  // overflow-x: scroll;
  //max-height: 70vh; //overflow: auto;
}
</style>
