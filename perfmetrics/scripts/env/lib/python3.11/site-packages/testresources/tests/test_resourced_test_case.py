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

import unittest
import testtools
import testresources
from testresources.tests import ResultWithResourceExtensions


def test_suite():
    loader = testresources.tests.TestUtil.TestLoader()
    result = loader.loadTestsFromName(__name__)
    return result


class MockResource(testresources.TestResource):
    """Resource used for testing ResourcedTestCase."""

    def __init__(self, resource):
        testresources.TestResource.__init__(self)
        self._resource = resource

    def make(self, dependency_resources):
        return self._resource


class MockResourceInstance(object):
    """A resource instance."""


class TestResourcedTestCase(testtools.TestCase):

    def setUp(self):
        super(TestResourcedTestCase, self).setUp()
        class Example(testresources.ResourcedTestCase):
            def test_example(self):
                pass
        self.resourced_case = Example('test_example')
        self.resource = self.getUniqueString()
        self.resource_manager = MockResource(self.resource)

    def testSetUpUsesSuper(self):
        class OtherBaseCase(unittest.TestCase):
            setUpCalled = False
            def setUp(self):
                self.setUpCalled = True
                super(OtherBaseCase, self).setUp()
        class OurCase(testresources.ResourcedTestCase, OtherBaseCase):
            def runTest(self):
                pass
        ourCase = OurCase()
        ourCase.setUp()
        self.assertTrue(ourCase.setUpCalled)

    def testTearDownUsesSuper(self):
        class OtherBaseCase(unittest.TestCase):
            tearDownCalled = False
            def tearDown(self):
                self.tearDownCalled = True
                super(OtherBaseCase, self).setUp()
        class OurCase(testresources.ResourcedTestCase, OtherBaseCase):
            def runTest(self):
                pass
        ourCase = OurCase()
        ourCase.setUp()
        ourCase.tearDown()
        self.assertTrue(ourCase.tearDownCalled)

    def testDefaults(self):
        self.assertEqual(self.resourced_case.resources, [])

    def testResultPassedToResources(self):
        result = ResultWithResourceExtensions()
        self.resourced_case.resources = [("foo", self.resource_manager)]
        self.resourced_case.run(result)
        self.assertEqual(4, len(result._calls))

    def testSetUpResourcesSingle(self):
        # setUpResources installs the resources listed in ResourcedTestCase.
        self.resourced_case.resources = [("foo", self.resource_manager)]
        testresources.setUpResources(self.resourced_case,
            self.resourced_case.resources, None)
        self.assertEqual(self.resource, self.resourced_case.foo)

    def testSetUpResourcesMultiple(self):
        # setUpResources installs the resources listed in ResourcedTestCase.
        self.resourced_case.resources = [
            ('foo', self.resource_manager),
            ('bar', MockResource('bar_resource'))]
        testresources.setUpResources(self.resourced_case,
            self.resourced_case.resources, None)
        self.assertEqual(self.resource, self.resourced_case.foo)
        self.assertEqual('bar_resource', self.resourced_case.bar)

    def testSetUpResourcesSetsUpDependences(self):
        resource = MockResourceInstance()
        self.resource_manager = MockResource(resource)
        self.resourced_case.resources = [('foo', self.resource_manager)]
        # Give the 'foo' resource access to a 'bar' resource
        self.resource_manager.resources.append(
            ('bar', MockResource('bar_resource')))
        testresources.setUpResources(self.resourced_case,
            self.resourced_case.resources, None)
        self.assertEqual(resource, self.resourced_case.foo)
        self.assertEqual('bar_resource', self.resourced_case.foo.bar)

    def testSetUpUsesResource(self):
        # setUpResources records a use of each declared resource.
        self.resourced_case.resources = [("foo", self.resource_manager)]
        testresources.setUpResources(self.resourced_case,
            self.resourced_case.resources, None)
        self.assertEqual(self.resource_manager._uses, 1)

    def testTearDownResourcesDeletesResourceAttributes(self):
        self.resourced_case.resources = [("foo", self.resource_manager)]
        self.resourced_case.setUpResources()
        self.resourced_case.tearDownResources()
        self.failIf(hasattr(self.resourced_case, "foo"))

    def testTearDownResourcesStopsUsingResource(self):
        # tearDownResources records that there is one less use of each
        # declared resource.
        self.resourced_case.resources = [("foo", self.resource_manager)]
        self.resourced_case.setUpResources()
        self.resourced_case.tearDownResources()
        self.assertEqual(self.resource_manager._uses, 0)

    def testTearDownResourcesStopsUsingDependencies(self):
        resource = MockResourceInstance()
        dep1 = MockResource('bar_resource')
        self.resource_manager = MockResource(resource)
        self.resourced_case.resources = [('foo', self.resource_manager)]
        # Give the 'foo' resource access to a 'bar' resource
        self.resource_manager.resources.append(
            ('bar', dep1))
        self.resourced_case.setUpResources()
        self.resourced_case.tearDownResources()
        self.assertEqual(dep1._uses, 0)

    def testSingleWithSetup(self):
        # setUp and tearDown invoke setUpResources and tearDownResources.
        self.resourced_case.resources = [("foo", self.resource_manager)]
        self.resourced_case.setUp()
        self.assertEqual(self.resourced_case.foo, self.resource)
        self.assertEqual(self.resource_manager._uses, 1)
        self.resourced_case.tearDown()
        self.failIf(hasattr(self.resourced_case, "foo"))
        self.assertEqual(self.resource_manager._uses, 0)
