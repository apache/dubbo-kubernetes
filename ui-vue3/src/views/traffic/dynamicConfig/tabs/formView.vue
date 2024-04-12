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
  <div class="__container_traffic_config_form">
    <a-card title="基础信息" class="dynamic-config-card">
      <a-descriptions :column="2" layout="vertical">
        <!-- ruleName -->
        <a-descriptions-item
          :label="$t('flowControlDomain.ruleName')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <p
            @click="copyIt('org.apache.dubbo.samples.UserService::.condition-router')"
            class="description-item-content with-card"
          >
            org.apache.dubbo.samples.UserService::.condition-router
            <CopyOutlined />
          </p>
        </a-descriptions-item>

        <!-- ruleGranularity -->
        <a-descriptions-item
          :label="$t('flowControlDomain.ruleGranularity')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph>服务</a-typography-paragraph>
        </a-descriptions-item>

        <!-- actionObject -->
        <a-descriptions-item
          :label="$t('flowControlDomain.actionObject')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <p
            @click="copyIt('org.apache.dubbo.samples.UserService')"
            class="description-item-content with-card"
          >
            org.apache.dubbo.samples.UserService
            <CopyOutlined />
          </p>
        </a-descriptions-item>

        <!-- timeOfTakingEffect -->
        <a-descriptions-item
          :label="$t('flowControlDomain.timeOfTakingEffect')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> 20230/12/19 22:09:34</a-typography-paragraph>
        </a-descriptions-item>

        <!-- enabledStatus -->
        <a-descriptions-item
          :label="$t('flowControlDomain.enabledStatus')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph>
            {{ $t('flowControlDomain.enabled') }}
          </a-typography-paragraph>
        </a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-card title="配置【1】" class="dynamic-config-card">
      <a-descriptions :column="2">
        <a-descriptions-item
          :label="$t('flowControlDomain.enabledStatus')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          {{ $t('flowControlDomain.enabled') }}
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.endOfAction')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          {{ 'provider' }}
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.actuatingRange')"
          :labelStyle="{ fontWeight: 'bold' }"
          :span="2"
        >
          <a-input-group compact>
            <a-input disabled value="address" style="width: 200px" />
            <a-input disabled value="=" style="width: 50px" />
            <a-input disabled value="10.255.10.11" style="width: 200px" />
          </a-input-group>
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.configurationItem')"
          :labelStyle="{ fontWeight: 'bold' }"
          :span="2"
        >
          <a-input-group compact>
            <a-input disabled value="retries" style="width: 200px" />
            <a-input disabled value="=" style="width: 50px" />
            <a-input disabled value="3" style="width: 200px" />
          </a-input-group>
        </a-descriptions-item>
      </a-descriptions>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import { type ComponentInternalInstance, getCurrentInstance } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

let __ = PRIMARY_COLOR
const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}
</script>

<style lang="less" scoped>
.__container_traffic_config_form {
  .dynamic-config-card {
    margin-bottom: 20px;
    .description-item-content {
      &.no-card {
        padding-left: 20px;
      }

      &.with-card:hover {
        color: v-bind('PRIMARY_COLOR');
      }
    }
  }
  
}
</style>
