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

import datetime as dt
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest


MODULE_PATH = Path(__file__).with_name("generate_release_notes.py")
SPEC = importlib.util.spec_from_file_location("generate_release_notes", MODULE_PATH)
assert SPEC and SPEC.loader
release_notes = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = release_notes
SPEC.loader.exec_module(release_notes)


class ReleaseNotesTest(unittest.TestCase):
    def context(self):
        return release_notes.ReleaseContext(
            tag="0.4.1",
            version="0.4.1",
            previous_tag="0.4.0",
            repo="apache/dubbo-kubernetes",
            date=dt.date(2026, 5, 18),
        )

    def commits(self):
        return [
            release_notes.Commit("feat: Add multicluster east-west gateway (#910)", "abc1234", "dev"),
            release_notes.Commit("fix: Repair remote webhook CA bundle (#911)", "def5678", "dev"),
            release_notes.Commit("docs: Update install guide", "abcd999", "dev"),
        ]

    def test_release_body_uses_istio_style_sections(self):
        body = release_notes.render_release_body(self.context(), self.commits())

        self.assertIn("# Announcing Kdubbo 0.4.1", body)
        self.assertIn("Kdubbo 0.4.1 patch release.", body)
        self.assertIn("May 18, 2026", body)
        self.assertIn("[Before you upgrade]", body)
        self.assertIn("[Source Changes](https://github.com/apache/dubbo-kubernetes/compare/0.4.0...0.4.1)", body)
        self.assertIn("### Features", body)
        self.assertIn("[#910](https://github.com/apache/dubbo-kubernetes/pull/910)", body)

    def test_writes_bilingual_docs_and_index_entry(self):
        with tempfile.TemporaryDirectory() as temp:
            docs = Path(temp) / "docs"
            release_dir = docs / "latest" / "release"
            release_dir.mkdir(parents=True)
            (release_dir / "index.md").write_text('# 公告栏\n\n=== "发布公告"\n\n    ## 0.3.x\n', encoding="utf-8")

            release_notes.write_docs(docs, self.context(), self.commits())

            zh = release_dir / "0.4.x" / "announcing-0.4.1.md"
            en = release_dir / "0.4.x" / "announcing-0.4.1.en.md"
            index = release_dir / "index.md"

            self.assertTrue(zh.exists())
            self.assertTrue(en.exists())
            self.assertIn("# Kdubbo 0.4.1 发布公告", zh.read_text(encoding="utf-8"))
            self.assertIn("# Announcing Kdubbo 0.4.1", en.read_text(encoding="utf-8"))
            self.assertIn("KDUBBO_RELEASE_NOTES:START", index.read_text(encoding="utf-8"))
            self.assertIn("[0.4.1](0.4.x/announcing-0.4.1.md)", index.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
