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

<script lang="ts" setup>
import { ref } from "vue"
import { Icon } from '@iconify/vue'
import type { SelectProps } from 'ant-design-vue';
import { useRouter, RouterLink } from "vue-router";
import type { TableColumnType, TableProps } from 'ant-design-vue';

const Router = useRouter()


// Search Options
const searchOptions = ref<SelectProps['options']>([
    {
        value: 'IP',
        label: 'IP',
    },
    {
        value: '名称',
        label: '名称',
    },
]);
// Select search options
const searchOption = ref("IP")


// search keywords
const keyword = ref("")

// Search instances
const onSearch = () => {

}



// defined types
type TableDataType = {
    instanceIP: string;
    instanceName: string;
    deployState: object;
    CPU: string;
    node: string;
    labels: Array<string>;
    memory: string,
    registerStates: object,
    registerCluster: string,
    startTime: string,
    registerTime: string
};


//  Configure instance list header
const columns: TableColumnType<TableDataType>[] = [
    {
        title: '实例ip',
        dataIndex: 'instanceIP',
        key: 'instanceIP',
    },
    {
        title: '实例名称',
        dataIndex: 'instanceName',
        key: 'instanceName',
    },
    {
        title: '部署状态',
        dataIndex: 'deployState',
        key: 'deployState',
        defaultSortOrder: 'descend',

    },
    {
        title: '注册状态',
        dataIndex: 'registerStates',
        key: 'registerStates',
        defaultSortOrder: 'descend',

    },
    {
        title: '注册集群',
        dataIndex: 'registerCluster',
        key: 'registerCluster',
        defaultSortOrder: 'descend',

    },
    {
        title: "CPU",
        dataIndex: "CPU",
        key: "CPU",
        defaultSortOrder: 'descend',
        sorter: (a: TableDataType, b: TableDataType) => parseFloat(a.CPU) - parseFloat(b.CPU),
    },
    {
        title: "内存",
        dataIndex: "memory",
        key: "memory",
        defaultSortOrder: 'descend',
        sorter: (a: TableDataType, b: TableDataType) => parseFloat(a.memory) - parseFloat(b.memory),
    },
    {
        title: "启动时间(k8s)",
        dataIndex: "startTime",
        key: "startTime",
        defaultSortOrder: 'descend',
        sorter: (a: TableDataType, b: TableDataType) => parseFloat(a.memory) - parseFloat(b.memory),
    },
    {
        title: "注册时间",
        dataIndex: "registerTime",
        key: "registerTime"
    },
    {
        title: "标签",
        dataIndex: "labels",
        key: "labels"
    }
]

// Instance List Data
const instanceList: TableDataType[] = [
    {
        "deployState": {
            "label": "Running",
            "value": "Running",
            "level": "healthy"
        },
        "registerStates":
        {
            "label": "Unregisted",
            "value": "Unregisted",
            "level": "warn"
        }
        ,
        CPU: "1.0",
        memory: "1022",
        "instanceIP": "45.7.37.227",
        "instanceName": "shop-user",
        "labels": [
            "app=shop-user",
            "version=v1",
            "region=beijing"
        ],
        "startTime": "2023/12/19  22:12:34",
        "registerTime": "2023/12/19   22:16:56",
        "registerCluster": "sz-ali-zk-f8otyo4r",
        "node": "30.33.0.1",
    },
    {
        "deployState": {
            "label": "Running",
            "value": "Running",
            "level": "healthy"
        },
        "registerStates":
        {
            "label": "Unregisted",
            "value": "Unregisted",
            "level": "warn"
        }
        ,
        CPU: "1.0",
        memory: "1022",
        "instanceIP": "45.7.37.227",
        "instanceName": "shop-user",
        "labels": [
            "app=shop-user",
            "version=v1",
            "region=beijing"
        ],
        "startTime": "2023/12/19  22:12:34",
        "registerTime": "2023/12/19   22:16:56",
        "registerCluster": "sz-ali-zk-f8otyo4r",
        "node": "30.33.0.1",
    },
    {
        "deployState": {
            "label": "Running",
            "value": "Running",
            "level": "healthy"
        },
        "registerStates":
        {
            "label": "Unregisted",
            "value": "Unregisted",
            "level": "warn"
        }
        ,
        CPU: "1.0",
        memory: "1022",
        "instanceIP": "45.7.37.227",
        "instanceName": "shop-user",
        "labels": [
            "app=shop-user",
            "version=v1",
            "region=beijing"
        ],
        "startTime": "2023/12/19  22:12:34",
        "registerTime": "2023/12/19   22:16:56",
        "registerCluster": "sz-ali-zk-f8otyo4r",
        "node": "30.33.0.1",
    },

]



// Page Sorter Data
const current = ref<number>(1);

const onChangePageNum = (pageNumber: number) => {
    console.log('Page: ', pageNumber);
};

// View instance details
const checkDetail = (instanceName: string) => {
    Router.push({
        path: "details/" + instanceName,
    })
}




</script>

<template>
    <div class="__container_home_index">
        <a-card :title="$t('instance')" style="width:100%">
            <template #extra>
                <a-space align="start">
                    <a-select ref="select" v-model:value="searchOption" style="width: 120px" :options="searchOptions">
                    </a-select>

                    <a-input-search v-model:value="keyword" placeholder="input search text" enter-button
                        @search="onSearch" />
                </a-space>

            </template>


            <!-- Instance List -->
            <a-table :dataSource="instanceList" :columns="columns" :pagination="false" :scroll="{ x: 1500 }">

                <!-- Table Header -->

                <template #headerCell="{ column }">
                    {{ $t(column.dataIndex) }}
                </template>


                <!-- body -->
                <template #bodyCell="{ column, text }">

                    <!-- The instance name can be clicked to jump to -->
                    <template v-if="column.key === 'instanceName'">
                        <router-link :to='`details/${text}`'> {{ text }} </router-link>
                    </template>

                    <!-- deployState -->
                    <template v-if="column.key === 'deployState'">
                        {{ text.label }}
                    </template>

                    <!-- registerStates -->
                    <template v-if="column.key === 'registerStates'">
                        {{ text.label }}
                    </template>

                    <!-- Used Memory -->
                    <template v-else-if="column.key === 'memory'">
                        {{ text }}MB
                    </template>
                    <!-- Instance label -->
                    <template v-else-if="column.key === 'labels'">
                        <a-tag v-for="tag in text">{{ tag }}</a-tag>
                    </template>
                </template>
            </a-table>
            <!--  pager -->
            <a-space style="width: 100%;display: flex;align-items: center;justify-content: center;margin-top: 10px;">
                <a-pagination v-model:current="current" show-quick-jumper show-less-items :total="500"
                    @change="onChangePageNum" />
            </a-space>
        </a-card>
    </div>
</template>

<style lang="less" scoped></style>