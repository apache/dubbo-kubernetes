#!/usr/bin/env python3
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License. You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from __future__ import annotations

import argparse
import dataclasses
import datetime as dt
import os
import re
import subprocess
from pathlib import Path
from typing import Iterable


VERSION_RE = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$")
PR_RE = re.compile(r"\(#(?P<num>\d+)\)")


@dataclasses.dataclass(frozen=True)
class Commit:
    subject: str
    sha: str
    author: str


@dataclasses.dataclass(frozen=True)
class ReleaseContext:
    tag: str
    version: str
    previous_tag: str | None
    repo: str
    date: dt.date


CATEGORY_ORDER = [
    ("security", "Security", "安全"),
    ("features", "Features", "新增功能"),
    ("traffic", "Traffic Management", "流量管理"),
    ("installation", "Installation and CLI", "安装与命令行"),
    ("fixes", "Bug Fixes", "问题修复"),
    ("tests", "Tests", "测试"),
    ("docs", "Documentation", "文档"),
    ("other", "Other Changes", "其他变更"),
]


def run_git(args: list[str], cwd: Path) -> str:
    return subprocess.check_output(["git", *args], cwd=cwd, text=True).strip()


def parse_version(tag: str) -> tuple[int, int, int]:
    match = VERSION_RE.match(strip_ref(tag))
    if not match:
        raise ValueError(f"tag must be a semantic version such as 0.4.0 or v0.4.0: {tag}")
    return tuple(int(part) for part in match.groups())


def strip_ref(tag: str) -> str:
    return tag.removeprefix("refs/tags/")


def display_version(tag: str) -> str:
    return strip_ref(tag).removeprefix("v")


def minor_series(version: str) -> str:
    major, minor, _ = parse_version(version)
    return f"{major}.{minor}.x"


def english_date(value: dt.date) -> str:
    months = [
        "Jan",
        "Feb",
        "Mar",
        "Apr",
        "May",
        "Jun",
        "Jul",
        "Aug",
        "Sep",
        "Oct",
        "Nov",
        "Dec",
    ]
    return f"{months[value.month - 1]} {value.day}, {value.year}"


def release_kind(version: str) -> tuple[str, str]:
    _, _, patch = parse_version(version)
    if patch == 0:
        return "minor release", "小版本发布"
    return "patch release", "补丁版本发布"


def source_changes_url(ctx: ReleaseContext) -> str:
    base = f"https://github.com/{ctx.repo}"
    if ctx.previous_tag:
        return f"{base}/compare/{ctx.previous_tag}...{ctx.tag}"
    return f"{base}/commits/{ctx.tag}"


def github_release_url(ctx: ReleaseContext) -> str:
    return f"https://github.com/{ctx.repo}/releases/tag/{ctx.tag}"


def previous_tag_for(tag: str, cwd: Path) -> str | None:
    current = parse_version(tag)
    tags = []
    for candidate in run_git(["tag", "--list"], cwd).splitlines():
        try:
            parsed = parse_version(candidate)
        except ValueError:
            continue
        if parsed < current:
            tags.append((parsed, candidate))
    if not tags:
        return None
    tags.sort()
    return tags[-1][1]


def git_log_range(ctx: ReleaseContext) -> str:
    if ctx.previous_tag:
        return f"{ctx.previous_tag}..{ctx.tag}"
    return ctx.tag


def collect_commits(ctx: ReleaseContext, cwd: Path) -> list[Commit]:
    fmt = "%s%x1f%h%x1f%an"
    range_ref = git_log_range(ctx)
    output = run_git(["log", "--no-merges", f"--pretty=format:{fmt}", range_ref], cwd)
    if not output:
        output = run_git(["log", f"--pretty=format:{fmt}", range_ref], cwd)
    commits = []
    for line in output.splitlines():
        parts = line.split("\x1f")
        if len(parts) == 3:
            commits.append(Commit(subject=parts[0], sha=parts[1], author=parts[2]))
    return [commit for commit in commits if not is_release_marker(commit.subject)]


def is_release_marker(subject: str) -> bool:
    normalized = subject.strip().lower()
    return bool(
        re.fullmatch(r"release[-: ]?v?\d+\.\d+\.\d+", normalized)
        or re.fullmatch(r"version v?\d+\.\d+\.\d+.*", normalized)
        or normalized in {"fix version", "bump version", "version bump"}
    )


