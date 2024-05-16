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

        <template v-if="column.dataIndex === 'deployCluster'">
          <a-tag color="grey">
            {{ text }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'deployState'">
          <a-tag :color="INSTANCE_DEPLOY_COLOR[text.toUpperCase()]">{{ text }}</a-tag>
        </template>

        <template v-if="column.dataIndex === 'registerStates'">
          <a-tag :color="INSTANCE_REGISTER_COLOR[t.level.toUpperCase()]" v-for="t in text">
            {{ t.label }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'registerClusters'">
          <a-tag v-for="t in text" color="grey">
            {{ t }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'labels'">
          <a-tag v-for="t in text" color="grey">
            {{ t }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'ip'">
          <span class="app-link" @click="router.replace(`detail/${record[column.key]}`)">
            <b>
              <Icon style="margin-bottom: -2px" icon="material-symbols:attach-file-rounded"></Icon>
              {{ text }}
            </b>
          </span>
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
import { INSTANCE_DEPLOY_COLOR, INSTANCE_REGISTER_COLOR } from '@/base/constants'
import router from '@/router'
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'

let __null = PRIMARY_COLOR
let columns = [
  {
    title: 'instanceDomain.instanceIP',
    key: 'ip',
    dataIndex: 'ip',
    sorter: (a: any, b: any) => sortString(a.ip, b.ip),
    width: 200
  },
  {
    title: 'instanceDomain.instanceName',
    key: 'name',
    dataIndex: 'name',
    sorter: (a: any, b: any) => sortString(a.name, b.name),
    width: 140
  },
  {
    title: 'instanceDomain.deployState',
    key: 'deployState',
    dataIndex: 'deployState',
    width: 120,
    sorter: (a: any, b: any) => sortString(a.deployState, b.deployState)
  },

  {
    title: 'instanceDomain.deployCluster',
    key: 'deployCluster',
    dataIndex: 'deployCluster',
    sorter: (a: any, b: any) => sortString(a.deployCluster, b.deployCluster),
    width: 120
  },
  {
    title: 'instanceDomain.registerStates',
    key: 'registerStates',
    dataIndex: 'registerStates',
    sorter: (a: any, b: any) => sortString(a.registerStates, b.registerStates),
    width: 120
  },
  {
    title: 'instanceDomain.registerCluster',
    key: 'registerClusters',
    dataIndex: 'registerClusters',
    sorter: (a: any, b: any) => sortString(a.registerClusters, b.registerClusters),
    width: 140
  },
  {
    title: 'instanceDomain.CPU',
    key: 'cpu',
    dataIndex: 'cpu',
    sorter: (a: any, b: any) => sortString(a.cpu, b.cpu),
    width: 140
  },
  {
    title: 'instanceDomain.memory',
    key: 'memory',
    dataIndex: 'memory',
    sorter: (a: any, b: any) => sortString(a.memory, b.memory),
    width: 100
  },
  {
    title: 'instanceDomain.startTime_k8s',
    key: 'startTime_k8s',
    dataIndex: 'startTime',
    sorter: (a: any, b: any) => sortString(a.startTime, b.startTime),
    width: 200
  },
  {
    title: 'instanceDomain.registerTime',
    key: 'registerTime',
    dataIndex: 'registerTime',
    sorter: (a: any, b: any) => sortString(a.registerTime, b.registerTime),
    width: 200
  },
  {
    title: 'instanceDomain.labels',
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

  .app-link {
    padding: 4px 10px 4px 4px;
    border-radius: 4px;
    color: v-bind('PRIMARY_COLOR');
    &:hover {
      cursor: pointer;
      background: rgba(133, 131, 131, 0.13);
    }
  }
}
</style>
