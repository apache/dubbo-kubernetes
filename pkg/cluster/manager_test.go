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

package cluster

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	defaultClusterID := ID("test-cluster")
	manager := NewManager(defaultClusterID)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.defaultCluster != defaultClusterID {
		t.Errorf("Expected default cluster %s, got %s", defaultClusterID, manager.defaultCluster)
	}

	if manager.clusters == nil {
		t.Error("Manager clusters map is nil")
	}

	if len(manager.clusters) != 0 {
		t.Errorf("Expected empty clusters map, got %d clusters", len(manager.clusters))
	}

	if manager.healthCheckInterval != 30*time.Second {
		t.Errorf("Expected health check interval 30s, got %v", manager.healthCheckInterval)
	}
}

func TestAddCluster(t *testing.T) {
	manager := NewManager("default")

	// Test adding a valid cluster
	config := ClusterConfig{
		ID:         "test-cluster-1",
		Name:       "Test Cluster 1",
		KubeConfig: "/path/to/kubeconfig",
		Context:    "test-context",
		Enabled:    true,
		Priority:   100,
		Region:     "us-west-1",
		Zone:       "us-west-1a",
	}

	err := manager.AddCluster(config)
	if err != nil {
		t.Errorf("Failed to add cluster: %v", err)
	}

	// Verify cluster was added
	clusterInfo, err := manager.GetCluster(ID(config.ID))
	if err != nil {
		t.Errorf("Failed to get added cluster: %v", err)
	}

	if clusterInfo.ID != ID(config.ID) {
		t.Errorf("Expected cluster ID %s, got %s", config.ID, clusterInfo.ID)
	}

	if clusterInfo.Name != config.Name {
		t.Errorf("Expected cluster name %s, got %s", config.Name, clusterInfo.Name)
	}

	if clusterInfo.Region != config.Region {
		t.Errorf("Expected cluster region %s, got %s", config.Region, clusterInfo.Region)
	}

	// Test adding duplicate cluster
	err = manager.AddCluster(config)
	if err == nil {
		t.Error("Expected error when adding duplicate cluster, got nil")
	}

	// Test adding disabled cluster
	disabledConfig := ClusterConfig{
		ID:      "disabled-cluster",
		Name:    "Disabled Cluster",
		Enabled: false,
		Region:  "us-east-1",
		Zone:    "us-east-1a",
	}

	err = manager.AddCluster(disabledConfig)
	if err != nil {
		t.Errorf("Failed to add disabled cluster: %v", err)
	}

	disabledCluster, err := manager.GetCluster(ID(disabledConfig.ID))
	if err != nil {
		t.Errorf("Failed to get disabled cluster: %v", err)
	}

	if disabledCluster.Enabled {
		t.Error("Expected disabled cluster to be disabled")
	}
}

func TestRemoveCluster(t *testing.T) {
	manager := NewManager("default")

	// Add a cluster first
	config := ClusterConfig{
		ID:      "test-cluster",
		Name:    "Test Cluster",
		Enabled: true,
		Region:  "us-west-1",
		Zone:    "us-west-1a",
	}

	err := manager.AddCluster(config)
	if err != nil {
		t.Fatalf("Failed to add cluster: %v", err)
	}

	// Test removing existing cluster
	err = manager.RemoveCluster(ID(config.ID))
	if err != nil {
		t.Errorf("Failed to remove cluster: %v", err)
	}

	// Verify cluster was removed
	_, err = manager.GetCluster(ID(config.ID))
	if err == nil {
		t.Error("Expected error when getting removed cluster, got nil")
	}

	// Test removing non-existent cluster
	err = manager.RemoveCluster(ID("non-existent"))
	if err == nil {
		t.Error("Expected error when removing non-existent cluster, got nil")
	}
}

func TestGetHealthyClusters(t *testing.T) {
	manager := NewManager("default")

	// Add healthy cluster
	healthyConfig := ClusterConfig{
		ID:      "healthy-cluster",
		Name:    "Healthy Cluster",
		Enabled: true,
		Region:  "us-west-1",
		Zone:    "us-west-1a",
	}

	err := manager.AddCluster(healthyConfig)
	if err != nil {
		t.Fatalf("Failed to add healthy cluster: %v", err)
	}

	// Add unhealthy cluster
	unhealthyConfig := ClusterConfig{
		ID:      "invalid-cluster",
		Name:    "Invalid Cluster",
		Enabled: true,
		Region:  "us-east-1",
		Zone:    "us-east-1a",
	}

	err = manager.AddCluster(unhealthyConfig)
	if err != nil {
		t.Fatalf("Failed to add unhealthy cluster: %v", err)
	}

	// Add disabled cluster
	disabledConfig := ClusterConfig{
		ID:      "disabled-cluster",
		Name:    "Disabled Cluster",
		Enabled: false,
		Region:  "eu-west-1",
		Zone:    "eu-west-1a",
	}

	err = manager.AddCluster(disabledConfig)
	if err != nil {
		t.Fatalf("Failed to add disabled cluster: %v", err)
	}

	// Get healthy clusters
	healthyClusters := manager.GetHealthyClusters()

	// Should only return enabled and healthy clusters
	expectedHealthyCount := 1 // Only healthy-cluster should be healthy
	if len(healthyClusters) != expectedHealthyCount {
		t.Errorf("Expected %d healthy clusters, got %d", expectedHealthyCount, len(healthyClusters))
	}

	// Verify the healthy cluster is the correct one
	if len(healthyClusters) > 0 {
		if healthyClusters[0].ID != ID(healthyConfig.ID) {
			t.Errorf("Expected healthy cluster ID %s, got %s", healthyConfig.ID, healthyClusters[0].ID)
		}
	}
}

