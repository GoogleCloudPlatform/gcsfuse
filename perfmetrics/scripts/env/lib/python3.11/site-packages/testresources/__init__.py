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
#  Unless required by applicable law or agreed to in writing, software
#  distributed under these licenses is distributed on an "AS IS" BASIS, WITHOUT
#  WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
#  license you chose for the specific language governing permissions and
#  limitations under that license.
#

"""TestResources: declarative management of external resources for tests."""

import heapq
import inspect
import unittest
try:
    import unittest2
except ImportError:
    unittest2 = None

# same format as sys.version_info: "A tuple containing the five components of
# the version number: major, minor, micro, releaselevel, and serial. All
# values except releaselevel are integers; the release level is 'alpha',
# 'beta', 'candidate', or 'final'. The version_info value corresponding to the
# Python version 2.0 is (2, 0, 0, 'final', 0)."  Additionally we use a
# releaselevel of 'dev' for unreleased under-development code.
#
# If the releaselevel is 'alpha' then the major/minor/micro components are not
# established at this point, and setup.py will use a version of next-$(revno).
# If the releaselevel is 'final', then the tarball will be major.minor.micro.
# Otherwise it is major.minor.micro~$(revno).

from pbr.version import VersionInfo
_version = VersionInfo('testresources')
__version__ = _version.semantic_version().version_tuple()
version = _version.release_string()


def test_suite():
    import testresources.tests
    return testresources.tests.test_suite()


def _digraph_to_graph(digraph, prime_node_mapping):
    """Convert digraph to a graph.

    :param digraph: A directed graph in the form
        {from:{to:value}}.
    :param prime_node_mapping: A mapping from every
        node in digraph to a new unique and not in digraph node.
    :return: A symmetric graph in the form {from:to:value}} created by
        creating edges in the result between every N to M-prime with the
        original N-M value and from every N to N-prime with a cost of 0.
        No other edges are created.
    """
    result = {}
    for from_node, from_prime_node in prime_node_mapping.items():
        result[from_node] = {from_prime_node: 0}
        result[from_prime_node] = {from_node: 0}
    for from_node, to_nodes in digraph.items():
        from_prime = prime_node_mapping[from_node]
        for to_node, value in to_nodes.items():
            to_prime = prime_node_mapping[to_node]
            result[from_prime][to_node] = value
            result[to_node][from_prime] = value
    return result


def _kruskals_graph_MST(graph):
    """Find the minimal spanning tree in graph using Kruskals algorithm.

    See http://en.wikipedia.org/wiki/Kruskal%27s_algorithm.
    :param graph: A graph in {from:{to:value}} form. Every node present in
        graph must be in the outer dict (because graph is not a directed graph.
    :return: A graph with all nodes and those vertices that are part of the MST
        for graph. If graph is not connected, then the result will also be a
        forest.
    """
    # forest contains all the nodes -> graph that node is in.
    forest = {}
    # graphs is the count of graphs we have yet to combine.
    for node in graph:
        forest[node] = {node: {}}
    graphs = len(forest)
    # collect edges: every edge is present twice (due to the graph
    # representation), so normalise.
    edges = set()
    for from_node, to_nodes in graph.items():
        for to_node, value in to_nodes.items():
            edge = (value,) + tuple(sorted([from_node, to_node]))
            edges.add(edge)
    edges = list(edges)
    heapq.heapify(edges)
    while edges and graphs > 1:
        # more edges to go and we haven't gotten a spanning tree yet.
        edge = heapq.heappop(edges)
        g1 = forest[edge[1]]
        g2 = forest[edge[2]]
        if g1 is g2:
            continue  # already joined
        # combine g1 and g2 into g1
        graphs -= 1
        for from_node, to_nodes in g2.items():
            #remember its symmetric, don't need to do 'to'.
            forest[from_node] = g1
            g1.setdefault(from_node, {}).update(to_nodes)
        # add edge
        g1[edge[1]][edge[2]] = edge[0]
        g1[edge[2]][edge[1]] = edge[0]
    # union the remaining graphs
    _, result = forest.popitem()
    for _, g2 in forest.items():
        if g2 is result:  # common case
            continue
        for from_node, to_nodes in g2.items():
            result.setdefault(from_node, {}).update(to_nodes)
    return result


