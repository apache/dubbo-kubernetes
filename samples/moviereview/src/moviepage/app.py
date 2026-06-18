# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from html import escape

import requests
from flask import Flask, redirect, request, session, url_for


app = Flask(__name__)
app.secret_key = os.getenv("MOVIEREVIEW_SESSION_SECRET", "moviereview-session-secret")

DETAILS_URL = os.getenv("DETAILS_URL", "http://details:9080/details")
REVIEWS_URL = os.getenv("REVIEWS_URL", "http://reviews:9080/reviews")


def forward_headers():
    headers = {}
    user = session.get("user")
    if user:
        headers["end-user"] = user
    return headers


def get_json(url, fallback, headers=None):
    try:
        response = requests.get(url, headers=headers or {}, timeout=2)
        response.raise_for_status()
        return response.json()
    except (requests.RequestException, ValueError) as exc:
        data = fallback.copy()
        data["error"] = str(exc)
        return data


@app.get("/healthz")
def healthz():
    return "ok\n"


@app.post("/login")
def login():
    username = request.form.get("username", "").strip()
    password = (request.form.get("passwd") or request.form.get("password") or "").strip()
    if username and password:
        session["user"] = username[:64]
    return redirect(request.referrer or url_for("index"))


@app.get("/logout")
def logout():
    session.pop("user", None)
    return redirect(request.referrer or url_for("index"))


