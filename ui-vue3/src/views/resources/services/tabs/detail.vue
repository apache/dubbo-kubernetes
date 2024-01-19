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
      <a-flex class="flex-column" vertical>
        <a-card title="服务名:" class="card-content" size="small">
          <p>org.apache.dubbo.samples.UserService</p>
        </a-card>
        <a-card title="服务版本&分组:" class="card-content" size="small">
          <div>
            <a-tag color="cyan">version=v1</a-tag>
            <a-tag color="cyan">version=2.0,group=group1</a-tag>
          </div>
        </a-card>
        <a-card title="RPC协议:" class="card-content" size="small">
          <p>triple</p>
        </a-card>
        <a-card title="延迟注册时间:" class="card-content" size="small">
          <p>3000ms</p>
        </a-card>
        <a-card title="超时时间:" class="card-content" size="small">
          <p>3000ms</p>
        </a-card>
        <a-card title="重试次数:" class="card-content" size="small">
          <p>3</p>
        </a-card>
      </a-flex>
      <a-flex class="flex-column" vertical>
        <a-card title="请求总量(1min):" class="card-content" size="small">
          <p>1384</p>
        </a-card>
        <a-card title="平均RT(1min):" class="card-content" size="small">
          <p>96ms</p>
        </a-card>
        <a-card title="平均qps(1min):" class="card-content" size="small">
          <p>12</p>
        </a-card>
        <a-card title="是否过时:" class="card-content" size="small">
          <p>false</p>
        </a-card>
      </a-flex>
    </a-flex>
  </div>
</template>

<script setup lang="ts"></script>
<style lang="less" scoped>
.__container_services_tabs_detail {
  .flex-column {
    width: 50%;
    .card-content {
      width: 90%;
      margin-bottom: 10px;
    }
  }
}
</style>
