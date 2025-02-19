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
            :label="$t('flowControlDomain.scope')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <span>{{ formViewData.basicInfo.scope }}</span>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.key')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <p
              @click="copyIt('org.apache.dubbo.samples.UserService')"
              class="description-item-content with-card"
            >
              {{ formViewData.basicInfo.key }}
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
            :label="$t('flowControlDomain.enabled')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <span>{{ formViewData.basicInfo.enabled ? '启用' : '不启用' }}</span>
          </a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else>
        <a-descriptions :column="2" layout="vertical">
          <a-descriptions-item
            :label="$t('flowControlDomain.scope')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-select
              v-model:value="formViewEdit.basicInfo.scope"
              style="min-width: 120px"
              disabled
            >
              <a-select-option :value="formViewEdit.basicInfo.scope"
                >{{ formViewData.basicInfo.scope }}
              </a-select-option>
            </a-select>
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.key')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-input v-model:value="formViewEdit.basicInfo.key" style="min-width: 300px" disabled />
          </a-descriptions-item>

          <a-descriptions-item
            :label="$t('flowControlDomain.enabled')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-switch
              v-model:checked="formViewEdit.basicInfo.enabled"
              checked-children="是"
              un-checked-children="否"
            />
          </a-descriptions-item>
        </a-descriptions>
      </div>
    </a-card>

    <template v-if="!isEdit">
      <a-card v-for="(config, index) in formViewData.config" class="dynamic-config-card">
        <template #title>
          配置【{{ index + 1 }}】
          <span style="font-weight: normal; font-size: 12px" :style="{ color: PRIMARY_COLOR }">
            对于{{ formViewData?.basicInfo?.scope === 'application' ? '应用' : '服务' }}的{{
              config.side === 'provider' ? '提供者' : '消费者'
            }}，将满足
            <a-tag
              :color="PRIMARY_COLOR"
              v-for="item in config.matches.map(
                (x: any) => x.key + ' ' + x.relation + ' ' + x.value
              )"
            >
              {{ item }}
            </a-tag>
            的实例，配置
            <a-tag
              :color="PRIMARY_COLOR"
              v-for="item in config.parameters.map((x: any) => x.key + ' = ' + x.value)"
            >
              {{ item }}
            </a-tag>
          </span>
        </template>
        <a-descriptions :column="2">
          <a-descriptions-item
            :label="$t('flowControlDomain.enabled')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            {{ config.enabled ? '启用' : '不启用' }}
          </a-descriptions-item>
          <a-descriptions-item
            :label="$t('flowControlDomain.side')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            {{ config.side }}
          </a-descriptions-item>
          <a-descriptions-item
            :label="$t('flowControlDomain.matches')"
            :labelStyle="{ fontWeight: 'bold' }"
            :span="2"
          >
            <a-input-group compact v-for="item in config.matches" :key="item.key">
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
            <a-input-group compact v-for="item in config.parameters" :key="item.key">
              <a-input disabled :value="item.key" style="width: 200px" />
              <a-input disabled :value="item.relation" style="width: 50px" />
              <a-input disabled :value="item.value" style="width: 200px" />
            </a-input-group>
          </a-descriptions-item>
        </a-descriptions>
      </a-card>
    </template>

    <template v-else>
      <a-spin :spinning="loading">
      <a-card v-for="(config, index) in formViewEdit.config" class="dynamic-config-card">
        <template #title>
          配置【{{ index + 1 }}】
          <span style="font-weight: normal; font-size: 12px" :style="{ color: PRIMARY_COLOR }">
            对于{{ formViewData?.basicInfo?.scope === 'application' ? '应用' : '服务' }}的{{
              config.side === 'provider' ? '提供者' : '消费者'
            }}，将满足
            <a-tag
              :color="PRIMARY_COLOR"
              v-for="item in config.matches.map(
                (x: any) => x.key + ' ' + x.relation + ' ' + x.value
              )"
            >
              {{ item }}
            </a-tag>
            的实例，配置
            <a-tag
              :color="PRIMARY_COLOR"
              v-for="item in config.parameters.map((x: any) => x.key + ' = ' + x.value)"
            >
              {{ item }}
            </a-tag>
          </span>
        </template>
        <a-descriptions :column="2">
          <a-descriptions-item
            :label="$t('flowControlDomain.enabled')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-switch
              v-model:checked="config.enabled"
              checked-children="是"
              un-checked-children="否"
            />
          </a-descriptions-item>
          <a-descriptions-item
            :label="$t('flowControlDomain.side')"
            :labelStyle="{ fontWeight: 'bold' }"
          >
            <a-radio-group v-model:value="config.side" :options="sideOptions" />
          </a-descriptions-item>
          <a-descriptions-item
            :label="$t('flowControlDomain.matches')"
            :labelStyle="{ fontWeight: 'bold' }"
            :span="2"
          >
            <div>
              <a-select
                ref="select"
                v-model:value="config.matchKeys"
                style="width: 500px"
                @change="handleChange(index, 'matches')"
                mode="multiple"
                :options="matchesArr.map((item) => ({ value: item }))"
              />
              <div v-for="item in config.matches" :key="item.key" style="margin-top: 20px">
                <a-input-group compact>
                  <a-input disabled :value="item.key" style="width: 160px" />
                  <a-input
                    placeholer="relation"
                    v-model:value="item.relation"
                    style="width: 100px"
                  />
                  <a-input placeholer="value" v-model:value="item.value" style="width: 240px" />
                </a-input-group>
              </div>
            </div>
          </a-descriptions-item>
          <a-descriptions-item
            :label="$t('flowControlDomain.configurationItem')"
            :labelStyle="{ fontWeight: 'bold' }"
            :span="2"
          >
            <div>
              <a-select
                ref="select"
                v-model:value="config.parameterKeys"
                style="width: 500px"
                @change="handleChange(index, 'parameters')"
                mode="multiple"
                :options="parametersArr.map((item) => ({ value: item }))"
              />
              <div v-for="item in config.parameters" :key="item.key" style="margin-top: 20px">
                <a-input-group compact>
                  <a-input disabled :value="item.key" style="width: 160px" />
                  <a-input
                    disabled
                    placeholer="relation"
                    :value="item.relation"
                    style="width: 100px"
                  />
                  <a-input placeholer="value" v-model:value="item.value" style="width: 240px" />
                </a-input-group>
              </div>
            </div>
          </a-descriptions-item>
        </a-descriptions>
      </a-card>
      </a-spin>
    </template>




    <a-button style="margin-bottom: 20px" v-if="isEdit" @click="addConfig">增加配置</a-button>

    <a-card class="footer">
      <a-flex v-if="isEdit">
        <a-button type="primary" @click="saveConfig" >确认</a-button>
        <a-button style="margin-left: 30px"
                  @click="router.replace('/traffic/dynamicConfig')">取消</a-button>
      </a-flex>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import type { ComponentInternalInstance } from 'vue'