def _resource_graph(resource_sets):
    """Convert an iterable of resource_sets into a graph.

    Each resource_set in the iterable is treated as a node, and each resource
    in that resource_set is used as an edge to other nodes.
    """
    nodes = {}
    edges = {}
    for resource_set in resource_sets:
        # put node in nodes
        node = frozenset(resource_set)
        nodes[node] = set()
        # put its contents in as edges
        for resource in resource_set:
            edges.setdefault(resource, []).append(node)
    # populate the adjacent members of nodes
    for node, connected in nodes.items():
        for resource in node:
            connected.update(edges[resource])
        connected.discard(node)
    return nodes


def split_by_resources(tests):
    """Split a list of tests by the resources that the tests use.

    :return: a dictionary mapping sets of resources to lists of tests
    using that combination of resources.  The dictionary always
    contains an entry for "no resources".
    """
    no_resources = frozenset()
    resource_set_tests = {no_resources: []}
    for test in tests:
        resources = getattr(test, "resources", ())
        all_resources = list(resource.neededResources()
                             for _, resource in resources)
        resource_set = set()
        for resource_list in all_resources:
            resource_set.update(resource_list)
        resource_set_tests.setdefault(frozenset(resource_set), []).append(test)
    return resource_set_tests


def _strongly_connected_components(graph, no_resources):
    """Find the strongly connected components in graph.

    This is essentially a nonrecursive flatterning of Tarjan's method. It
    may be worth profiling against an actual Tarjan's implementation at some
    point, but sets are often faster than python calls.

    graph gets consumed, but that could be changed easily enough.
    """
    partitions = []
    while graph:
        node, pending = graph.popitem()
        current_partition = set([node])
        while pending:
            # add all the nodes connected to a connected node to pending.
            node = pending.pop()
            current_partition.add(node)
            pending.update(graph.pop(node, []))
            # don't try to process things we've allready processed.
            pending.difference_update(current_partition)
        partitions.append(current_partition)
    return partitions


