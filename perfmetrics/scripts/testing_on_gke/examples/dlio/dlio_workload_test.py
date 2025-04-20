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

"""This file defines unit tests for functionalities in dlio_workload.py"""

import unittest
from dlio_workload import DlioWorkload, validateDlioWorkload


class DlioWorkloadTest(unittest.TestCase):

  def test_validate_dlio_workload_empty(self):
    self.assertFalse(validateDlioWorkload(({}), "empty-dlio-workload"))

  def test_validate_dlio_workload_invalid_missing_bucket(self):
    self.assertFalse(
        validateDlioWorkload(
            ({"dlioWorkload": {}, "gcsfuseMountOptions": ""}),
            "invalid-dlio-workload-missing-bucket",
        )
    )

  def test_validate_dlio_workload_invalid_bucket_contains_space(self):
    self.assertFalse(
        validateDlioWorkload(
            ({"dlioWorkload": {}, "gcsfuseMountOptions": "", "bucket": " "}),
            "invalid-dlio-workload-bucket-contains-space",
        )
    )

  def test_validate_dlio_workload_invalid_no_dlioWorkloadSpecified(self):
    self.assertFalse(
        validateDlioWorkload(({"bucket": {}}), "invalid-dlio-workload-2")
    )

  def test_validate_dlio_workload_invalid_commented_out_dlioWorkload(self):
    self.assertFalse(
        validateDlioWorkload(
            ({
                "_dlioWorkload": {},
                "bucket": "dummy-bucket",
                "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
            }),
            "commented-out-dlio-workload",
        )
    )

  def test_validate_dlio_workload_invalid_mixed_dlioWorkload_fioWorkload(self):
    self.assertFalse(
        validateDlioWorkload(
            ({
                "dlioWorkload": {},
                "fioWorkload": {},
                "bucket": "dummy-bucket",
                "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
            }),
            "mixed-dlio/fio-workload",
        )
    )

  def test_validate_dlio_workload_invalid_missing_numFilesTrain(self):
    workload = dict({
        "dlioWorkload": {
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-numFilesTrain"
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_numFilesTrain(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": "1000",
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-numFilesTrain"
        )
    )

  def test_validate_dlio_workload_invalid_missing_recordLength(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-recordLength"
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_recordLength(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": "10000",
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-recordLength"
        )
    )

  def test_validate_dlio_workload_invalid_missing_gcsfuseMountOptions(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 100,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-gcsfuseMountOptions"
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_gcsfuseMountOptions(
      self,
  ):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": 100,
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-gcsfuseMountOptions1"
        )
    )

  def test_validate_dlio_workload_invalid_gcsfuseMountOptions_contains_space(
      self,
  ):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "abc def",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload,
            "invalid-dlio-workload-unsupported-gcsfuseMountOptions-contains-space",
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_numEpochs(
      self,
  ):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs",
        "numEpochs": False,
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-numEpochs"
        )
    )

  def test_validate_dlio_workload_invalid_numEpochs_toolow(
      self,
  ):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs",
        "numEpochs": -1,
    })
    self.assertFalse(
        validateDlioWorkload(
            workload,
            "invalid-dlio-workload-unsupported-numEpochs-too-low",
        )
    )

  def test_validate_dlio_workload_invalid_missing_batchSizes(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-batchSizes"
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_batchSizes1(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": ["100"],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-batchSizes1"
        )
    )

  def test_validate_dlio_workload_invalid_unsupported_batchSizes2(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [0, -1],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-unsupported-batchSizes2"
        )
    )

  def test_validate_dlio_workload_valid_single_batchSize(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertTrue(validateDlioWorkload(workload, "valid-dlio-workload-2"))

  def test_validate_dlio_workload_valid_multiple_batchSizes(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": 10000,
            "batchSizes": [100, 200],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertTrue(validateDlioWorkload(workload, "valid-dlio-workload-2"))


if __name__ == "__main__":
  unittest.main()
