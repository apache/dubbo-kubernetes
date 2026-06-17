// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type movieDetails struct {
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	Director string   `json:"director"`
	Genres   []string `json:"genres"`
	Summary  string   `json:"summary"`
}

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/details" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, movieDetails{
			Title:    "银河补习班",
			Year:     2019,
			Director: "Deng Chao",
			Genres:   []string{"Drama", "Family"},
			Summary:  "A father and son story used here as a small, normal microservice domain.",
		})
	})
	log.Fatal(http.ListenAndServe(":9080", nil))
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
