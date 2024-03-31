<script setup lang="ts"></script>

<template>
  <a-card>
    <a-row>
      <a-descriptions title="基础信息" />
      <a-col :span="12">
        <a-descriptions title="" layout="vertical" :column="1">
          <a-descriptions-item label="规则名" :labelStyle="{ fontWeight: 'bold', width: '100px' }">
            shop-user/StandardRouter
          </a-descriptions-item>
          <a-descriptions-item
            label="作用对象"
            :labelStyle="{ fontWeight: 'bold', width: '100px' }"
          >
            shop-user
          </a-descriptions-item>
        </a-descriptions>
      </a-col>
      <a-col :span="12">
        <a-descriptions title="" layout="vertical" :column="1">
          <a-descriptions-item
            label="创建时间"
            :labelStyle="{ fontWeight: 'bold', width: '100px' }"
          >
            2024/2/19/ 11:38:21
          </a-descriptions-item>
        </a-descriptions>
      </a-col>
    </a-row>
  </a-card>

  <a-card>
    <a-descriptions title="路由列表" />

    <a-card title="路由【1】">
      <a-space size="middle" direction="vertical">
        <span>名称：未指定</span>
        <a-space size="small">
          <span>服务生效范围:</span>
          <a-space>
            <a-space-compact direction="horizontal">
              <a-button type="primary">exact</a-button>
              <a-button type="primary">org.apache.dubbo.samples.UserService</a-button>
            </a-space-compact>
          </a-space>
        </a-space>
        <a-space size="middle">
          <span>路由:</span>
          <a-card style="min-width: 400px">
            <a-space direction="vertical" size="middle">
              <span>名称：未指定</span>
              <a-space size="middle">
                <span>匹配条件:</span>
              </a-space>
            </a-space>
          </a-card>
        </a-space>
      </a-space>
    </a-card>
  </a-card>
</template>

<style scoped lang="less">
.description-item-content {
  margin-left: 20px;
  width: 90%;
}
</style>