class OptimisingTestSuite(unittest.TestSuite):
    """A resource creation optimising TestSuite."""

    known_suite_classes = None

    def adsorbSuite(self, test_case_or_suite):
        """Deprecated. Use addTest instead."""
        self.addTest(test_case_or_suite)

    def addTest(self, test_case_or_suite):
        """Add `test_case_or_suite`, unwrapping standard TestSuites.

        This means that any containing unittest.TestSuites will be removed,
        while any custom test suites will be 'distributed' across their
        members. Thus addTest(CustomSuite([a, b])) will result in
        CustomSuite([a]) and CustomSuite([b]) being added to this suite.
        """
        try:
            tests = iter(test_case_or_suite)
        except TypeError:
            unittest.TestSuite.addTest(self, test_case_or_suite)
            return
        if test_case_or_suite.__class__ in self.__class__.known_suite_classes:
            for test in tests:
                self.adsorbSuite(test)
        else:
            for test in tests:
                unittest.TestSuite.addTest(
                    self, test_case_or_suite.__class__([test]))

    def cost_of_switching(self, old_resource_set, new_resource_set):
        """Cost of switching from 'old_resource_set' to 'new_resource_set'.

        This is calculated by adding the cost of tearing down unnecessary
        resources to the cost of setting up the newly-needed resources.

        Note that resources which are always dirtied may skew the predicted
        skew the cost of switching because they are considered common, even
        when reusing them may actually be equivalent to a teardown+setup
        operation.
        """
        new_resources = new_resource_set - old_resource_set
        gone_resources = old_resource_set - new_resource_set
        return (sum(resource.setUpCost for resource in new_resources) +
            sum(resource.tearDownCost for resource in gone_resources))

    def switch(self, old_resource_set, new_resource_set, result):
        """Switch from 'old_resource_set' to 'new_resource_set'.

        Tear down resources in old_resource_set that aren't in
        new_resource_set and set up resources that are in new_resource_set but
        not in old_resource_set.

        :param result: TestResult object to report activity on.
        """
        new_resources = new_resource_set - old_resource_set
        old_resources = old_resource_set - new_resource_set
        for resource in old_resources:
            resource.finishedWith(resource._currentResource, result)
        for resource in new_resources:
            resource.getResource(result)

    def run(self, result):
        self.sortTests()
        current_resources = set()
        for test in self._tests:
            if result.shouldStop:
                break
            resources = getattr(test, 'resources', [])
            new_resources = set()
            for name, resource in resources:
                new_resources.update(resource.neededResources())
            self.switch(current_resources, new_resources, result)
            current_resources = new_resources
            test(result)
        self.switch(current_resources, set(), result)
        return result

    def sortTests(self):
        """Attempt to topographically sort the contained tests.

        This function biases to reusing a resource: it assumes that resetting
        a resource is usually cheaper than a teardown + setup; and that most
        resources are not dirtied by most tests.

        Feel free to override to improve the sort behaviour.
        """
        # We group the tests by the resource combinations they use,
        # since there will usually be fewer resource combinations than
        # actual tests and there can never be more: This gives us 'nodes' or
        # 'resource_sets' that represent many tests using the same set of
        # resources.
        resource_set_tests = split_by_resources(self._tests)
        # Partition into separate sets of resources, there is no ordering
        # preference between sets that do not share members. Rationale:
        # If resource_set A and B have no common resources, AB and BA are
        # equally good - the global setup/teardown sums are identical. Secondly
        # if A shares one or more resources with C, then pairing AC|CA is
        # better than having B between A and C, because the shared resources
        # can be reset or reused. Having partitioned we can use connected graph
        # logic on each partition.
        resource_set_graph = _resource_graph(resource_set_tests)
        no_resources = frozenset()
        # A list of resource_set_tests, all fully internally connected.
        partitions = _strongly_connected_components(resource_set_graph,
            no_resources)
        result = []
        for partition in partitions:
            # we process these at the end for no particularly good reason (it
            # makes testing slightly easier).
            if partition == [no_resources]:
                continue
            order = self._makeOrder(partition)
            # Spit this partition out into result
            for resource_set in order:
                result.extend(resource_set_tests[resource_set])
        result.extend(resource_set_tests[no_resources])
        self._tests = result

    def _getGraph(self, resource_sets):
        """Build a graph of the resource-using nodes.

        This special cases set(['root']) to be a node with no resources and
        edges to everything.

        :return: A complete directed graph of the switching costs
            between each resource combination. Note that links from N to N are
            not included.
        """
        no_resources = frozenset()
        graph = {}
        root = set(['root'])
        # bottom = set(['bottom'])
        for from_set in resource_sets:
            graph[from_set] = {}
            if from_set == root:
                from_resources = no_resources
            #elif from_set == bottom:
            #    continue # no links from bottom
            else:
                from_resources = from_set
            for to_set in resource_sets:
                if from_set is to_set:
                    continue  # no self-edges
                #if to_set == bottom:
                #   if from_set == root:
                #       continue # no short cuts!
                #   to_resources = no_resources
                #el
                if to_set == root:
                    continue  # no links to root
                else:
                    to_resources = to_set
                graph[from_set][to_set] = self.cost_of_switching(
                        from_resources, to_resources)
        return graph

    def _makeOrder(self, partition):
        """Return a order for the resource sets in partition."""
        # This problem is NP-C - find the lowest cost hamiltonian path. It
        # also meets the triangle inequality, so we can use an approximation.
        # TODO: implement Christofides.
        # See:
        #   http://en.wikipedia.org/wiki/Travelling_salesman_problem#Metric_TSP

        # We need a root
        root = frozenset(['root'])
        partition.add(root)
        # and an end
        # partition.add(frozenset(['bottom']))
        # get rid of 'noresources'
        partition.discard(frozenset())
        digraph = self._getGraph(partition)
        # build a prime map
        primes = {}
        prime = frozenset(['prime'])
        for node in digraph:
            primes[node] = node.union(prime)
        graph = _digraph_to_graph(digraph, primes)
        mst = _kruskals_graph_MST(graph)
        # Because the representation is a digraph, we can build an Eulerian
        # cycle directly from the representation by just following the links:
        # a node with only 1 'edge' has two directed edges; and we can only
        # enter and leave it once, so the edge lookups will match precisely.
        # As the mst is a spanning tree, the graph will become disconnected
        # (we choose non-disconnecting edges first)
        #  - for a stub node (1 outgoing link): when exiting it unless it is
        #    the first node started at
        # - for a non-stub node if choosing an outgoing link where some other
        #   endpoints incoming link has not been traversed. [exit by a
        #   different node than entering, until all exits taken].
        # We don't need the mst after, so it gets modified in place.
        node = root
        cycle = [node]
        steps = 2 * (len(mst) - 1)
        for step in range(steps):
            found = False
            outgoing = None  # For clearer debugging.
            for outgoing in mst[node]:
                if node in mst[outgoing]:
                    # we have a return path: take it
                    # print node, '->', outgoing, ' can return'
                    del mst[node][outgoing]
                    node = outgoing
                    cycle.append(node)
                    found = True
                    break
            if not found:
                # none of the outgoing links have an incoming, so follow an
                # arbitrary one (the last examined outgoing)
                # print node, '->', outgoing
                del mst[node][outgoing]
                node = outgoing
                cycle.append(node)
        # Convert to a path:
        visited = set()
        order = []
        for node in cycle:
            if node in visited:
                continue
            if node in primes:
                order.append(node)
            visited.add(node)
        assert order[0] == root
        return order[1:]


