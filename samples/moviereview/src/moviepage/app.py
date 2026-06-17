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
from flask import Flask


app = Flask(__name__)

DETAILS_URL = os.getenv("DETAILS_URL", "http://details:9080/details")
REVIEWS_URL = os.getenv("REVIEWS_URL", "http://reviews:9080/reviews")


def get_json(url, fallback):
    try:
        response = requests.get(url, timeout=2)
        response.raise_for_status()
        return response.json()
    except requests.RequestException as exc:
        data = fallback.copy()
        data["error"] = str(exc)
        return data


@app.get("/healthz")
def healthz():
    return "ok\n"


@app.get("/")
def index():
    details = get_json(DETAILS_URL, {"title": "Unknown", "director": "-", "year": "-", "genres": []})
    reviews = get_json(REVIEWS_URL, {"version": "unavailable", "items": [], "rating": None})
    title = escape(str(details.get("title", "Unknown")))
    year = escape(str(details.get("year", "-")))
    director = escape(str(details.get("director", "-")))
    genres = " / ".join(escape(str(genre)) for genre in details.get("genres", []))
    summary = escape(str(details.get("summary", "")))
    version = escape(str(reviews.get("version", "unavailable")))
    review_items = "".join(
        f"<article class='review-card'><span></span><p>{escape(str(item))}</p></article>"
        for item in reviews.get("items", [])
    )
    rating = reviews.get("rating")
    rating_html = "<p class='rating-empty'>No rating in this version</p>"
    if rating:
        color = escape(str(reviews.get("starColor", "#222")))
        stars = "★" * int(rating.get("stars", 0))
        rating_html = f"""
        <p class='stars' style='color: {color}'>{stars}</p>
        <p class='rating-source'>ratings service · movie-1</p>
        """

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{title}</title>
  <style>
    :root {{
      --ink: #171313;
      --paper: #f7efe1;
      --muted: #766957;
      --line: rgba(23, 19, 19, .14);
      --wine: #8f2434;
      --teal: #126e72;
      --gold: #d6a84f;
      --night: #201b1a;
    }}

    * {{ box-sizing: border-box; }}

    body {{
      min-height: 100vh;
      margin: 0;
      color: var(--ink);
      font-family: "Avenir Next", "Gill Sans", "Trebuchet MS", sans-serif;
      background:
        linear-gradient(90deg, rgba(32, 27, 26, .05) 1px, transparent 1px),
        linear-gradient(rgba(32, 27, 26, .05) 1px, transparent 1px),
        var(--paper);
      background-size: 36px 36px;
    }}

    body::before {{
      position: fixed;
      inset: 0;
      z-index: -1;
      pointer-events: none;
      content: "";
      background:
        radial-gradient(circle at 18% 14%, rgba(214, 168, 79, .22), transparent 28%),
        linear-gradient(135deg, rgba(143, 36, 52, .13), transparent 34%, rgba(18, 110, 114, .13));
    }}

    main {{
      width: min(1120px, calc(100% - 32px));
      margin: 0 auto;
      padding: 44px 0;
    }}

    .shell {{
      display: grid;
      grid-template-columns: minmax(280px, 380px) minmax(0, 1fr);
      gap: 34px;
      align-items: stretch;
    }}

    .poster {{
      position: relative;
      min-height: 620px;
      overflow: hidden;
      color: #fff8ea;
      background:
        linear-gradient(150deg, rgba(143, 36, 52, .92), rgba(18, 110, 114, .80) 55%, #151010),
        var(--night);
      border: 1px solid rgba(255, 255, 255, .24);
      border-radius: 8px;
      box-shadow: 0 28px 70px rgba(30, 20, 16, .32);
      isolation: isolate;
    }}

    .poster::before {{
      position: absolute;
      inset: 22px;
      content: "";
      border: 1px solid rgba(255, 248, 234, .32);
      border-radius: 4px;
    }}

    .poster::after {{
      position: absolute;
      right: -44px;
      bottom: -32px;
      z-index: -1;
      content: "影";
      font-family: Georgia, "Times New Roman", serif;
      font-size: 240px;
      line-height: 1;
      color: rgba(255, 248, 234, .10);
    }}

    .poster-inner {{
      position: relative;
      display: flex;
      min-height: 620px;
      flex-direction: column;
      justify-content: space-between;
      padding: 36px;
    }}

    .label {{
      width: max-content;
      padding: 6px 10px;
      border: 1px solid rgba(255, 248, 234, .36);
      color: #fff8ea;
      font-size: 12px;
      font-weight: 800;
      text-transform: uppercase;
    }}

    h1 {{
      margin: 0;
      font-family: Georgia, "Times New Roman", serif;
      font-size: 58px;
      line-height: .98;
      font-weight: 700;
    }}

    .meta {{
      margin: 18px 0 0;
      color: rgba(255, 248, 234, .76);
      font-size: 15px;
      line-height: 1.6;
    }}

    .content {{
      display: grid;
      gap: 18px;
    }}

    .summary,
    .reviews,
    .rating {{
      border: 1px solid var(--line);
      border-radius: 8px;
      background: rgba(255, 252, 244, .72);
      box-shadow: 0 20px 42px rgba(70, 48, 34, .10);
      backdrop-filter: blur(8px);
    }}

    .summary {{
      padding: 28px;
    }}

    .section-kicker {{
      margin: 0 0 10px;
      color: var(--teal);
      font-size: 12px;
      font-weight: 900;
      text-transform: uppercase;
    }}

    .summary p {{
      max-width: 68ch;
      margin: 0;
      color: #3f342b;
      font-size: 18px;
      line-height: 1.75;
    }}

    .reviews {{
      padding: 24px;
    }}

    .reviews-head {{
      display: flex;
      gap: 16px;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 18px;
      border-bottom: 1px solid var(--line);
      padding-bottom: 16px;
    }}

    .reviews-head h2 {{
      margin: 0;
      font-family: Georgia, "Times New Roman", serif;
      font-size: 32px;
      line-height: 1;
    }}

    .version {{
      flex: 0 0 auto;
      padding: 8px 12px;
      border-radius: 4px;
      background: var(--wine);
      color: #fff8ea;
      font-size: 13px;
      font-weight: 900;
    }}

    .review-list {{
      display: grid;
      gap: 12px;
    }}

    .review-card {{
      display: grid;
      grid-template-columns: 8px minmax(0, 1fr);
      gap: 14px;
      align-items: start;
      min-height: 74px;
      margin: 0;
      padding: 16px;
      border: 1px solid rgba(23, 19, 19, .10);
      border-radius: 6px;
      background: #fffaf0;
    }}

    .review-card span {{
      width: 8px;
      height: 100%;
      min-height: 42px;
      border-radius: 999px;
      background: linear-gradient(var(--gold), var(--teal));
    }}

    .review-card p {{
      margin: 0;
      color: #302820;
      font-size: 17px;
      line-height: 1.65;
    }}

    .rating {{
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 20px;
      align-items: center;
      padding: 22px 24px;
    }}

    .rating h2 {{
      margin: 0;
      font-family: Georgia, "Times New Roman", serif;
      font-size: 28px;
    }}

    .stars {{
      margin: 0;
      font-size: 34px;
      letter-spacing: 0;
      text-shadow: 0 2px 0 rgba(0, 0, 0, .08);
      white-space: nowrap;
    }}

    .rating-source,
    .rating-empty {{
      margin: 6px 0 0;
      color: var(--muted);
      font-size: 13px;
    }}

    @media (max-width: 820px) {{
      main {{ padding: 20px 0; }}
      .shell {{ grid-template-columns: 1fr; }}
      .poster,
      .poster-inner {{ min-height: 420px; }}
      h1 {{ font-size: 42px; }}
      .rating {{ grid-template-columns: 1fr; }}
      .reviews-head {{ align-items: flex-start; flex-direction: column; }}
    }}
  </style>
</head>
<body>
  <main>
    <div class="shell">
      <section class="poster">
        <div class="poster-inner">
          <span class="label">Movie Review</span>
          <div>
            <h1>{title}</h1>
            <p class="meta">{year} · {director}<br>{genres}</p>
          </div>
        </div>
      </section>
      <section class="content">
        <article class="summary">
          <p class="section-kicker">Synopsis</p>
          <p>{summary}</p>
        </article>
        <article class="reviews">
          <div class="reviews-head">
            <h2>Audience Notes</h2>
            <span class="version">reviews {version}</span>
          </div>
          <div class="review-list">{review_items}</div>
        </article>
        <article class="rating">
          <div>
            <p class="section-kicker">Score</p>
            <h2>Viewer Rating</h2>
          </div>
          <div>{rating_html}</div>
        </article>
      </section>
    </div>
  </main>
</body>
</html>"""


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9080)
