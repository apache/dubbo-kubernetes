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
  <div class="__container_tagRule_detail">
    <a-flex style="width: 100%">
      <a-col :span="isDrawerOpened ? 24 - sliderSpan : 24" class="left">
        <a-card>
          <a-space style="width: 100%" direction="vertical" size="middle">
            <a-row>
              <a-flex justify="end" style="width: 100%">
                <a-button
                  type="text"
                  style="color: #0a90d5"
                  @click="isDrawerOpened = !isDrawerOpened"
                >
                  字段说明
                  <DoubleLeftOutlined v-if="!isDrawerOpened" />
                  <DoubleRightOutlined v-else />
                </a-button>
              </a-flex>
              <a-card title="基础信息" style="width: 100%" class="_detail">
                <a-form layout="horizontal">
                  <a-row style="width: 100%">
                    <a-col :span="12">
                      <a-form-item label="规则粒度" required> 应用 </a-form-item>
                      <a-form-item label="容错保护">
                        <a-switch
                          v-model:checked="baseInfo.faultTolerantProtection"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                      <a-form-item label="运行时生效">
                        <a-switch
                          v-model:checked="baseInfo.runtime"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                    </a-col>
                    <a-col :span="12">
                      <a-form-item label="作用对象" required>
                        <a-input v-model:value="baseInfo.objectOfAction" style="width: 200px" />
                      </a-form-item>
                      <a-form-item label="立即启用">
                        <a-switch
                          v-model:checked="baseInfo.enable"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                      <a-form-item label="优先级">
                        <a-input-number min="1" v-model:value="baseInfo.priority" />
                      </a-form-item>
                    </a-col>
                  </a-row>
                </a-form>
              </a-card>
            </a-row>

            <a-card title="标签列表" style="width: 100%" class="_detail">
              <a-card v-for="(tagItem, tagItemIndex) in tagList" :key="tagItemIndex">
                <template #title>
                  <a-space align="center">
                    <div>路由【{{ tagItemIndex + 1 }}】</div>
                    <div>
                      对于服务org.apahe.dubbo.samples.UserService，将满足请求方法等于login，且该方法的第一个参数等于dubbo的请求，导向带有标签version=v1的实例
                    </div>
                  </a-space>
                </template>

                <a-form layout="horizontal">
                  <a-space style="width: 100%" direction="vertical" size="large">
                    <a-flex justify="end">
                      <Icon
                        @click="deleteTagItem(tagItemIndex)"
                        class="action-icon"
                        icon="tdesign:delete"
                      />
                    </a-flex>
                    <a-form-item label="标签名" required>
                      <a-input placeholder="隔离环境名" v-model:value="tagItem.tagName" />
                    </a-form-item>
                    <a-form-item label="作用范围" required>
                      <a-card>
                        <a-space style="width: 100%" direction="vertical">
                          <a-form-item label="匹配条件类型">
                            <a-radio-group
                              v-model:value="tagItem.scope.type"
                              :options="matchConditionTypeOptions"
                            />
                          </a-form-item>
                          <a-space align="start" style="width: 100%" direction="horizontal">
                            <a-tag :bordered="false" color="processing">
                              {{ tagItem.scope.type }}
                            </a-tag>
                            <a-table
                              v-if="tagItem.scope.type === 'labels'"
                              :pagination="false"
                              :columns="labelsColumns"
                              :data-source="tagItem.scope?.labels"
                            >
                              <template #bodyCell="{ column, record, text, index: labelItemIndex }">
                                <template v-if="column.key === 'myKey'">
                                  <a-input placeholder="label key" v-model:value="record.myKey" />
                                </template>
                                <template v-if="column.key === 'condition'">
                                  <a-select
                                    v-model:value="record.condition"
                                    style="width: 120px"
                                    :options="labelConditionOptions"
                                  ></a-select>
                                </template>
                                <template v-if="column.key === 'value'">
                                  <a-input placeholder="label value" v-model:value="record.value" />
                                </template>
                                <template v-else-if="column.key === 'operation'">
                                  <a-space align="center">
                                    <Icon
                                      icon="tdesign:remove"
                                      class="action-icon"
                                      @click="deleteLabelItem(tagItemIndex, labelItemIndex)"
                                    />
                                    <Icon
                                      class="action-icon"
                                      icon="tdesign:add"
                                      @click="addLabelItem(tagItemIndex)"
                                    />
                                  </a-space>
                                </template>
                              </template>
                            </a-table>
                            <a-space v-else align="start">
                              <a-select
                                style="width: 120px"
                                v-model:value="tagItem.scope.addresses.condition"
                                :options="conditionOptions"
                              />
                              <a-textarea
                                style="width: 500px"
                                v-model:value="tagItem.scope.addresses.addressesStr"
                                placeholder='地址列表，如有多个用"，"隔开'
                              />
                            </a-space>
                          </a-space>
                        </a-space>
                      </a-card>
                    </a-form-item>
                  </a-space>
                </a-form>
              </a-card>
            </a-card>
            <a-button @click="addTag" type="primary"> 增加标签</a-button>
          </a-space>
        </a-card>
      </a-col>

      <a-col :span="isDrawerOpened ? sliderSpan : 0" class="right">
        <a-card v-if="isDrawerOpened" class="sliderBox">
          <div>
            <a-descriptions title="字段说明" :column="1">
              <a-descriptions-item label="key">
                作用对象<br />
                可能的值：Dubbo应用名或者服务名
              </a-descriptions-item>
              <a-descriptions-item label="scope">
                规则粒度<br />
                可能的值：application, service
              </a-descriptions-item>
              <a-descriptions-item label="force">
                容错保护<br />
                可能的值：true, false<br />
                描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用
              </a-descriptions-item>
              <a-descriptions-item label="runtime">
                运行时生效<br />
                可能的值：true, false<br />
                描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </a-card>
      </a-col>
    </a-flex>
    <a-affix :offset-bottom="10">
      <div class="bottom-action-footer">
        <a-space align="center" size="large">
          <a-button type="primary" @click="addTagRule"> 确认</a-button>
          <a-button> 取消</a-button>
        </a-space>
      </div>
    </a-affix>
  </div>
