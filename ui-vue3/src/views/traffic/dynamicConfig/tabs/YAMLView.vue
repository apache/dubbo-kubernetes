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
  <a-card>
    <a-spin :spinning="loading">
      <a-flex style="width: 100%">
        <a-col :span="isDrawerOpened ? 24 - sliderSpan : 24" class="left">
          <a-flex vertical align="end">
            <a-row style="width: 100%" justify="space-between">
              <a-col :span="12">
<!--                <a-button-->
<!--                  type="primary"-->
<!--                  size="small"-->
<!--                  style="width: 80px; float: left"-->
<!--                  @click="saveConfig"-->
<!--                >-->
<!--                  {{ $t('form.save') }}-->
<!--                </a-button>-->
                <a-tag v-if='modify' color="red" :bordered=false>*改动未保存</a-tag>
                <a-tag v-else :color="PRIMARY_COLOR" :bordered=false>配置无改动</a-tag>
              </a-col>
              <a-col :span="12">
                <!--                todo 版本记录后续添加-->
                <a-button
                  type="text"
                  style="color: #0a90d5; float: right; margin-top: -5px"
                  @click="isDrawerOpened = !isDrawerOpened"
                >
                  {{ $t('flowControlDomain.versionRecords') }}
                  <DoubleLeftOutlined v-if="!isDrawerOpened" />
                  <DoubleRightOutlined v-else />
                </a-button>
              </a-col>
            </a-row>

            <div class="editorBox">
              <MonacoEditor
                v-model:modelValue="YAMLValue"
                theme="vs-dark"
                :height="500"
                language="yaml"
                :readonly="!isEdit"
              />
            </div>
          </a-flex>
        </a-col>

        <a-col :span="isDrawerOpened ? sliderSpan : 0" class="right">
          <a-card v-if="isDrawerOpened" class="sliderBox">
            <a-card v-for="i in 2" :key="i">
              <p>修改时间: 2024/3/20 15:20:31</p>
              <p>版本号: xo842xqpx834</p>

              <a-flex justify="flex-end">
                <a-button type="text" style="color: #0a90d5">查看</a-button>
                <a-button type="text" style="color: #0a90d5">回滚</a-button>
              </a-flex>
            </a-card>
          </a-card>
        </a-col>
      </a-flex>
    </a-spin>
  </a-card>

  <a-flex v-if="isEdit" style="margin-top: 30px">
    <a-button type="primary" @click="saveConfig">保存</a-button>
    <a-button style="margin-left: 30px" @click="resetConfig">重置</a-button>
  </a-flex>
</template>

<script setup lang="ts">
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import {computed, inject, onMounted, reactive, ref} from 'vue'
import { useRoute } from 'vue-router'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { getConfiguratorDetail, saveConfiguratorDetail } from '@/api/service/traffic'
// @ts-ignore
import yaml from 'js-yaml'
import { message } from 'ant-design-vue'
import {PRIMARY_COLOR} from "@/base/constants";

const route = useRoute()
const isEdit = ref(route.params.isEdit === '1')
const isDrawerOpened = ref(false)
const loading = ref(false)
const sliderSpan = ref(8)

const YAMLValue = ref()
const initValue = ref()
onMounted(async () => {
  await initConfig();
})
const modify = computed(()=>{
  console.log(initValue.value)
  console.log(JSON.stringify(YAMLValue.value))
  return initValue.value !== JSON.stringify(YAMLValue.value)
})
async function initConfig(){
  const res = await getConfiguratorDetail({ name: route.params?.pathId })
  const json = yaml.dump(res.data) // 输出为 json 格式
  initValue.value = JSON.stringify(json)
  YAMLValue.value = json
}
async function resetConfig(){
  loading.value = true
  try {
    await initConfig();
    message.success('config reset success')
  }finally {
    loading.value = false
  }
}
async function saveConfig() {
  loading.value = true
  let newVal = yaml.load(YAMLValue.value)
  try {
    let res = await saveConfiguratorDetail({ name: route.params?.pathId }, newVal)
    message.success('config save success')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="less">
.editorBox {
  border-radius: 0.3rem;
  overflow: hidden;
  width: 100%;
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
</style>
