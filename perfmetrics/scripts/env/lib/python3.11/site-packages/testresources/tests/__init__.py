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

from unittest import TestResult

import testresources
from testresources.tests import TestUtil

def test_suite():
    import testresources.tests.test_optimising_test_suite
    import testresources.tests.test_resourced_test_case
    import testresources.tests.test_test_loader
    import testresources.tests.test_test_resource
    import testresources.tests.test_resource_graph
    result = TestUtil.TestSuite()
    result.addTest(testresources.tests.test_test_loader.test_suite())
    result.addTest(testresources.tests.test_test_resource.test_suite())
    result.addTest(testresources.tests.test_resourced_test_case.test_suite())
    result.addTest(testresources.tests.test_resource_graph.test_suite())
    result.addTest(
        testresources.tests.test_optimising_test_suite.test_suite())
    return result


class ResultWithoutResourceExtensions(object):
    """A test fake which does not have resource extensions."""


class ResultWithResourceExtensions(TestResult):
    """A test fake which has resource extensions."""

    def __init__(self):
        TestResult.__init__(self)
        self._calls = []

    def startCleanResource(self, resource):
        self._calls.append(("clean", "start", resource))

    def stopCleanResource(self, resource):
        self._calls.append(("clean", "stop", resource))

    def startMakeResource(self, resource):
        self._calls.append(("make", "start", resource))

    def stopMakeResource(self, resource):
        self._calls.append(("make", "stop", resource))

    def startResetResource(self, resource):
        self._calls.append(("reset", "start", resource))

    def stopResetResource(self, resource):
        self._calls.append(("reset", "stop", resource))
