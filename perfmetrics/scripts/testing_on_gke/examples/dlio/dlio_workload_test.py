"""This file defines unit tests for functionalities in dlio_workload.py"""

import unittest
from dlio_workload import DlioWorkload, validateDlioWorkload


class DlioWorkloadTest(unittest.TestCase):

  def test_validate_dlio_workload_empty(self):
    self.assertFalse(validateDlioWorkload(({}), "empty-dlio-workload"))

  def test_validate_dlio_workload_invalid_no_bucket(self):
    self.assertFalse(
        validateDlioWorkload(({"dlioWorkload": {}}), "invalid-dlio-workload-1")
    )

  def test_validate_dlio_workload_invalid_no_dlioWorkloadSpecified(self):
    self.assertFalse(
        validateDlioWorkload(({"bucket": {}}), "invalid-dlio-workload-2")
    )

  def test_validate_dlio_workload_invalid_commented_out_dlioWorkload(self):
    self.assertFalse(
        validateDlioWorkload(
            ({"_dlioWorkload": {}, "bucket": "dummy-bucket"}),
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
            }),
            "mixed-dlio/fio-workload",
        )
    )

  def test_validate_dlio_workload_invalid_missing_numFilesTrain(self):
    workload = dict({
        "dlioWorkload": {
            # "numFilesTrain": 1000,
            "recordLength": "10kb",
            "batchSizes": "100,200",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-blockSize"
        )
    )
    pass

  def test_validate_dlio_workload_invalid_missing_recordLength(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            # "recordLength": "10kb",
            "batchSizes": "100,200",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-filesPerThread"
        )
    )
    pass

  def test_validate_dlio_workload_invalid_missing_batchSizes(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": "10kb",
            # "batchSizes": "100,200",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateDlioWorkload(
            workload, "invalid-dlio-workload-missing-numThreads"
        )
    )
    pass

  def test_validate_dlio_workload_valid_single_batchSize(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": "10kb",
            "batchSizes": "100",
        },
        "bucket": "dummy-bucket",
    })
    self.assertTrue(validateDlioWorkload(workload, "valid-dlio-workload-2"))
    pass

  def test_validate_dlio_workload_valid_multiple_batchSizes(self):
    workload = dict({
        "dlioWorkload": {
            "numFilesTrain": 1000,
            "recordLength": "10kb",
            "batchSizes": "100,200",
        },
        "bucket": "dummy-bucket",
    })
    self.assertTrue(validateDlioWorkload(workload, "valid-dlio-workload-2"))
    pass


if __name__ == "__main__":
  unittest.main()
