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
  <div class="__container_routingRule_detail">
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
                      <a-form-item label="规则粒度" required>
                        <a-select
                          v-model:value="baseInfo.ruleGranularity"
                          style="width: 120px"
                          :options="ruleGranularityOptions"
                        ></a-select>
                      </a-form-item>
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
                    </a-col>
                  </a-row>
                </a-form>
              </a-card>
            </a-row>

            <a-card title="路由列表" style="width: 100%" class="_detail">
              <a-card v-for="(routeItem, routeItemIndex) in routeList">
                <template #title>
                  <a-space align="center">
                    <div>路由【{{ routeItemIndex + 1 }}】</div>
                    <div>
                      对于服务org.apahe.dubbo.samples.UserService，将满足请求方法等于login，且该方法的第一个参数等于dubbo的请求，导向带有标签version=v1的实例
                    </div>
                  </a-space>
                </template>

                <a-form layout="horizontal">
                  <a-space style="width: 100%" direction="vertical" size="large">
                    <a-form-item label="请求匹配">
                      <a-card v-if="routeItem.requestMatch.length > 0">
                        <a-space style="width: 100%" direction="vertical" size="small">
                          <a-flex align="center" justify="space-between">
                            <a-form-item label="匹配条件类型">
                              <a-select
                                v-model:value="routeItem.selectedMatchConditionTypes"
                                :options="matchConditionTypeOptions"
                                mode="multiple"
                                style="min-width: 200px"
                              />
                            </a-form-item>
                            <Icon
                              @click="deleteRequestMatch(routeItemIndex)"
                              class="action-icon"
                              icon="tdesign:delete"
                            />
                          </a-flex>
                          <template
                            v-for="(conditionItem, conditionItemIndex) in routeItem.requestMatch"
                          >
                            <!--                        host-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('host') &&
                                conditionItem.type === 'host'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input v-model="conditionItem.value" placeholder="请求来源ip" />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--application-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('application') &&
                                conditionItem.type === 'application'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input v-model="conditionItem.value" placeholder="请求来源应用名" />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--                      method-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('method') &&
                                conditionItem.type === 'method'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input v-model="conditionItem.value" placeholder="方法值" />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--                      arguments-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('arguments') &&
                                conditionItem.type === 'arguments'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-table
                                :pagination="false"
                                :columns="argumentsColumns"
                                :data-source="routeItem.requestMatch[conditionItemIndex].list"
                              >
                                <template #bodyCell="{ column, record, text }">
                                  <template v-if="column.key === 'index'">
                                    <a-input v-model:value="record.index" placeholder="index" />
                                  </template>
                                  <template v-else-if="column.key === 'condition'">
                                    <a-select
                                      v-model:value="record.condition"
                                      :options="conditionOptions"
                                    />
                                  </template>
                                  <template v-else-if="column.key === 'value'">
                                    <a-input v-model:value="record.value" placeholder="value" />
                                  </template>
                                  <template v-else-if="column.key === 'operation'">
                                    <a-space align="center">
                                      <Icon
                                        @click="
                                          deleteArgumentsItem(
                                            routeItemIndex,
                                            conditionItemIndex,
                                            addArgumentsItemIndex
                                          )
                                        "
                                        icon="tdesign:remove"
                                        class="action-icon"
                                      />
                                      <Icon
                                        class="action-icon"
                                        @click="
                                          addArgumentsItem(routeItemIndex, conditionItemIndex)
                                        "
                                        icon="tdesign:add"
                                      />
                                    </a-space>
                                  </template>
                                </template>
                              </a-table>
                            </a-space>
                            <!--                      attachments-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('attachments') &&
                                conditionItem.type === 'attachments'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-table
                                :pagination="false"
                                :columns="attachmentsColumns"
                                :data-source="routeItem.requestMatch[conditionItemIndex].list"
                              >
                                <template #bodyCell="{ column, record, text }">
                                  <template v-if="column.key === 'myKey'">
                                    <a-input v-model:value="record.myKey" placeholder="key" />
                                  </template>
                                  <template v-else-if="column.key === 'condition'">
                                    <a-select
                                      v-model:value="record.condition"
                                      :options="conditionOptions"
                                    />
                                  </template>
                                  <template v-else-if="column.key === 'value'">
                                    <a-input v-model:value="record.value" placeholder="value" />
                                  </template>
                                  <template v-else-if="column.key === 'operation'">
                                    <a-space align="center">
                                      <Icon
                                        @click="
                                          deleteAttachmentsItem(
                                            routeItemIndex,
                                            conditionItemIndex,
                                            record.index
                                          )
                                        "
                                        icon="tdesign:remove"
                                        class="action-icon"
                                      />
                                      <Icon
                                        class="action-icon"
                                        @click="
                                          addAttachmentsItem(routeItemIndex, conditionItemIndex)
                                        "
                                        icon="tdesign:add"
                                      />
                                    </a-space>
                                  </template>
                                </template>
                              </a-table>
                            </a-space>
                            <!--                      other-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('other') &&
                                conditionItem.type === 'other'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                              </a-tag>
                              <a-table
                                :pagination="false"
                                :columns="otherColumns"
                                :data-source="routeItem.requestMatch[conditionItemIndex].list"
                              >
                                <template #bodyCell="{ column, record, text }">
                                  <template v-if="column.key === 'myKey'">
                                    <a-input v-model:value="record.myKey" placeholder="key" />
                                  </template>
                                  <template v-else-if="column.key === 'condition'">
                                    <a-select
                                      v-model:value="record.condition"
                                      :options="conditionOptions"
                                    />
                                  </template>
                                  <template v-else-if="column.key === 'value'">
                                    <a-input v-model:value="record.value" placeholder="value" />
                                  </template>
                                  <template v-else-if="column.key === 'operation'">
                                    <a-space align="center">
                                      <Icon
                                        @click="
                                          deleteOtherItem(
                                            routeItemIndex,
                                            conditionItemIndex,
                                            record.index
                                          )
                                        "
                                        icon="tdesign:remove"
                                        class="action-icon"
                                      />
                                      <Icon
                                        @click="addOtherItem(routeItemIndex, conditionItemIndex)"
                                        icon="tdesign:add"
                                        class="action-icon"
                                      />
                                    </a-space>
                                  </template>
                                </template>
                              </a-table>
                            </a-space>
                          </template>
                        </a-space>
                      </a-card>
                      <a-button
                        @click="addRequestMatch(routeItemIndex)"
                        v-else
                        type="dashed"
                        size="large"
                      >
                        <template #icon>
                          <Icon icon="tdesign:add" />
                        </template>
                        增加匹配条件
                      </a-button>
                    </a-form-item>
                    <a-form-item label="路由分发" required>
                      <a-card>
                        <a-space style="width: 100%" direction="vertical" size="small">
                          <a-flex>
                            <a-form-item label="匹配条件类型">
                              <a-select
                                v-model:value="routeItem.selectedRouteDistributeMatchTypes"
                                :options="routeDistributionTypeOptions"
                                mode="multiple"
                                style="min-width: 200px"
                              />
                            </a-form-item>
                          </a-flex>
                          <template
                            v-for="(conditionItem, conditionItemIndex) in routeItem.routeDistribute"
                            :key="conditionItemIndex"
                          >
                            <!--                        host-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedRouteDistributeMatchTypes.includes('host') &&
                                conditionItem.type === 'host'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input v-model="conditionItem.value" placeholder="请求来源ip" />

                              <Icon
                                @click="
                                  deleteRouteDistributeMatchTypeItem(
                                    conditionItem?.type,
                                    routeItemIndex
                                  )
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>

                            <!--                      other-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedRouteDistributeMatchTypes.includes('other') &&
                                conditionItem.type === 'other'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                              </a-tag>
                              <a-table
                                :pagination="false"
                                :columns="otherColumns"
                                :data-source="routeItem.routeDistribute[conditionItemIndex].list"
                              >
                                <template #bodyCell="{ column, record, text }">
                                  <template v-if="column.key === 'myKey'">
                                    <a-input v-model:value="record.myKey" placeholder="key" />
                                  </template>
                                  <template v-else-if="column.key === 'condition'">
                                    <a-select
                                      v-model:value="record.condition"
                                      :options="conditionOptions"
                                    />
                                  </template>
                                  <template v-else-if="column.key === 'value'">
                                    <a-input v-model:value="record.value" placeholder="value" />
                                  </template>
                                  <template v-else-if="column.key === 'operation'">
                                    <a-space align="center">
                                      <Icon
                                        @click="
                                          deleteRouteDistributeOtherItem(
                                            routeItemIndex,
                                            conditionItemIndex,
                                            record.index
                                          )
                                        "
                                        icon="tdesign:remove"
                                        class="action-icon"
                                      />
                                      <Icon
                                        @click="
                                          addRouteDistributeOtherItem(
                                            routeItemIndex,
                                            conditionItemIndex
                                          )
                                        "
                                        icon="tdesign:add"
                                        class="action-icon"
                                      />
                                    </a-space>
                                  </template>
                                </template>
                              </a-table>
                            </a-space>
                          </template>
                        </a-space>
                      </a-card>
                    </a-form-item>
                  </a-space>
                </a-form>
              </a-card>
            </a-card>
            <a-button @click="addRoute" type="primary"> 增加路由 </a-button>
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
          <a-button type="primary"> 确认 </a-button>
          <a-button> 取消 </a-button>
        </a-space>
      </div>
    </a-affix>
  </div>