@app.get("/")
def index():
    headers = forward_headers()
    user = session.get("user", "")
    details = get_json(
        DETAILS_URL,
        {
            "title": "Unknown",
            "director": "-",
            "year": "-",
            "genres": [],
            "runtime": "147 min",
            "language": "Mandarin",
            "region": "China",
            "movieId": "movie-1",
        },
        headers,
    )
    reviews = get_json(REVIEWS_URL, {"version": "unavailable", "items": [], "rating": None}, headers)
    page_status = 200
    title = escape(str(details.get("title", "Unknown")))
    year = escape(str(details.get("year", "-")))
    pages = escape(str(page_status))
    director = escape(str(details.get("director", "-")))
    genres = " / ".join(escape(str(genre)) for genre in details.get("genres", []))
    runtime = escape(str(details.get("runtime", "147 min")))
    language = escape(str(details.get("language", "Mandarin")))
    region = escape(str(details.get("region", "China")))
    movie_id = escape(str(details.get("movieId", "movie-1")))
    summary = escape(str(details.get("summary", "")))
    version_raw = str(reviews.get("version", "unavailable"))
    version = escape(version_raw)
    version_class = f"version-{version_raw}" if version_raw in {"v1", "v2", "v3"} else "version-default"
    detail_rows = "".join(
        f"<div class='metric'><span>{label}</span><strong>{value}</strong></div>"
        for label, value in [
            ("Release", year),
            ("Pages", pages),
            ("Director", director),
            ("Genre", genres),
            ("Runtime", runtime),
            ("Language", language),
            ("Region", region),
            ("Movie ID", movie_id),
        ]
    )
    review_items = "".join(
        f"""
        <article class="review-item">
          <p>{escape(str(item))}</p>
          <span>Viewer {index}</span>
        </article>
        """
        for index, item in enumerate(reviews.get("items", []), start=1)
    )
    if not review_items:
        review_items = "<p class='review-empty'>No audience notes available.</p>"
    rating = reviews.get("rating")
    rating_html = "<p class='rating-empty'>No rating in this version</p>"
    if isinstance(rating, dict):
        score = rating.get("score", "9.0")
        try:
            score = f"{float(score):.1f}"
        except (TypeError, ValueError):
            score = escape(str(score))
        rating_html = f"""
        <p class='score-number'>{score}</p>
        <p class='rating-source'>ratings service · {movie_id}</p>
        """
    if user:
        auth_html = f"""
        <div class="identity">
          <span>{escape(str(user))}</span>
          <a href="/logout">Sign out</a>
        </div>
        """
    else:
        auth_html = """
        <form class="signin" method="post" action="/login">
          <input type="text" name="username" placeholder="Username" autocomplete="username" required>
          <input type="password" name="passwd" placeholder="Password" autocomplete="current-password" required>
          <button type="submit">Sign in</button>
        </form>
        """

    html = f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{title}</title>
  <style>
    :root {{
      --ink: #333;
      --muted: #666;
      --line: #ddd;
      --link: #337ab7;
    }}

    * {{ box-sizing: border-box; }}

    body {{
      min-height: 100vh;
      margin: 0;
      color: var(--ink);
      background: #fff;
      font-family: Arial, Helvetica, sans-serif;
    }}

    body.version-v1 {{
      --version-bg: #337ab7;
      --version-soft: #f5f9fc;
    }}

    body.version-v2 {{
      --version-bg: #8a6d3b;
      --version-soft: #fcf8e3;
    }}

    body.version-v3 {{
      --version-bg: #a94442;
      --version-soft: #f9f2f4;
    }}

    body.version-default {{
      --version-bg: #555d66;
      --version-soft: rgba(85, 93, 102, .10);
    }}

    .topbar {{
      width: 100%;
      border-bottom: 1px solid var(--line);
      background: #f8f8f8;
    }}

    .topbar-inner {{
      display: flex;
      min-height: 58px;
      width: min(1120px, calc(100% - 48px));
      margin: 0 auto;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
    }}

    .brand {{
      color: var(--ink);
      font-size: 18px;
      font-weight: 500;
      text-decoration: none;
    }}

    .signin,
    .identity {{
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: 8px;
    }}

    .identity {{
      margin-left: auto;
    }}

    .signin input {{
      width: 128px;
      height: 34px;
      border: 1px solid #ccc;
      border-radius: 4px;
      padding: 0 10px;
      color: var(--ink);
      background: #fff;
      font: inherit;
      font-size: 13px;
      outline: none;
    }}

    .signin input:focus {{ border-color: var(--version-bg); }}

    .signin button,
    .identity a {{
      display: inline-flex;
      align-items: center;
      justify-content: center;
      height: 34px;
      border: 1px solid #ccc;
      border-radius: 4px;
      padding: 0 12px;
      color: #333;
      background: #fff;
      font: inherit;
      font-size: 13px;
      line-height: 1;
      text-decoration: none;
      cursor: pointer;
    }}

    .identity span {{
      color: var(--muted);
      font-size: 13px;
    }}

    main {{
      width: min(1120px, calc(100% - 48px));
      margin: 0 auto;
      padding: 32px 0 42px;
    }}

    .movie-hero {{
      display: grid;
      grid-template-columns: minmax(0, 1fr) 220px;
      gap: 32px;
      align-items: end;
      padding-bottom: 22px;
      border-bottom: 1px solid var(--line);
    }}

    .kicker {{
      margin: 0 0 8px;
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
    }}

    h1 {{
      margin: 0;
      color: var(--link);
      font-size: 30px;
      font-weight: 500;
      line-height: 1.18;
    }}

    .summary {{
      max-width: 760px;
      margin: 12px 0 0;
      color: var(--ink);
      font-size: 16px;
      line-height: 1.5;
    }}

    .score-panel {{
      padding: 0;
      border: 0;
      background: transparent;
      text-align: right;
    }}

    .version {{
      display: inline-block;
      margin-bottom: 10px;
      padding: 4px 7px;
      color: #fff;
      background: var(--version-bg);
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
    }}

    .score-number {{
      margin: 0;
      color: var(--version-bg);
      font-size: 36px;
      font-weight: 700;
      line-height: 1;
      white-space: nowrap;
    }}

    .rating-source,
    .rating-empty {{
      margin: 6px 0 0;
      color: var(--muted);
      font-size: 12px;
    }}

    .meta-strip {{
      display: grid;
      grid-template-columns: repeat(8, minmax(0, 1fr));
      gap: 0;
      padding: 14px 0;
      border-bottom: 1px solid var(--line);
    }}

    .metric {{
      min-width: 0;
      padding: 0 12px;
      border-left: 1px solid var(--line);
    }}

    .metric:first-child {{ border-left: 0; padding-left: 0; }}

    .metric span {{
      display: block;
      margin-bottom: 5px;
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
    }}

    .metric strong {{
      display: block;
      overflow-wrap: anywhere;
      color: var(--ink);
      font-size: 14px;
      font-weight: 400;
      line-height: 1.35;
    }}

    .review-section {{
      display: grid;
      grid-template-columns: 210px minmax(0, 1fr);
      gap: 32px;
      padding-top: 28px;
    }}

    .section-head h2 {{
      margin: 0;
      color: var(--link);
      font-size: 22px;
      font-weight: 500;
      line-height: 1.2;
    }}

    .review-list {{
      display: grid;
      gap: 18px;
    }}

    .review-item {{
      padding-bottom: 18px;
      border-bottom: 1px solid var(--line);
    }}

    .review-item p {{
      margin: 0 0 8px;
      color: var(--ink);
      font-size: 16px;
      line-height: 1.5;
    }}

    .review-item span,
    .review-empty {{
      color: var(--muted);
      font-size: 13px;
    }}

    @media (max-width: 900px) {{
      .topbar-inner {{
        width: min(100% - 32px, 1120px);
        align-items: center;
        flex-direction: row;
        flex-wrap: wrap;
        justify-content: space-between;
        padding: 12px 0;
      }}

      .signin {{
        width: auto;
        margin-left: auto;
        flex-wrap: wrap;
        justify-content: flex-end;
      }}

      main {{
        width: min(100% - 32px, 1120px);
        padding-top: 28px;
      }}

      .movie-hero,
      .review-section {{
        grid-template-columns: 1fr;
        gap: 20px;
      }}

      h1 {{ font-size: 26px; }}

      .summary {{ font-size: 15px; }}

      .score-panel {{
        max-width: 320px;
        text-align: left;
      }}

      .meta-strip {{
        grid-template-columns: repeat(2, minmax(0, 1fr));
        row-gap: 14px;
      }}

      .metric:nth-child(odd) {{
        border-left: 0;
        padding-left: 0;
      }}
    }}

    @media (max-width: 560px) {{
      .topbar-inner {{
        align-items: flex-start;
        flex-direction: column;
        justify-content: center;
      }}

      .signin {{
        width: 100%;
        margin-left: 0;
        justify-content: flex-start;
      }}

      .identity {{
        margin-left: 0;
      }}
    }}
  </style>
</head>
<body class="{version_class}">
  <header class="topbar">
    <div class="topbar-inner">
      <a class="brand" href="/">MovieReview Sample</a>
      {auth_html}
    </div>
  </header>
  <main>
    <section class="movie-hero">
      <div>
        <p class="kicker">Movie Review</p>
        <h1>{title}</h1>
        <p class="summary">{summary}</p>
      </div>
      <div class="score-panel">
        <span class="version">reviews {version}</span>
        {rating_html}
      </div>
    </section>
    <section class="meta-strip" aria-label="Movie details">{detail_rows}</section>
    <section class="review-section">
      <div class="section-head">
        <h2>Audience Reviews</h2>
      </div>
      <div class="review-list">{review_items}</div>
    </section>
  </main>
</body>
</html>"""
    return html, page_status


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9080)
