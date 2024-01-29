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
  <div class="__container_services_tabs_detail">
    <a-flex>
      <a-descriptions class="description-column" :column="1" layout="vertical">
        <a-descriptions-item label="服务名">
          <p class="description-item-content">org.apache.dubbo.samples.UserService</p>
        </a-descriptions-item>
        <a-descriptions-item label="服务版本&分组">
          <div class="description-item-content">
            <a-tag color="cyan">version=v1</a-tag>
            <a-tag color="cyan">version=2.0,group=group1</a-tag>
          </div>
        </a-descriptions-item>
        <a-descriptions-item label="RPC协议">
          <p class="description-item-content">triple</p>
        </a-descriptions-item>
        <a-descriptions-item label="延迟注册时间">
          <p class="description-item-content">3000ms</p>
        </a-descriptions-item>
        <a-descriptions-item label="超时时间">
          <p class="description-item-content">3000ms</p>
        </a-descriptions-item>
        <a-descriptions-item label="重试次数">
          <p class="description-item-content">3</p>
        </a-descriptions-item>
      </a-descriptions>
      <a-descriptions class="description-column" :column="1" layout="vertical">
        <a-descriptions-item label="请求总量(1min)">
          <p class="description-item-content">1384</p>
        </a-descriptions-item>
        <a-descriptions-item label="平均RT(1min)">
          <p class="description-item-content">96ms</p>
        </a-descriptions-item>
        <a-descriptions-item label="平均qps(1min)">
          <p class="description-item-content">12</p>
        </a-descriptions-item>
        <a-descriptions-item label="是否过时">
          <p class="description-item-content">false</p>
        </a-descriptions-item>
      </a-descriptions>
    </a-flex>
  </div>
</template>

<script setup lang="ts"></script>
<style lang="less" scoped>
.__container_services_tabs_detail {
  .description-item-content {
    margin-left: 20px;
    width: 90%;
  }
}
</style>
