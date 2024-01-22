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
      <a-row justify="start">
        <a-col :span="10">
          <a-form-item :label="$t('applicationDomain.name')">
            <a-input-group compact>
              <a-input-search
                v-model:value="searchDomain.params.appName"
                placeholder="input search text"
                enter-button
                @search="searchDomain.onSearch()"
              ></a-input-search>
            </a-input-group>
          </a-form-item>
        </a-col>
        <a-col :span="14"> </a-col>
      </a-row>
    </a-form>

    <div class="search-table-container">
      <a-table
        :pagination="pagination"
        :scroll="{ y: '55vh' }"
        :columns="searchDomain?.table.columns"
        :data-source="searchDomain?.result"
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
import { getCurrentInstance, inject } from 'vue'

import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { SearchDomain } from '@/utils/SearchUtil'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const searchDomain: SearchDomain | any = inject(PROVIDE_INJECT_KEY.SEARCH_DOMAIN)
console.log(searchDomain)
const pagination = {
  showTotal: (v: any) =>
    globalProperties.$t('searchDomain.total') +
    ': ' +
    v +
    ' ' +
    globalProperties.$t('searchDomain.unit')
}
</script>
<style lang="less" scoped></style>
