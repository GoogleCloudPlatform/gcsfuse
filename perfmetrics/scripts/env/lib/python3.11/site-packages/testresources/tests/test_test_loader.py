#  testresources: extensions to python unittest to allow declaritive use
#  of resources by test cases.
#
#  Copyright (c) 2005-2010 Testresources Contributors
#  
#  Licensed under either the Apache License, Version 2.0 or the BSD 3-clause
#  license at the users choice. A copy of both licenses are available in the
#  project source as Apache-2.0 and BSD. You may not use this file except in
#  compliance with one of these two licences.
#  
#  Unless required by applicable law or agreed to in writing, software distributed
#  under these licenses is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
#  CONDITIONS OF ANY KIND, either express or implied.  See the license you chose
#  for the specific language governing permissions and limitations under that
#  license.
#

import testtools
from testresources import TestLoader, OptimisingTestSuite
from testresources.tests import TestUtil


def test_suite():
    loader = TestUtil.TestLoader()
    result = loader.loadTestsFromName(__name__)
    return result


class TestTestLoader(testtools.TestCase):

    def testSuiteType(self):
        # The testresources TestLoader loads tests into an
        # OptimisingTestSuite.
        loader = TestLoader()
        suite = loader.loadTestsFromName(__name__)
        self.assertIsInstance(suite, OptimisingTestSuite)
