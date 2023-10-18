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

from fixtures.tests.helpers import LoggingFixture
import testtools

import testresources
from testresources.tests import (
    ResultWithResourceExtensions,
    ResultWithoutResourceExtensions,
    )


def test_suite():
    loader = testresources.tests.TestUtil.TestLoader()
    result = loader.loadTestsFromName(__name__)
    return result


class MockResourceInstance(object):

    def __init__(self, name):
        self._name = name

    def __eq__(self, other):
        return self.__dict__ == other.__dict__

    def __cmp__(self, other):
        return cmp(self.__dict__, other.__dict__)

    def __repr__(self):
        return self._name


class MockResource(testresources.TestResourceManager):
    """Mock resource that logs the number of make and clean calls."""

    def __init__(self):
        super(MockResource, self).__init__()
        self.makes = 0
        self.cleans = 0

    def clean(self, resource):
        self.cleans += 1

    def make(self, dependency_resources):
        self.makes += 1
        return MockResourceInstance("Boo!")


class MockResettableResource(MockResource):
    """Mock resource that logs the number of reset calls too."""

    def __init__(self):
        super(MockResettableResource, self).__init__()
        self.resets = 0

    def _reset(self, resource, dependency_resources):
        self.resets += 1
        resource._name += "!"
        self._dirty = False
        return resource


