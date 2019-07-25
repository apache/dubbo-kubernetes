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

package com.alibaba.dubbo.registry.kubernetes;

import com.alibaba.dubbo.common.URL;
import com.alibaba.dubbo.common.utils.NamedThreadFactory;
import com.alibaba.dubbo.common.utils.StringUtils;
import com.alibaba.dubbo.registry.NotifyListener;
import com.alibaba.dubbo.registry.support.FailbackRegistry;
import com.google.common.base.Preconditions;
import io.fabric8.kubernetes.api.model.Endpoints;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watcher;

import java.net.URI;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * kuberntes registry : treat kubernetes as the registry
 * <p>
 * 1） for register, it is done automatically by the processing of orchestration
 * 2)  for service address discovery, watch the resource and update accordingly
 * </p>
 */
public class KubernetesRegistry extends FailbackRegistry {
    //default master url https://kubernetes.default.svc
    private final static ScheduledExecutorService TIMER_SERVICE = Executors.newSingleThreadScheduledExecutor(new NamedThreadFactory("KubernetesRegistry"));

    private final KubernetesClient kubernetesClient;

    public KubernetesRegistry(URL url) {
        super(url);
        kubernetesClient = new DefaultKubernetesClient(url.getAddress());
    }

    @Override
    protected void doRegister(URL url) {

    }

    @Override
    protected void doUnregister(URL url) {

    }

    @Override
    protected void doSubscribe(final URL url, final NotifyListener notifyListener) {
        String[] serviceDesc = url.getServiceKey().split("\\.");
        String kubernatesTarget = System.getenv(serviceDesc[serviceDesc.length - 1] + "_target");
        if (StringUtils.isEmpty(kubernatesTarget)) {
            return;
        }
        URI targetUri = URI.create(kubernatesTarget);
        String targetPath = Preconditions.checkNotNull(targetUri.getPath(), "targetPath");
        Preconditions.checkArgument(targetPath.startsWith("/"),
                "the path component (%s) of the target (%s) must start with '/'", targetPath, targetUri);

        String[] parts = targetPath.split("/");
        if (parts.length != 4) {
            throw new IllegalArgumentException("Must be formatted like kubernetes:///{namespace}/{service}/{port}");
        }

        try {
            int targetPort = Integer.valueOf(parts[3]);
            String targetName = parts[1];
            String targetNameSpace = parts[2];


            Endpoints endpoints = kubernetesClient.endpoints().inNamespace(targetNameSpace)
                    .withName(targetName)
                    .get();

            if (endpoints == null) {
                // Didn't find anything, retrying
                TIMER_SERVICE.schedule(() -> {
                    doSubscribe(url, notifyListener);
                }, 30, TimeUnit.SECONDS);
                return;
            }

            update(endpoints, notifyListener, targetPort);
            watch(notifyListener, targetNameSpace, targetName, targetPort);

        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("Unable to parse port number", e);
        }
    }

    private void update(Endpoints endpoints, final NotifyListener notifyListener, int targetPort) {
        List<URL> servers = new ArrayList<>();
        endpoints.getSubsets().stream().forEach(subset -> {
            long matchingPorts = subset.getPorts().stream().filter(p -> {
                return p.getPort() == targetPort;
            }).count();
            if (matchingPorts > 0) {
                subset.getAddresses().stream().map(address -> {
                    return new URL("", address.getIp(), targetPort);
                }).forEach(address -> {
                    servers.add(address);
                });
            }
        });

        notifyListener.notify(servers);
    }

    private void watch(final NotifyListener notifyListener, String targetNameSpace, String targetName, int targetPort) {
        kubernetesClient.endpoints().inNamespace(targetNameSpace)
                .withName(targetName)
                .watch(new Watcher<Endpoints>() {
                    @Override
                    public void eventReceived(Action action, Endpoints endpoints) {
                        switch (action) {
                            case MODIFIED:
                            case ADDED:
                                update(endpoints, notifyListener, targetPort);
                                return;
                            case DELETED:
                                //TODO
                                //notifyListener.notify(Collections.emptyList());
                                return;
                        }
                    }

                    @Override
                    public void onClose(KubernetesClientException e) {

                    }
                });
    }

    @Override
    protected void doUnsubscribe(URL url, NotifyListener notifyListener) {
        //TODO
        //Need cancel the watcher
    }

    @Override
    public boolean isAvailable() {
        return true;
    }

    @Override
    public void destroy() {
        super.destroy();
        kubernetesClient.close();
    }
}
