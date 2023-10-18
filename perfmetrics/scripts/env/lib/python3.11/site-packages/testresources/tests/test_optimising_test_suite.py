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
import random
import testresources
from testresources import split_by_resources
from testresources.tests import ResultWithResourceExtensions
import unittest
try:
    import unittest2
except ImportError:
    unittest2 = None


def test_suite():
    from testresources.tests import TestUtil
    loader = TestUtil.TestLoader()
    result = loader.loadTestsFromName(__name__)
    return result


class CustomSuite(unittest.TestSuite):
    """Custom TestSuite that's comparable using == and !=."""

    def __eq__(self, other):
        return (self.__class__ == other.__class__
                and self._tests == other._tests)
    def __ne__(self, other):
        return not self.__eq__(other)


class MakeCounter(testresources.TestResource):
    """Test resource that counts makes and cleans."""

    def __init__(self):
        testresources.TestResource.__init__(self)
        self.cleans = 0
        self.makes = 0
        self.calls = []

    def clean(self, resource):
        self.cleans += 1
        self.calls.append(('clean', resource))

    def make(self, dependency_resources):
        self.makes += 1
        resource = "boo %d" % self.makes
        self.calls.append(('make', resource))
        return resource


class TestOptimisingTestSuite(testtools.TestCase):

    def makeTestCase(self, test_running_hook=None):
        """Make a normal TestCase."""
        class TestCaseForTesting(unittest.TestCase):
            def runTest(self):
                if test_running_hook:
                    test_running_hook(self)
        return TestCaseForTesting('runTest')

    def makeResourcedTestCase(self, resource_manager, test_running_hook):
        """Make a ResourcedTestCase."""
        class ResourcedTestCaseForTesting(testresources.ResourcedTestCase):
            def runTest(self):
                test_running_hook(self)
        test_case = ResourcedTestCaseForTesting('runTest')
        test_case.resources = [('_default', resource_manager)]
        return test_case

    def setUp(self):
        super(TestOptimisingTestSuite, self).setUp()
        self.optimising_suite = testresources.OptimisingTestSuite()

    def testAddTest(self):
        # Adding a single test case is the same as adding one using the
        # standard addTest.
        case = self.makeTestCase()
        self.optimising_suite.addTest(case)
        self.assertEqual([case], self.optimising_suite._tests)

    def testAddTestSuite(self):
        # Adding a standard test suite is the same as adding all the tests in
        # that suite.
        case = self.makeTestCase()
        suite = unittest.TestSuite([case])
        self.optimising_suite.addTest(suite)
        self.assertEqual([case], self.optimising_suite._tests)

    @testtools.skipIf(unittest2 is None, "Unittest2 needed")
    def testAddUnittest2TestSuite(self):
        # Adding a unittest2 test suite is the same as adding all the tests in
        # that suite.
        case = self.makeTestCase()
        suite = unittest2.TestSuite([case])
        self.optimising_suite.addTest(suite)
        self.assertEqual([case], self.optimising_suite._tests)

    def testAddTestOptimisingTestSuite(self):
        # when adding an optimising test suite, it should be unpacked.
        case = self.makeTestCase()
        suite1 = testresources.OptimisingTestSuite([case])
        suite2 = testresources.OptimisingTestSuite([case])
        self.optimising_suite.addTest(suite1)
        self.optimising_suite.addTest(suite2)
        self.assertEqual([case, case], self.optimising_suite._tests)

    def testAddFlattensStandardSuiteStructure(self):
        # addTest will get rid of all unittest.TestSuite structure when adding
        # a test, no matter how much nesting is going on.
        case1 = self.makeTestCase()
        case2 = self.makeTestCase()
        case3 = self.makeTestCase()
        suite = unittest.TestSuite(
            [unittest.TestSuite([case1, unittest.TestSuite([case2])]),
             case3])
        self.optimising_suite.addTest(suite)
        self.assertEqual([case1, case2, case3], self.optimising_suite._tests)

    def testAddDistributesNonStandardSuiteStructure(self):
        # addTest distributes all non-standard TestSuites across their
        # members.
        case1 = self.makeTestCase()
        case2 = self.makeTestCase()
        inner_suite = unittest.TestSuite([case2])
        suite = CustomSuite([case1, inner_suite])
        self.optimising_suite.addTest(suite)
        self.assertEqual(
            [CustomSuite([case1]), CustomSuite([inner_suite])],
            self.optimising_suite._tests)

    def testAddPullsNonStandardSuitesUp(self):
        # addTest flattens standard TestSuites, even those that contain custom
        # suites. When it reaches the custom suites, it distributes them
        # across their members.
        case1 = self.makeTestCase()
        case2 = self.makeTestCase()
        inner_suite = CustomSuite([case1, case2])
        self.optimising_suite.addTest(
            unittest.TestSuite([unittest.TestSuite([inner_suite])]))
        self.assertEqual(
            [CustomSuite([case1]), CustomSuite([case2])],
            self.optimising_suite._tests)

    def testSingleCaseResourceAcquisition(self):
        sample_resource = MakeCounter()
        def getResourceCount(test):
            self.assertEqual(sample_resource._uses, 2)
        case = self.makeResourcedTestCase(sample_resource, getResourceCount)
        self.optimising_suite.addTest(case)
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 1)
        self.assertEqual(result.wasSuccessful(), True)
        self.assertEqual(sample_resource._uses, 0)

    def testResourceReuse(self):
        make_counter = MakeCounter()
        def getResourceCount(test):
            self.assertEqual(make_counter._uses, 2)
        case = self.makeResourcedTestCase(make_counter, getResourceCount)
        case2 = self.makeResourcedTestCase(make_counter, getResourceCount)
        self.optimising_suite.addTest(case)
        self.optimising_suite.addTest(case2)
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 2)
        self.assertEqual(result.wasSuccessful(), True)
        self.assertEqual(make_counter._uses, 0)
        self.assertEqual(make_counter.makes, 1)
        self.assertEqual(make_counter.cleans, 1)

    def testResultPassedToResources(self):
        resource_manager = MakeCounter()
        test_case = self.makeTestCase(lambda x:None)
        test_case.resources = [('_default', resource_manager)]
        self.optimising_suite.addTest(test_case)
        result = ResultWithResourceExtensions()
        self.optimising_suite.run(result)
        # We should see the resource made and cleaned once. As its not a
        # resource aware test, it won't make any calls itself.
        self.assertEqual(4, len(result._calls))

    def testOptimisedRunNonResourcedTestCase(self):
        case = self.makeTestCase()
        self.optimising_suite.addTest(case)
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 1)
        self.assertEqual(result.wasSuccessful(), True)

    def testSortTestsCalled(self):
        # OptimisingTestSuite.run() calls sortTests on the suite.
        class MockOptimisingTestSuite(testresources.OptimisingTestSuite):
            def sortTests(self):
                self.sorted = True

        suite = MockOptimisingTestSuite()
        suite.sorted = False
        suite.run(None)
        self.assertEqual(suite.sorted, True)

    def testResourcesDroppedForNonResourcedTestCase(self):
        sample_resource = MakeCounter()
        def resourced_case_hook(test):
            self.assertTrue(sample_resource._uses > 0)
        self.optimising_suite.addTest(self.makeResourcedTestCase(
            sample_resource, resourced_case_hook))
        def normal_case_hook(test):
            # The resource should not be acquired when the normal test
            # runs.
            self.assertEqual(sample_resource._uses, 0)
        self.optimising_suite.addTest(self.makeTestCase(normal_case_hook))
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 2)
        self.assertEqual([], result.failures)
        self.assertEqual([], result.errors)
        self.assertEqual(result.wasSuccessful(), True)

    def testDirtiedResourceNotRecreated(self):
        make_counter = MakeCounter()
        def dirtyResource(test):
            make_counter.dirtied(test._default)
        case = self.makeResourcedTestCase(make_counter, dirtyResource)
        self.optimising_suite.addTest(case)
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 1)
        self.assertEqual(result.wasSuccessful(), True)
        # The resource should only have been made once.
        self.assertEqual(make_counter.makes, 1)

    def testDirtiedResourceCleanedUp(self):
        make_counter = MakeCounter()
        def testOne(test):
            make_counter.calls.append('test one')
            make_counter.dirtied(test._default)
        def testTwo(test):
            make_counter.calls.append('test two')
        case1 = self.makeResourcedTestCase(make_counter, testOne)
        case2 = self.makeResourcedTestCase(make_counter, testTwo)
        self.optimising_suite.addTest(case1)
        self.optimising_suite.addTest(case2)
        result = unittest.TestResult()
        self.optimising_suite.run(result)
        self.assertEqual(result.testsRun, 2)
        self.assertEqual(result.wasSuccessful(), True)
        # Two resources should have been created and cleaned up
        self.assertEqual(make_counter.calls,
                         [('make', 'boo 1'),
                          'test one',
                          ('clean', 'boo 1'),
                          ('make', 'boo 2'),
                          'test two',
                          ('clean', 'boo 2')])


