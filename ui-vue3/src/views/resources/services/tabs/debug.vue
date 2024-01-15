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
  <div class="__container_services_tabs_debug">
    <a-tabs v-model:activeKey="activeKey" tab-position="left" :style="{ height: '800px' }" :tabBarStyle="{ width: '200px' }">
      <a-tab-pane v-for="i in 100" :key="i" :tab="`Tab-${i}`">
        <a-flex vertical>
          <a-flex>
            <div class="div-column">
              <a-card title="接口" class="card-content">
                <p>org.apache.dubbo.samples.UserService:getPermissions</p>
              </a-card>
            </div>
            <div class="div-column">
              <a-card title="接口" class="card-content">
                <a-select v-model:value="value" size="large" :options="options" style="width: 100%;"></a-select>
              </a-card>
            </div>
          </a-flex>
          <a-flex>
            <div class="div-column">
              <a-card title="入参类型" class="card-content">
                <a-textarea v-model:value="responseValue" placeholder="请输入" :rows="4" />
              </a-card>
            </div>
            <div class="div-column">
              <a-card title="实例IP" class="card-content">
                <a-textarea v-model:value="responseValue" placeholder="请输入" :rows="4" />
              </a-card>
            </div>
          </a-flex>
          <a-flex>
            <div class="div-column">
              <a-card class="card-content">
                <a-textarea v-model:value="requestValue" placeholder="请输入" :rows="4" />
                <template #title>
                  <a-flex class="request-card-title">
                    <span>请求</span>
                    <span>选择生产者：<a-select v-model:value="value" :options="options" style="width: 200px;"></a-select></span>
                  </a-flex>
                </template>
              </a-card>
            </div>
            <div class="div-column">
              <a-card title="响应" class="card-content">
                <a-textarea v-model:value="responseValue" placeholder="请输入" :rows="4" />
              </a-card>
            </div>
          </a-flex>
        </a-flex>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
const activeKey = ref(1)

const options = reactive([
  {
    value: 'version=,group='
  },
  {
    value: 'version=1.0.0,group='
  },
  {
    value: 'version=,group=group1'
  },
  {
    value: 'version=1.0.0,group=group1'
  }
])

const value = ref(options[0].value)

const requestValue = ref('');
const responseValue = ref('');
</script>
<style lang="less" scoped>
.__container_services_tabs_debug {
  width: 100%;
  .div-column {
    width: 50%;
    .card-content {
      width: 80%;
      .request-card-title{
        align-items: center;
        justify-content: space-between;
      }
    }
  }
}
</style>
