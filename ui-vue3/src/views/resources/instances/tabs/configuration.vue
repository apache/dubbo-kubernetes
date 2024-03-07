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
  <a-descriptions title="" layout="vertical" :column="2">
    <!-- execution log -->
    <a-descriptions-item :labelStyle="{ fontWeight: 'bold' }">
      <template v-slot:label>
        {{ $t('instanceDomain.executionLog') }}
        <a-tooltip placement="topLeft">
          <template #title>
            {{ $t('instanceDomain.enableAppInstanceLogs') }}(provider.accesslog)
          </template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </template>

      <span :class="{ active: !state }" :style="{ color: 'black' }">
        {{ $t('instanceDomain.close') }}
      </span>
      <a-switch v-model:checked="state" :loading="loading" />
      <span :class="{ active: state }" :style="{ color: state ? PRIMARY_COLOR : 'black' }">
        {{ $t('instanceDomain.enable') }}
      </span>
    </a-descriptions-item>

    <!-- retry count -->
    <a-descriptions-item :labelStyle="{ fontWeight: 'bold' }">
      <template v-slot:label>
        {{ $t('instanceDomain.retryCount') }}
        <a-tooltip placement="topLeft">
          <template #title>{{ $t('instanceDomain.appServiceRetries') }}</template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </template>
      <a-typography-paragraph editable v-model:content="retryCount"> </a-typography-paragraph>
    </a-descriptions-item>

    <!-- Load Balance -->
    <a-descriptions-item :labelStyle="{ fontWeight: 'bold' }">
      <template v-slot:label>
        {{ $t('instanceDomain.loadBalance') }}
        <a-tooltip placement="topLeft">
          <template #title
            >{{ $t('instanceDomain.appServiceLoadBalance') }}(provider.loadbalance)</template
          >
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </template>
      <a-typography-paragraph editable v-model:content="loadBalance"> </a-typography-paragraph>
    </a-descriptions-item>

    <!-- timeout -->
    <a-descriptions-item :labelStyle="{ fontWeight: 'bold' }">
      <template v-slot:label>
        {{ $t('instanceDomain.timeout_ms') }}
        <a-tooltip placement="topLeft">
          <template #title>
            {{ $t('instanceDomain.appServiceTimeout') }}(provider.timeout)
          </template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </template>

      <a-typography-paragraph editable v-model:content="timeout"> </a-typography-paragraph>
    </a-descriptions-item>

    <!-- Cluster approach -->
    <a-descriptions-item :labelStyle="{ fontWeight: 'bold' }">
      <template v-slot:label>
        {{ $t('instanceDomain.clusterApproach') }}
        <a-tooltip placement="topLeft">
          <template #title>{{ $t('instanceDomain.appServiceNegativeClusteringMethod') }}</template>
          <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
        </a-tooltip>
      </template>

      <a-typography-paragraph editable v-model:content="clusterApproach"> </a-typography-paragraph>
    </a-descriptions-item>
  </a-descriptions>
</template>

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

<style lang="less" scoped>
.active {
  font-size: 13px;
  font-weight: bold;
}

.iconStyle {
  font-size: 17px;
}
</style>
