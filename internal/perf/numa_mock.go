// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package perf

import "github.com/lrita/numa"

var numaLib NumaLibrary = &realNumaLibrary{}

type NumaLibrary interface {
	Available() bool
	NodeMask() numa.Bitmask
	RunOnNode(node int) error
	GetCPUAndNode() (cpu int, node int)
}

type realNumaLibrary struct{}

func (l *realNumaLibrary) Available() bool {
	return numa.Available()
}

func (l *realNumaLibrary) NodeMask() numa.Bitmask {
	return numa.NodeMask()
}

func (l *realNumaLibrary) RunOnNode(node int) error {
	return numa.RunOnNode(node)
}

func (l *realNumaLibrary) GetCPUAndNode() (cpu int, node int) {
	return numa.GetCPUAndNode()
}

type mockNumaLibrary struct {
	available    bool
	nodeMask     numa.Bitmask
	runOnNodeErr error
	cpu, node    int
}

func (l *mockNumaLibrary) Available() bool {
	return l.available
}

func (l *mockNumaLibrary) NodeMask() numa.Bitmask {
	return l.nodeMask
}

func (l *mockNumaLibrary) RunOnNode(node int) error {
	return l.runOnNodeErr
}

func (l *mockNumaLibrary) GetCPUAndNode() (cpu int, node int) {
	return l.cpu, l.node
}
