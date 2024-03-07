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
      <a-descriptions class="description-column" :column="2" layout="vertical">
        <a-descriptions-item
          v-for="(v, key, idx) in apiData.detail?.data"
          v-show="!!v"
          :labelStyle="{ fontWeight: 'bold' }"
          :label="$t('applicationDomain.' + key)"
        >
          <template #title> 111 </template>
          <template v-if="v.length < 3">
            <p v-for="item in v" class="description-item-content no-card">
              {{ item }}
            </p>
          </template>

          <a-card class="description-item-card" v-else>
            <p v-for="item in v" @click="copyIt(item)" class="description-item-content with-card">
              {{ item }} <CopyOutlined />
            </p>
          </a-card>
        </a-descriptions-item>
      </a-descriptions>
    </a-flex>
  </div>
</template>

<script setup lang="ts">
import { PRIMARY_COLOR } from '@/base/constants'
import { getApplicationDetail } from '@/api/service/app'
import { computed, onMounted, reactive, getCurrentInstance } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import type { ComponentInternalInstance } from 'vue'
const apiData: any = reactive({})
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

let __ = PRIMARY_COLOR
onMounted(async () => {
  apiData.detail = await getApplicationDetail({})
})
const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}
</script>
<style lang="less" scoped>
.__container_app_detail {
  .description-item-content {
    &.no-card {
      padding-left: 20px;
    }
    &.with-card:hover {
      color: v-bind('PRIMARY_COLOR');
    }
  }
  .description-item-card {
    width: 80%;
  }
}
</style>
