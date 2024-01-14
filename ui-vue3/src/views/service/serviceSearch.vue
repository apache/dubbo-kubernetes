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
    <a-table :columns="columns" :data-source="dataSource">
      <template #bodyCell="{ column, text }">
        <template v-if="column.dataIndex === 'service'">
          <a-button type="link" @click="viewDetail(text)">{{ text }}</a-button>
        </template>
      </template>
      <template
        #customFilterDropdown="{ setSelectedKeys, selectedKeys, confirm, clearFilters, column }"
      >
        <div style="padding: 8px">
          <a-input
            ref="searchInput"
            :placeholder="`Search ${column.dataIndex}`"
            :value="selectedKeys[0]"
            style="width: 188px; margin-bottom: 8px; display: block"
            @change="e => setSelectedKeys(e.target.value ? [e.target.value] : [])"
            @pressEnter="handleSearch(selectedKeys, confirm, column.dataIndex)"
          />
          <a-button
            type="primary"
            size="small"
            style="width: 90px; margin-right: 8px"
            @click="handleSearch(selectedKeys, confirm, column.dataIndex)"
          >
            <template #icon><SearchOutlined /></template>
            Search
          </a-button>
          <a-button size="small" style="width: 90px" @click="handleReset(clearFilters)">
            Reset
          </a-button>
        </div>
      </template>
      <template #customFilterIcon="{ filtered }">
        <search-outlined :style="{ color: filtered ? '#108ee9' : undefined }" />
      </template>
    </a-table>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { SearchOutlined } from '@ant-design/icons-vue';
import { reactive, ref } from 'vue';

const router = useRouter()
const searchInput = ref();
const columns = [
  {
    title: '服务',
    dataIndex: 'service',
    sorter: true,
    width: '30%',
    customFilterDropdown: true,
    onFilter: (value: string, record: string) => record.toString().toLowerCase().includes(value.toLowerCase()),
    onFilterDropdownOpenChange: visible => {
      if (visible) {
        setTimeout(() => {
          searchInput.value.focus();
        }, 100);
      }
    },
  },
  {
    title: '接口数',
    dataIndex: 'interfaceNumber',
    width: '10%'
  },
  {
    title: '生产者/消费者实例',
    dataIndex: 'instance',
    width: '15%'
  },
  {
    title: '近 1min QPS',
    dataIndex: 'QPS',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min RT',
    dataIndex: 'RT',
    sorter: true,
    width: '15%'
  },
  {
    title: '近 1min 请求总量',
    dataIndex: 'requestTotal',
    sorter: true,
    width: '15%'
  }
]

const state = reactive({
  searchText: '',
  searchedColumn: '',
});

const handleSearch = (selectedKeys, confirm, dataIndex) => {
  confirm();
  state.searchText = selectedKeys[0];
  state.searchedColumn = dataIndex;
};

const handleReset = clearFilters => {
  clearFilters({ confirm: true });
  state.searchText = '';
};

const dataSource = [
  {},
  {},
  {
    service: '123123'
  },
  {},
  {},
  {}
]

const viewDetail = (service: string) => {
  router.push('/serviceDetail');
}
</script>
<style lang="less" scoped></style>
