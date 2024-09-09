import unittest
from run_tests_common import escape_commas_in_string
# import utils


class RunTestsCommonTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    pass

  def test_escape_commas_in_string(self):
    tcs = [
        {"input": "a:b,c=d,", "expected_output": "a:b\,c=d\,"},
        {"input": "", "expected_output": ""},
    ]
    for tc in tcs:
      self.assertEqual(
          tc["expected_output"], escape_commas_in_string(tc["input"])
      )
      pass


if __name__ == "__main__":
  unittest.main()
