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

Mock.mock('/mock/instance/search', 'get', () => {
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

Mock.mock('/mock/instance/detail', 'get', () => {
  return {
    code: 200,
    message: 'success',
    data: {
      deployState: {
        label: 'pariatur do nulla',
        value: 'ut',
        level: 'ullamco veniam laboris ex',
        tip: '246.179.217.170'
      },
      registerStates: [
        {
          label: 'magna Duis non',
          value: 'laboris',
          level: 'et dolore pariatur ipsum adipisicing',
          tip: '204.174.144.51'
        }
      ],
      dubboPort: 35,
      ip: '15.1.144.52',
      appName: '式团划',
      workload: 'in labore enim',
      labels: [null],
      createTime: '2000-11-12 05:59:48',
      startTime: '1986-03-29 11:48:17',
      registerTime: '2000-01-26 19:09:48',
      registerCluster: 'qui commodo dolore',
      deployCluster: 'dolore laborum',
      node: 'aliquip',
      image: 'http://dummyimage.com/400x400',
      probes: {
        startupProbe: {
          type: 'pariatur in quis',
          port: 92,
          open: false
        },
        readinessProbe: {
          type: 'aute',
          port: 52,
          open: false
        },
        livenessPronbe: {
          type: 'reprehenderit aute nostrud',
          port: 66,
          open: false
        }
      }
    }
  }
})

Mock.mock('/mock/instance/metrics', 'get', () => {
  return {
    code: 200,
    message: 'success',
    data: 'http://8.147.104.101:3000/d/dcf5defe-d198-4704-9edf-6520838880e9/instance?orgId=1&refresh=1m&from=1710644821536&to=1710731221536&theme=light'
  }
})
