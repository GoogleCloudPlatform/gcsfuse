"""This file defines a DlioWorkload (a DLIO Unet3d workload) and provides utility for parsing a json

test-config file for a list of them.
"""

import json


def validateDlioWorkload(workload: dict, name: str):
  """Validates the given json workload object."""
  for requiredWorkloadAttribute, expectedType in {
      'bucket': str,
      'gcsfuseMountOptions': str,
      'dlioWorkload': dict,
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

  if 'fioWorkload' in workload:
    print(f"{name} has 'fioWorkload' key in it, which is unexpected.")
    return False

  dlioWorkload = workload['dlioWorkload']
  for requiredAttribute, expectedType in {
      'numFilesTrain': int,
      'recordLength': int,
      'batchSizes': list,
  }.items():
    if requiredAttribute not in dlioWorkload:
      print(
          f'In {name}, dlioWorkload for {name} does not have'
          f' {requiredAttribute} in it'
      )
      return False
    if not type(dlioWorkload[requiredAttribute]) is expectedType:
      print(
          f'In {name}, dlioWorkload[{requiredAttribute}] is of type'
          f' {type(dlioWorkload[requiredAttribute])}, expected:'
          f' {expectedType} '
      )
      return False

  batchSizes = dlioWorkload['batchSizes']
  for batchSize in batchSizes:
    if not type(batchSize) is int:
      print(
          f'In {name}, one of the batch-size values in'
          f" dlioWorkload['batchSizes'] is '{batchSize}', which is of type"
          f' {type(batchSize)}, not int'
      )
      return False
    if batchSize < 1:
      print(
          f'In {name}, one of the batch-size values in'
          f" dlioWorkload['batchSizes'] is '{batchSize}' < 1, which is not"
          ' supported.'
      )
      return False

  return True


class DlioWorkload:
  """DlioWorkload holds data needed to define a DLIO Unet3d workload

  (essentially the data needed to create a job file for DLIO run).

  Members:
  1. scenario (string): One of "local-ssd", "gcsfuse-generic",
  "gcsfuse-file-cache" and "gcsfuse-no-file-cache".
  2. numFilesTrain (int): DLIO numFilesTrain argument e.g. 500000 etc.
  3. recordLength (int): DLIO recordLength argument e.g. 100, 1000000 etc.
  4. bucket (str): Name of a GCS bucket to read input files from.
  5. batchSizes (set of ints): a set of ints representing multiple batchsize
  values to test.
  6. gcsfuseMountOptions (str): gcsfuse mount options as a single
  comma-separated string.
  """

  def __init__(
      self,
      scenario: str,
      numFilesTrain: int,
      recordLength: int,
      bucket: str,
      batchSizes: list,
      gcsfuseMountOptions: str,
  ):
    self.scenario = scenario
    self.numFilesTrain = numFilesTrain
    self.recordLength = recordLength
    self.bucket = bucket
    self.batchSizes = set(batchSizes)
    self.gcsfuseMountOptions = gcsfuseMountOptions


def ParseTestConfigForDlioWorkloads(testConfigFileName: str):
  """Parses the given workload test configuration file for DLIO workloads."""
  print(f'Parsing {testConfigFileName} for DLIO workloads ...')
  with open(testConfigFileName) as f:
    file = json.load(f)
    testConfig = file['TestConfig']
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
        print(f'workloads#{i} is not a valid DLIO workload, so ignoring it.')
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
                  dlioWorkload['batchSizes'],
                  workload['gcsfuseMountOptions'],
              )
          )
  return dlioWorkloads


def DlioChartNamePodName(
    dlioWorkload: DlioWorkload, instanceID: str, batchSize: int
) -> (str, str, str):
  shortenScenario = {
      'local-ssd': 'ssd',
      'gcsfuse-generic': 'gcsfuse',
  }
  shortForScenario = (
      shortenScenario[dlioWorkload.scenario]
      if dlioWorkload.scenario in shortenScenario
      else 'other'
  )

  hashOfWorkload = str(hash((instanceID, batchSize, dlioWorkload))).replace(
      '-', ''
  )
  return (
      f'dlio-unet3d-{shortForScenario}-{dlioWorkload.recordLength}-{hashOfWorkload}',
      f'dlio-tester-{shortForScenario}-{dlioWorkload.recordLength}-{hashOfWorkload}',
      f'{instanceID}/{dlioWorkload.numFilesTrain}-{dlioWorkload.recordLength}-{batchSize}-{hashOfWorkload}/{dlioWorkload.scenario}',
  )
