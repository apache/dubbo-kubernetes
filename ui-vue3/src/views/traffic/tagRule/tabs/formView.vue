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
  <div class="__container_app_detail">
    <a-flex>
      <a-card class="_detail">
        <a-descriptions :column="2" layout="vertical" title="">
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
            <a-typography-paragraph> 服务 </a-typography-paragraph>
          </a-descriptions-item>

          <!-- actionObject -->
          <a-descriptions-item
            :label="$t('flowControlDomain.actionObject')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <p @click="copyIt('shop-user')" class="description-item-content with-card">
              shop-user
              <CopyOutlined />
            </p>
          </a-descriptions-item>

          <!-- effectTime -->
          <a-descriptions-item
            :label="$t('flowControlDomain.effectTime')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph> 20230/12/19 22:09:34</a-typography-paragraph>
          </a-descriptions-item>

          <!-- faultTolerantProtection -->
          <a-descriptions-item
            :label="$t('flowControlDomain.faultTolerantProtection')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph>
              {{ $t('flowControlDomain.opened') }}
            </a-typography-paragraph>
          </a-descriptions-item>

          <!-- enabledState -->
          <a-descriptions-item
            :label="$t('flowControlDomain.enabledState')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph>
              {{ $t('flowControlDomain.enabled') }}
            </a-typography-paragraph>
          </a-descriptions-item>

          <!-- runTimeEffective -->
          <a-descriptions-item
            :label="$t('flowControlDomain.runTimeEffective')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph>
              {{ $t('flowControlDomain.opened') }}
            </a-typography-paragraph>
          </a-descriptions-item>

          <!-- priority -->
          <a-descriptions-item
            :label="$t('flowControlDomain.priority')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph>
              {{ '未设置' }}
            </a-typography-paragraph>
          </a-descriptions-item>
        </a-descriptions>
      </a-card >
    </a-flex>

    <a-card  title="标签【1】" style="margin-top: 10px"  class="_detail">
      <a-space align="center">
        <a-typography-title :level="5">
          {{ $t('flowControlDomain.labelName') }}:
          <a-typography-text class="labelName">gray</a-typography-text>
        </a-typography-title>
      </a-space>
      <a-space align="start" style="width: 100%">
        <a-typography-title :level="5"
          >{{ $t('flowControlDomain.actuatingRange') }}:</a-typography-title
        >
        <a-tag color="#2db7f5">version=v1</a-tag>
      </a-space>
    </a-card >
  </div>
</template>

<script lang="ts" setup>
import { CopyOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { type ComponentInternalInstance, getCurrentInstance } from 'vue'
import { PRIMARY_COLOR } from '@/base/constants'

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
.description-item-content {
  &.no-card {
    padding-left: 20px;
  }

  &.with-card:hover {
    color: v-bind('PRIMARY_COLOR');
  }
}
</style>
