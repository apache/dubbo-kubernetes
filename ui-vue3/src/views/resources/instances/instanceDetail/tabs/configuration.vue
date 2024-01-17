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

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'

// 执行日志开关
const state = ref('')
const loading = ref(false)
const loadBalance = ref('random')
const clusterApproach = ref('failover')
const retryCount = ref('2次')
const timeout = ref('1000 ms')
</script>

<template>
  <!-- configuration -->
  <div class="configBox">
    <!-- left -->
    <a-typography>
      <!-- execution log -->
      <a-typography-title :level="5">
        {{ $t('executionLog') }}
        <a-tooltip placement="topLeft">
          <template #title
            >{{
              $t('enable_access_logs_for_all_instances_of_this_application')
            }}(provider.accesslog)</template
          >
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </a-typography-title>
      <a-typography-paragraph>
        <span :class="{ active: !state }" :style="{ color: 'black' }">{{ $t('close') }}</span>
        <a-switch v-model:checked="state" :loading="loading" />
        <span :class="{ active: state }" :style="{ color: state ? PRIMARY_COLOR : 'black' }">{{
          $t('enabled')
        }}</span>
      </a-typography-paragraph>

      <!-- Load Balance -->
      <a-typography-title :level="5">
        {{ $t('loadBalance') }}
        <a-tooltip placement="topLeft">
          <template #title
            >{{
              $t('adjust_the_load_balancing_strategy_for_application_services')
            }}(provider.loadbalance)</template
          >
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </a-typography-title>
      <a-typography-paragraph editable v-model:content="loadBalance"> </a-typography-paragraph>

      <!-- Cluster approach -->
      <a-typography-title :level="5">
        {{ $t('clusterApproach') }}
        <a-tooltip placement="topLeft">
          <template #title>{{
            $t('adjusting_the_negative_clustering_method_for_application_service_provision')
          }}</template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </a-typography-title>
      <a-typography-paragraph editable v-model:content="clusterApproach"> </a-typography-paragraph>
    </a-typography>

    <!-- right -->
    <a-typography>
      <!-- retry count -->
      <a-typography-title :level="5">
        {{ $t('retryCount') }}
        <a-tooltip placement="topLeft">
          <template #title>{{
            $t('adjusting_the_number_of_retries_for_application_provided_services')
          }}</template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </a-typography-title>
      <a-typography-paragraph editable v-model:content="retryCount"> </a-typography-paragraph>

      <!-- timeout -->
      <a-typography-title :level="5">
        {{ $t('timeout') }}
        <a-tooltip placement="topLeft">
          <template #title>
            {{ $t('adjusting_the_timeout_for_application_service_provision') }}(provider.timeout)
          </template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </a-typography-title>
      <a-typography-paragraph editable v-model:content="timeout"> </a-typography-paragraph>
    </a-typography>
  </div>
</template>

<style lang="less" scoped>
.configBox {
  margin: 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;

  .iconStyle {
    font-size: 17px;
  }

  .active {
    font-size: 15px;
    font-weight: bold;
  }
}
</style>
