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
        <template v-if="column.dataIndex === 'ruleName'">
          <a-button type="link" @click="router.replace(`formview/${record.ruleName}`)">{{
            text
          }}</a-button>
        </template>

        <template v-if="column.dataIndex === 'enable'">
          {{ text ? '是' : '否' }}
        </template>
        <template v-if="column.dataIndex === 'protection'">
          {{ text ? '是' : '否' }}
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, provide, reactive } from 'vue'
import { searchRoutingRule } from '@/api/service/traffic'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain, sortString } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import router from '@/router'
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'

let __null = PRIMARY_COLOR
let columns = [
  {
    title: 'ruleName',
    key: 'ruleName',
    dataIndex: 'ruleName',
    sorter: (a: any, b: any) => sortString(a.appName, b.appName),
    width: 140
  },
  {
    title: 'ruleGranularity',
    key: 'ruleGranularity',
    dataIndex: 'ruleGranularity',
    width: 100,
    sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },
  {
    title: 'enable',
    key: 'enable',
    dataIndex: 'enable',
    render: (text, record) => (record.enable ? '是' : '否'),
    width: 120
  },
  {
    title: 'effectiveTime',
    key: 'effectiveTime',
    dataIndex: 'effectiveTime',
    width: 120
  },
  {
    title: 'protection',
    key: 'protection',
    dataIndex: 'protection',
    render: (text, record) => (record.protection ? '是' : '否'),
    width: 200
  }
]
const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'serviceGovernance',
        param: 'serviceGovernance',
        placeholder: 'typeRoutingRules',
        style: {
          width: '200px'
        }
      }
    ],
    searchRoutingRule,
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
