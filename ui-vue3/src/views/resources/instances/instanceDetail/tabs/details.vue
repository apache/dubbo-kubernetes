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
import { ref } from 'vue'

// instance name
const instanceName = ref('shop-user-base-7e33f1e')

// Click on the application name to view the application
const checkApplication = () => {}
</script>

<template>
  <!-- detail -->
  <div class="detailBox">
    <!-- left-->
    <a-typography>
      <!-- instanceName -->
      <a-typography-title :level="5"> {{ $t('instanceName') }}: </a-typography-title>
      <a-typography-paragraph copyable>
        {{ instanceName }}
      </a-typography-paragraph>

      <!-- deployState -->
      <a-typography-title :level="5"> {{ $t('deployState') }}: </a-typography-title>
      <a-typography-paragraph type="success"> Running </a-typography-paragraph>
      <a-typography-paragraph type="danger"> Stop </a-typography-paragraph>

      <!-- registerStates -->
      <a-typography-title :level="5"> {{ $t('registerStates') }}: </a-typography-title>
      <a-typography-paragraph type="success"> Registed </a-typography-paragraph>
      <a-typography-paragraph type="danger"> UnRegisted </a-typography-paragraph>

      <!-- instanceIP -->
      <a-typography-title :level="5"> {{ $t('instanceIP') }}: </a-typography-title>
      <a-typography-paragraph copyable> 192.168.0.1 </a-typography-paragraph>

      <!-- Dubbo Port -->
      <a-typography-title :level="5"> {{ $t('dubboPort') }}: </a-typography-title>
      <a-typography-paragraph copyable> 2088 </a-typography-paragraph>

      <!-- whichApplication -->
      <a-typography-title :level="5"> {{ $t('whichApplication') }}: </a-typography-title>
      <a-typography-link @click="checkApplication()"> shop-b </a-typography-link>

      <!-- Owning workload(k8s) -->
      <a-typography-title :level="5"> {{ $t('owningWorkload_k8s') }}: </a-typography-title>
      <a-typography-paragraph> shop-user-base </a-typography-paragraph>

      <!-- instanceLabel -->
      <a-typography-title :level="5"> {{ $t('instanceLabel') }}: </a-typography-title>
      <div class="tagsBox">
        <a-tag>app=shop-user</a-tag>
        <a-tag>version=v1</a-tag>
        <a-tag>region=shenzhen</a-tag>
      </div>
    </a-typography>

    <!-- right -->
    <a-typography>
      <!-- Creation time -->
      <a-typography-title :level="5"> {{ $t('creationTime_k8s') }}: </a-typography-title>
      <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>

      <!-- Start time -->
      <a-typography-title :level="5"> {{ $t('startTime_k8s') }}: </a-typography-title>
      <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>

      <!-- Register Time -->
      <a-typography-title :level="5"> {{ $t('registerTime') }}: </a-typography-title>
      <a-typography-paragraph> 20230/12/19 22:09:34 </a-typography-paragraph>

      <!-- deploy cluster -->
      <a-typography-title :level="5"> {{ $t('deployCluster') }}: </a-typography-title>
      <a-typography-paragraph> sz-ali-zk-f8otyo4r </a-typography-paragraph>

      <!-- Register cluster -->
      <a-typography-title :level="5"> {{ $t('registerCluster') }}: </a-typography-title>
      <a-typography-paragraph> sz-ali-zk-f8otyo4r </a-typography-paragraph>

      <!-- Node IP -->
      <a-typography-title :level="5"> {{ $t('nodeIP') }}: </a-typography-title>
      <a-typography-paragraph copyable> 30.33.0.1 </a-typography-paragraph>

      <!-- image -->
      <a-typography-title :level="5"> {{ $t('instanceImage_k8s') }}: </a-typography-title>
      <a-typography-paragraph copyable>
        apache/org.apahce.dubbo.samples.shop-user:v1
      </a-typography-paragraph>

      <!-- health examination -->
      <a-typography-title :level="5"> {{ $t('healthExamination') }}: </a-typography-title>
      <div class="healthBox">
        <p class="white_space">启动探针（StartupProbe）：关闭</p>
        <p class="white_space">就绪探针（ReadinessProbe）：开启 类型: Http 端口：22222</p>
        <p class="white_space">存活探针（LivenessProbe）：开启 类型: Http 端口：22222</p>
      </div>
    </a-typography>
  </div>
</template>

<style lang="less" scoped>
.white_space {
  white-space: pre;
}

.detailBox {
  margin: 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;

  .tagsBox {
    border: 1px solid black;
    border-radius: 10px;
    padding: 10px;
    display: flex;
    align-items: center;
    justify-content: space-around;
  }

  .healthBox {
    border: 1px solid black;
    border-radius: 10px;
    padding: 10px;
    display: flex;
    align-items: flex-start;
    flex-direction: column;
    justify-content: center;
  }
}
</style>