func TestSetDefaultCluster(t *testing.T) {
	manager := NewManager("original-default")

	// Add a cluster
	config := ClusterConfig{
		ID:      "new-default",
		Name:    "New Default Cluster",
		Enabled: true,
		Region:  "us-west-1",
		Zone:    "us-west-1a",
	}

	err := manager.AddCluster(config)
	if err != nil {
		t.Fatalf("Failed to add cluster: %v", err)
	}

	// Test setting valid default cluster
	err = manager.SetDefaultCluster(ID(config.ID))
	if err != nil {
		t.Errorf("Failed to set default cluster: %v", err)
	}

	if manager.GetDefaultCluster() != ID(config.ID) {
		t.Errorf("Expected default cluster %s, got %s", config.ID, manager.GetDefaultCluster())
	}

	// Test setting invalid default cluster
	err = manager.SetDefaultCluster(ID("non-existent"))
	if err == nil {
		t.Error("Expected error when setting non-existent default cluster, got nil")
	}

	// Verify default cluster didn't change
	if manager.GetDefaultCluster() != ID(config.ID) {
		t.Error("Default cluster should not have changed after failed set operation")
	}
}

func TestClustersByRegion(t *testing.T) {
	manager := NewManager("default")

	// Add clusters in different regions
	configs := []ClusterConfig{
		{
			ID:      "us-west-1-cluster",
			Name:    "US West 1 Cluster",
			Enabled: true,
			Region:  "us-west-1",
			Zone:    "us-west-1a",
		},
		{
			ID:      "us-west-2-cluster",
			Name:    "US West 2 Cluster",
			Enabled: true,
			Region:  "us-west-1", // Same region as first
			Zone:    "us-west-1b",
		},
		{
			ID:      "us-east-1-cluster",
			Name:    "US East 1 Cluster",
			Enabled: true,
			Region:  "us-east-1",
			Zone:    "us-east-1a",
		},
		{
			ID:      "disabled-cluster",
			Name:    "Disabled Cluster",
			Enabled: false,
			Region:  "us-west-1",
			Zone:    "us-west-1c",
		},
	}

	for _, config := range configs {
		err := manager.AddCluster(config)
		if err != nil {
			t.Fatalf("Failed to add cluster %s: %v", config.ID, err)
		}
	}

	// Test getting clusters by region (this would require implementing ClustersByRegion method)
	// For now, we'll test the concept by manually filtering
	allClusters := manager.GetAllClusters()
	usWest1Clusters := make([]*ClusterInfo, 0)

	for _, cluster := range allClusters {
		if cluster.Region == "us-west-1" && cluster.Enabled {
			usWest1Clusters = append(usWest1Clusters, cluster)
		}
	}

	expectedCount := 2 // Two enabled clusters in us-west-1
	if len(usWest1Clusters) != expectedCount {
		t.Errorf("Expected %d clusters in us-west-1 region, got %d", expectedCount, len(usWest1Clusters))
	}

	// Verify cluster IDs
	foundIDs := make(map[string]bool)
	for _, cluster := range usWest1Clusters {
		foundIDs[string(cluster.ID)] = true
	}

	expectedIDs := []string{"us-west-1-cluster", "us-west-2-cluster"}
	for _, expectedID := range expectedIDs {
		if !foundIDs[expectedID] {
			t.Errorf("Expected to find cluster %s in us-west-1 region", expectedID)
		}
	}

	// Test empty region
	emptyRegionClusters := make([]*ClusterInfo, 0)
	for _, cluster := range allClusters {
		if cluster.Region == "nonexistent-region" {
			emptyRegionClusters = append(emptyRegionClusters, cluster)
		}
	}

	if len(emptyRegionClusters) != 0 {
		t.Errorf("Expected 0 clusters in nonexistent region, got %d", len(emptyRegionClusters))
	}
}

func TestConcurrentOperations(t *testing.T) {
	manager := NewManager("default")

	// Test concurrent add operations
	var wg sync.WaitGroup
	numClusters := 10

	for i := 0; i < numClusters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			config := ClusterConfig{
				ID:      fmt.Sprintf("cluster-%d", id),
				Name:    fmt.Sprintf("Cluster %d", id),
				Enabled: true,
				Region:  "us-west-1",
				Zone:    "us-west-1a",
			}
			err := manager.AddCluster(config)
			if err != nil {
				t.Errorf("Failed to add cluster %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all clusters were added
	allClusters := manager.GetAllClusters()
	if len(allClusters) != numClusters {
		t.Errorf("Expected %d clusters, got %d", numClusters, len(allClusters))
	}

	// Test concurrent read operations
	for i := 0; i < numClusters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clusterID := ID(fmt.Sprintf("cluster-%d", id))
			_, err := manager.GetCluster(clusterID)
			if err != nil {
				t.Errorf("Failed to get cluster %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}
