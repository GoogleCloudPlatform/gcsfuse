"""This file defines a FioWorkload and provides utility for parsing a json

test-config file for a list of them.
"""

import json


def validateFioWorkload(workload: dict, name: str):
  """Validates the given json workload object."""
  if 'fioWorkload' not in workload:
    print(f"{name} does not have 'fioWorkload' key in it.")
    return False

  if 'bucket' not in workload:
    print(f"{name} does not have 'bucket' key in it.")
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
  7. readTypes (list of strings): a list containing multiple values out of
  'read', 'randread'.
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
  ):
    self.scenario = scenario
    self.fileSize = fileSize
    self.blockSize = blockSize
    self.filesPerThread = filesPerThread
    self.numThreads = numThreads
    self.bucket = bucket
    self.readTypes = readTypes

  def PPrint(self):
    print(
        f'scenario:{self.scenario}, fileSize:{self.fileSize},'
        f' blockSize:{self.blockSize}, filesPerThread:{self.filesPerThread},'
        f' numThreads:{self.numThreads}, bucket:{self.bucket},'
        f' readTypes:{self.readTypes}'
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
        pass
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
              )
          )
  return fioWorkloads
