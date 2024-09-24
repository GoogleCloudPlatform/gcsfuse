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

import os
from os import open
from os.path import isfile
import random
from random import choices
import string
import unittest
from gsheet import append_data_to_gsheet, download_gcs_object_as_tempfile, url


class GsheetTest(unittest.TestCase):

  # @classmethod
  # def setUpClass(self):
  # self.project_id = 'gcs-fuse-test'

  def test_append_data_to_gsheet(self):
    _DEFAULT_GSHEET_ID = '1UghIdsyarrV1HVNc6lugFZS1jJRumhdiWnPgoEC8Fe4'

    def _default_service_account_key_file(
        project_id: str, localfile: bool
    ) -> str:
      if localfile:
        if project_id == 'gcs-fuse-test':
          return '20240919-gcs-fuse-test-bc1a2c0aac45.json'
        elif project_id == 'gcs-fuse-test-ml':
          return '20240919-gcs-fuse-test-ml-d6e0247b2cf1.json'
        else:
          raise Exception(f'Unknown project-id: {project_id}')
      else:
        if project_id in ['gcs-fuse-test', 'gcs-fuse-test-ml']:
          return f'gs://gcsfuse-aiml-test-outputs/creds/{project_id}.json'
        else:
          raise Exception(f'Unknown project-id: {project_id}')

    for project_id in ['gcs-fuse-test', 'gcs-fuse-test-ml']:
      for worksheet in ['fio-test', 'dlio-test']:
        for localkeyfile in [False]:
          serviceAccountKeyFile = _default_service_account_key_file(
              project_id, localkeyfile
          )
          append_data_to_gsheet(
              worksheet=worksheet,
              data={
                  'header': ('Column1', 'Column2'),
                  'values': [(
                      ''.join(random.choices(string.ascii_letters, k=9)),
                      random.random(),
                  )],
              },
              serviceAccountKeyFile=serviceAccountKeyFile,
              gsheet_id=_DEFAULT_GSHEET_ID,
          )

  def test_gsheet_url(self):
    gsheet_id = ''.join(random.choices(string.ascii_letters, k=20))
    gsheet_url = url(gsheet_id)
    self.assertTrue(gsheet_id in gsheet_url)
    self.assertTrue(len(gsheet_id) < len(gsheet_url))

  def test_download_gcs_object_as_tempfile(self):
    gcs_object = 'gs://gcsfuse-aiml-test-outputs/creds/gcs-fuse-test.json'
    localfile = download_gcs_object_as_tempfile(gcs_object)
    self.assertTrue(localfile)
    self.assertTrue(localfile.strip())
    os.stat(localfile)
    os.remove(localfile)

  def test_download_gcs_object_as_tempfile_nonexistent(self):
    gcs_object = 'gs://non/existing/gcs/object'
    localfile = download_gcs_object_as_tempfile(gcs_object)
    self.assertIsNone(localfile)


if __name__ == '__main__':
  unittest.main()
