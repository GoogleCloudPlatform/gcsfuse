"""This file defines unit tests for functionalities in fio_workload.py"""

import unittest
from fio_workload import FioWorkload, validateFioWorkload


class FioWorkloadTest(unittest.TestCase):

  def test_validate_fio_workload_empty(self):
    self.assertFalse(validateFioWorkload(({}), "empty-fio-workload"))

  def test_validate_fio_workload_invalid_no_bucket(self):
    self.assertFalse(
        validateFioWorkload(({"fioWorkload": {}}), "invalid-fio-workload-1")
    )

  def test_validate_fio_workload_invalid_no_fioWorkloadSpecified(self):
    self.assertFalse(
        validateFioWorkload(({"bucket": {}}), "invalid-fio-workload-2")
    )

  def test_validate_fio_workload_invalid_commented_out_fioWorkload(self):
    self.assertFalse(
        validateFioWorkload(
            ({"_fioWorkload": {}, "bucket": "dummy-bucket"}),
            "commented-out-fio-workload",
        )
    )

  def test_validate_fio_workload_invalid_mixed_fioWorkload_dlioWorkload(self):
    self.assertFalse(
        validateFioWorkload(
            ({"fioWorkload": {}, "dlioWorkload": {}, "bucket": "dummy-bucket"}),
            "mixed-fio/dlio-workload",
        )
    )

  def test_validate_fio_workload_invalid_missing_fileSize(self):
    workload = dict({
        "fioWorkload": {
            # "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-fileSize")
    )
    pass

  def test_validate_fio_workload_invalid_missing_blockSize(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
            # "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-blockSize")
    )
    pass

  def test_validate_fio_workload_invalid_missing_filesPerThread(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            # "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-missing-filesPerThread"
        )
    )
    pass

  def test_validate_fio_workload_invalid_missing_numThreads(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            # "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-numThreads")
    )
    pass

  def test_validate_fio_workload_valid_without_readTypes(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
    })
    self.assertTrue(validateFioWorkload(workload, "valid-fio-workload-1"))
    pass

  def test_validate_fio_workload_valid_with_readTypes(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
            "readTypes": "read,randread",
        },
        "bucket": "dummy-bucket",
    })
    self.assertTrue(validateFioWorkload(workload, "valid-fio-workload-2"))
    pass


if __name__ == "__main__":
  unittest.main()
