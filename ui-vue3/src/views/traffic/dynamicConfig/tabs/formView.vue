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
      <div v-if="!isEdit">
        <a-descriptions :column="2" layout="vertical">
          <a-descriptions-item
            :label="$t('flowControlDomain.ruleName')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <p @click="copyIt(formViewData.ruleName)" class="description-item-content with-card">
              {{ formViewData.basicInfo.ruleName }}
              <CopyOutlined />
            </p>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.ruleGranularity')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <span>{{ formViewData.basicInfo.ruleGranularity }}</span>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.actionObject')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <p
              @click="copyIt('org.apache.dubbo.samples.UserService')"
              class="description-item-content with-card"
            >
              {{ formViewData.basicInfo.actionObject }}
              <CopyOutlined />
            </p>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.effectTime')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <span>{{ formViewData.basicInfo.effectTime }}</span>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.enabledState')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <span>{{ formViewData.basicInfo.enabledState ? '启用' : '不启用' }}</span>
          </a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else>
        <a-descriptions :column="2" layout="vertical">
          <a-descriptions-item
            :label="$t('flowControlDomain.ruleGranularity')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-select v-model:value="formViewData.basicInfo.ruleGranularity" style="min-width: 120px" disabled>
              <a-select-option :value="formViewData.basicInfo.ruleGranularity">{{ formViewData.basicInfo.ruleGranularity }}</a-select-option>
            </a-select>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.actionObject')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-input v-model:value="formViewData.basicInfo.actionObject" style="min-width: 300px" disabled />
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.enabledState')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-switch v-model:checked="enabledState" checked-children="是" un-checked-children="否" />
          </a-descriptions-item>
        </a-descriptions>
      </div>
    </a-card>

    <a-card
      :title="'配置【' + (index + 1) + '】'"
      class="dynamic-config-card"
      v-for="(config, index) in formViewData.config"
    >
      <a-descriptions :column="2">
        <a-descriptions-item
          :label="$t('flowControlDomain.enabledState')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          {{ config.enabledState ? '启用' : '不启用' }}
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.endOfAction')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          {{ config.endOfAction }}
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.actuatingRange')"
          :labelStyle="{ fontWeight: 'bold' }"
          :span="2"
        >
          <a-input-group compact v-for="item in config.actuatingRange">
            <a-input disabled :value="item.key" style="width: 200px" />
            <a-input disabled :value="item.relation" style="width: 50px" />
            <a-input disabled :value="item.value" style="width: 200px" />
          </a-input-group>
        </a-descriptions-item>
        <a-descriptions-item
          :label="$t('flowControlDomain.configurationItem')"
          :labelStyle="{ fontWeight: 'bold' }"
          :span="2"
        >
          <a-input-group compact v-for="item in config.configItem">
            <a-input disabled :value="item.key" style="width: 200px" />
            <a-input disabled :value="item.relation" style="width: 50px" />
            <a-input disabled :value="item.value" style="width: 200px" />
          </a-input-group>
        </a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-button v-if="isEdit">增加配置</a-button>

    <a-flex v-if="isEdit" style="margin-top: 30px">
      <a-button type="primary">确认</a-button>
      <a-button style="margin-left: 30px">取消</a-button>
    </a-flex>
  </div>
</template>

<script lang="ts" setup>
import type { ComponentInternalInstance } from 'vue'
import { getCurrentInstance, ref } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { useRoute } from 'vue-router'

let __ = PRIMARY_COLOR
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const route = useRoute()
const isEdit = ref(route.params.isEdit === '1')

const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

const formViewData = ref({
  basicInfo: {
    ruleName: 'org.apache.dubbo.samples.UserService::.condition-router',
    ruleGranularity: '服务',
    actionObject: 'org.apache.dubbo.samples.UserService',
    effectTime: '20230/12/19 22:09:34',
    enabledState: true
  },
  config: [
    {
      enabledState: true,
      endOfAction: 'provider',
      actuatingRange: [
        {
          key: 'address',
          relation: '=',
          value: '10.255.10.11'
        }
      ],
      configItem: [
        {
          key: 'retries',
          relation: '=',
          value: '2'
        }
      ]
    }
  ]
})

const enabledState = ref(formViewData.value.basicInfo.enabledState);
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