import { getCurrentInstance, nextTick, onMounted, reactive, ref } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { useRoute, useRouter } from 'vue-router'
import {getConfiguratorDetail, saveConfiguratorDetail} from '@/api/service/traffic'
import gsap from 'gsap'

let __ = PRIMARY_COLOR
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const route = useRoute()
const router = useRouter()
const isEdit = ref(route.params.isEdit === '1')

const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

const formViewData: any = reactive({
  basicInfo: {
    ruleName: 'org.apache.dubbo.samples.UserService::.condition-router',
    scope: '服务',
    key: 'org.apache.dubbo.samples.UserService',
    effectTime: '20230/12/19 22:09:34',
    enabled: true
  },
  config: [
    {
      enabled: true,
      side: 'provider',
      matchKeys: ['address'],
      matches: [
        {
          key: 'address',
          relation: '=',
          value: '10.255.10.11'
        }
      ],
      parameterKeys: ['retries'],
      parameters: [
        {
          key: 'retries',
          relation: '=',
          value: '2'
        }
      ]
    }
  ]
})

const matchesArr = ['address', 'providerAddress', 'service', 'app', 'param']
const parametersArr = ['retries', 'timeout', 'accesslog', 'weight', '其他']

const formViewEdit: any = formViewData

const sideOptions = [
  {
    label: 'provider',
    value: 'provider'
  },
  {
    label: 'consumer',
    value: 'consumer'
  }
]

const addConfig = () => {
  const container: any = document.getElementById('layout-tab-body')
  formViewEdit.config.push({
    enabled: true,
    side: 'provider',
    matches: [],
    parameters: []
  })
  nextTick(() => {
    gsap.to(container, {
      duration: 1, // 动画持续时间（秒）
      scrollTop: container.scrollHeight,
      ease: 'power2.out'
    })
  })
}

const handleChange = (index: number, name: string) => {
  const config: any = formViewData.config[index]
  config[name] = config[name].filter((item: any) => {
    return config[name + 'Keys'].find((i: any) => {
      return i === item.key
    })
  })
  config[name + 'Keys'].forEach((item: any) => {
    if (
      !config[name].find((i: any) => {
        return i.key === item
      })
    ) {
      config[name].push({
        key: item,
        relation: name === 'parameters' ? '=' : 'exact',
        value: ''
      })
    }
  })
}

onMounted(async () => {
  const res = await getConfiguratorDetail({ name: route.params?.pathId })
  // console.log(formViewData.config)
  const data = res.data
  if (data) {
    formViewData.basicInfo.configVerison = data.configVerison
    formViewData.basicInfo.scope = data.scope
    formViewData.basicInfo.key = data.key
    formViewData.basicInfo.enabled = data.enabled
    formViewData.config = data.configs.map((x: any) => {
      let matches = []
      for (let matchKey in x.match) {
        let relation = Object.keys(x.match[matchKey])[0]
        matches.push({
          key: matchKey,
          relation: relation,
          value: x.match[matchKey][relation]
        })
      }
      let parameters = []
      for (let paramKey in x.parameters) {
        parameters.push({
          key: paramKey,
          relation: '=',
          value: x.parameters[paramKey]
        })
      }

      return {
        enabled: x.enabled,
        side: x.side,
        matchKeys: Object.keys(x.match),
        matches: matches,
        parameterKeys: Object.keys(x.parameters),
        parameters: parameters
      }
    })
  }
})
const loading = ref(false)
async function saveConfig() {
  loading.value = true
  let newVal = {
    scope: formViewEdit.basicInfo.scope,
    key: formViewEdit.basicInfo.key,
    enabled: formViewEdit.basicInfo.enabled,
    configVersion: formViewEdit.basicInfo.configVerison,
    configs: formViewEdit.config.map((x:any)=>{
      const match:any = {}
      const parameters:any = {}
      for (let m of x.matches) {
        match[m.key] = {[m.relation]: m.value  }
      }
      for (let m of x.parameters) {
        parameters[m.key] =  m.value
      }

      return {
        match,
        parameters,
        enabled: x.enabled,
        side: x.side,
      }
    })
  }
  try {
    let res = await saveConfiguratorDetail({ name: route.params?.pathId }, newVal)
    message.success('config save success')
  } finally {
    loading.value = false
  }
}
</script>

<style lang="less" scoped>
.__container_traffic_config_form {
  position: relative;
  width: 100%;

  .footer {
    //position: sticky; //bottom: 0; //width: 100%;
  }

  .dynamic-config-card {
    :deep(.ant-descriptions-item-label) {
      width: 120px;
      text-align: right;
    }

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
