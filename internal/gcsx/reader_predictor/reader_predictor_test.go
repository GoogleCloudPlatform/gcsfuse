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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const EPS = 0.000001

type HeuristicReaderPredictorSuite struct {
	suite.Suite
}

func TestHeuristicReaderPredictorSuite(t *testing.T) {
	suite.Run(t, new(HeuristicReaderPredictorSuite))
}

func (hrp *HeuristicReaderPredictorSuite) TestHeuristicReaderPredictor() {
	predictor := NewHeuristicReaderPredictor()

	assert.NotNil(hrp.T(), predictor)
}

func (hrp *HeuristicReaderPredictorSuite) TestRecordRead() {
	predictor := NewHeuristicReaderPredictor()

	input := NewReaderPredictorInput(Sequential)
	predictor.RecordRead(input)

	assert.True(hrp.T(), math.Abs(0.5*0.2+0.001*0.8-predictor.(*heuristicReaderPredictor).runningScore) < EPS)
	assert.Equal(hrp.T(), Sequential, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_Random() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of random reads
	for i := 0; i < 10; i++ {
		input := NewReaderPredictorInput(Random)
		predictor.RecordRead(input)
	}

	assert.Equal(hrp.T(), Random, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_Sequential() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of sequential reads
	for i := 0; i < 10; i++ {
		input := NewReaderPredictorInput(Sequential)
		predictor.RecordRead(input)
	}

	assert.Equal(hrp.T(), Sequential, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_MixReadSequential() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of random reads
	for i := 0; i < 5; i++ {
		input := NewReaderPredictorInput(Random)
		predictor.RecordRead(input)
	}

	// Now simulate more sequential reads
	for i := 0; i < 10; i++ {
		input := NewReaderPredictorInput(Sequential)
		predictor.RecordRead(input)
	}

	assert.Equal(hrp.T(), Sequential, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_MixReadRandom() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of sequential reads
	for i := 0; i < 5; i++ {
		input := NewReaderPredictorInput(Sequential)
		predictor.RecordRead(input)
	}

	// Now simulate more random reads
	for i := 0; i < 10; i++ {
		input := NewReaderPredictorInput(Random)
		predictor.RecordRead(input)
	}

	assert.Equal(hrp.T(), Random, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_SmallFile() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of sequential reads
	for i := 0; i < 10; i++ {
		sizePredictorInput := NewSizePredictorInput(1 << 20) // 1 MB
		predictor.RecordRead(sizePredictorInput)
	}

	assert.Equal(hrp.T(), Sequential, predictor.PredictReadType())
}

func (hrp *HeuristicReaderPredictorSuite) TestPredictReadType_LargeFile() {
	predictor := NewHeuristicReaderPredictor()

	// Simulate a series of sequential reads
	for i := 0; i < 10; i++ {
		sizePredictorInput := NewSizePredictorInput(1 << 30) // 1 GB
		predictor.RecordRead(sizePredictorInput)
	}

	assert.Equal(hrp.T(), Sequential, predictor.PredictReadType())
}