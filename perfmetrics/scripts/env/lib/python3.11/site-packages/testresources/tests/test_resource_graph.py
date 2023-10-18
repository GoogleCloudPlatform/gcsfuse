#
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

"""Test _resource_graph(resource_sets)."""

import testtools
import testresources
from testresources import split_by_resources, _resource_graph
from testresources.tests import ResultWithResourceExtensions
import unittest


def test_suite():
    from testresources.tests import TestUtil
    loader = TestUtil.TestLoader()
    result = loader.loadTestsFromName(__name__)
    return result


class TestResourceGraph(testtools.TestCase):

    def test_empty(self):
        no_resources = frozenset()
        resource_sets = [no_resources]
        self.assertEqual({no_resources:set([])}, _resource_graph(resource_sets))

    def test_discrete(self):
        resset1 = frozenset([testresources.TestResourceManager()])
        resset2 = frozenset([testresources.TestResourceManager()])
        resource_sets = [resset1, resset2]
        result = _resource_graph(resource_sets)
        self.assertEqual({resset1:set([]), resset2:set([])}, result)

    def test_overlapping(self):
        res1 = testresources.TestResourceManager()
        res2 = testresources.TestResourceManager()
        resset1 = frozenset([res1])
        resset2 = frozenset([res2])
        resset3 = frozenset([res1, res2])
        resource_sets = [resset1, resset2, resset3]
        result = _resource_graph(resource_sets)
        self.assertEqual(
            {resset1:set([resset3]),
             resset2:set([resset3]),
             resset3:set([resset1, resset2])},
            result)


class TestDigraphToGraph(testtools.TestCase):

    def test_wikipedia_example(self):
        """Converting a digraph mirrors it in the XZ axis (matrix view).

        See http://en.wikipedia.org/wiki/Travelling_salesman_problem \
        #Solving_by_conversion_to_Symmetric_TSP
        """
        #       A   B   C
        #   A       1   2
        #   B   6       3
        #   C   5   4   
        A = "A"
        Ap = "A'"
        B = "B"
        Bp = "B'"
        C = "C"
        Cp = "C'"
        digraph = {A:{     B:1, C:2},
                   B:{A:6,      C:3},
                   C:{A:5, B:4     }}
        # and the output
        #       A   B   C   A'  B'  C'
        #   A               0   6   5
        #   B               1   0   4
        #   C               2   3   0
        #   A'  0   1   2           
        #   B'  6   0   3           
        #   C'  5   4   0
        expected = {
            A :{              Ap:0, Bp:6, Cp:5},
            B :{              Ap:1, Bp:0, Cp:4},
            C :{              Ap:2, Bp:3, Cp:0},
            Ap:{A:0, B:1, C:2                 },
            Bp:{A:6, B:0, C:3                 },
            Cp:{A:5, B:4, C:0                 }}
        self.assertEqual(expected,
            testresources._digraph_to_graph(digraph, {A:Ap, B:Bp, C:Cp}))


class TestKruskalsMST(testtools.TestCase):

    def test_wikipedia_example(self):
        """Performing KruskalsMST on a graph returns a spanning tree.

        See http://en.wikipedia.org/wiki/Kruskal%27s_algorithm.
        """
        A = "A"
        B = "B"
        C = "C"
        D = "D"
        E = "E"
        F = "F"
        G = "G"
        graph = {
            A:{     B:7,      D:5},
            B:{A:7,      C:8, D:9,  E:7},
            C:{     B:8,            E:5},
            D:{A:5, B:9,            E:15, F:6},
            E:{     B:7, C:5, D:15,       F:8, G:9},
            F:{               D:6,  E:8,       G:11},
            G:{                     E:9, F:11}}
        expected = {
            A:{     B:7,      D:5},
            B:{A:7,                 E:7},
            C:{                     E:5},
            D:{A:5,                      F:6},
            E:{     B:7, C:5,                  G:9},
            F:{               D:6},
            G:{                     E:9}}
        result = testresources._kruskals_graph_MST(graph)
        e_weight = sum(sum(row.values()) for row in expected.values())
        r_weight = sum(sum(row.values()) for row in result.values())
        self.assertEqual(e_weight, r_weight)
        self.assertEqual(expected,
            testresources._kruskals_graph_MST(graph))
