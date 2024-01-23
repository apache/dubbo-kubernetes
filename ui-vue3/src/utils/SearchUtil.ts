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

import type { TableColumnsType } from 'ant-design-vue'
import { reactive } from 'vue'

export class SearchDomain {
  // form of search
  queryForm: any
  params: [
    {
      label: string
      param: string
      defaultValue: string
      dict: [
        {
          label: string
          value: string
        }
      ]
    }
  ]
  searchApi: Function
  result: any
  table: {
    columns: TableColumnsType
  } = { columns: [] }
  paged = {
    curPage: 1,
    total: 0,
    pageSize: 10
  }

  constructor(query: any, searchApi: any, columns: TableColumnsType) {
    this.params = query
    this.queryForm = reactive({})
    this.table.columns = columns
    this.searchApi = searchApi
    this.onSearch()
  }

  async onSearch() {
    let res = (await this.searchApi(this.queryForm || {})).data
    this.result = res.data
    this.paged.total = res.total
    this.paged.curPage = res.curPage
    this.paged.pageSize = res.pageSize
  }
}

export function sortString(a: any, b: any) {
  if (!isNaN(a - b)) {
    return a - b
  }
  return a.localeCompare(b)
}
