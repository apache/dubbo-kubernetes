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
  <div class="__container_services_index">
    <search-table :search-domain="searchDomain">
      <template #bodyCell="{ column, text }">
        <template v-if="column.dataIndex === 'serviceName'">
          <span class="service-link" @click="viewDetail(text)">
            <b>
              <Icon style="margin-bottom: -2px" icon="material-symbols:attach-file-rounded"></Icon>
              <span class="service-link-text">{{ text }}</span>
            </b>
          </span>
        </template>
        <template v-else-if="column.dataIndex === 'versionGroupSelect'">
          <a-select
            v-model:value="text.versionGroupValue"
            :bordered="false"
            style="width: 80%"
          >
            <a-select-option v-for="(item, index) in text.versionGroupArr" :value="item" :key="index">
              {{ item }}
            </a-select-option>
          </a-select>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { reactive, provide } from 'vue'
import { searchService } from '@/api/service/service.ts'
import { SearchDomain } from '@/utils/SearchUtil'
import SearchTable from '@/components/SearchTable.vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { PRIMARY_COLOR } from '@/base/constants'
import { Icon } from '@iconify/vue'

let __null = PRIMARY_COLOR
const router = useRouter()
const columns = [
  {
    title: 'service',
    key: 'service',
    dataIndex: 'serviceName',
    sorter: true,
    width: '30%'
  },
  {
    title: 'versionGroup',
    key: 'versionGroup',
    dataIndex: 'versionGroupSelect',
    width: '25%'
  },
  {
    title: 'avgQPS',
    key: 'avgQPS',
    dataIndex: 'avgQPS',
    sorter: true,
    width: '15%'
  },
  {
    title: 'avgRT',
    key: 'avgRT',
    dataIndex: 'avgRT',
    sorter: true,
    width: '15%'
  },
  {
    title: 'requestTotal',
    key: 'requestTotal',
    dataIndex: 'requestTotal',
    sorter: true,
    width: '15%'
  }
]

const handleResult = (result: any) => {
  return result.map(service => {
    service.versionGroupSelect = {}
    service.versionGroupSelect.versionGroupArr = service.versionGroup.map((item: any) => {
      return item.versionGroup = (item.version ? 'version: ' + item.version + ', ' : '') + (item.group ? 'group: ' + item.group : '') || '无'
    })
    service.versionGroupSelect.versionGroupValue = service.versionGroupSelect.versionGroupArr[0];
    return service;
  })
}

const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'serviceName',
        param: 'serviceName',
        placeholder: 'typeAppName',
        style: {
          width: '200px'
        }
      }
    ],
    searchService,
    columns,
    undefined,
    undefined,
    handleResult
  )
)

searchDomain.onSearch(handleResult)

const viewDetail = (serviceName: string) => {
  router.push({ name: 'detail', params: { serviceName } })
}

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped>
.__container_services_index {
  .service-link {
    padding: 4px 10px 4px 4px;
    border-radius: 4px;
    color: v-bind('PRIMARY_COLOR');
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    &:hover {
      cursor: pointer;
      background: rgba(133, 131, 131, 0.13);
    }
  }
}
</style>
