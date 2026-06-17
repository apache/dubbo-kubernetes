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

const http = require("http");

const ratingsUrl = process.env.RATINGS_URL || "http://ratings:9080/ratings";

function write(res, status, contentType, body) {
  res.writeHead(status, { "content-type": `${contentType}; charset=utf-8` });
  res.end(body);
}

function getRating() {
  const url = new URL(ratingsUrl);
  url.searchParams.set("movieId", "movie-1");
  return new Promise((resolve, reject) => {
    const req = http.get(url, (res) => {
      let body = "";
      res.setEncoding("utf8");
      res.on("data", (chunk) => {
        body += chunk;
      });
      res.on("end", () => {
        try {
          resolve(JSON.parse(body));
        } catch (err) {
          reject(err);
        }
      });
    });
    req.setTimeout(2000, () => {
      req.destroy(new Error("ratings request timed out"));
    });
    req.on("error", reject);
  });
}

const server = http.createServer(async (req, res) => {
  const path = new URL(req.url, "http://localhost").pathname;
  if (path === "/healthz") {
    write(res, 200, "text/plain", "ok\n");
    return;
  }
  if (path === "/") {
    write(res, 200, "text/plain", "reviews v3\n");
    return;
  }
  if (path !== "/reviews") {
    write(res, 404, "text/plain", "not found\n");
    return;
  }

  try {
    const rating = await getRating();
    write(
      res,
      200,
      "application/json",
      JSON.stringify({
        version: "v3",
        items: ["评论样式更醒目。", "这个版本也调用 ratings 服务。"],
        rating,
        starColor: "#dc2626",
      }),
    );
  } catch (err) {
    write(
      res,
      502,
      "application/json",
      JSON.stringify({ version: "v3", items: [], error: err.message }),
    );
  }
});

server.listen(9080, "0.0.0.0");
