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

package app

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

var (
	requestTimeoutMillis int
	periodMillis         int
	url                  string

	waitCmd = &cobra.Command{
		Use:   "wait",
		Short: "Waits until the Proxy XDS is ready",
		RunE: func(c *cobra.Command, args []string) error {
			client := &http.Client{
				Timeout: time.Duration(requestTimeoutMillis) * time.Millisecond,
			}
			klog.Infof("Waiting for Proxy XDS to be ready")

			var err error

			for {
				select {
				case <-time.After(time.Duration(periodMillis) * time.Millisecond):
					err = checkIfReady(client, url)
					if err == nil {
						klog.Info("Proxy XDS is ready")
						return nil
					}
					klog.Errorf("Not ready yet: %v\n", err)
				}
			}
		},
	}
)

func checkIfReady(client *http.Client, url string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status code %v", resp.StatusCode)
	}
	return nil
}

func init() {
	waitCmd.PersistentFlags().IntVar(&requestTimeoutMillis, "requestTimeoutMillis", 500, "number of milliseconds to wait for response")
	waitCmd.PersistentFlags().IntVar(&periodMillis, "periodMillis", 500, "number of milliseconds to wait between attempts")
	waitCmd.PersistentFlags().StringVar(&url, "url", "http://localhost:15020/healthz/ready", "URL to use in requests")
}
