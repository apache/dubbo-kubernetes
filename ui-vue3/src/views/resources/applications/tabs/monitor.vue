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
  <div class="__container_tabDemo3">
    <!--    <div class="option">-->
    <!--      <a-button class="btn" @click="refresh"> refresh </a-button>-->
    <!--      <a-button class="btn" @click="newPageForGrafana"> grafana </a-button>-->
    <!--    </div>-->
    <a-spin class="spin" :spinning="!showIframe">
      <div class="__container_iframe_container">
        <iframe
          :onload="onIframeLoad"
          v-show="showIframe"
          id="grafanaIframe"
          :src="grafanaUrl"
          frameborder="0"
        ></iframe>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getApplicationMetricsInfo } from '@/api/service/app'
import { useRoute } from 'vue-router'

const route = useRoute()
let appNameParam: any = route.params?.pathId

let grafanaUrl = ref('')
let showIframe = ref(true)
onMounted(async () => {
  let res = await getApplicationMetricsInfo({})

  showIframe.value = false
  grafanaUrl.value = `${window.location.origin}/grafana/d/${res.data?.baseURL.split('/d/')[1].split('?')[0]}?var-application=${appNameParam}&kiosk=tv`
  console.log(grafanaUrl)
})

function refresh() {
  showIframe.value = false
  setTimeout(() => {
    showIframe.value = true
  }, 200)
}

function tryDo(handle: any) {
  try {
    handle()
  } catch (e) {
    console.log(e)
  }
}

function onIframeLoad() {
  console.log('The iframe has been loaded.')
  setTimeout(() => {
    try {
      let iframeDocument = document.querySelector('#grafanaIframe').contentDocument
      tryDo(() => {
        iframeDocument.querySelector('header').remove()
      })
      tryDo(() => {
        iframeDocument.querySelector(`[data-testid*='controls']`).remove()
      })
      setTimeout(() => {
        tryDo(() => {
          iframeDocument.querySelector(`[data-testid*='navigation mega-menu']`).remove()
        })
        tryDo(() => {
          for (let querySelectorAllElement of iframeDocument.querySelectorAll(
            `[data-testid*='Panel menu']`
          )) {
            console.log(querySelectorAllElement)
            querySelectorAllElement.remove()
          }
        })
      }, 1000)
    } catch (e) {}
    showIframe.value = true
  }, 1000)
}

function newPageForGrafana() {
  window.open(grafanaUrl.value, '_blank')
}
</script>
<style lang="less" scoped></style>
