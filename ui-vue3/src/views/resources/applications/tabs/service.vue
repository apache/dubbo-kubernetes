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
  <div class="__container_app_service">
    <a-flex wrap="wrap" gap="small" :vertical="false" justify="start" align="left">
      <a-card class="statistic-card" v-for="(v, k) in clusterInfo.report">
        <a-flex gap="middle" :vertical="false" justify="space-between" align="center">
          <a-statistic :value="v.value" class="statistic">
            <template #prefix>
              <Icon class="statistic-icon" icon="solar:target-line-duotone"></Icon>
            </template>
            <template #title> {{ $t(k.toString()) }}</template>
          </a-statistic>
          <div class="statistic-icon-big">
            <Icon :icon="v.icon"></Icon>
          </div>
        </a-flex>
      </a-card>
    </a-flex>
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
import { computed, onMounted, reactive } from 'vue'
import ServiceList from '@/views/resources/services/search.vue'
import { getClusterInfo } from '@/api/service/clusterInfo'
import { getMetricsMetadata } from '@/api/service/serverInfo'
import { Chart } from '@antv/g2'
import { PRIMARY_COLOR } from '@/base/constants'
import { Icon } from '@iconify/vue'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain } from '@/utils/SearchUtil'
import { searchService } from '@/api/service/service'
import { provide } from 'vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { useRoute, useRouter } from 'vue-router'

let __null = PRIMARY_COLOR
let clusterInfo = reactive({
  info: <{ [key: string]: string }>{},
  report: <{ [key: string]: { value: string; icon: string } }>{}
})

let metricsMetadata = reactive({
  info: <{ [key: string]: string }>{}
})

onMounted(async () => {
  let clusterData = (await getClusterInfo({})).data
  metricsMetadata.info = <{ [key: string]: string }>(await getMetricsMetadata({})).data
  clusterInfo.info = <{ [key: string]: string }>clusterData
  clusterInfo.report = {
    providers: {
      icon: 'carbon:branch',
      value: clusterInfo.info.providers
    },
    consumers: {
      icon: 'mdi:merge',
      value: clusterInfo.info.consumers
    }
  }
})

const columns = [
  {
    title: 'idx',
    key: 'idx'
  },
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
        label: '',
        param: 'type',
        defaultValue: 1,
        dict: [
          { label: 'providers', value: 1 },
          { label: 'consumers', value: 2 }
        ],
        dictType: 'BUTTON'
      },
      {
        label: 'serviceName',
        param: 'serviceName'
      }
    ],
    searchService,
    columns,
    { pageSize: 4 }
  )
)
searchDomain.onSearch()
const route = useRoute()
const router = useRouter()

const viewDetail = (serviceName: string) => {
  router.push('/resources/services/detail/' + serviceName)
}

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped>
.__container_app_service {
  .statistic {
    width: 8vw;
  }
  :deep(.ant-card-body) {
    padding: 12px;
  }

  .statistic-card {
    border: 1px solid v-bind("(PRIMARY_COLOR) + '22'");
    margin-bottom: 30px;
  }

  .statistic-icon {
    color: v-bind(PRIMARY_COLOR);
    margin-bottom: -3px;
    font-weight: bold;
  }

  .statistic-icon-big {
    width: 38px;
    height: 38px;
    background: v-bind('PRIMARY_COLOR');
    line-height: 38px;
    vertical-align: middle;
    text-align: center;
    border-radius: 5px;
    font-size: 20px;
    padding-top: 2px;
    color: white;
  }
}
</style>
