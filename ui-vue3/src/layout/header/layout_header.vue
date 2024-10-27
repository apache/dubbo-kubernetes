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
  <div class="__container_layout_header">
    <a-layout-header class="header">
      <a-row>
        <a-col :span="2">
          <menu-unfold-outlined
            v-if="collapsed"
            class="trigger"
            @click="() => (collapsed = !collapsed)"
          />
          <menu-fold-outlined v-else class="trigger" @click="() => (collapsed = !collapsed)" />
        </a-col>
        <a-col :span="3"></a-col>
        <a-col :span="10" class="search-group">
          <a-input-group compact>
            <a-select v-model:value="searchType" class="select-type">
              <a-select-option v-for="option in searchTypeOptions" :value="option.value"
                >{{ option.label }}
              </a-select-option>
            </a-select>
            <a-auto-complete
              v-model:value="keywords"
              class="input-keywords"
              :placeholder="$t('globalSearchTip')"
              :options="candidates"
              @select="onSelect"
              @search="inputChange"
            />
            <a-button :icon="h(SearchOutlined)" class="search-icon" @click="inputChange"></a-button>
          </a-input-group>
        </a-col>
        <a-col :span="3"></a-col>
        <a-col :span="2">
          <a-segmented v-model:value="locale" :options="i18nConfig.opts" />
        </a-col>
        <a-col :span="2">
          <color-picker
            :pureColor="PRIMARY_COLOR"
            @pureColorChange="changeTheme"
            format="hex6"
            shape="circle"
            useType="pure"
          ></color-picker>
          <a-popover>
            <template #content>reset the theme</template>
            <Icon
              class="reset-icon"
              icon="material-symbols:reset-tv-outline"
              @click="resetTheme"
            ></Icon>
          </a-popover>
        </a-col>
        <a-col :span="2">
          <a-avatar @click="devTool.todo('avatar and user info')">
            <template #icon>
              <UserOutlined />
            </template>
          </a-avatar>
          <span class="username">张三</span>
        </a-col>
      </a-row>
    </a-layout-header>
  </div>
</template>

<script setup lang="ts">
import { MenuFoldOutlined, MenuUnfoldOutlined, UserOutlined } from '@ant-design/icons-vue'
import type { ComponentInternalInstance } from 'vue'
import { inject, ref, reactive, watch, h, getCurrentInstance, computed } from 'vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { changeLanguage, localeConfig } from '@/base/i18n'
import {
  LOCAL_STORAGE_LOCALE,
  LOCAL_STORAGE_THEME,
  PRIMARY_COLOR,
  PRIMARY_COLOR_DEFAULT
} from '@/base/constants'
import devTool from '@/utils/DevToolUtil'
import { Icon } from '@iconify/vue'
import { SearchOutlined } from '@ant-design/icons-vue'
import { globalSearch } from '@/api/service/globalSearch'
import { debounce } from 'lodash'
import type { SelectOption } from '@/types/common.ts'
import { searchApplications } from '@/api/service/app'
import { searchInstances } from '@/api/service/instance'
import { searchService } from '@/api/service/service'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

let __null = PRIMARY_COLOR
const collapsed = inject(PROVIDE_INJECT_KEY.COLLAPSED)
const i18nConfig = <typeof localeConfig>inject(PROVIDE_INJECT_KEY.LOCALE)
let locale = ref(localeConfig.locale)

function changeTheme(val: string) {
  localStorage.setItem(LOCAL_STORAGE_THEME, val)
  PRIMARY_COLOR.value = val
}

function resetTheme(val: string) {
  localStorage.removeItem(LOCAL_STORAGE_THEME)
  PRIMARY_COLOR.value = PRIMARY_COLOR_DEFAULT
}

watch(locale, (value) => {
  changeLanguage(value)
})

const searchTypeOptions = reactive([
  // Temporarily hidden, awaiting improvement
  // {
  //   label: 'IP',
  //   value: 'ip'
  // },
  {
    label: computed(() => globalProperties.$t('application')),
    value: 'appName'
  },
  {
    label: computed(() => globalProperties.$t('instance')),
    value: 'instanceName'
  },
  {
    label: computed(() => globalProperties.$t('service')),
    value: 'serviceName'
  }
])
const searchType = ref(searchTypeOptions[0].value)

const keywords = ref('')

const onSearch = async () => {
  const params = {
    keywords: keywords.value
  }

  // Public search processing function
  const globalSearch = async (searchFunc: Function, labelKey: string) => {
    const {
      data: { list }
    } = await searchFunc(params)

    // Using map instead of forEach is more concise.
    candidates.value = list.map((item: any) => ({
      label: item[labelKey],
      value: item[labelKey]
    }))
    console.log('candidates', candidates.value)
  }

  // Various types of search functions
  const globalSearchByIp = () => {
    // The IP search logic is undefined and left blank.
  }

  switch (searchType.value) {
    case 'ip':
      globalSearchByIp()
      break
    case 'appName':
      await globalSearch(searchApplications, 'appName')
      break
    case 'instanceName':
      await globalSearch(searchInstances, 'name')
      break
    case 'serviceName':
      await globalSearch(searchService, 'serviceName')
      break
    default:
      break
  }
}

// Listen for changes in searchType and trigger a search.
watch(searchType, async (newType) => {
  await onSearch() // When a change is detected, re-call the search function.
})

const candidates = ref<Array<SelectOption>>([])

const inputChange = debounce(onSearch, 300)

const onSelect = () => {}
</script>
<style lang="less" scoped>
.__container_layout_header {
  .header {
    background: v-bind('PRIMARY_COLOR');
    padding: 0;

    .search-group {
      display: flex;
      align-items: center;

      .select-type {
        width: 120px;
      }

      .input-keywords {
        width: calc(100% - 152px);
      }

      .search-icon {
        width: 32px;
      }
    }
  }

  .trigger {
    font-size: 20px;
    margin-left: 20px;
    color: white;
  }

  .username {
    color: white;
    padding: 5px;
  }

  .reset-icon {
    font-size: 25px;
    color: white;
    margin-bottom: -9px;
  }
}
</style>
