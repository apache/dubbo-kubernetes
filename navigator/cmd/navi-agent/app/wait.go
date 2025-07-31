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
	"net/http"
	"time"
)

var (
	timeoutSeconds       int
	requestTimeoutMillis int
	periodMillis         int
	url                  string

	waitCmd = &cobra.Command{
		Use:   "wait",
		Short: "Waits until the Envoy proxy is ready",
		RunE: func(c *cobra.Command, args []string) error {
			client := &http.Client{
				Timeout: time.Duration(requestTimeoutMillis) * time.Millisecond,
			}
			fmt.Printf("Waiting for Envoy proxy to be ready (timeout: %d seconds)...", timeoutSeconds)

			var err error
			timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

			for {
				select {
				case <-timeout:
					return fmt.Errorf("timeout waiting for Envoy proxy to become ready. Last error: %v", err)
				case <-time.After(time.Duration(periodMillis) * time.Millisecond):
					err = checkIfReady(client, url)
					if err == nil {
						fmt.Println("Envoy is ready!")
						return nil
					}
					fmt.Printf("Not ready yet: %v", err)
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
