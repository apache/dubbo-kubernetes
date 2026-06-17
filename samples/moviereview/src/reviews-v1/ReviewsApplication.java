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

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

public class ReviewsApplication {
    public static void main(String[] args) throws IOException {
        HttpServer server = HttpServer.create(new InetSocketAddress(9080), 0);
        server.createContext("/healthz", exchange -> write(exchange, "text/plain", "ok\n"));
        server.createContext("/", ReviewsApplication::handle);
        server.start();
    }

    private static void handle(HttpExchange exchange) throws IOException {
        String path = exchange.getRequestURI().getPath();
        if ("/".equals(path)) {
            write(exchange, "text/plain", "reviews v1\n");
            return;
        }
        if (!"/reviews".equals(path)) {
            exchange.sendResponseHeaders(404, -1);
            exchange.close();
            return;
        }
        write(exchange, "application/json", """
            {
              "version": "v1",
              "items": [
                "故事完整，节奏朴素。",
                "这个版本只返回评论文本。"
              ]
            }
            """);
    }

    private static void write(HttpExchange exchange, String contentType, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", contentType + "; charset=utf-8");
        exchange.sendResponseHeaders(200, bytes.length);
        exchange.getResponseBody().write(bytes);
        exchange.close();
    }
}
