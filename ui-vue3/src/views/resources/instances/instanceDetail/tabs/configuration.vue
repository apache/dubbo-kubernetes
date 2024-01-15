<script lang="ts" setup>
import { ref, onMounted } from "vue"
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'


// 执行日志开关
const state = ref("")
const loading = ref(false)
const loadBalance = ref("random")
const clusterApproach = ref("failover")
const retryCount = ref('2次')
const timeout = ref("1000 ms")





</script>

<template>
    <!-- configuration -->
    <div class="configBox">
        <!-- left -->
        <a-typography>
            <!-- execution log -->
            <a-typography-title :level="5">
                {{ $t('executionLog') }}
                <a-tooltip placement="topLeft">
                    <template #title>开启该应用所有实例的访问日志(provider.accesslog)</template>
                    <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
                </a-tooltip>
            </a-typography-title>
            <a-typography-paragraph>
                <span :class="{ active: !state }"
                    :style="{ color: (!state ? PRIMARY_COLOR : 'black') }">{{ $t('close') }}</span>
                <a-switch v-model:checked="state" :loading="loading" />
                <span :class="{ active: state }"
                    :style="{ color: (state ? PRIMARY_COLOR : 'black') }">{{ $t('enabled') }}</span>
            </a-typography-paragraph>

            <!-- Load Balance -->
            <a-typography-title :level="5">
                {{ $t('loadBalance') }}
                <a-tooltip placement="topLeft">
                    <template #title>调整应用提供服务的负载均衡策略(provider.loadbalance)</template>
                    <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
                </a-tooltip>
            </a-typography-title>
            <a-typography-paragraph editable v-model:content="loadBalance">

            </a-typography-paragraph>

            <!-- Cluster approach -->
            <a-typography-title :level="5">
                {{ $t('clusterApproach') }}
                <a-tooltip placement="topLeft">
                    <template #title>调整应用提供服务的负集群方式</template>
                    <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
                </a-tooltip>
            </a-typography-title>
            <a-typography-paragraph editable v-model:content="clusterApproach">

            </a-typography-paragraph>



        </a-typography>

        <!-- right -->
        <a-typography>


            <!-- retry count -->
            <a-typography-title :level="5">
                {{ $t('retryCount') }}
                <a-tooltip placement="topLeft">
                    <template #title>调整应用提供服务的重试次数</template>
                    <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
                </a-tooltip>
            </a-typography-title>
            <a-typography-paragraph editable v-model:content="retryCount">
            </a-typography-paragraph>

            <!-- timeout -->
            <a-typography-title :level="5">
                {{ $t('timeout') }}
                <a-tooltip placement="topLeft">
                    <template #title>调整应用提供服务的负集群方式</template>
                    <Icon icon="bitcoin-icons:info-circle-outline" class="iconStyle" />
                </a-tooltip>
            </a-typography-title>
            <a-typography-paragraph editable v-model:content="timeout">

            </a-typography-paragraph>



        </a-typography>


    </div>
</template>

<style lang="less" scoped>
.configBox {
    margin: 0 auto;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;

    .iconStyle {
        font-size: 17px;
    }

    .active {
        font-size: 15px;
        font-weight: bold;

    }


}
</style>