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
  <div class="__container_app_detail">
    <a-flex>
      <a-descriptions title="" layout="vertical" :column="2">
        <!-- instanceName -->
        <a-descriptions-item
          :label="$t('instanceDomain.instanceName')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph copyable>
            {{ instanceName }}
          </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Creation time -->
        <a-descriptions-item
          :label="$t('instanceDomain.creationTime_k8s')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- deployState -->
        <a-descriptions-item
          :label="$t('instanceDomain.deployState')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph type="success"> Running </a-typography-paragraph>
          <a-typography-paragraph type="danger"> Stop </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Start time -->
        <a-descriptions-item
          :label="$t('instanceDomain.startTime_k8s')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- registerStates -->
        <a-descriptions-item
          :label="$t('instanceDomain.registerStates')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph type="success"> Registed </a-typography-paragraph>
          <a-typography-paragraph type="danger"> UnRegisted </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Register Time -->
        <a-descriptions-item
          :label="$t('instanceDomain.registerTime')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- instanceIP -->
        <a-descriptions-item
          :label="$t('instanceDomain.instanceIP')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph copyable> 192.168.0.1 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- deploy cluster -->
        <a-descriptions-item
          :label="$t('instanceDomain.deployCluster')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> sz-ali-zk-f8otyo4r </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Dubbo Port -->
        <a-descriptions-item
          :label="$t('instanceDomain.dubboPort')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph copyable> 2088 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Register cluster -->
        <a-descriptions-item
          :label="$t('instanceDomain.registerCluster')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> sz-ali-zk-f8otyo4r </a-typography-paragraph>
        </a-descriptions-item>

        <!-- whichApplication -->
        <a-descriptions-item
          :label="$t('instanceDomain.whichApplication')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-link @click="checkApplication()"> shop-b </a-typography-link>
        </a-descriptions-item>

        <!-- Node IP -->
        <a-descriptions-item
          :label="$t('instanceDomain.node')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph copyable> 30.33.0.1 </a-typography-paragraph>
        </a-descriptions-item>

        <!-- Owning workload(k8s) -->
        <a-descriptions-item
          :label="$t('instanceDomain.owningWorkload_k8s')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph> shop-user-base </a-typography-paragraph>
        </a-descriptions-item>

        <!-- image -->
        <a-descriptions-item
          :label="$t('instanceDomain.instanceImage_k8s')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-typography-paragraph copyable>
            apache/org.apahce.dubbo.samples.shop-user:v1
          </a-typography-paragraph>
        </a-descriptions-item>

        <!-- instanceLabel -->
        <a-descriptions-item
          :label="$t('instanceDomain.instanceLabel')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-card class="description-item-card">
            <a-tag>app=shop-user</a-tag>
            <a-tag>version=v1</a-tag>
            <a-tag>region=shenzhen</a-tag>
          </a-card>
        </a-descriptions-item>

        <!-- health examination -->
        <a-descriptions-item
          :label="$t('instanceDomain.healthExamination_k8s')"
          :labelStyle="{ fontWeight: 'bold' }"
        >
          <a-card class="description-item-card">
            <p class="white_space">启动探针(StartupProbe):关闭</p>
            <p class="white_space">就绪探针(ReadinessProbe):开启 类型: Http 端口:22222</p>
            <p class="white_space">存活探针(LivenessProbe):开启 类型: Http 端口:22222</p>
          </a-card>
        </a-descriptions-item>
      </a-descriptions>
    </a-flex>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

// instance name
const instanceName = ref('shop-user-base-7e33f1e')

// Click on the application name to view the application
const checkApplication = () => {}
</script>

<style lang="less" scoped></style>