</template>

<script setup lang="ts">
import { type ComponentInternalInstance, getCurrentInstance, onMounted, reactive, ref } from 'vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { addTagRuleAPI, getTagRuleDetailAPI, updateTagRuleAPI } from '@/api/service/traffic'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()
const route = useRoute()

const isDrawerOpened = ref(false)

const sliderSpan = ref(8)

let __ = PRIMARY_COLOR

const toClipboard = useClipboard().toClipboard

const router = useRouter()

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

// base info
const baseInfo = reactive({
  ruleGranularity: 'application',
  objectOfAction: '',
  enable: true,
  faultTolerantProtection: true,
  runtime: true,
  priority: 1,
  configVersion: 'v3.0'
})

const matchConditionTypeOptions = ref([
  {
    label: 'labels',
    value: 'labels'
  },
  {
    label: 'addresses',
    value: 'addresses'
  }
])

const labelConditionOptions = ref([
  {
    label: 'exact',
    value: 'exact'
  },
  {
    label: 'regex',
    value: 'regex'
  },
  {
    label: 'prefix',
    value: 'prefix'
  },
  {
    label: 'noempty',
    value: 'noempty'
  },
  {
    label: 'empty',
    value: 'empty'
  },
  {
    label: 'wildcard',
    value: 'wildcard'
  }
])

const conditionOptions = ref([
  {
    label: '=',
    value: '='
  },
  {
    label: '!=',
    value: '!='
  }
])

const labelsColumns = ref([
  {
    title: '键',
    dataIndex: 'myKey',
    key: 'myKey'
  },
  {
    title: '关系',
    dataIndex: 'condition',
    key: 'condition'
  },
  {
    title: '值',
    dataIndex: 'value',
    key: 'value'
  },
  {
    title: '操作',
    dataIndex: 'operation',
    key: 'operation'
  }
])

// tag list
const tagList: any[] = ref([])

const deleteLabelItem = (tagItemIndex: number, labelItemIndex: number) => {
  if (tagList.value[tagItemIndex].scope.labels.length === 1) {
    tagList.value[tagItemIndex].scope.type = 'addresses'
    return
  }
  tagList.value[tagItemIndex].scope.labels.splice(labelItemIndex, 1)
}

const addLabelItem = (tagItemIndex: number) => {
  tagList.value[tagItemIndex].scope.labels.push({
    myKey: '',
    condition: 'exact',
    value: ''
  })
}

const addTag = () => {
  tagList.value.push({
    tagName: '',
    scope: {
      type: 'labels',
      labels: [
        {
          myKey: '',
          condition: '',
          value: ''
        }
      ],
      addresses: {
        condition: '',
        addressesStr: ''
      }
    }
  })
}

const deleteTagItem = (tagItemIndex: number) => {
  tagList.value.splice(tagItemIndex, 1)
}

const addTagRule = async () => {
  const {
    ruleGranularity,
    objectOfAction,
    enable,
    faultTolerantProtection,
    runtime,
    priority,
    configVersion
  } = baseInfo
  const data = {
    configVersion,
    scope: ruleGranularity,
    key: objectOfAction,
    enabled: enable,
    runtime,
    tags: []
  }
  tagList.value.forEach((tagItem, tagIndex) => {
    const tag = {
      name: tagItem.tagName,
      match: []
    }
    tagItem.scope.labels.forEach((labelItem, labelIndex) => {
      const matchItem = {
        key: labelItem.myKey,
        value: {}
      }
      matchItem.value[labelItem.condition] = labelItem.value
      tag.match.push(matchItem)
    })
    data.tags.push(tag)
  })
  let ruleName = ''
  if (ruleGranularity == 'application') {
    ruleName = `${objectOfAction}.tag-router`
  } else {
    ruleName = `${objectOfAction}:${configVersion}.tag-router`
  }
  const res = await addTagRuleAPI(ruleName, data)
  if (res.code === 200) {
    router.push('/traffic/tagRule')
  }
}
</script>

<style scoped lang="less">
.__container_tagRule_detail {
  .action-icon {
    font-size: 17px;
    margin-left: 10px;
    cursor: pointer;
  }

  .match-condition-type-label {
    min-width: 100px;
    text-align: center;
  }

  .bottom-action-footer {
    width: 100%;
    background-color: white;
    height: 50px;
    display: flex;
    align-items: center;
    padding-left: 20px;
    box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.1); /* 添加顶部阴影 */
  }

  .sliderBox {
    margin-left: 5px;
    max-height: 530px;
    overflow: auto;
  }

  &:deep(.left.ant-col) {
    transition: all 0.5s ease;
  }

  &:deep(.right.ant-col) {
    transition: all 0.5s ease;
  }
}
</style>
