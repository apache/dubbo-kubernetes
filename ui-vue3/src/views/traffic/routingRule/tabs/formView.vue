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
      <a-card>
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

          <!-- faultTolerantProtection -->
          <a-descriptions-item
            :label="$t('flowControlDomain.faultTolerantProtection')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-typography-paragraph>
              {{ $t('flowControlDomain.opened') }}
            </a-typography-paragraph>
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
              {{ $t('flowControlDomain.notSet') }}
            </a-typography-paragraph>
          </a-descriptions-item>
        </a-descriptions>
      </a-card>
    </a-flex>

    <a-card title="路由【1】">
      <a-space align="start" style="width: 100%">
        <a-typography-title :level="5"
          >{{ $t('flowControlDomain.requestParameterMatching') }}:</a-typography-title
        >

        <a-space align="center" direction="horizontal" size="middle">
          <a-tag color="#2db7f5">method=login</a-tag>
          <a-tag color="#2db7f5">arguments[0]=dubbo</a-tag>
        </a-space>
      </a-space>

      <a-space align="start" style="width: 100%">
        <a-typography-title :level="5"
          >{{ $t('flowControlDomain.addressSubsetMatching') }}:</a-typography-title
        >
        <a-tag color="#87d068">version=v1</a-tag>
      </a-space>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import { type ComponentInternalInstance, getCurrentInstance, ref } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
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
