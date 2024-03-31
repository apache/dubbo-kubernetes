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
  <div class="__container_app_config">
    <config-page :options="options">
      <template v-slot:form_log="{ current }">
        <a-form-item :label="$t('applicationDomain.operatorLog')" name="logFlag">
          <a-switch v-model:checked="current.form.logFlag"></a-switch>
        </a-form-item>
      </template>
      <template v-slot:form_flow="{ current }"> 2 </template>
      <template v-slot:form_gray="{ current }"> 3 </template>
    </config-page>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import ConfigPage from '@/components/ConfigPage.vue'
let options: any = reactive({
  list: [
    {
      title: 'applicationDomain.operatorLog',
      key: 'log',
      form: {
        logFlag: false
      },
      submit: (form: {}) => {
        return new Promise((resolve) => {
          setTimeout(() => {
            resolve(1)
          }, 1000)
        })
      },
      reset(form: any) {
        form.logFlag = false
      }
    },
    {
      title: 'applicationDomain.flowWeight',
      form: {},
      key: 'flow',
      submit(form: {}) {
        console.log(form)
      }
    },
    {
      title: 'applicationDomain.gray',
      form: {},
      key: 'gray',
      submit(form: {}) {
        console.log(form)
      }
    }
  ],
  current: [0]
})
onMounted(() => {
  console.log(333)
})
</script>
<style lang="less" scoped>
.__container_app_config {
}
</style>
