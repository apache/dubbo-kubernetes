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

import type { RouterMeta } from '@/router/RouterMeta'
import type { RouteRecordRaw } from 'vue-router'

export declare type RouteRecordType = RouteRecordRaw & {
  key?: string
  name: string
  children?: RouteRecordType[]
  meta?: RouterMeta
}

export const routes: Readonly<RouteRecordType[]> = [
  {
    path: '/',
    name: 'Root',
    redirect: 'home',
    component: () => import('../layout/index.vue'),
    meta: {
      skip: true
    },
    children: [
      {
        path: '/home',
        name: 'homePage',
        component: () => import('../views/home/index.vue'),
        meta: {
          icon: 'carbon:web-services-cluster'
        }
      },
      {
        path: '/service',
        name: 'serviceSearch',
        component: () => import('../views/service/index.vue'),
        meta: {
          icon: 'material-symbols-light:screen-search-desktop-outline-rounded'
        }
      },
      {
        path: '/traffic',
        name: 'trafficManagement',
        meta: {
          icon: 'carbon:traffic-flow'
        },

        children: [
          {
            path: '/timeout',
            name: 'trafficTimeout',
            component: () => import('../views/traffic/timeout/index.vue'),
            meta: {}
          },
          {
            path: '/retry',
            name: 'trafficRetry',
            component: () => import('../views/traffic/retry/index.vue'),
            meta: {}
          },
          {
            path: '/region',
            name: 'trafficRegion',
            component: () => import('../views/traffic/region/index.vue'),
            meta: {}
          },
          {
            path: '/weight',
            name: 'trafficWeight',
            component: () => import('../views/traffic/weight/index.vue'),
            meta: {}
          },
          {
            path: '/arguments',
            name: 'trafficArguments',
            component: () => import('../views/traffic/arguments/index.vue'),
            meta: {}
          },
          {
            path: '/mock',
            name: 'trafficMock',
            component: () => import('../views/traffic/mock/index.vue'),
            meta: {}
          },
          {
            path: '/accesslog',
            name: 'trafficAccesslog',
            component: () => import('../views/traffic/accesslog/index.vue'),
            meta: {}
          },
          {
            path: '/gray',
            name: 'trafficGray',
            component: () => import('../views/traffic/gray/index.vue'),
            meta: {}
          },
          {
            path: '/routingRule',
            name: 'routingRule',
            component: () => import('../views/traffic/routingRule/index.vue'),
            meta: {}
          },
          {
            path: '/tagRule',
            name: 'tagRule',
            component: () => import('../views/traffic/tagRule/index.vue'),
            meta: {}
          },
          {
            path: '/dynamicConfig',
            name: 'dynamicConfig',
            component: () => import('../views/traffic/dynamicConfig/index.vue'),
            meta: {}
          }
        ]
      },
      {
        path: '/test',
        name: 'serviceManagement',
        redirect: '/test',
        meta: {
          icon: 'file-icons:testcafe'
        },
        children: [
          {
            path: '/test',
            name: 'serviceTest',
            component: () => import('../views/test/test/index.vue')
          },
          {
            path: '/mock',
            name: 'serviceMock',
            component: () => import('../views/test/mock/index.vue')
          }
        ]
      },
      {
        path: '/metrics',
        name: 'serviceMetrics',
        component: () => import('../views/metrics/index.vue'),
        meta: {
          icon: 'material-symbols-light:screen-search-desktop-outline-rounded'
        }
      },
      {
        path: '/kubernetes',
        name: 'kubernetes',
        component: () => import('../views/kubernetes/index.vue'),
        meta: {
          icon: 'carbon:logo-kubernetes'
        }
      }
    ]
  }
]
