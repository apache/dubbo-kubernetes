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

Mock.mock('/mock/instance/search', 'get', {
  code: 200,
  message: 'laborum qui',
  data: {
    total: 66,
    curPage: 82,
    pageSize: 31,
    data: [
      {
        ip: '205.216.185.96',
        name: '用省中际解理',
        deployState: {
          label: 'dolor',
          value: 'in amet',
          level: 'amet nisi incididunt',
          tip: '133.16.55.40'
        },
        deployCluster: 'veniam elit irure',
        registerStates: [
          {
            label: 'in consequat est',
            value: 'esse non Lorem',
            level: 'sit',
            tip: '122.249.164.252'
          }
        ],
        registerClusters: ['cupidatat'],
        cpu: 'officia cupidatat reprehenderit magna ex',
        memory: 'mollit',
        startTime: '2016-07-31 19:20:31',
        registerTime: '2014-02-09 04:02:41',
        labels: ['cupidat']
      },
      {
        ip: '117.23.142.162',
        name: '之受力即此',
        deployState: {
          label: 'sint culpa elit quis id',
          value: 'amet',
          level: 'adipisicing do',
          tip: '112.176.231.68'
        },
        deployCluster: 'esse sit',
        registerStates: [
          {
            label: 'ut',
            value: 'eu sit',
            level: 'in eiusmod ullamco',
            tip: '220.153.108.236'
          }
        ],
        registerClusters: ['ea consectetur'],
        cpu: 'dolor sint deserunt',
        memory: 'sint eu commodo proident',
        startTime: '1994-12-22 18:24:57',
        registerTime: '1986-07-24 03:18:24'
      },
      {
        ip: '41.215.196.61',
        name: '值青给值',
        deployState: {
          label: 'sunt',
          value: 'consectetur in',
          level: 'culpa dolore',
          tip: '142.182.249.124'
        },
        deployCluster: 'cupidatat eu nostrud',
        registerStates: [
          {
            label: 'ad quis',
            value: 'Excepteur esse dolore Ut dolore',
            level: 'ipsum ad quis',
            tip: '220.55.203.4'
          }
        ],
        registerClusters: ['Excepteur sit laboris'],
        cpu: 'fugiat pariatur laborum ut',
        memory: 'Lorem adipisicing sunt',
        startTime: '1984-04-25 12:22:51',
        registerTime: '1976-06-06 19:58:58'
      }
    ]
  }
})