class TestSplitByResources(testtools.TestCase):
    """Tests for split_by_resources."""

    def makeTestCase(self):
        return unittest.TestCase('run')

    def makeResourcedTestCase(self, has_resource=True):
        case = testresources.ResourcedTestCase('run')
        if has_resource:
            case.resources = [('resource', testresources.TestResource())]
        return case

    def testNoTests(self):
        self.assertEqual({frozenset(): []}, split_by_resources([]))

    def testJustNormalCases(self):
        normal_case = self.makeTestCase()
        resource_set_tests = split_by_resources([normal_case])
        self.assertEqual({frozenset(): [normal_case]}, resource_set_tests)

    def testJustResourcedCases(self):
        resourced_case = self.makeResourcedTestCase()
        resource = resourced_case.resources[0][1]
        resource_set_tests = split_by_resources([resourced_case])
        self.assertEqual({frozenset(): [],
                          frozenset([resource]): [resourced_case]},
                         resource_set_tests)

    def testMultipleResources(self):
        resource1 = testresources.TestResource()
        resource2 = testresources.TestResource()
        resourced_case = self.makeResourcedTestCase(has_resource=False)
        resourced_case.resources = [('resource1', resource1),
                                    ('resource2', resource2)]
        resource_set_tests = split_by_resources([resourced_case])
        self.assertEqual({frozenset(): [],
                          frozenset([resource1, resource2]): [resourced_case]},
                         resource_set_tests)

    def testDependentResources(self):
        resource1 = testresources.TestResource()
        resource2 = testresources.TestResource()
        resource1.resources = [('foo', resource2)]
        resourced_case = self.makeResourcedTestCase(has_resource=False)
        resourced_case.resources = [('resource1', resource1)]
        resource_set_tests = split_by_resources([resourced_case])
        self.assertEqual({frozenset(): [],
                          frozenset([resource1, resource2]): [resourced_case]},
                         resource_set_tests)

    def testResourcedCaseWithNoResources(self):
        resourced_case = self.makeResourcedTestCase(has_resource=False)
        resource_set_tests = split_by_resources([resourced_case])
        self.assertEqual({frozenset(): [resourced_case]}, resource_set_tests)

    def testMixThemUp(self):
        normal_cases = [self.makeTestCase() for i in range(3)]
        normal_cases.extend([
            self.makeResourcedTestCase(has_resource=False) for i in range(3)])
        resourced_cases = [self.makeResourcedTestCase() for i in range(3)]
        all_cases = normal_cases + resourced_cases
        # XXX: Maybe I shouldn't be using random here.
        random.shuffle(all_cases)
        resource_set_tests = split_by_resources(all_cases)
        self.assertEqual(set(normal_cases),
                         set(resource_set_tests[frozenset()]))
        for case in resourced_cases:
            resource = case.resources[0][1]
            self.assertEqual([case], resource_set_tests[frozenset([resource])])


