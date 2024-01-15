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
  <div class="__container_services_tabs_distribution">
    <a-flex vertical>
      <a-flex class="service-filter">
        <a-flex>
          <div>
            <span>版本&分组：</span>
            <a-select v-model:value="value" :options="options" class="service-filter-select"></a-select>
          </div>
          <div>
            <a-radio-group v-model:value="value1" button-style="solid" class="service-filter-radios">
              <a-radio-button value="producer">生产者</a-radio-button>
              <a-radio-button value="consumer">消费者</a-radio-button>
            </a-radio-group>
          </div>
        </a-flex>
        <a-input-search
          v-model:value="searchValue"
          placeholder="请输入"
          style="width: 300px"
          @search="()=>{}"
          enter-button
        />
      </a-flex>
      <a-table :columns="tableColumns" :data-source="dataSource"></a-table>
    </a-flex>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';

const searchValue = ref('');
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
const value1 = ref('producer')
const tableColumns = [
  {
    title: '应用名',
    dataIndex: 'applicationName',
    width: '25%'
  },
  {
    title: '实例数',
    dataIndex: 'instanceNum',
    width: '25%'
  },
  {
    title: '实例ip',
    dataIndex: 'instanceIP',
    width: '50%'
  }
]
const dataSource = reactive([])
</script>
<style lang="less" scoped>
.__container_services_tabs_distribution {
  .service-filter {
    justify-content: space-between;
    margin-bottom: 20px;
    .service-filter-select {
      width: 300px;
    }
    .service-filter-radios {
      margin-left: 50px;
    }
  }
}
</style>
