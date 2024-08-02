#!/usr/bin/env python

# Copyright 2018 The Kubernetes Authors.
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""This program takes in a json dlio test-config file and generates and deploys helm charts."""

import json

# import os

from absl import app


def validateDlioWorkload(workload, name):
  """Validates the given dlio workload config."""
  if (
      'dlioWorkload' not in workload
      or 'fioWorkload' in workload
      or 'bucket' not in workload
  ):
    print(
        f"{name} does not have 'dlioWorkload' or 'bucket' key in it, or"
        " has 'fioWorkload' key in it"
    )
    return False
  dlioWorkload = workload['dlioWorkload']
  for requiredField in ['numFilesTrain', 'recordLength', 'batchSizes']:
    if requiredField not in dlioWorkload:
      print(f'dlioWorkload for {name} does not have {requiredField} in it')
      return False
  return True


class DlioWorkload:

  def __init__(self, scenario, numFilesTrain, recordLength, bucket, batchSizes):
    self.scenario = scenario
    self.numFilesTrain = numFilesTrain
    self.recordLength = recordLength
    self.bucket = bucket
    self.batchSizes = batchSizes


def ParseTestConfigForDlioWorkloads(testConfigFileName):
  with open(testConfigFileName) as f:
    d = json.load(f)
    testConfig = d['TestConfig']
    workloadConfig = testConfig['workloadConfig']
    workloads = workloadConfig['workloads']
    dlioWorkloads = []
    scenarios = (
        ['local-ssd', 'gcsfuse-generic']
        if ('runOnSSD' not in workloadConfig or workloadConfig['runOnSSD'])
        else ['gcsfuse-generic']
    )
    for i in range(len(workloads)):
      workload = workloads[i]
      if not validateDlioWorkload(workload, f'workload#{i}'):
        print(f'workloads#{i} is not a valid dlio workload, so ignoring it.')
        pass
      else:
        for scenario in scenarios:
          dlioWorkload = workload['dlioWorkload']
          dlioWorkloads.append(
              DlioWorkload(
                  scenario,
                  dlioWorkload['numFilesTrain'],
                  dlioWorkload['recordLength'],
                  workload['bucket'],
                  (
                      dlioWorkload['batchSizes'].split(',')
                      if (
                          'batchSizes' in dlioWorkload
                          and dlioWorkload['batchSizes']
                          and not str.isspace(dlioWorkload['batchSizes'])
                      )
                      else []
                  ),
              )
          )
  return dlioWorkloads
