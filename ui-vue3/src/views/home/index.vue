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
  <div class="__container_home_index">
    <h1>{{ $t(routeName) }}</h1>
    <a-flex wrap="wrap" gap="small" :vertical="false" justify="space-between" align="center">
      <a-card class="statistic-card" v-for="(v, k) in clusterInfo.report">
        <a-flex gap="middle" :vertical="false" justify="space-between" align="center">
          <a-statistic :value="v.value" class="statistic">
            <template #prefix>
              <Icon class="statistic-icon" icon="svg-spinners:pulse-ring"></Icon>
            </template>
            <template #title> {{ $t(k.toString()) }}</template>
          </a-statistic>
          <div class="statistic-icon-big">
            <Icon :icon="v.icon"></Icon>
          </div>
        </a-flex>
      </a-card>
    </a-flex>
    <a-descriptions
      title=" "
      bordered
      :column="{ xxl: 2, xl: 2, lg: 2, md: 2, sm: 1, xs: 1 }"
      layout="horizontal"
    >
      <a-descriptions-item label="versions">
        <a-tag :color="PRIMARY_COLOR" v-for="v in metricsMetadata.info.versions">{{ v }}</a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="protocols">
        <a-tag :color="PRIMARY_COLOR" v-for="v in metricsMetadata.info.protocols">{{ v }}</a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="configCenter">{{
        metricsMetadata.info.configCenter
      }}</a-descriptions-item>
      <a-descriptions-item label="registry">{{
        metricsMetadata.info.registry
      }}</a-descriptions-item>
      <a-descriptions-item label="metadataCenter">{{
        metricsMetadata.info.metadataCenter
      }}</a-descriptions-item>
      <a-descriptions-item label="grafana">{{ metricsMetadata.info.grafana }}</a-descriptions-item>
      <a-descriptions-item label="prometheus">{{
        metricsMetadata.info.prometheus
      }}</a-descriptions-item>
      <a-descriptions-item label="Remark">empty</a-descriptions-item>
      <a-descriptions-item label="rules">
        <a-tag :color="PRIMARY_COLOR" v-for="v in metricsMetadata.info.rules">{{ v }}</a-tag>
      </a-descriptions-item>
    </a-descriptions>
    <div id="report_container"></div>
  </div>
</template>

<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { onMounted, reactive } from 'vue'
import { getClusterInfo } from '@/api/service/clusterInfo'
import { getMetricsMetadata } from '@/api/service/serverInfo'
import { useRoute } from 'vue-router'
import { Chart } from '@antv/g2'

let __null = PRIMARY_COLOR

const routeName = <string>useRoute().name
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
    all: {
      icon: 'ic:outline-all-inclusive',
      value: clusterInfo.info.all
    },
    application: {
      icon: 'cil:applications-settings',
      value: clusterInfo.info.application
    },
    services: {
      icon: 'carbon:microservices-1',
      value: clusterInfo.info.services
    },
    providers: {
      icon: 'arcticons:newsprovider',
      value: clusterInfo.info.providers
    },
    consumers: {
      icon: 'iconoir:consumable',
      value: clusterInfo.info.consumers
    }
  }

  const chart = new Chart({
    container: 'report_container',
    autoFit: true
  })

  chart
    .interval()
    .data([
      { name: 'all', value: clusterInfo.info.all },
      { name: 'application', value: clusterInfo.info.application },
      { name: 'services', value: clusterInfo.info.services },
      { name: 'providers', value: clusterInfo.info.providers },
      { name: 'consumers', value: clusterInfo.info.consumers }
    ])
    .encode('x', 'name')
    .encode('y', 'value')
    .encode('color', 'name')
    .encode('size', 40)
    .style('radiusTopLeft', 5)
    .style('radiusTopRight', 10)
    .style('radiusBottomRight', 15)
    .style('radiusBottomLeft', 20)
  chart.render()
})
</script>
<style lang="less" scoped>
.__container_home_index {
  .statistic {
    width: 8vw;
  }

  .statistic-card {
    border: 1px solid v-bind("(PRIMARY_COLOR) + '22'");
  }

  .statistic-icon {
    color: v-bind(PRIMARY_COLOR);
    margin-bottom: -3px;
  }

  .statistic-icon-big {
    width: 40px;
    height: 40px;
    background: v-bind('PRIMARY_COLOR');
    line-height: 44px;
    vertical-align: middle;
    text-align: center;
    border-radius: 5px;
    font-size: 20px;
    color: white;
  }
}
</style>