</template>

<script lang="ts" setup>
import { type ComponentInternalInstance, getCurrentInstance, reactive, ref } from 'vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'

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
  runtime: true
})

const matchConditionTypeOptions = ref([
  {
    label: 'host',
    value: 'host'
  },
  {
    label: 'application',
    value: 'application'
  },
  {
    label: 'method',
    value: 'method'
  },
  {
    label: 'arguments',
    value: 'arguments'
  },
  {
    label: 'attachments',
    value: 'attachments'
  },
  {
    label: '其他',
    value: 'other'
  }
])

const routeDistributionTypeOptions = ref([
  {
    label: 'host',
    value: 'host'
  },
  {
    label: '其他',
    value: 'other'
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

// rule granularity options
const ruleGranularityOptions = ref([
  {
    label: '应用',
    value: 'application'
  },
  {
    label: '服务',
    value: 'service'
  }
])

// route list
const routeList = ref([
  {
    selectedMatchConditionTypes: [],
    requestMatch: [],
    selectedRouteDistributeMatchTypes: [],
    routeDistribute: [
      {
        type: 'host',
        condition: '=',
        value: '127.0.0.1'
      },
      {
        type: 'other',
        list: [
          {
            myKey: 'key',
            condition: '=',
            value: 'value'
          }
        ]
      }
    ]
  }
])

const addRoute = () => {
  routeList.value.push({
    selectedMatchConditionTypes: [],
    requestMatch: [],
    selectedRouteDistributeMatchTypes: [],
    routeDistribute: [
      {
        type: 'host',
        condition: '=',
        value: '127.0.0.1'
      },
      {
        type: 'other',
        list: [
          {
            myKey: 'key',
            condition: '=',
            value: 'value'
          }
        ]
      }
    ]
  })
}

const deleteRequestMatch = (index: number) => {
  routeList.value[index].requestMatch = []
}

const addRequestMatch = (index: number) => {
  routeList.value[index].requestMatch = [
    {
      type: 'host',
      condition: '=',
      value: '127.0.0.1'
    },
    {
      type: 'application',
      condition: '=',
      value: 'appName'
    },
    {
      type: 'method',
      condition: '=',
      value: 'methodName'
    },
    {
      type: 'arguments',
      list: [
        {
          index: 0,
          condition: '=',
          value: 'arg0'
        }
      ]
    },
    {
      type: 'attachments',
      list: [
        {
          myKey: 'key',
          condition: '=',
          value: 'value'
        }
      ]
    },
    {
      type: 'other',
      list: [
        {
          myKey: 'key',
          condition: '=',
          value: 'value'
        }
      ]
    }
  ]
}

const deleteMatchConditionTypeItem = (type: string, index: number) => {
  console.log(type, index)
  routeList.value[index].selectedMatchConditionTypes = routeList.value[
    index
  ].selectedMatchConditionTypes.filter((item) => item !== type)
}

const deleteRouteDistributeMatchTypeItem = (type: string, index: number) => {
  routeList.value[index].selectedRouteDistributeMatchTypes = routeList.value[
    index
  ].selectedRouteDistributeMatchTypes.filter((item) => item !== type)
}

const argumentsColumns = [
  {
    dataIndex: 'index',
    key: 'index',
    title: '参数索引'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

// add argumentsItem
const addArgumentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    index: 0,
    condition: '=',
    value: ''
  })
}

// deleteArgumentsItem
const deleteArgumentsItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  argumentsIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'arguments')
    return
  }

  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(argumentsIndex, 1)
}

// attachments
const attachmentsColumns = [
  {
    dataIndex: 'myKey',
    key: 'myKey',
    title: '键'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

const addAttachmentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    key: 'key',
    condition: '=',
    value: ''
  })
}

const deleteAttachmentsItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  attachmentsItemIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'attachments')
    return
  }
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(
    attachmentsItemIndex,
    1
  )
}

// other
const otherColumns = [
  {
    dataIndex: 'myKey',
    key: 'myKey',
    title: '键'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

const addOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    myKey: '',
    condition: '=',
    value: ''
  })
}

const deleteOtherItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  otherItemIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'other')
    return
  }
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(otherItemIndex, 1)
}

const addRouteDistributeOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.push({
    myKey: '',
    condition: '=',
    value: ''
  })
}

const deleteRouteDistributeOtherItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  otherItemIndex: number
) => {
  if (routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedRouteDistributeMatchTypes = routeList.value[
      routeItemIndex
    ].selectedRouteDistributeMatchTypes.filter((item) => item !== 'other')
    return
  }

  routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.splice(otherItemIndex, 1)
}
</script>

<style lang="less" scoped>
.__container_routingRule_detail {
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
