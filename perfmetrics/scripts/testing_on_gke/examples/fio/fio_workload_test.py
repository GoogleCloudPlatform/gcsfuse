"""This file defines unit tests for functionalities in fio_workload.py"""

import unittest
from fio_workload import FioWorkload, validateFioWorkload


class FioWorkloadTest(unittest.TestCase):

  def test_validate_fio_workload_empty(self):
    self.assertFalse(validateFioWorkload(({}), "empty-fio-workload"))

  def test_validate_fio_workload_invalid_missing_bucket(self):
    self.assertFalse(
        validateFioWorkload(
            ({"fioWorkload": {}, "gcsfuseMountOptions": ""}),
            "invalid-fio-workload-missing-bucket",
        )
    )

  def test_validate_fio_workload_invalid_bucket_contains_space(self):
    self.assertFalse(
        validateFioWorkload(
            ({"fioWorkload": {}, "gcsfuseMountOptions": "", "bucket": " "}),
            "invalid-fio-workload-bucket-contains-space",
        )
    )

  def test_validate_fio_workload_invalid_no_fioWorkloadSpecified(self):
    self.assertFalse(
        validateFioWorkload(({"bucket": {}}), "invalid-fio-workload-2")
    )

  def test_validate_fio_workload_invalid_commented_out_fioWorkload(self):
    self.assertFalse(
        validateFioWorkload(
            ({
                "_fioWorkload": {},
                "bucket": "dummy-bucket",
                "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
            }),
            "commented-out-fio-workload",
        )
    )

  def test_validate_fio_workload_invalid_mixed_fioWorkload_dlioWorkload(self):
    self.assertFalse(
        validateFioWorkload(
            ({
                "fioWorkload": {},
                "dlioWorkload": {},
                "bucket": "dummy-bucket",
                "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
            }),
            "mixed-fio/dlio-workload",
        )
    )

  def test_validate_fio_workload_invalid_missing_fileSize(self):
    workload = dict({
        "fioWorkload": {
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-fileSize")
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_fileSize(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": 1000,
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-fileSize"
        )
    )
    pass

  def test_validate_fio_workload_invalid_missing_blockSize(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-blockSize")
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_blockSize(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "blockSize": 1000,
            "filesPerThread": 2,
            "numThreads": 100,
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-blockSize"
        )
    )
    pass

  def test_validate_fio_workload_invalid_missing_filesPerThread(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-missing-filesPerThread"
        )
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_filesPerThread(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": "1k",
            "numThreads": 100,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-filesPerThread"
        )
    )
    pass

  def test_validate_fio_workload_invalid_missing_numThreads(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(workload, "invalid-fio-workload-missing-numThreads")
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_numThreads(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": "1k",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-numThreads"
        )
    )
    pass

  def test_validate_fio_workload_invalid_missing_gcsfuseMountOptions(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": "1k",
        },
        "bucket": "dummy-bucket",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-missing-gcsfuseMountOptions"
        )
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_gcsfuseMountOptions(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": "1k",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": 100,
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-numThreads"
        )
    )
    pass

  def test_validate_fio_workload_invalid_gcsfuseMountOptions_contains_space(
      self,
  ):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": "1k",
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "abc def",
    })
    self.assertFalse(
        validateFioWorkload(
            workload,
            "invalid-fio-workload-unsupported-gcsfuseMountOptions-contains-space",
        )
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_readTypes_1(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": 10,
            "readTypes": True,
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-readTypes-1"
        )
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_readTypes_2(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": 10,
            "readTypes": ["read", 1],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-readTypes-2"
        )
    )
    pass

  def test_validate_fio_workload_invalid_unsupported_readTypes_3(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "blockSize": "1kb",
            "numThreads": 10,
            "readTypes": ["read", "write"],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertFalse(
        validateFioWorkload(
            workload, "invalid-fio-workload-unsupported-readTypes-3"
        )
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
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
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
            "readTypes": ["read", "randread"],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertTrue(validateFioWorkload(workload, "valid-fio-workload-2"))
    pass

  def test_validate_fio_workload_valid_with_single_readType(self):
    workload = dict({
        "fioWorkload": {
            "fileSize": "1kb",
            "filesPerThread": 2,
            "numThreads": 100,
            "blockSize": "1kb",
            "readTypes": ["randread"],
        },
        "bucket": "dummy-bucket",
        "gcsfuseMountOptions": "implicit-dirs,cache-max-size:-1",
    })
    self.assertTrue(validateFioWorkload(workload, "valid-fio-workload-2"))
    pass


if __name__ == "__main__":
  unittest.main()
