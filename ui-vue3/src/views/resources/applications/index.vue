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
        <template v-if="column.dataIndex === 'registerClusters'">
          <a-tag v-for="t in text" color="grey">
            {{ t }}
          </a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'deployCluster'">
          {{ text }}
        </template>
        <template v-else-if="column.dataIndex === 'appName'">
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
import { searchApplications } from '@/api/service/app'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain, sortString } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { Icon } from '@iconify/vue'
import router from '@/router'
import { PRIMARY_COLOR } from '@/base/constants'

let __null = PRIMARY_COLOR
let columns = [
  // {
  //   title: 'idx',
  //   key: 'idx',
  //   dataIndex: 'idx',
  //   width: 50
  // },
  {
    title: 'appName',
    key: 'appName',
    dataIndex: 'appName',
    sorter: (a: any, b: any) => sortString(a.appName, b.appName),
    width: 140
  },
  {
    title: 'instanceNum',
    key: 'instanceNum',
    dataIndex: 'instanceNum',
    width: 100,
    sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },

  {
    title: 'deployCluster',
    key: 'deployCluster',
    dataIndex: 'deployCluster',
    width: 120
  },
  {
    title: 'registerClusters',
    key: 'registerClusters',
    dataIndex: 'registerClusters',
    width: 200
  }
]
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
    searchApplications,
    columns
  )
)

onMounted(() => {
  searchDomain.onSearch()
})

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped>
.search-table-container {
  min-height: 60vh;
  //max-height: 70vh; //overflow: auto;
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
