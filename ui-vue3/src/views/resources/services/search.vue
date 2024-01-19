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
          @search="onSearch"
          enter-button
        />
      </a-flex>
      <a-table :columns="columns" :data-source="dataSource" :pagination="{ pageSize: 5 }">
        <template #bodyCell="{ column, text }">
          <template v-if="column.dataIndex === 'serviceName'">
            <a-button type="link" @click="viewDetail(text)">{{ text }}</a-button>
          </template>
        </template>
        <template #customFilterIcon="{ filtered }">
          <search-outlined :style="{ color: filtered ? '#108ee9' : undefined }" />
        </template>
      </a-table>
    </a-flex>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { SearchOutlined } from '@ant-design/icons-vue'
import { ref } from 'vue'

const serviceName = ref('')

const onSearch = () => {}

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
    dataIndex: 'QPS',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min RT',
    dataIndex: 'RT',
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

const dataSource = [
  {
    serviceName: 'org.apache.dubbo.samples.UserService',
    interfaceNum: 4,
    QPS: 6,
    RT: '194ms',
    requestTotal: 200
  },
  {
    serviceName: 'org.apache.dubbo.samples.OrderService',
    interfaceNum: 12,
    QPS: 13,
    RT: '189ms',
    requestTotal: 164
  },
  {
    serviceName: 'org.apache.dubbo.samples.DetailService',
    interfaceNum: 14,
    QPS: 0.5,
    RT: '268ms',
    requestTotal: 1324
  },
  {
    serviceName: 'org.apache.dubbo.samples.PayService',
    interfaceNum: 8,
    QPS: 9,
    RT: '346ms',
    requestTotal: 189
  },
  {
    serviceName: 'org.apache.dubbo.samples.CommentService',
    interfaceNum: 9,
    QPS: 8,
    RT: '936ms',
    requestTotal: 200
  },
  {
    serviceName: 'org.apache.dubbo.samples.RepayService',
    interfaceNum: 16,
    QPS: 17,
    RT: '240ms',
    requestTotal: 146
  },
  {
    serviceName: 'org.apche.dubbo.samples.TransportService',
    interfaceNum: 5,
    QPS: 43,
    RT: '89ms',
    requestTotal: 367
  },
  {
    serviceName: 'org.apche.dubbo.samples.DistributionService',
    interfaceNum: 5,
    QPS: 4,
    RT: '78ms',
    requestTotal: 145
  }
]

const viewDetail = (serviceName: string) => {
  router.push({ name: 'detail', params: { serviceName } })
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