OptimisingTestSuite.known_suite_classes = (
    unittest.TestSuite, OptimisingTestSuite)
if unittest2 is not None:
    OptimisingTestSuite.known_suite_classes += (unittest2.TestSuite,)


class TestLoader(unittest.TestLoader):
    """Custom TestLoader to set the right TestSuite class."""
    suiteClass = OptimisingTestSuite


class TestResourceManager(object):
    """A manager for resources that can be shared across tests.

    ResourceManagers can report activity to a TestResult. The methods
     - startCleanResource(resource)
     - stopCleanResource(resource)
     - startMakeResource(resource)
     - stopMakeResource(resource)
    will be looked for and if present invoked before and after cleaning or
    creation of resource objects takes place.

    :cvar resources: The same as the resources list on an instance, the default
        constructor will look for the class instance and copy it. This is a
        convenience to avoid needing to define __init__ solely to alter the
        dependencies list.
    :ivar resources: The resources that this resource needs. Calling
        neededResources will return the closure of this resource and its needed
        resources. The resources list is in the same format as resources on a
        test case - a list of tuples (attribute_name, resource).
    :ivar setUpCost: The relative cost to construct a resource of this type.
         One good approach is to set this to the number of seconds it normally
         takes to set up the resource.
    :ivar tearDownCost: The relative cost to tear down a resource of this
         type. One good approach is to set this to the number of seconds it
         normally takes to tear down the resource.
    """

    setUpCost = 1
    tearDownCost = 1

    def __init__(self):
        """Create a TestResourceManager object."""
        self._dirty = False
        self._uses = 0
        self._currentResource = None
        self.resources = list(getattr(self.__class__, "resources", []))

    def _call_result_method_if_exists(self, result, methodname, *args):
        """Call a method on a TestResult that may exist."""
        method = getattr(result, methodname, None)
        if callable(method):
            method(*args)

    def _clean_all(self, resource, result):
        """Clean the dependencies from resource, and then resource itself."""
        self._call_result_method_if_exists(result, "startCleanResource", self)
        self.clean(resource)
        for name, manager in self.resources:
            manager.finishedWith(getattr(resource, name))
        self._call_result_method_if_exists(result, "stopCleanResource", self)

    def clean(self, resource):
        """Override this to class method to hook into resource removal."""

    def dirtied(self, resource):
        """Mark the resource as having been 'dirtied'.

        A resource is dirty when it is no longer suitable for use by other
        tests.

        e.g. a shared database that has had rows changed.
        """
        self._dirty = True

    def finishedWith(self, resource, result=None):
        """Indicate that 'resource' has one less user.

        If there are no more registered users of 'resource' then we trigger
        the `clean` hook, which should do any resource-specific
        cleanup.

        :param resource: A resource returned by
            `TestResourceManager.getResource`.
        :param result: An optional TestResult to report resource changes to.
        """
        self._uses -= 1
        if self._uses == 0:
            self._clean_all(resource, result)
            self._setResource(None)

    def getResource(self, result=None):
        """Get the resource for this class and record that it's being used.

        The resource is constructed using the `make` hook.

        Once done with the resource, pass it to `finishedWith` to indicated
        that it is no longer needed.
        :param result: An optional TestResult to report resource changes to.
        """
        if self._uses == 0:
            self._setResource(self._make_all(result))
        elif self.isDirty():
            self._setResource(self.reset(self._currentResource, result))
        self._uses += 1
        return self._currentResource

    def isDirty(self):
        """Return True if this managers cached resource is dirty.

        Calling when the resource is not currently held has undefined
        behaviour.
        """
        if self._dirty:
            return True
        for name, mgr in self.resources:
            if mgr.isDirty():
                return True
            res = mgr.getResource()
            try:
                if res is not getattr(self._currentResource, name):
                    return True
            finally:
                mgr.finishedWith(res)

    def _make_all(self, result):
        """Make the dependencies of this resource and this resource."""
        self._call_result_method_if_exists(result, "startMakeResource", self)
        dependency_resources = {}
        for name, resource in self.resources:
            dependency_resources[name] = resource.getResource()
        resource = self.make(dependency_resources)
        for name, value in dependency_resources.items():
            setattr(resource, name, value)
        self._call_result_method_if_exists(result, "stopMakeResource", self)
        return resource

    def make(self, dependency_resources):
        """Override this to construct resources.

        :param dependency_resources: A dict mapping name -> resource instance
            for the resources specified as dependencies.
        :return: The made resource.
        """
        raise NotImplementedError(
            "Override make to construct resources.")

    def neededResources(self):
        """Return the resources needed for this resource, including self.

        :return: A list of needed resources, in topological deepest-first
            order.
        """
        seen = set([self])
        result = []
        for name, resource in self.resources:
            for resource in resource.neededResources():
                if resource in seen:
                    continue
                seen.add(resource)
                result.append(resource)
        result.append(self)
        return result

    def reset(self, old_resource, result=None):
        """Return a clean version of old_resource.

        By default, the resource will be cleaned then remade if it had
        previously been `dirtied` by the helper self._reset() - which is the
        extension point folk should override to customise reset behaviour.

        This function takes the dependent resource stack into consideration as
        _make_all and _clean_all do. The inconsistent naming is because reset
        is part of the public interface, but _make_all and _clean_all is not.

        Note that if a resource A holds a lock or other blocking thing on
        a dependency D, reset will result in this call sequence over a 
        getResource(), dirty(), getResource(), finishedWith(), finishedWith()
        sequence:
        B.make(), A.make(), B.reset(), A.reset(), A.clean(), B.clean()
        Thus it is important that B.reset not assume that A has been cleaned or
        reset before B is reset: it should arrange to reference count, lazy
        cleanup or forcibly reset resource in some fashion.

        As an example, consider that B is a database with sample data, and
        A is an application server serving content from it. B._reset() should
        disconnect all database clients, reset the state of the database, and
        A._reset() should tell the application server to dump any internal
        caches it might have.

        In principle we might make a richer API to allow before-and-after
        reset actions, but so far that hasn't been needed.

        :return: The possibly new resource.
        :param result: An optional TestResult to report resource changes to.
        """
        # Core logic:
        #  - if neither we nor any parent is dirty, do nothing.
        # otherwise
        #  - emit a signal to the test result
        #  - reset all dependencies all, getting new attributes.
        #  - call self._reset(old_resource, dependency_attributes)
        #    [the default implementation does a clean + make]
        if not self.isDirty():
            return old_resource
        self._call_result_method_if_exists(result, "startResetResource", self)
        dependency_resources = {}
        for name, mgr in self.resources:
            dependency_resources[name] = mgr.reset(
                getattr(old_resource, name), result)
        resource = self._reset(old_resource, dependency_resources)
        for name, value in dependency_resources.items():
            setattr(resource, name, value)
        self._call_result_method_if_exists(result, "stopResetResource", self)
        return resource

    def _reset(self, resource, dependency_resources):
        """Override this to reset resources other than via clean+make.

        This method should reset the self._dirty flag (assuming the manager can
        ever be clean) and return either the old resource cleaned or a fresh
        one.

        :param resource: The resource to reset.
        :param dependency_resources: A dict mapping name -> resource instance
            for the resources specified as dependencies.
        """
        self.clean(resource)
        return self.make(dependency_resources)

    def _setResource(self, new_resource):
        """Set the current resource to a new value."""
        self._currentResource = new_resource
        self._dirty = False
