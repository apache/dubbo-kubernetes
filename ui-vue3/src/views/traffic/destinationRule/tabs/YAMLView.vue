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
  <!--  editor-->
  <a-flex style="width: 100%">
    <a-col :span="isDrawerOpened ? 16 : 24">
      <a-flex vertical align="end">
        <a-button type="text" style="color: #0a90d5" @click="isDrawerOpened = !isDrawerOpened">
          {{ $t('flowControlDomain.versionRecords') }}
          <DoubleLeftOutlined v-if="!isDrawerOpened" />
          <DoubleRightOutlined v-else />
        </a-button>

        <MonacoEditor
          :modelValue="YAMLValue"
          theme="vs-dark"
          :height="280"
          language="yaml"
          :readonly="isReadonly"
        />
      </a-flex>
    </a-col>

    <a-col :span="isDrawerOpened ? 8 : 0">
      <a-card v-show="isDrawerOpened">
        <a-card>
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
</template>

<script setup lang="ts">
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import { ref } from 'vue'

const isReadonly = ref(true)

const isDrawerOpened = ref(false)

const YAMLValue = ref(
  'configVersion: v3.0\n' +
    'force: true\n' +
    'enabled: true\n' +
    'key: shop-detail\n' +
    'tags:\n' +
    '  - name: gray\n' +
    '    match:\n' +
    '      - key: env\n' +
    '        value:\n' +
    '          exact: gray'
)
</script>

<style scoped lang="less">
.editor {
  width: fit-content;
}

.drawer {
  width: 0;
  min-height: 300px;
  max-height: 100%;
  transition: all 0.3s ease-in-out;
}

.drawerOpened {
  width: 250px;
}
</style>
