// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type request struct {
	ID     any            `json:"id"`
	Method string         `json:"method"`
	Model  string         `json:"model"`
	Params map[string]any `json:"params"`
}

func main() {
	mode := env("MOCK_MODE", "http")
	name := env("MOCK_NAME", mode)
	port := env("PORT", "8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "http":
			writeJSON(w, map[string]any{"service": "http", "path": r.URL.Path})
		case "openai":
			var in request
			_ = json.NewDecoder(r.Body).Decode(&in)
			writeJSON(w, map[string]any{
				"id": "chatcmpl-mock", "object": "chat.completion", "model": in.Model,
				"provider_authorization": r.Header.Get("Authorization"),
				"choices":                []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": "openai-mock"}, "finish_reason": "stop"}},
				"usage":                  map[string]any{"prompt_tokens": 2, "completion_tokens": 3, "total_tokens": 5},
			})
		case "anthropic":
			var in request
			_ = json.NewDecoder(r.Body).Decode(&in)
			writeJSON(w, map[string]any{
				"id": "msg-mock", "type": "message", "role": "assistant", "model": in.Model,
				"content":     []any{map[string]any{"type": "text", "text": "anthropic-mock"}},
				"stop_reason": "end_turn",
				"usage":       map[string]any{"input_tokens": 4, "output_tokens": 6},
			})
		case "mcp":
			var in request
			_ = json.NewDecoder(r.Body).Decode(&in)
			result := map[string]any{}
			switch in.Method {
			case "initialize":
				result = map[string]any{"protocolVersion": "2025-03-26", "serverInfo": map[string]any{"name": name, "version": "1.0.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}
			case "tools/list":
				result = map[string]any{"tools": []any{map[string]any{"name": name, "description": name + " mock tool", "inputSchema": map[string]any{"type": "object"}}}}
			case "tools/call":
				result = map[string]any{"content": []any{map[string]any{"type": "text", "text": name + "-ok"}}}
			}
			writeJSON(w, map[string]any{"jsonrpc": "2.0", "id": in.ID, "result": result})
		case "a2a":
			if r.Method == http.MethodGet {
				writeJSON(w, map[string]any{"name": "planner", "version": "1.0.0", "url": "http://mock-a2a:8080/a2a"})
				return
			}
			var in request
			_ = json.NewDecoder(r.Body).Decode(&in)
			writeJSON(w, map[string]any{"jsonrpc": "2.0", "id": in.ID, "result": map[string]any{"id": "task-mock", "status": map[string]any{"state": "completed"}}})
		default:
			http.Error(w, fmt.Sprintf("unknown MOCK_MODE %q", mode), http.StatusInternalServerError)
		}
	})
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("agent mock mode=%s name=%s addr=%s", mode, name, server.Addr)
	log.Fatal(server.ListenAndServe())
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