TestResource = TestResourceManager


class GenericResource(TestResourceManager):
    """A TestResourceManager that decorates an external helper of some kind.

    GenericResource can be used to adapt an external resource so that
    testresources can use it. By default the setUp and tearDown methods are
    called when making and cleaning the resource, and the resource is
    considered permanently dirty, so it is torn down and brought up again
    between every use.

    The constructor method is called with the dependency resources dict::
        resource_factory(**dependency_resources)
    This permits naming those resources to match the contract of the setUp
    method.
    """

    def __init__(self, resource_factory, setup_method_name='setUp',
        teardown_method_name='tearDown'):
        """Create a GenericResource

        :param resource_factory: A factory to create a new resource.
        :param setup_method_name: Optional method name to call to setup the
            resource. Defaults to 'setUp'.
        :param teardown_method_name: Optional method name to call to tear down
            the resource. Defaults to 'tearDown'.
        """
        super(GenericResource, self).__init__()
        self.resource_factory = resource_factory
        self.setup_method_name = setup_method_name
        self.teardown_method_name = teardown_method_name

    def clean(self, resource):
        getattr(resource, self.teardown_method_name)()

    def make(self, dependency_resources):
        result = self.resource_factory(**dependency_resources)
        getattr(result, self.setup_method_name)()
        return result

    def isDirty(self):
        return True


