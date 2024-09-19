"""This file defines unit tests for functionalities in utils.py"""

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
from gsheet import append_to_gsheet
import utils


class UtilsTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    self.project_id = 'gcs-fuse-test'

  def test_append_to_gsheet(self):
    append_to_gsheet(
        worksheet='fio',
        data=[
            ('Column1', 'Column2', 'Column3', 'Column4'),
            ('d', 2, 0.33, 'beta'),
        ],
        project_id='gcs-fuse-test',
    )


if __name__ == '__main__':
  unittest.main()