class TestCostOfSwitching(testtools.TestCase):
    """Tests for cost_of_switching."""

    def setUp(self):
        super(TestCostOfSwitching, self).setUp()
        self.suite = testresources.OptimisingTestSuite()

    def makeResource(self, setUpCost=1, tearDownCost=1):
        resource = testresources.TestResource()
        resource.setUpCost = setUpCost
        resource.tearDownCost = tearDownCost
        return resource

    def testNoResources(self):
        # The cost of switching from no resources to no resources is 0.
        self.assertEqual(0, self.suite.cost_of_switching(set(), set()))

    def testSameResources(self):
        # The cost of switching to the same set of resources is also 0.
        a = self.makeResource()
        b = self.makeResource()
        self.assertEqual(0, self.suite.cost_of_switching(set([a]), set([a])))
        self.assertEqual(
            0, self.suite.cost_of_switching(set([a, b]), set([a, b])))

    # XXX: The next few tests demonstrate the current behaviour of the system.
    # We'll change them later.

    def testNewResources(self):
        a = self.makeResource()
        b = self.makeResource()
        self.assertEqual(1, self.suite.cost_of_switching(set(), set([a])))
        self.assertEqual(
            1, self.suite.cost_of_switching(set([a]), set([a, b])))
        self.assertEqual(2, self.suite.cost_of_switching(set(), set([a, b])))

    def testOldResources(self):
        a = self.makeResource()
        b = self.makeResource()
        self.assertEqual(1, self.suite.cost_of_switching(set([a]), set()))
        self.assertEqual(
            1, self.suite.cost_of_switching(set([a, b]), set([a])))
        self.assertEqual(2, self.suite.cost_of_switching(set([a, b]), set()))

    def testCombo(self):
        a = self.makeResource()
        b = self.makeResource()
        c = self.makeResource()
        self.assertEqual(2, self.suite.cost_of_switching(set([a]), set([b])))
        self.assertEqual(
            2, self.suite.cost_of_switching(set([a, c]), set([b, c])))


