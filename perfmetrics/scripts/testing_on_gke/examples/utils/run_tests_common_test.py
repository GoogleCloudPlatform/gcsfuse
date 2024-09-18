# Copyright 2018 The Kubernetes Authors.
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest
from run_tests_common import escape_commas_in_string


class RunTestsCommonTest(unittest.TestCase):

  def test_escape_commas_in_string(self):
    tcs = [
        {"input": "a:b,c=d,", "expected_output": "a:b\,c=d\,"},
        {"input": "", "expected_output": ""},
    ]
    for tc in tcs:
      self.assertEqual(
          tc["expected_output"], escape_commas_in_string(tc["input"])
      )


if __name__ == "__main__":
  unittest.main()
