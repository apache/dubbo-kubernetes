/*
 * Copyright 2016 Google, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.alibaba.dubbo.registry.dns;

import com.alibaba.dubbo.common.URL;
import com.alibaba.dubbo.registry.NotifyListener;
import com.alibaba.dubbo.registry.support.FailbackRegistry;

/**
 * static registry : nothing to do whenever register or subscribe
 */
public class DNSRegistry extends FailbackRegistry {
    public DNSRegistry(URL url) {
        super(url);
    }

    @Override
    protected void doRegister(URL url) {

    }

    @Override
    protected void doUnregister(URL url) {

    }

    @Override
    protected void doSubscribe(URL url, NotifyListener notifyListener) {

    }

    @Override
    protected void doUnsubscribe(URL url, NotifyListener notifyListener) {

    }

    @Override
    public boolean isAvailable() {
        return true;
    }
}
