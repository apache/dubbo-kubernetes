// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

CREATE TABLE `node_data` (
                             `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
                             `nodeName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '节点名',
                             `nodeIP` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '节点IP',
                             `sn` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '序列号',
                             `clusterName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NULL DEFAULT NULL COMMENT '集群名',
                             `moduleName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '模块名',
                             `reason` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NULL DEFAULT NULL COMMENT '维护原因',
                             `restart` int(10) UNSIGNED NULL DEFAULT NULL COMMENT '重启标志',
                             `repair` int(10) UNSIGNED NULL DEFAULT NULL COMMENT '修复标志',
                             `repairTicketUrl` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NULL DEFAULT NULL COMMENT '修复单跳转 URL',
                             `firstDate` datetime(0) NULL DEFAULT NULL COMMENT '最早开始维护时间',
                             `lastDate` datetime(0) DEFAULT NULL COMMENT '最后维护日期',
                             `createTime` datetime(0) NULL DEFAULT CURRENT_TIMESTAMP(0) COMMENT '创建时间',
                             `updateTime` datetime(0) NULL DEFAULT CURRENT_TIMESTAMP(0) ON UPDATE CURRENT_TIMESTAMP(0) COMMENT '更新时间',
                             PRIMARY KEY (`id`) USING BTREE,
                             UNIQUE INDEX `idx_unique_node_module` (`nodeName`, `moduleName`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 CHARACTER SET=utf8mb3 COLLATE=utf8mb3_general_ci ROW_FORMAT=Dynamic;

CREATE TABLE `pod_data` (
                                 `id` int unsigned NOT NULL AUTO_INCREMENT,
                                 `podName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT 'Pod 名',
                                 `podIP` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT 'ip地址',
                                 `sn` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '序列号',
                                 `nodeName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '节点名',
                                 `clusterName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '集群名',
                                 `moduleName` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '模块名',
                                 `reason` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '维护原因',
                                 `restart` int DEFAULT NULL COMMENT '重启标志位',
                                 `repair` int DEFAULT NULL COMMENT '修复标志位',
                                 `repairTicketUrl` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '修复单跳转 URL',
                                 `firstDate` datetime DEFAULT NULL COMMENT '第一次维护日期',
                                 `lastDate` datetime DEFAULT NULL COMMENT '最后维护日期',
                                 `createTime` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建日期',
                                 `updateTime` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新日期',
                                 PRIMARY KEY (`id`) USING BTREE,
                                 UNIQUE KEY `idxUniqueName` (`podName`, `moduleName`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC;