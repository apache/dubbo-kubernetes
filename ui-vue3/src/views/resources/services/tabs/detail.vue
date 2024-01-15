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
    <a-flex horizontal>
      <a-flex class="flex-column" gap="middle" vertical>
        <a-card title="服务名：" class="card-content">
          <p>org.apache.dubbo.samples.UserService</p>
        </a-card>
        <a-card title="服务版本&分组：" class="card-content">
          <div>
            <a-tag color="cyan">version=v1</a-tag>
            <a-tag color="cyan">version=2.0,group=group1</a-tag>
          </div>
        </a-card>
        <a-card title="自定义参数：" class="card-content">
          <p>test</p>
        </a-card>
      </a-flex>
      <a-flex class="flex-column" gap="middle" vertical>
        <a-card title="近1min请求量：" class="card-content">
          <p>test</p>
        </a-card>
        <a-card title="近1min RT：" class="card-content">
          <p>test</p>
        </a-card>
        <a-card title="近1min QPS：" class="card-content">
          <p>test</p>
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
      width: 80%;
    }
  }
}
</style>
