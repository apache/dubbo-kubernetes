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
          <a-button type="link" @click="viewDetail(text)">{{ text }}</a-button>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { reactive, provide } from 'vue'
import { searchService } from '@/api/service/service'
import { SearchDomain } from '@/utils/SearchUtil'
import SearchTable from '@/components/SearchTable.vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'

const router = useRouter()
const columns = [
  {
    title: '服务',
    dataIndex: 'serviceName',
    key: 'serviceName',
    sorter: true,
    width: '30%'
  },
  {
    title: '接口数',
    dataIndex: 'interfaceNum',
    key: 'interfaceNum',
    sorter: true,
    width: '10%'
  },
  {
    title: '近 1min QPS',
    dataIndex: 'avgQPS',
    key: 'avgQPS',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min RT',
    dataIndex: 'avgRT',
    key: 'avgRT',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min 请求总量',
    dataIndex: 'requestTotal',
    key: 'requestTotal',
    sorter: true,
    width: '15%'
  }
]

const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: '服务名',
        param: 'serviceName',
        placeholder: '请输入',
        style: {
          width: '200px'
        }
      }
    ],
    searchService,
    columns
  )
)

searchDomain.onSearch()

const viewDetail = (serviceName: string) => {
  router.push({ name: 'detail', params: { serviceName } })
}

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped></style>
