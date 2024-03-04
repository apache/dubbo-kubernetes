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
  <div class="__container_search_table">
    <a-form>
      <a-flex wrap="wrap" gap="large">
        <template v-for="q in searchDomain.params">
          <a-form-item :label="$t(q.label)">
            <template v-if="q.dict && q.dict.length > 0">
              <a-radio-group
                button-style="solid"
                v-model:value="searchDomain.queryForm[q.param]"
                v-if="q.dictType === 'BUTTON'"
              >
                <a-radio-button v-for="item in q.dict" :value="item.value">
                  {{ $t(item.label) }}
                </a-radio-button>
              </a-radio-group>
              <a-select
                v-else
                class="select-type"
                :style="q.style"
                v-model:value="searchDomain.queryForm[q.param]"
              >
                <a-select-option
                  :value="item.value"
                  v-for="item in [...q.dict, { label: 'none', value: '' }]"
                >
                  {{ $t(item.label) }}
                </a-select-option>
              </a-select>
            </template>

            <a-input
              v-else
              :style="q.style"
              :placeholder="$t('placeholder.' + (q.placeholder || `typeDefault`))"
              v-model:value="searchDomain.queryForm[q.param]"
            ></a-input>
          </a-form-item>
        </template>
        <a-form-item :label="''">
          <a-button type="primary" @click="searchDomain.onSearch()">
            <Icon
              style="margin-bottom: -2px; font-size: 1.3rem"
              icon="ic:outline-manage-search"
            ></Icon>
          </a-button>
        </a-form-item>
      </a-flex>
    </a-form>

    <div class="search-table-container">
      <a-table
        :loading="searchDomain.table.loading"
        :pagination="pagination"
        :scroll="{
          scrollToFirstRowOnChange: true,
          y: searchDomain.tableStyle?.scrollY || '55vh',
          x: searchDomain.tableStyle?.scrollX || ''
        }"
        :columns="searchDomain?.table.columns"
        :data-source="searchDomain?.result"
        @change="handleTableChange"
      >
        <template #bodyCell="{ text, record, index, column }">
          <span v-if="column.key === 'idx'">{{ index + 1 }}</span>
          <slot
            name="bodyCell"
            :text="text"
            :record="record"
            :index="index"
            :column="column"
            v-else
          >
          </slot>
        </template>
      </a-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ComponentInternalInstance } from 'vue'
import { computed, getCurrentInstance, inject } from 'vue'

import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { SearchDomain } from '@/utils/SearchUtil'
import { Icon } from '@iconify/vue'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const searchDomain: SearchDomain | any = inject(PROVIDE_INJECT_KEY.SEARCH_DOMAIN)
searchDomain.table.columns.forEach((column: any) => {
  if (column.title) {
    const tmp = column.title
    column.title = computed(() => globalProperties.$t(tmp))
  }
})
const pagination = computed(() => {
  console.log(pagination)
  return {
    pageSize: searchDomain.paged.pageSize,
    current: searchDomain.paged.curPage,
    showTotal: (v: any) =>
      globalProperties.$t('searchDomain.total') +
      ': ' +
      v +
      ' ' +
      globalProperties.$t('searchDomain.unit')
  }
})

const handleTableChange = (
  pag: { pageSize: number; current: number },
  filters: any,
  sorter: any
) => {
  searchDomain.paged.pageSize = pag.pageSize
  searchDomain.paged.curPage = pag.current
  searchDomain.onSearch()
  return
}
</script>
<style lang="less" scoped>
.__container_search_table {
  .select-type {
    width: 200px;
  }
}
</style>
