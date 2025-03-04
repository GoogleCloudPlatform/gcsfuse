# Copyright 2018 The Kubernetes Authors.
# Copyright 2024 Google LLC
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

"""This file defines a FioWorkload and provides utility for parsing a json

test-config file for a list of them.
"""

import json


DefaultNumEpochs = 4


def validateFioWorkload(workload: dict, name: str):
  """Validates the given json workload object."""
  for requiredWorkloadAttribute, expectedType in {
      'bucket': str,
      'gcsfuseMountOptions': str,
      'fioWorkload': dict,
  }.items():
    if requiredWorkloadAttribute not in workload:
      print(f"{name} does not have '{requiredWorkloadAttribute}' key in it.")
      return False
    if not type(workload[requiredWorkloadAttribute]) is expectedType:
      print(
          f"In {name}, the type of '{requiredWorkloadAttribute}' is of type"
          f" '{type(workload[requiredWorkloadAttribute])}', not {expectedType}"
      )
      return False
    if expectedType == str and ' ' in workload[requiredWorkloadAttribute]:
      print(f"{name} has space in the value of '{requiredWorkloadAttribute}'")
      return False

  if 'numEpochs' in workload:
    if not type(workload['numEpochs']) is int:
      print(
          f"In {name}, the type of workload['numEpochs'] is of type"
          f" {type(workload['numEpochs'])}, not {int}"
      )
      return False
    if int(workload['numEpochs']) < 0:
      print(f"In {name}, the value of workload['numEpochs'] < 0, expected: >=0")
      return False

  if 'dlioWorkload' in workload:
    print(f"{name} has 'dlioWorkload' key in it, which is unexpected.")
    return False

  fioWorkload = workload['fioWorkload']
  for requiredAttribute, expectedType in {
      'fileSize': str,
      'blockSize': str,
      'filesPerThread': int,
      'numThreads': int,
  }.items():
    if requiredAttribute not in fioWorkload:
      print(f'In {name}, fioWorkload does not have {requiredAttribute} in it')
      return False
    if not type(fioWorkload[requiredAttribute]) is expectedType:
      print(
          f'In {name}, fioWorkload[{requiredAttribute}] is of type'
          f' {type(fioWorkload[requiredAttribute])}, expected:'
          f' {expectedType} '
      )
      return False

  if 'readTypes' in fioWorkload:
    readTypes = fioWorkload['readTypes']
    if not type(readTypes) is list:
      print(
          f"In {name}, fioWorkload['readTypes'] is of type {type(readTypes)},"
          " not 'list'."
      )
      return False
    for readType in readTypes:
      if not type(readType) is str:
        print(
            f'In {name}, one of the values in'
            f" fioWorkload['readTypes'] is '{readType}', which is of type"
            f' {type(readType)}, not str'
        )
        return False
      if not readType == 'read' and not readType == 'randread':
        print(
            f"In {name}, one of the values in fioWorkload['readTypes'] is"
            f" '{readType}' which is not a supported value. Supported values"
            ' are read, randread'
        )
        return False

  return True


