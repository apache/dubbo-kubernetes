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
  <div class="__container_menu">
    <a-menu mode="inline" :items="items" @click="handleClick">
      <!--      <a-sub-menu v-if="!route.meta.hidden">-->
      <!--        <template #title>{{ $t(route.name) }}</template>-->
      <!--        <template v-if="route.children&&route.children.length>0">-->
      <!--          <layout_sub_menu v-for="item in route.children" :menu-info="item" :key="item.key"/>-->
      <!--        </template>-->
      <!--      </a-sub-menu>-->
    </a-menu>
  </div>
</template>

<script setup lang="ts">
import type {RouteRecordType} from '@/router/defaultRoutes'
import {routes as defautlRoutes} from '@/router/defaultRoutes'
import {deepCopy} from '@/utils/ObjectUtil'
import type {ItemType, MenuProps} from 'ant-design-vue'
import type {ComponentInternalInstance} from 'vue'
import {computed, getCurrentInstance, h, reactive} from 'vue'
import _ from 'lodash-es'
import {Icon} from '@iconify/vue'
import {useRouter} from 'vue-router'

const {
  appContext: {
    config: {globalProperties}
  }
} = <ComponentInternalInstance>getCurrentInstance()

const routesForMenu = deepCopy(defautlRoutes)

function getItem(
    label: any,
    title: any,
    key?: string,
    icon?: any,
    children?: ItemType[],
    type?: 'group'
): ItemType {
  if (icon) {
    icon = h(Icon, {icon: icon})
  }

  return {
    key,
    title,
    icon,
    children,
    label: computed(() => globalProperties.$t(label)),
    type
  } as ItemType
}

const items: ItemType[] = reactive([])

/**
 * pre handle routes
 * 1. if a route doesn't have any child, skip loop
 * 2. if a route has only one child, replace
 *
 * @param arr
 * @param arr2
 */
function prepareRoutes(arr: RouteRecordType[] | any, arr2: ItemType[]) {
  if (!arr || arr.length === 0) return
  for (let r of arr) {
    if (r.meta?.skip) {
      prepareRoutes(r.children, arr2)
      continue
    }
    if (!r.meta?.hidden) {
      let key = _.uniqueId('route_menu_')
      if (!r.children || r.children.length === 0) {
        arr2.push(getItem(r.name, r.path, key, r.meta?.icon))
      } else if (r.children.length === 1) {
        arr2.push(getItem(r.children[0].name, r.path, key, r.children[0].meta?.icon))
      } else {
        const tmp: ItemType[] = reactive([])
        prepareRoutes(r.children, tmp)
        arr2.push(getItem(r.name, r.path, key, r.meta?.icon, tmp))
      }
    }
  }
}

prepareRoutes(routesForMenu, items)

const router = useRouter()

const handleClick: MenuProps['onClick'] = (e) => {
  router.push(<string>e.item?.title)
}
</script>
<style lang="less" scoped>
.__container_menu {
  .icon-wrapper {
    .icon {
      font-size: 20px;
      margin-right: 5px;
      margin-bottom: -4px;
      font-weight: 700;
    }
  }
}
</style>
