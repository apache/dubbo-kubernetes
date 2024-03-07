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

Mock.mock(devTool.mockUrl('/mock/service/search'), 'get', {
  code: 200,
  message: 'success',
  data: {
    total: 8,
    curPage: 1,
    pageSize: 5,
    data: [
      {
        serviceName: 'org.apache.dubbo.samples.UserService',
        interfaceNum: 4,
        avgQPS: 6,
        avgRT: '194ms',
        requestTotal: 200
      },
      {
        serviceName: 'org.apache.dubbo.samples.OrderService',
        interfaceNum: 12,
        avgQPS: 13,
        avgRT: '189ms',
        requestTotal: 164
      },
      {
        serviceName: 'org.apache.dubbo.samples.DetailService',
        interfaceNum: 14,
        avgQPS: 0.5,
        avgRT: '268ms',
        requestTotal: 1324
      },
      {
        serviceName: 'org.apache.dubbo.samples.PayService',
        interfaceNum: 8,
        avgQPS: 9,
        avgRT: '346ms',
        requestTotal: 189
      },
      {
        serviceName: 'org.apache.dubbo.samples.CommentService',
        interfaceNum: 9,
        avgQPS: 8,
        avgRT: '936ms',
        requestTotal: 200
      },
      {
        serviceName: 'org.apache.dubbo.samples.RepayService',
        interfaceNum: 16,
        avgQPS: 17,
        avgRT: '240ms',
        requestTotal: 146
      },
      {
        serviceName: 'org.apche.dubbo.samples.TransportService',
        interfaceNum: 5,
        avgQPS: 43,
        avgRT: '89ms',
        requestTotal: 367
      },
      {
        serviceName: 'org.apche.dubbo.samples.DistributionService',
        interfaceNum: 5,
        avgQPS: 4,
        avgRT: '78ms',
        requestTotal: 145
      }
    ]
  }
})
