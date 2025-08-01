// Create a read predictor which will take input prediction input and provide a method to predict if the pattern was sequential or random.
// Copyright 2025 Google LLC
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

package reader_predictor

import (
	"sync"
)

type ReadType int64

const (
	Sequential ReadType = iota
	Random
)

type PredictorInput interface {
	Score() float64
}

type ReaderPredictor interface {
	RecordRead(PredictorInput)
	PredictReadType() ReadType
}

type heuristicReaderPredictor struct {
	runningScore float64
	decayFactor  float64 // Not used in this implementation, but can be used for future enhancements.
	mu           sync.RWMutex
}

func NewHeuristicReaderPredictor() ReaderPredictor {
	return &heuristicReaderPredictor{
		runningScore: 0.5,
		decayFactor:  0.8,
		mu:           sync.RWMutex{},
	}
}

func (p *heuristicReaderPredictor) RecordRead(input PredictorInput) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.runningScore = p.runningScore*(1-p.decayFactor) + p.decayFactor*input.Score()
}

func (p *heuristicReaderPredictor) PredictReadType() ReadType {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.runningScore > 0.7 {
		return Random
	}
	return Sequential
}

type readerPredictorInput struct {
	PredictorInput
	readType ReadType
}

func (r *readerPredictorInput) Score() float64 {
	if r.readType == Sequential {
		return 0.001
	}
	return 0.999
}

func NewReaderPredictorInput(readType ReadType) PredictorInput {
	return &readerPredictorInput{
		readType: readType,
	}
}

type sizePredictorInput struct {
	PredictorInput
	objectSize int64 // Size of the object in bytes.
}

func (r *sizePredictorInput) Score() float64 {
	if r.objectSize < 1024 {
		return 0.001
	}

	if r.objectSize < 1048576 { // Less than 1 MB
		return 0.01
	}

	if r.objectSize < 10485760 {
		return 0.2
	}

	// Otherwise, it can be anything.
	return 0.5
}

func NewSizePredictorInput(objectSize int64) PredictorInput {
	return &sizePredictorInput{
		objectSize: objectSize,
	}
}