def classify(subject: str) -> str:
    value = subject.lower()
    if any(token in value for token in ["cve", "security", "auth", "jwt", "tls", "mtls", "rbac"]):
        return "security"
    if any(token in value for token in ["feat", "feature", "add ", "added", "new "]):
        return "features"
    if any(
        token in value
        for token in [
            "gateway",
            "xds",
            "route",
            "traffic",
            "meshservice",
            "multicluster",
            "east-west",
            "proxyless",
            "eds",
            "lds",
            "rds",
        ]
    ):
        return "traffic"
    if any(token in value for token in ["helm", "chart", "install", "dubboctl", "operator", "manifest"]):
        return "installation"
    if any(token in value for token in ["fix", "bug", "repair", "resolve", "correct"]):
        return "fixes"
    if any(token in value for token in ["test", "e2e", "coverage"]):
        return "tests"
    if any(token in value for token in ["doc", "readme", "website"]):
        return "docs"
    return "other"


def clean_subject(subject: str) -> str:
    value = subject.strip()
    value = re.sub(r"^(feat|fix|docs|test|chore|refactor|perf)(\([^)]+\))?:\s*", "", value, flags=re.I)
    return value[:1].upper() + value[1:] if value else subject


def format_commit(commit: Commit, repo: str) -> str:
    subject = clean_subject(commit.subject)
    match = PR_RE.search(subject)
    if match:
        pr = match.group("num")
        subject = PR_RE.sub("", subject).strip()
        return f"{subject} ([#{pr}](https://github.com/{repo}/pull/{pr}))"
    return f"{subject} ({commit.sha})"


def grouped_changes(commits: Iterable[Commit], repo: str) -> dict[str, list[str]]:
    groups: dict[str, list[str]] = {key: [] for key, _, _ in CATEGORY_ORDER}
    seen: set[str] = set()
    for commit in commits:
        item = format_commit(commit, repo)
        if item in seen:
            continue
        seen.add(item)
        groups[classify(commit.subject)].append(item)
    return {key: values for key, values in groups.items() if values}


def render_changes(groups: dict[str, list[str]], language: str) -> str:
    if not groups:
        return "- No user-facing changes were detected in the commit range.\n" if language == "en" else "- 未检测到面向用户的变更。\n"
    lines: list[str] = []
    for key, en_title, zh_title in CATEGORY_ORDER:
        items = groups.get(key)
        if not items:
            continue
        title = en_title if language == "en" else zh_title
        lines.append(f"### {title}")
        lines.append("")
        for item in items:
            lines.append(f"- {item}")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def render_release_body(ctx: ReleaseContext, commits: list[Commit]) -> str:
    en_kind, _ = release_kind(ctx.version)
    groups = grouped_changes(commits, ctx.repo)
    previous = f"Kdubbo {display_version(ctx.previous_tag)}" if ctx.previous_tag else "the initial release"
    return f"""# Announcing Kdubbo {ctx.version}

Kdubbo {ctx.version} {en_kind}.

{english_date(ctx.date)}

This release note describes what is different between {previous} and Kdubbo {ctx.version}.

[Before you upgrade](https://kdubbo.github.io/operation/) | [Download]({github_release_url(ctx)}) | [Docs](https://kdubbo.github.io/) | [Source Changes]({source_changes_url(ctx)})

## Changes

{render_changes(groups, "en")}"""


def render_docs_page(ctx: ReleaseContext, commits: list[Commit], language: str) -> str:
    en_kind, zh_kind = release_kind(ctx.version)
    groups = grouped_changes(commits, ctx.repo)
    if language == "en":
        previous = f"Kdubbo {display_version(ctx.previous_tag)}" if ctx.previous_tag else "the initial release"
        return f"""# Announcing Kdubbo {ctx.version}

Kdubbo {ctx.version} {en_kind}.

{english_date(ctx.date)}

This release note describes what is different between {previous} and Kdubbo {ctx.version}.

[Before you upgrade](../../../operation/) | [Download]({github_release_url(ctx)}) | [Docs](https://kdubbo.github.io/) | [Source Changes]({source_changes_url(ctx)})

## Changes

{render_changes(groups, "en")}"""

    previous_zh = f"Kdubbo {display_version(ctx.previous_tag)}" if ctx.previous_tag else "初始版本"
    return f"""# Kdubbo {ctx.version} 发布公告

Kdubbo {ctx.version} {zh_kind}。

发布日期：{ctx.date.isoformat()}

本发布说明描述 {previous_zh} 和 Kdubbo {ctx.version} 之间的差异。

[升级前注意事项](../../../operation/) | [下载]({github_release_url(ctx)}) | [文档](https://kdubbo.github.io/) | [源码变更]({source_changes_url(ctx)})

## 变更

{render_changes(groups, "zh")}"""


def docs_paths(docs_root: Path, version: str) -> tuple[Path, Path]:
    base = docs_root / "latest" / "release" / minor_series(version)
    return base / f"announcing-{version}.md", base / f"announcing-{version}.en.md"