class TestTestResource(testtools.TestCase):

    def testUnimplementedGetResource(self):
        # By default, TestResource raises NotImplementedError on getResource
        # because make is not defined initially.
        resource_manager = testresources.TestResource()
        self.assertRaises(NotImplementedError, resource_manager.getResource)

    def testInitiallyNotDirty(self):
        resource_manager = testresources.TestResource()
        self.assertEqual(False, resource_manager._dirty)

    def testInitiallyUnused(self):
        resource_manager = testresources.TestResource()
        self.assertEqual(0, resource_manager._uses)

    def testInitiallyNoCurrentResource(self):
        resource_manager = testresources.TestResource()
        self.assertEqual(None, resource_manager._currentResource)

    def testneededResourcesDefault(self):
        # Calling neededResources on a default TestResource returns the
        # resource.
        resource = testresources.TestResource()
        self.assertEqual([resource], resource.neededResources())

    def testneededResourcesDependenciesFirst(self):
        # Calling neededResources on a TestResource with dependencies puts the
        # dependencies first.
        resource = testresources.TestResource()
        dep1 = testresources.TestResource()
        dep2 = testresources.TestResource()
        resource.resources.append(("dep1", dep1))
        resource.resources.append(("dep2", dep2))
        self.assertEqual([dep1, dep2, resource], resource.neededResources())

    def testneededResourcesClosure(self):
        # Calling neededResources on a TestResource with dependencies includes
        # the needed resources of the needed resources.
        resource = testresources.TestResource()
        dep1 = testresources.TestResource()
        dep2 = testresources.TestResource()
        resource.resources.append(("dep1", dep1))
        dep1.resources.append(("dep2", dep2))
        self.assertEqual([dep2, dep1, resource], resource.neededResources())

    def testDefaultCosts(self):
        # The base TestResource costs 1 to set up and to tear down.
        resource_manager = testresources.TestResource()
        self.assertEqual(resource_manager.setUpCost, 1)
        self.assertEqual(resource_manager.tearDownCost, 1)

    def testGetResourceReturnsMakeResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        self.assertEqual(resource_manager.make({}), resource)

    def testGetResourceIncrementsUses(self):
        resource_manager = MockResource()
        resource_manager.getResource()
        self.assertEqual(1, resource_manager._uses)
        resource_manager.getResource()
        self.assertEqual(2, resource_manager._uses)

    def testGetResourceDoesntDirty(self):
        resource_manager = MockResource()
        resource_manager.getResource()
        self.assertEqual(resource_manager._dirty, False)

    def testGetResourceSetsCurrentResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        self.assertIs(resource_manager._currentResource, resource)

    def testGetResourceTwiceReturnsIdenticalResource(self):
        resource_manager = MockResource()
        resource1 = resource_manager.getResource()
        resource2 = resource_manager.getResource()
        self.assertIs(resource1, resource2)

    def testGetResourceCallsMakeResource(self):
        resource_manager = MockResource()
        resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)

    def testIsDirty(self):
        resource_manager = MockResource()
        r = resource_manager.getResource()
        resource_manager.dirtied(r)
        self.assertTrue(resource_manager.isDirty())
        resource_manager.finishedWith(r)

    def testIsDirtyIsTrueIfDependenciesChanged(self):
        resource_manager = MockResource()
        dep1 = MockResource()
        dep2 = MockResource()
        dep3 = MockResource()
        resource_manager.resources.append(("dep1", dep1))
        resource_manager.resources.append(("dep2", dep2))
        resource_manager.resources.append(("dep3", dep3))
        r = resource_manager.getResource()
        dep2.dirtied(r.dep2)
        r2 =dep2.getResource()
        self.assertTrue(resource_manager.isDirty())
        resource_manager.finishedWith(r)
        dep2.finishedWith(r2)

    def testIsDirtyIsTrueIfDependenciesAreDirty(self):
        resource_manager = MockResource()
        dep1 = MockResource()
        dep2 = MockResource()
        dep3 = MockResource()
        resource_manager.resources.append(("dep1", dep1))
        resource_manager.resources.append(("dep2", dep2))
        resource_manager.resources.append(("dep3", dep3))
        r = resource_manager.getResource()
        dep2.dirtied(r.dep2)
        self.assertTrue(resource_manager.isDirty())
        resource_manager.finishedWith(r)

    def testRepeatedGetResourceCallsMakeResourceOnceOnly(self):
        resource_manager = MockResource()
        resource_manager.getResource()
        resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)

    def testGetResourceResetsUsedResource(self):
        resource_manager = MockResettableResource()
        resource_manager.getResource()
        resource = resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)
        resource_manager.dirtied(resource)
        resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)
        self.assertEqual(1, resource_manager.resets)
        resource_manager.finishedWith(resource)

    def testIsResetIfDependenciesAreDirty(self):
        resource_manager = MockResource()
        dep1 = MockResettableResource()
        resource_manager.resources.append(("dep1", dep1))
        r = resource_manager.getResource()
        dep1.dirtied(r.dep1)
        # if we get the resource again, it should be cleaned.
        r = resource_manager.getResource()
        self.assertFalse(resource_manager.isDirty())
        self.assertFalse(dep1.isDirty())
        resource_manager.finishedWith(r)
        resource_manager.finishedWith(r)

    def testUsedResourceResetBetweenUses(self):
        resource_manager = MockResettableResource()
        # take two refs; like happens with OptimisingTestSuite.
        resource_manager.getResource()
        resource = resource_manager.getResource()
        resource_manager.dirtied(resource)
        resource_manager.finishedWith(resource)
        # Get again, but its been dirtied.
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        resource_manager.finishedWith(resource)
        # The resource is made once, reset once and cleaned once.
        self.assertEqual(1, resource_manager.makes)
        self.assertEqual(1, resource_manager.resets)
        self.assertEqual(1, resource_manager.cleans)

    def testFinishedWithDecrementsUses(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource = resource_manager.getResource()
        self.assertEqual(2, resource_manager._uses)
        resource_manager.finishedWith(resource)
        self.assertEqual(1, resource_manager._uses)
        resource_manager.finishedWith(resource)
        self.assertEqual(0, resource_manager._uses)

    def testFinishedWithResetsCurrentResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        self.assertIs(None, resource_manager._currentResource)

    def testFinishedWithCallsCleanResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        self.assertEqual(1, resource_manager.cleans)

    def testUsingTwiceMakesAndCleansTwice(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        self.assertEqual(2, resource_manager.makes)
        self.assertEqual(2, resource_manager.cleans)

    def testFinishedWithCallsCleanResourceOnceOnly(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        self.assertEqual(0, resource_manager.cleans)
        resource_manager.finishedWith(resource)
        self.assertEqual(1, resource_manager.cleans)

    def testFinishedWithMarksNonDirty(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource_manager.dirtied(resource)
        resource_manager.finishedWith(resource)
        self.assertEqual(False, resource_manager._dirty)

    def testResourceAvailableBetweenFinishedWithCalls(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        self.assertIs(resource, resource_manager._currentResource)
        resource_manager.finishedWith(resource)

    def testDirtiedSetsDirty(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        self.assertEqual(False, resource_manager._dirty)
        resource_manager.dirtied(resource)
        self.assertEqual(True, resource_manager._dirty)

    def testDirtyingResourceTriggersCleanOnGet(self):
        resource_manager = MockResource()
        resource1 = resource_manager.getResource()
        resource2 = resource_manager.getResource()
        resource_manager.dirtied(resource2)
        resource_manager.finishedWith(resource2)
        self.assertEqual(0, resource_manager.cleans)
        resource3 = resource_manager.getResource()
        self.assertEqual(1, resource_manager.cleans)
        resource_manager.finishedWith(resource3)
        resource_manager.finishedWith(resource1)
        self.assertEqual(2, resource_manager.cleans)

    def testDefaultResetMethodPreservesCleanResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)
        self.assertEqual(False, resource_manager._dirty)
        resource_manager.reset(resource)
        self.assertEqual(1, resource_manager.makes)
        self.assertEqual(0, resource_manager.cleans)

    def testDefaultResetMethodRecreatesDirtyResource(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        self.assertEqual(1, resource_manager.makes)
        resource_manager.dirtied(resource)
        resource_manager.reset(resource)
        self.assertEqual(2, resource_manager.makes)
        self.assertEqual(1, resource_manager.cleans)

    def testDefaultResetResetsDependencies(self):
        resource_manager = MockResettableResource()
        dep1 = MockResettableResource()
        dep2 = MockResettableResource()
        resource_manager.resources.append(("dep1", dep1))
        resource_manager.resources.append(("dep2", dep2))
        # A typical OptimisingTestSuite workflow
        r_outer = resource_manager.getResource()
        # test 1
        r_inner = resource_manager.getResource()
        dep2.dirtied(r_inner.dep2)
        resource_manager.finishedWith(r_inner)
        # test 2
        r_inner = resource_manager.getResource()
        dep2.dirtied(r_inner.dep2)
        resource_manager.finishedWith(r_inner)
        resource_manager.finishedWith(r_outer)
        # Dep 1 was clean, doesn't do a reset, and should only have one
        # make+clean.
        self.assertEqual(1, dep1.makes)
        self.assertEqual(1, dep1.cleans)
        self.assertEqual(0, dep1.resets)
        # Dep 2 was dirty, so _reset happens, and likewise only one make and
        # clean.
        self.assertEqual(1, dep2.makes)
        self.assertEqual(1, dep2.cleans)
        self.assertEqual(1, dep2.resets)
        # The top layer should have had a reset happen, and only one make and
        # clean.
        self.assertEqual(1, resource_manager.makes)
        self.assertEqual(1, resource_manager.cleans)
        self.assertEqual(1, resource_manager.resets)

    def testDirtyingWhenUnused(self):
        resource_manager = MockResource()
        resource = resource_manager.getResource()
        resource_manager.finishedWith(resource)
        resource_manager.dirtied(resource)
        self.assertEqual(1, resource_manager.makes)
        resource = resource_manager.getResource()
        self.assertEqual(2, resource_manager.makes)

    def testFinishedActivityForResourceWithoutExtensions(self):
        result = ResultWithoutResourceExtensions()
        resource_manager = MockResource()
        r = resource_manager.getResource()
        resource_manager.finishedWith(r, result)

    def testFinishedActivityForResourceWithExtensions(self):
        result = ResultWithResourceExtensions()
        resource_manager = MockResource()
        r = resource_manager.getResource()
        expected = [("clean", "start", resource_manager),
            ("clean", "stop", resource_manager)]
        resource_manager.finishedWith(r, result)
        self.assertEqual(expected, result._calls)

    def testGetActivityForResourceWithoutExtensions(self):
        result = ResultWithoutResourceExtensions()
        resource_manager = MockResource()
        r = resource_manager.getResource(result)
        resource_manager.finishedWith(r)

    def testGetActivityForResourceWithExtensions(self):
        result = ResultWithResourceExtensions()
        resource_manager = MockResource()
        r = resource_manager.getResource(result)
        expected = [("make", "start", resource_manager),
            ("make", "stop", resource_manager)]
        resource_manager.finishedWith(r)
        self.assertEqual(expected, result._calls)

    def testResetActivityForResourceWithoutExtensions(self):
        result = ResultWithoutResourceExtensions()
        resource_manager = MockResource()
        resource_manager.getResource()
        r = resource_manager.getResource()
        resource_manager.dirtied(r)
        resource_manager.finishedWith(r)
        r = resource_manager.getResource(result)
        resource_manager.dirtied(r)
        resource_manager.finishedWith(r)
        resource_manager.finishedWith(resource_manager._currentResource)

    def testResetActivityForResourceWithExtensions(self):
        result = ResultWithResourceExtensions()
        resource_manager = MockResource()
        expected = [("reset", "start", resource_manager),
            ("reset", "stop", resource_manager),
            ]
        resource_manager.getResource()
        r = resource_manager.getResource()
        resource_manager.dirtied(r)
        resource_manager.finishedWith(r)
        r = resource_manager.getResource(result)
        resource_manager.dirtied(r)
        resource_manager.finishedWith(r)
        resource_manager.finishedWith(resource_manager._currentResource)
        self.assertEqual(expected, result._calls)


class TestGenericResource(testtools.TestCase):

    def test_default_uses_setUp_tearDown(self):
        calls = []
        class Wrapped:
            def setUp(self):
                calls.append('setUp')
            def tearDown(self):
                calls.append('tearDown')
        mgr = testresources.GenericResource(Wrapped)
        resource = mgr.getResource()
        self.assertEqual(['setUp'], calls)
        mgr.finishedWith(resource)
        self.assertEqual(['setUp', 'tearDown'], calls)
        self.assertIsInstance(resource, Wrapped)

    def test_dependencies_passed_to_factory(self):
        calls = []
        class Wrapped:
            def __init__(self, **args):
                calls.append(args)
            def setUp(self):pass
            def tearDown(self):pass
        class Trivial(testresources.TestResource):
            def __init__(self, thing):
                testresources.TestResource.__init__(self)
                self.thing = thing
            def make(self, dependency_resources):return self.thing
            def clean(self, resource):pass
        mgr = testresources.GenericResource(Wrapped)
        mgr.resources = [('foo', Trivial('foo')), ('bar', Trivial('bar'))]
        resource = mgr.getResource()
        self.assertEqual([{'foo':'foo', 'bar':'bar'}], calls)
        mgr.finishedWith(resource)

    def test_setup_teardown_controllable(self):
        calls = []
        class Wrapped:
            def start(self):
                calls.append('setUp')
            def stop(self):
                calls.append('tearDown')
        mgr = testresources.GenericResource(Wrapped,
            setup_method_name='start', teardown_method_name='stop')
        resource = mgr.getResource()
        self.assertEqual(['setUp'], calls)
        mgr.finishedWith(resource)
        self.assertEqual(['setUp', 'tearDown'], calls)
        self.assertIsInstance(resource, Wrapped)

    def test_always_dirty(self):
        class Wrapped:
            def setUp(self):pass
            def tearDown(self):pass
        mgr = testresources.GenericResource(Wrapped)
        resource = mgr.getResource()
        self.assertTrue(mgr.isDirty())
        mgr.finishedWith(resource)


class TestFixtureResource(testtools.TestCase):

    def test_uses_setUp_cleanUp(self):
        fixture = LoggingFixture()
        mgr = testresources.FixtureResource(fixture)
        resource = mgr.getResource()
        self.assertEqual(fixture, resource)
        self.assertEqual(['setUp'], fixture.calls)
        mgr.finishedWith(resource)
        self.assertEqual(['setUp', 'cleanUp'], fixture.calls)

    def test_always_dirty(self):
        fixture = LoggingFixture()
        mgr = testresources.FixtureResource(fixture)
        resource = mgr.getResource()
        self.assertTrue(mgr.isDirty())
        mgr.finishedWith(resource)

    def test_reset_called(self):
        fixture = LoggingFixture()
        mgr = testresources.FixtureResource(fixture)
        resource = mgr.getResource()
        mgr.reset(resource)
        mgr.finishedWith(resource)
        self.assertEqual(
            ['setUp', 'reset', 'cleanUp'], fixture.calls)
