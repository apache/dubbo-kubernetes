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
  <div class="__container_router_tab_index" :key="key">
    <a-tabs
      v-if="tabRoute.meta.tab"
      @change="router.push({ name: activeKey || '' })"
      v-model:activeKey="activeKey"
    >
      <a-tab-pane :key="v.name" v-for="v in tabRouters">
        <template #tab>
          <span>
            <Icon style="margin-bottom: -2px" :icon="v.meta.icon"></Icon>
            {{ $t(v.name) }}
          </span>
        </template>
        <router-view :key="key" />
      </a-tab-pane>
    </a-tabs>
    <router-view v-if="!tabRoute.meta.tab" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Icon } from '@iconify/vue'
import { useRoute, useRouter } from 'vue-router'
import _ from 'lodash'
import { PRIMARY_COLOR } from '@/base/constants'
import type { RouterMeta } from '@/router/RouterMeta'
const router = useRouter()
const tabRoute = useRoute()
const tabRouters = computed(() => {
  let meta: any = tabRoute.meta
  return meta?.parent?.children?.filter((x: any): any => x.meta.tab)
})
let activeKey = ref(tabRoute.name)
let transitionFlag = ref(true)
let key = _.uniqueId('__tab_page')
router.beforeEach((to, from, next) => {
  key = _.uniqueId('__tab_page')
  transitionFlag.value = false
  activeKey.value = <string>to.name
  next()
  setTimeout(() => {
    transitionFlag.value = true
  }, 500)
})
console.log(tabRoute)
</script>
<style lang="less" scoped></style>
