/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import Mock from 'mockjs'
import devTool from '@/utils/DevToolUtil'

Mock.mock('/mock/application/metrics', 'get', () => {
  return {
    code: 200,
    message: 'success',
    data: 'http://101.201.225.179:3000/d/a0b114ca-edf7-4dfe-ac2c-34a4fc545fed/application?orgId=1&refresh=1m&from=1710644821536&to=1710731221536&theme=light'
  }
})

Mock.mock('/mock/application/search', 'get', () => {
  let total = Mock.mock('@integer(8, 1000)')
  let list = []
  for (let i = 0; i < total; i++) {
    let tmp: any = {
      registerClusters: []
    }
    let num = Mock.mock('@integer(1,3)')
    for (let j = 0; j < num; j++) {
      let r = Mock.mock('@string(5)')
      tmp.registerClusters.push(`cluster_${r}`)
    }
    list.push({
      appName: 'app_' + Mock.mock('@string(2,10)'),
      instanceNum: Mock.mock('@integer(80, 200)'),
      deployCluster: 'cluster_' + Mock.mock('@string(5)'),
      ...tmp
    })
  }
  return {
    code: 200,
    message: 'success',
    data: {
      total: total,
      curPage: 1,
      pageSize: 10,
      data: list
    }
  }
})
Mock.mock('/mock/application/instance/statistics', 'get', () => {
  return {
    code: 1000,
    message: 'success',
    data: {
      instanceTotal: 43,
      versionTotal: 4,
      cpuTotal: '56c',
      memoryTotal: '108.2GB'
    }
  }
})

Mock.mock(devTool.mockUrl('/mock/application/instance/info'), 'get', () => {
  let total = Mock.mock('@integer(8, 1000)')
  let list = []
  for (let i = 0; i < total; i++) {
    list.push({
      ip: '121.90.211.162',
      name: 'shop-user',
      deployState: Mock.Random.pick(['Running', 'Pending', 'Terminating', 'Crashing']),
      deployCluster: 'tx-shanghai-1',
      registerStates: [
        {
          label: 'Registed',
          value: 'Registed',
          level: 'healthy'
        }
      ],
      registerClusters: ['ali-hangzhou-1', 'ali-hangzhou-2'],
      cpu: '1.2c',
      memory: '2349MB',
      startTime: '2023-06-09 03:47:10',
      registerTime: '2023-06-09 03:48:20',
      labels: {
        region: 'beijing',
        version: 'v1'
      }
    })
  }
  return {
    code: 200,
    message: 'success',
    data: Mock.mock({
      total: total,
      curPage: 1,
      pageSize: 10,
      data: list
    })
  }
})

Mock.mock('/mock/application/detail', 'get', () => {
  return {
    code: 200,
    message: 'success',
    data: {
      appName: ['shop-user'],
      rpcProtocols: ['dubbo 2.0.2'],
      dubboVersions: ['Dubbo 3.2.10', 'Dubbo 2.7.4.1'],
      dubboPorts: ['20880'],
      serialProtocols: ['fastjson2'],
      appTypes: ['无状态'],
      images: [
        'harbor.apche.org/dubbo-samples-shop-user:v1.0',
        'harbor.apche.org/dubbo-samples-shop-user:v1.1',
        'harbor.apche.org/dubbo-samples-shop-user:v1.2'
      ],
      workloads: [
        'dubbo-samples-shop-user-base',
        'dubbo-samples-shop-user-gray',
        'dubbo-samples-shop-user-gray',
        'dubbo-samples-shop-user-gray'
      ],
      deployCluster: ['ali-shanghai-1', 'tx-shanghai-2'],
      registerCluster: ['nacos-cluster-1', 'nacos-cluster-2'],
      registerMode: ['应用级', '接口级']
    }
  }
})

Mock.mock('/mock/application/event', 'get', () => {
  let list = Mock.mock({
    'list|10': [
      {
        desc: `Scaled down replica set shop-detail-v1-5847b7cdfd to @integer(3,10) from @integer(3,10)`,
        time: '@DATETIME("yyyy-MM-dd HH:mm:ss")',
        type: 'deployment-controller'
      }
    ]
  })
  return {
    code: 200,
    message: 'success',
    data: {
      ...list
    }
  }
})