class FixtureResource(TestResourceManager):
    """A TestResourceManager that decorates a ``fixtures.Fixture``.

    The fixture has its setUp and cleanUp called as expected, and
    reset is called between uses.

    Due to the API of fixtures, dependency_resources are not
    accessible to the wrapped fixture. However, if you are using
    resource optimisation, you should wrap any dependencies in a
    FixtureResource and set the resources attribute appropriately.
    Note that when this is done, testresources will take care of
    calling setUp and cleanUp on the dependency fixtures and so
    the fixtures should not implicitly setUp or cleanUp their
    dependencies (that have been mapped).

    See the ``fixtures`` documentation for information on managing
    dependencies within the ``fixtures`` API.

    :ivar fixture: The wrapped fixture.
    """

    def __init__(self, fixture):
        """Create a FixtureResource

        :param fixture: The fixture to wrap.
        """
        super(FixtureResource, self).__init__()
        self.fixture = fixture

    def clean(self, resource):
        resource.cleanUp()

    def make(self, dependency_resources):
        self.fixture.setUp()
        return self.fixture

    def _reset(self, resource, dependency_resources):
        self.fixture.reset()
        return self.fixture

    def isDirty(self):
        return True

    _dirty = property(lambda _:True, lambda _, _1:None)


class ResourcedTestCase(unittest.TestCase):
    """A TestCase parent or utility that enables cross-test resource usage.

    ResourcedTestCase is a thin wrapper around the
    testresources.setUpResources and testresources.tearDownResources helper
    functions. It should be trivially reimplemented where a different base
    class is neded, or you can use multiple inheritance and call into
    ResourcedTestCase.setUpResources and ResourcedTestCase.tearDownResources
    from your setUp and tearDown (or whatever cleanup idiom is used).

    :ivar resources: A list of (name, resource) pairs, where 'resource' is a
        subclass of `TestResourceManager` and 'name' is the name of the
        attribute that the resource should be stored on.
    """

    resources = []

    def setUp(self):
        super(ResourcedTestCase, self).setUp()
        self.setUpResources()

    def setUpResources(self):
        setUpResources(self, self.resources, _get_result())

    def tearDown(self):
        self.tearDownResources()
        super(ResourcedTestCase, self).tearDown()

    def tearDownResources(self):
        tearDownResources(self, self.resources, _get_result())


def setUpResources(test, resources, result):
    """Set up resources for test.

    :param test: The test to setup resources for.
    :param resources: The resources to setup.
    :param result: A result object for tracing resource activity.
    """
    for resource in resources:
        setattr(test, resource[0], resource[1].getResource(result))


def tearDownResources(test, resources, result):
    """Tear down resources for test.

    :param test: The test to tear down resources from.
    :param resources: The resources to tear down.
    :param result: A result object for tracing resource activity.
    """
    for resource in resources:
        resource[1].finishedWith(getattr(test, resource[0]), result)
        delattr(test, resource[0])


def _get_result():
    """Find a TestResult in the stack.

    unittest hides the result. This forces us to look up the stack.
    The result is passed to a run() or a __call__ method 4 or more frames
    up: that method is what calls setUp and tearDown, and they call their
    parent setUp etc. Its not guaranteed that the parameter to run will
    be calls result as its not required to be a keyword parameter in
    TestCase. However, in practice, this works.
    """
    stack = inspect.stack()
    for frame in stack[2:]:
        if frame[3] in ('run', '__call__'):
            # Not all frames called 'run' will be unittest. It could be a
            # reactor in trial, for instance.
            result = frame[0].f_locals.get('result')
            if (result is not None and
                getattr(result, 'startTest', None) is not None):
                return result