class TestCostGraph(testtools.TestCase):
    """Tests for calculating the cost graph of resourced test cases."""

    def makeResource(self, setUpCost=1, tearDownCost=1):
        resource = testresources.TestResource()
        resource.setUpCost = setUpCost
        resource.tearDownCost = tearDownCost
        return resource

    def testEmptyGraph(self):
        suite = testresources.OptimisingTestSuite()
        graph = suite._getGraph([])
        self.assertEqual({}, graph)

    def testSingletonGraph(self):
        resource = self.makeResource()
        suite = testresources.OptimisingTestSuite()
        graph = suite._getGraph([frozenset()])
        self.assertEqual({frozenset(): {}}, graph)

    def testTwoCasesInGraph(self):
        res1 = self.makeResource()
        res2 = self.makeResource()

        set1 = frozenset([res1, res2])
        set2 = frozenset([res2])
        no_resources = frozenset()

        suite = testresources.OptimisingTestSuite()
        graph = suite._getGraph([no_resources, set1, set2])
        self.assertEqual({no_resources: {set1: 2, set2: 1},
                          set1: {no_resources: 2, set2: 1},
                          set2: {no_resources: 1, set1: 1 }}, graph)


class TestGraphStuff(testtools.TestCase):

    def setUp(self):
        super(TestGraphStuff, self).setUp()
        class MockTest(unittest.TestCase):
            def __repr__(self):
                """The representation is the tests name.

                This makes it easier to debug sorting failures.
                """
                return self.id().split('.')[-1]
            def test_one(self):
                pass
            def test_two(self):
                pass
            def test_three(self):
                pass
            def test_four(self):
                pass

        self.case1 = MockTest("test_one")
        self.case2 = MockTest("test_two")
        self.case3 = MockTest("test_three")
        self.case4 = MockTest("test_four")
        self.cases = []
        self.cases.append(self.case1)
        self.cases.append(self.case2)
        self.cases.append(self.case3)
        self.cases.append(self.case4)

    def sortTests(self, tests):
        suite = testresources.OptimisingTestSuite()
        suite.addTests(tests)
        suite.sortTests()
        return suite._tests

    def _permute_four(self, cases):
        case1, case2, case3, case4 = cases
        permutations = []
        permutations.append([case1, case2, case3, case4])
        permutations.append([case1, case2, case4, case3])
        permutations.append([case1, case3, case2, case4])
        permutations.append([case1, case3, case4, case2])
        permutations.append([case1, case4, case2, case3])
        permutations.append([case1, case4, case3, case2])

        permutations.append([case2, case1, case3, case4])
        permutations.append([case2, case1, case4, case3])
        permutations.append([case2, case3, case1, case4])
        permutations.append([case2, case3, case4, case1])
        permutations.append([case2, case4, case1, case3])
        permutations.append([case2, case4, case3, case1])

        permutations.append([case3, case2, case1, case4])
        permutations.append([case3, case2, case4, case1])
        permutations.append([case3, case1, case2, case4])
        permutations.append([case3, case1, case4, case2])
        permutations.append([case3, case4, case2, case1])
        permutations.append([case3, case4, case1, case2])

        permutations.append([case4, case2, case3, case1])
        permutations.append([case4, case2, case1, case3])
        permutations.append([case4, case3, case2, case1])
        permutations.append([case4, case3, case1, case2])
        permutations.append([case4, case1, case2, case3])
        permutations.append([case4, case1, case3, case2])
        return permutations

    def testBasicSortTests(self):
        # Test every permutation of inputs, with legacy tests.
        # Cannot use equal costs because of the use of
        # a 2*optimal heuristic for sorting: with equal
        # costs the wrong sort order is < twice the optimal
        # weight, and thus can be selected.
        resource_one = testresources.TestResource()
        resource_two = testresources.TestResource()
        resource_two.setUpCost = 5
        resource_two.tearDownCost = 5
        resource_three = testresources.TestResource()

        self.case1.resources = [
            ("_one", resource_one), ("_two", resource_two)]
        self.case2.resources = [
            ("_two", resource_two), ("_three", resource_three)]
        self.case3.resources = [("_three", resource_three)]
        # acceptable sorted orders are:
        # 1, 2, 3, 4
        # 3, 2, 1, 4

        for permutation in self._permute_four(self.cases):
            self.assertIn(
                self.sortTests(permutation), [
                    [self.case1, self.case2, self.case3, self.case4],
                    [self.case3, self.case2, self.case1, self.case4]],
                "failed with permutation %s" % (permutation,))

    def testGlobalMinimum(self):
        # When a local minimum leads to a global non-minum, the global
        # non-minimum is still reached. We construct this by having a resource
        # that appears very cheap (it has a low setup cost) but is very
        # expensive to tear down. Then we have it be used twice: the global
        # minimum depends on only tearing it down once. To prevent it 
        # accidentally being chosen twice, we make one use of it be
        # on its own, and another with a resource to boost its cost,
        # finally we put a resource which is more expensive to setup
        # than the expensive teardown is to teardown, but less expensive
        # than it + the small booster to setup.
        # valid results are - the expensive setup, then both expensive
        # teardowns, and the legacy fourth, or
        # both expensive teardowns and then the expensive setup (and the legacy
        # fourth)
        # case1 has expensive setup (one)
        # case2 has expensive teardown (two)
        # case3 has expensive teardown + boost (three)
        resource_one = testresources.TestResource()
        resource_one.setUpCost = 20
        resource_two = testresources.TestResource()
        resource_two.tearDownCost = 50
        resource_three = testresources.TestResource()
        resource_three.setUpCost = 72
        # node costs:
        #  ->1 = r1.up                    = 20
        #  ->2 = r2.up                    = 1
        #  ->3 = r2.up + r3.up            = 122
        # 1->2 = r1.down + r2.up          = 2
        # 1->3 = r1.down + r2.up + r3.up  = 93
        # 2->1 = r2.down + r1.up          = 70
        # 2->3 = r3.up                    = 72
        # 3->1 = r1.up + r2.down + r3.down= 71
        # 3->2 = r3.down                  = 1
        # 1->  = r1.down                  = 1
        # 2->  = r2.down                  = 50
        # 3->  = r3.down + r3.down        = 51
        # naive path = 2, 1, 3 = 1 + 70 + 93 + 51 = 215
        # better    = 2, 3, 1 = 1 + 72 + 71 + 1 = 145
        acceptable_orders = [
            [self.case1, self.case2, self.case3, self.case4],
            [self.case1, self.case3, self.case2, self.case4],
            [self.case2, self.case3, self.case1, self.case4],
            [self.case3, self.case2, self.case1, self.case4],
            ]

        self.case1.resources = [
            ("_one", resource_one)]
        self.case2.resources = [
            ("_two", resource_two)]
        self.case3.resources = [("_two", resource_two),
            ("_three", resource_three)]
        for permutation in self._permute_four(self.cases):
            self.assertIn(self.sortTests(permutation), acceptable_orders)

    def testSortIsStableWithinGroups(self):
        """Tests with the same resources maintain their relative order."""
        resource_one = testresources.TestResource()
        resource_two = testresources.TestResource()

        self.case1.resources = [("_one", resource_one)]
        self.case2.resources = [("_one", resource_one)]
        self.case3.resources = [("_one", resource_one), ("_two", resource_two)]
        self.case4.resources = [("_one", resource_one), ("_two", resource_two)]

        for permutation in self._permute_four(self.cases):
            sorted = self.sortTests(permutation)
            self.assertEqual(
                permutation.index(self.case1) < permutation.index(self.case2),
                sorted.index(self.case1) < sorted.index(self.case2))
            self.assertEqual(
                permutation.index(self.case3) < permutation.index(self.case4),
                sorted.index(self.case3) < sorted.index(self.case4))

    def testSortingTwelveIndependentIsFast(self):
        # Given twelve independent resource sets, my patience is not exhausted.
        managers = []
        for pos in range(12):
            managers.append(testresources.TestResourceManager())
        # Add more sample tests
        cases = [self.case1, self.case2, self.case3, self.case4]
        for pos in range(5,13):
            cases.append(
                testtools.clone_test_with_new_id(cases[0], 'case%d' % pos))
        # We care that this is fast in this test, so we don't need to have
        # overlapping resource usage
        for case, manager in zip(cases, managers):
            case.resources = [('_resource', manager)]
        # Any sort is ok, as long as its the right length :)
        result = self.sortTests(cases)
        self.assertEqual(12, len(result))

    def testSortingTwelveOverlappingIsFast(self):
        # Given twelve connected resource sets, my patience is not exhausted.
        managers = []
        for pos in range(12):
            managers.append(testresources.TestResourceManager())
        # Add more sample tests
        cases = [self.case1, self.case2, self.case3, self.case4]
        for pos in range(5,13):
            cases.append(
                testtools.clone_test_with_new_id(cases[0], 'case%d' % pos))
        tempdir = testresources.TestResourceManager()
        # give all tests a tempdir, enough to provoke a single partition in
        # the current code.
        for case, manager in zip(cases, managers):
            case.resources = [('_resource', manager), ('tempdir', tempdir)]
        # Any sort is ok, as long as its the right length :)
        result = self.sortTests(cases)
        self.assertEqual(12, len(result))

    def testSortConsidersDependencies(self):
        """Tests with different dependencies are sorted together."""
        # We test this by having two resources (one and two) that share a very
        # expensive dependency (dep). So one and two have to sort together. By
        # using a cheap resource directly from several tests we can force the
        # optimise to choose between keeping the cheap resource together or
        # keeping the expensive dependency together.
        # Test1, res_one, res_common_one
        # Test2, res_two, res_common_two
        # Test3, res_common_one, res_common_two
        # In a dependency naive sort, we will have test3 between test1 and
        # test2 always. In a dependency aware sort, test1 and two will
        # always group.
         
        resource_one = testresources.TestResource()
        resource_two = testresources.TestResource()
        resource_one_common = testresources.TestResource()
        # make it cheaper to keep a _common resource than to switch both
        # resources (when dependencies are ignored)
        resource_one_common.setUpCost = 2
        resource_one_common.tearDownCost = 2
        resource_two_common = testresources.TestResource()
        resource_two_common.setUpCost = 2
        resource_two_common.tearDownCost = 2
        dep = testresources.TestResource()
        dep.setUpCost = 20
        dep.tearDownCost = 20
        resource_one.resources.append(("dep1", dep))
        resource_two.resources.append(("dep2", dep))

        self.case1.resources = [("withdep", resource_one), ("common", resource_one_common)]
        self.case2.resources = [("withdep", resource_two), ("common", resource_two_common)]
        self.case3.resources = [("_one", resource_one_common), ("_two", resource_two_common)]
        self.case4.resources = []

        acceptable_orders = [
            [self.case1, self.case2, self.case3, self.case4],
            [self.case2, self.case1, self.case3, self.case4],
            [self.case3, self.case1, self.case2, self.case4],
            [self.case3, self.case2, self.case1, self.case4],
            ]

        for permutation in self._permute_four(self.cases):
            self.assertIn(self.sortTests(permutation), acceptable_orders)