def render_index_entry(ctx: ReleaseContext, commits: list[Commit]) -> str:
    groups = grouped_changes(commits, ctx.repo)
    first_items = []
    for key, _, zh_title in CATEGORY_ORDER:
        if key in groups:
            first_items.append(f"{zh_title}：{groups[key][0]}")
        if len(first_items) == 3:
            break
    summary = "\n".join(f"    - {item}" for item in first_items) or "    - 查看完整发布说明"
    page = f"{minor_series(ctx.version)}/announcing-{ctx.version}.md"
    return f"""    ### [{ctx.version}]({page})

    发布日期：{ctx.date.isoformat()}

{summary}

    [查看完整发布说明]({page})
    [查看 GitHub Release]({github_release_url(ctx)})

    ---"""


def update_docs_index(docs_root: Path, ctx: ReleaseContext, commits: list[Commit]) -> None:
    index = docs_root / "latest" / "release" / "index.md"
    start = "    <!-- KDUBBO_RELEASE_NOTES:START -->"
    end = "    <!-- KDUBBO_RELEASE_NOTES:END -->"
    entry = render_index_entry(ctx, commits)
    content = index.read_text(encoding="utf-8")
    if start in content and end in content:
        before, rest = content.split(start, 1)
        block, after = rest.split(end, 1)
        old_entries = [part.strip("\n") for part in block.strip().split("\n\n    ---\n\n") if part.strip()]
        old_entries = [part for part in old_entries if f"announcing-{ctx.version}.md" not in part]
        normalized = [entry.rstrip()]
        for part in old_entries:
            normalized.append(part.rstrip() + "\n\n    ---")
        new_block = "\n\n" + "\n\n".join(normalized).rstrip() + "\n\n"
        index.write_text(before + start + new_block + end + after, encoding="utf-8")
        return

    anchor = '=== "发布公告"'
    if anchor not in content:
        raise ValueError(f"{index} does not contain the release announcement tab")
    replacement = f'{anchor}\n\n{start}\n\n{entry}\n\n{end}\n'
    index.write_text(content.replace(anchor, replacement, 1), encoding="utf-8")


def write_docs(docs_root: Path, ctx: ReleaseContext, commits: list[Commit]) -> None:
    zh_path, en_path = docs_paths(docs_root, ctx.version)
    zh_path.parent.mkdir(parents=True, exist_ok=True)
    zh_path.write_text(render_docs_page(ctx, commits, "zh"), encoding="utf-8")
    en_path.write_text(render_docs_page(ctx, commits, "en"), encoding="utf-8")
    update_docs_index(docs_root, ctx, commits)


def build_context(args: argparse.Namespace, cwd: Path) -> ReleaseContext:
    tag = strip_ref(args.tag or os.environ.get("GITHUB_REF_NAME", ""))
    if not tag:
        tag = strip_ref(run_git(["describe", "--tags", "--abbrev=0"], cwd))
    version = display_version(tag)
    parse_version(version)
    previous_tag = args.previous_tag if args.previous_tag is not None else previous_tag_for(tag, cwd)
    date_value = dt.date.fromisoformat(args.date) if args.date else dt.date.today()
    repo = args.repo or os.environ.get("GITHUB_REPOSITORY", "apache/dubbo-kubernetes")
    return ReleaseContext(tag=tag, version=version, previous_tag=previous_tag, repo=repo, date=date_value)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate Kdubbo release notes and kdubbo.github.io pages.")
    parser.add_argument("--tag", help="Release tag, for example 0.4.0 or v0.4.0.")
    parser.add_argument("--previous-tag", help="Previous release tag. Auto-detected from git tags when omitted.")
    parser.add_argument("--repo", help="GitHub repository, for example apache/dubbo-kubernetes.")
    parser.add_argument("--date", help="Release date in YYYY-MM-DD format. Defaults to today.")
    parser.add_argument("--release-body", type=Path, help="Write the GitHub Release body markdown to this file.")
    parser.add_argument("--docs-root", type=Path, help="Path to kdubbo.github.io/docs to update.")
    parser.add_argument("--print", action="store_true", help="Print the GitHub Release body to stdout.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    cwd = Path.cwd()
    ctx = build_context(args, cwd)
    commits = collect_commits(ctx, cwd)
    body = render_release_body(ctx, commits)
    if args.release_body:
        args.release_body.parent.mkdir(parents=True, exist_ok=True)
        args.release_body.write_text(body, encoding="utf-8")
    if args.docs_root:
        write_docs(args.docs_root, ctx, commits)
    if args.print:
        print(body)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
