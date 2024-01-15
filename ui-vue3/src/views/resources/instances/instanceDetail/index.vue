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
<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref } from 'vue';

import { useRouter } from "vue-router";

const Router = useRouter()

// 实例名称
const instanceName = ref("shop-user-base-7e33f1e")

// 当前标签页索引
const activeKey = ref('details');

// tab-pane配置项
const tabPanesOption = [
  {
    key: "details",
    tab: "详情"
  },
  {
    key: "monitor",
    tab: "监控"
  },
  {
    key: "linkTracking",
    tab: "链路追踪"
  },
  {
    key: "configuration",
    tab: "配置"
  },
  {
    key: "event",
    tab: "事件"
  },
]

// 切换tab页面
const changeTab = (activeKey: string) => {
  console.log(activeKey);

  Router.push({
    name: activeKey,
    query: {
      instanceName: "name"
    }
  })
}
</script>



<template>
  <div class="__container_home_index">
    <a-card :title="instanceName" style="width: 100%">
      <template #extra>

      </template>
      <a-tabs v-model:activeKey="activeKey" @change="changeTab" >
        <a-tab-pane :key="tab.key" :tab="$t(tab.key)" v-for="tab in tabPanesOption">

         

        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>
  

<style lang="less" scoped></style>
  