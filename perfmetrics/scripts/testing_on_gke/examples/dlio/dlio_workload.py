"""This file defines a DlioWorkload (a DLIO Unet3d workload) and provides utility for parsing a json

test-config file for a list of them.
"""

import json


def validateDlioWorkload(workload: dict, name: str):
  """Validates the given json workload object."""
  if 'dlioWorkload' not in workload:
    print(f"{name} does not have 'dlioWorkload' key in it.")
    return False

  if 'bucket' not in workload:
    print(f"{name} does not have 'bucket' key in it.")
    return False

  if 'fioWorkload' in workload:
    print(f"{name} has 'fioWorkload' key in it, which is unexpected.")
    return False

  dlioWorkload = workload['dlioWorkload']
  for requiredAttribute, _type in {
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
    if not type(dlioWorkload[requiredAttribute]) is _type:
      print(
          f'In {name}, dlioWorkload[{requiredAttribute}] is of type'
          f' {type(dlioWorkload[requiredAttribute])}, not of type {_type} '
      )
      return False

  for batchSize in dlioWorkload['batchSizes']:
    if not type(batchSize) is int:
      print(
          f'In {name}, one of the batch-size values in'
          f" dlioWorkload['batchSizes'] is '{batchSize}', which is of type"
          f' {type("batchSize")}, not int'
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
  2. numFilesTrain (string): DLIO numFilesTrain argument in string format e.g.
  '500000' etc.
  3. recordLength (string): DLIO recordLength argument in string format e.g.
  '100', '1000000', '10M' etc.
  4. bucket (string): Name of a GCS bucket to read input files from.
  5. batchSizes (list of strings): a string containing multiple comma-separated
  integer values.
  """

  def __init__(self, scenario, numFilesTrain, recordLength, bucket, batchSizes):
    self.scenario = scenario
    self.numFilesTrain = numFilesTrain
    self.recordLength = recordLength
    self.bucket = bucket
    self.batchSizes = batchSizes


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
              )
          )
  return dlioWorkloads