class FioWorkload:
  """FioWorkload holds data needed to define a FIO workload

  (essentially the data needed to create a job file for FIO run).

  Members:
  1. scenario (string): One of "local-ssd", "gcsfuse-generic",
  "gcsfuse-file-cache" and "gcsfuse-no-file-cache".
  2. fileSize (string): fio filesize field in string format e.g. '100', '10K',
  '10M' etc.
  3. blockSize (string): equivalent of bs field in fio job file e.g. '8K',
  '128K', '1M' etc.
  4. filesPerThreads (int): equivalent of nrfiles in fio job file. Must be
  greater than 0.
  5. numThreads (int): equivalent of numjobs in fio job file. Must be greater
  than 0.
  6. bucket (string): Name of a GCS bucket to read input files from.
  7. readTypes (set of strings): a set containing multiple values out of
  'read', 'randread'.
  8. gcsfuseMountOptions (str): gcsfuse mount options as a single
  string in compact stringified format, to be used for the
  test scenario "gcsfuse-generic". The individual config/cli flag values should
  be separated by comma. Each cli flag should be of the form "<flag>[=<value>]",
  while each config-file flag should be of form
  "<config>[:<subconfig>[:<subsubconfig>[...]]]:<value>". For example, a legal
  value would be:
  "implicit-dirs,file_mode=777,file-cache:enable-parallel-downloads:true,metadata-cache:ttl-secs:true".
  9. numEpochs: Number of runs of the fio workload. Default is DefaultNumEpochs
  if missing.
  """

  def __init__(
      self,
      scenario: str,
      fileSize: str,
      blockSize: str,
      filesPerThread: int,
      numThreads: int,
      bucket: str,
      readTypes: list,
      gcsfuseMountOptions: str,
      numEpochs: int = DefaultNumEpochs,
  ):
    self.scenario = scenario
    self.fileSize = fileSize
    self.blockSize = blockSize
    self.filesPerThread = filesPerThread
    self.numThreads = numThreads
    self.bucket = bucket
    self.readTypes = set(readTypes)
    self.gcsfuseMountOptions = gcsfuseMountOptions
    self.numEpochs = numEpochs

  def PPrint(self):
    print(
        f'scenario:{self.scenario}, fileSize:{self.fileSize},'
        f' blockSize:{self.blockSize}, filesPerThread:{self.filesPerThread},'
        f' numThreads:{self.numThreads}, bucket:{self.bucket},'
        f' readTypes:{self.readTypes}, gcsfuseMountOptions:'
        f' {gcsfuseMountOptions}, numEpochs: {self.numEpochs}'
    )


def ParseTestConfigForFioWorkloads(fioTestConfigFile: str):
  """Parses the given workload test configuration file for FIO workloads."""
  print(f'Parsing {fioTestConfigFile} for FIO workloads ...')
  with open(fioTestConfigFile) as f:
    file = json.load(f)
    testConfig = file['TestConfig']
    workloadConfig = testConfig['workloadConfig']
    workloads = workloadConfig['workloads']
    fioWorkloads = []
    scenarios = (
        ['local-ssd', 'gcsfuse-generic']
        if ('runOnSSD' not in workloadConfig or workloadConfig['runOnSSD'])
        else ['gcsfuse-generic']
    )
    for i in range(len(workloads)):
      workload = workloads[i]
      if not validateFioWorkload(workload, f'workload#{i}'):
        print(f'workloads#{i} is not a valid FIO workload, so ignoring it.')
      else:
        for scenario in scenarios:
          fioWorkload = workload['fioWorkload']
          fioWorkloads.append(
              FioWorkload(
                  scenario,
                  fioWorkload['fileSize'],
                  fioWorkload['blockSize'],
                  fioWorkload['filesPerThread'],
                  fioWorkload['numThreads'],
                  workload['bucket'],
                  (
                      fioWorkload['readTypes']
                      if 'readTypes' in fioWorkload
                      else ['read', 'randread']
                  ),
                  workload['gcsfuseMountOptions'],
                  numEpochs=(
                      workload['numEpochs']
                      if 'numEpochs' in workload
                      else DefaultNumEpochs
                  ),
              )
          )
  return fioWorkloads


def FioChartNamePodName(
    fioWorkload: FioWorkload, instanceID: str, readType: str
) -> (str, str, str):
  shortenScenario = {
      'local-ssd': 'ssd',
      'gcsfuse-generic': 'gcsfuse',
  }
  shortForScenario = (
      shortenScenario[fioWorkload.scenario]
      if fioWorkload.scenario in shortenScenario
      else 'other'
  )
  readTypeToShortReadType = {'read': 'sr', 'randread': 'rr'}
  shortForReadType = (
      readTypeToShortReadType[readType]
      if readType in readTypeToShortReadType
      else 'ur'
  )

  hashOfWorkload = str(hash((fioWorkload, instanceID, readType))).replace(
      '-', ''
  )
  return (
      f'fio-load-{shortForScenario}-{shortForReadType}-{fioWorkload.fileSize.lower()}-{hashOfWorkload}',
      f'fio-tester-{shortForScenario}-{shortForReadType}-{fioWorkload.fileSize.lower()}-{hashOfWorkload}',
      f'{instanceID}/{fioWorkload.fileSize}-{fioWorkload.blockSize}-{fioWorkload.numThreads}-{fioWorkload.filesPerThread}-{hashOfWorkload}/{fioWorkload.scenario}/{readType}',
  )
