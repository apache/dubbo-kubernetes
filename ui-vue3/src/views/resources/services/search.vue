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
    <a-flex vertical>
      <a-flex class="service-filter">
        <a-input-search
          v-model:value="serviceName"
          placeholder="请输入"
          class="service-name-input"
          @search="debounceSearch"
          enter-button
        />
      </a-flex>
      <a-table
        :columns="columns"
        :data-source="dataSource"
        :pagination="pagination"
        :scroll="{ y: '55vh' }"
      >
        <template #bodyCell="{ column, text }">
          <template v-if="column.dataIndex === 'serviceName'">
            <a-button type="link" @click="viewDetail(text)">{{ text }}</a-button>
          </template>
        </template>
      </a-table>
    </a-flex>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { ComponentInternalInstance } from 'vue'
import { ref, getCurrentInstance } from 'vue'
import { searchService } from '@/api/service/service.ts'
import { debounce } from 'lodash'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const serviceName = ref('')

const router = useRouter()
const columns = [
  {
    title: '服务',
    dataIndex: 'serviceName',
    sorter: true,
    width: '30%'
  },
  {
    title: '接口数',
    dataIndex: 'interfaceNum',
    sorter: true,
    width: '10%'
  },
  {
    title: '近 1min QPS',
    dataIndex: 'avgQPS',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min RT',
    dataIndex: 'avgRT',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min 请求总量',
    dataIndex: 'requestTotal',
    sorter: true,
    width: '15%'
  }
]

const dataSource = ref([])

const onSearch = async () => {
  let { data } = await searchService({})
  dataSource.value = data.data
}

onSearch()

const debounceSearch = debounce(onSearch, 300)

const viewDetail = (serviceName: string) => {
  router.push({ name: 'detail', params: { serviceName } })
}

const pagination = {
  showTotal: (v: any) =>
    globalProperties.$t('searchDomain.total') +
    ': ' +
    v +
    ' ' +
    globalProperties.$t('searchDomain.unit')
}
</script>
<style lang="less" scoped>
.__container_services_index {
  .service-filter {
    margin-bottom: 20px;
    .service-name-input {
      width: 500px;
    }
  }
}
</style>
