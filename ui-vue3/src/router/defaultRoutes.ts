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
import LayoutTab from '../layout/tab/layout_tab.vue'
import _ from 'lodash'

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
        path: '/resources',
        name: 'resources',
        meta: {
          icon: 'carbon:web-services-cluster'
        },
        children: [
          {
            path: '/applications',
            name: 'applications',
            component: LayoutTab,
            redirect: 'index',
            meta: {
              tab_parent: true
            },
            children: [
              {
                path: '/index',
                name: 'index',
                component: () => import('../views/resources/applications/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/detail/:pathId',
                name: 'applicationDomain.detail',
                component: () => import('../views/resources/applications/tabs/tab1.vue'),
                meta: {
                  tab: true,
                  icon: 'material-symbols:view-in-ar'
                }
              },
              {
                path: '/detail2/:pathId',
                name: 'application-tab2',
                component: () => import('../views/resources/applications/tabs/tab2.vue'),
                meta: {
                  tab: true
                }
              }
            ]
          },
          {
            path: '/instances',
            name: 'instances',
            component: () => import('../views/resources/instances/index.vue'),
            meta: {}
          },
          {
            path: '/services',
            name: 'services',
            redirect: 'search',
            component: () => import('../views/resources/services/index.vue'),
            meta: {
              tab_parent: true
            },
            children: [
              {
                path: '/search',
                name: 'search',
                component: () => import('../views/resources/services/search.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/detail/:serviceName',
                name: 'detail',
                component: () => import('../views/resources/services/tabs/detail.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/debug/:serviceName',
                name: 'debug',
                component: () => import('../views/resources/services/tabs/debug.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/distribution/:serviceName',
                name: 'distribution',
                component: () => import('../views/resources/services/tabs/distribution.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/monitor/:serviceName',
                name: 'monitor',
                component: () => import('../views/resources/services/tabs/monitor.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/tracing/:serviceName',
                name: 'tracing',
                component: () => import('../views/resources/services/tabs/tracing.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/event/:serviceName',
                name: 'event',
                component: () => import('../views/resources/services/tabs/event.vue'),
                meta: {
                  tab: true
                }
              }
            ]
          }
        ]
      },
      {
        path: '/common',
        name: 'commonDemo',
        redirect: 'tab',
        meta: {
          icon: 'tdesign:play-demo'
        },
        children: [
          {
            path: '/tab',
            name: 'tabDemo',
            component: LayoutTab,
            redirect: 'index',
            meta: {
              tab_parent: true
            },
            children: [
              {
                path: '/index',
                name: 'tab_demo_index',
                component: () => import('../views/common/tab_demo/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/tab1/:pathId',
                name: 'tab1',
                component: () => import('../views/common/tab_demo/tab1.vue'),
                meta: {
                  icon: 'simple-icons:podman',
                  tab: true
                }
              },
              {
                path: '/tab2/:pathId',
                name: 'tab2',
                component: () => import('../views/common/tab_demo/tab2.vue'),
                meta: {
                  icon: 'fontisto:docker',
                  tab: true
                }
              }
            ]
          },
          {
            path: '/placeholder',
            name: 'placeholder_demo',
            component: () => import('../views/common/placeholder_demo/index.vue'),
            meta: {}
          }
        ]
      }
    ]
  },
  {
    path: '/:catchAll(.*)',
    name: 'notFound',
    component: () => import('../views/error/notFound.vue'),
    meta: {
      skip: true
    }
  }
]

function handlePath(...paths: any[]) {
  return paths.join('/').replace(/\/+/g, '/')
}

function handleRoutes(
  routes: readonly RouteRecordType[] | undefined,
  parent: RouteRecordType | undefined
) {
  if (!routes) return
  for (let route of routes) {
    if (parent) {
      route.path = handlePath(parent?.path, route.path)
    }
    if (route.redirect) {
      route.redirect = handlePath(route.path, route.redirect || '')
    }

    if (route.meta) {
      route.meta._router_key = _.uniqueId('__router_key')
      route.meta.parent = parent
      // fixme, its really useful for tab_router judging how to  show tab
      route.meta.skip = route.meta.skip === true ? true : false
    } else {
      route.meta = {
        _router_key: _.uniqueId('__router_key'),
        skip: false
      }
    }
    handleRoutes(route.children, route)
  }
}

handleRoutes(routes, undefined)
console.log(routes)
