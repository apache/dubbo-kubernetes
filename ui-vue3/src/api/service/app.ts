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

import request from '@/base/http/request'

export const searchApplications = (params: any): Promise<any> => {
  return request({
    url: '/application/search',
    method: 'get',
    params
  })
}

export const getApplicationDetail = (params: any): Promise<any> => {
  return request({
    url: '/application/detail',
    method: 'get',
    params
  })
}

export const getApplicationInstanceStatistics = (params: any): Promise<any> => {
  return request({
    url: '/application/instance/statistics',
    method: 'get',
    params
  })
}
export const getApplicationInstanceInfo = (params: any): Promise<any> => {
  return request({
    url: '/application/instance/info',
    method: 'get',
    params
  })
}

export const getApplicationServiceForm = (params: any): Promise<any> => {
  return request({
    url: '/application/service/form',
    method: 'get',
    params
  })
}

export const getApplicationMetricsInfo = (params: any): Promise<any> => {
  return request({
    url: '/application/metrics',
    method: 'get',
    params
  })
}
export const listApplicationEvent = (params: any): Promise<any> => {
  return request({
    url: '/application/event',
    method: 'get',
    params
  })
}
