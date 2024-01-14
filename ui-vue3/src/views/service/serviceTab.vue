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
  <div class="__container_home_index">
    <a-tabs v-model:activeKey="activeKey">
      <a-tab-pane v-for="tab in tabs" :key="tab.title" :tab="tab.title" @click="() => {
        console.log(123)
        router.push(tab.title)
      }">
        <router-view/>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

  const tabs = [
  {
    title: 'serviceDetails'
  },
  {
    title: 'serviceDebug'
  },
  {
    title: 'serviceDistribution',
  },
  {
    title: 'monitor',
  },
  {
    title: 'tracing',
  },
  {
    title: 'event',
  }
]
</script>
<style lang="less" scoped></style>
