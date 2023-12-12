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

  <div class="__dashboard_echarts_container">
    <div class="__gauge_for_home" :id="`__gauge_for_home_${value.name}`"></div>
  </div>
</template>

<script>
import * as echarts from 'echarts';

export default {
  name: 'home-echarts',
  props: {
    value: {
      type: Object,
      default: {
        number: 0,
        name: 'demo'
      }
    },

  },
  data() {
    return {
      option: {
        tooltip: {
          trigger: 'item'
        },
        // legend: {
        //   top: '5%',
        //   left: 'center'
        // },
        series: [
          {
            name: 'Cluster',
            type: 'pie',
            radius: ['40%', '70%'],
            avoidLabelOverlap: false,
            itemStyle: {
              borderRadius: 10,
              borderColor: '#fff',
              borderWidth: 2
            },
            label: {
              //文本样式
              normal: {
                textStyle: {
                  fontSize: 28,
                  fontWeight: 'bolder',
                  color: '#FC6679'
                },
                formatter: "{c}",
                position: "center",
                show: true,
              },
            },
            emphasis: {
              label: {
                show: true,
                fontSize: 40,
                fontWeight: 'bold'
              }
            },
            labelLine: {
              show: false
            },
            color:[
              new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: "#FC6679 " },
                { offset: 0.4, color: "#FCC581 " },
                { offset: 1, color: "#FDC581 " }
              ])
            ],
            data: [
              { value: 100, name: 'demo' },
            ]
          }
        ]
      }
    }
  },
  mounted() {
    const chartDom = document.getElementById(`__gauge_for_home_${this.value.name}`);
    const myChart = echarts.init(chartDom);
    this.option.series[0].data[0].value = this.value.number
    this.option.series[0].data[0].name = this.value.name
    console.log(this.option.series[0])
    myChart.setOption(this.option);
  }
}
</script>

<style lang="css">

.__dashboard_echarts_container .__gauge_for_home {
  height: 100%;
  width: 100%;

}

</style>
