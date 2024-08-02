"""This file defines a FioWorkload and provides logic for parsing a bunch of them from a json test-config file."""

import json
import pprint


def validateFioWorkload(workload, name):
  """Validates the given fio workload config."""
  if (
      'fioWorkload' not in workload
      or 'dlioWorkload' in workload
      or 'bucket' not in workload
  ):
    print(
        f"{name} does not have 'fioWorkload' or 'bucket' key in it, or"
        " has 'dlioWorkload' key in it"
    )
    return False
  fioWorkload = workload['fioWorkload']
  for requiredField in [
      'fileSize',
      'blockSize',
      'filesPerThread',
      'numThreads',
  ]:
    if requiredField not in fioWorkload:
      print(
          f'fioWorkload for {name} does not have value for'
          f' {requiredField} in it'
      )
      return False
  return True


class FioWorkload:

  def __init__(
      self,
      scenario,
      fileSize,
      blockSize,
      filesPerThread,
      numThreads,
      bucket,
      readTypes,
  ):
    self.scenario = scenario
    self.fileSize = fileSize
    self.blockSize = blockSize
    self.filesPerThread = filesPerThread
    self.numThreads = numThreads
    self.bucket = bucket
    self.readTypes = readTypes

  def PPrint(self):
    # pprint.pp(self)
    print(
        f'scenario:{self.scenario}, fileSize:{self.fileSize},'
        f' blockSize:{self.blockSize}, filesPerThread:{self.filesPerThread},'
        f' numThreads:{self.numThreads}, bucket:{self.bucket},'
        f' readTypes:{self.readTypes}'
    )


def ParseTestConfigForFioWorkloads(fioTestConfigFile):
  print(f'Parsing {fioTestConfigFile}')
  with open(fioTestConfigFile) as f:
    d = json.load(f)
    testConfig = d['TestConfig']
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
        print(f'workloads#{i} is not a valid fio workload, so ignoring it.')
        pass
      else:
        for scenario in scenarios:
          # print(f'workload#{i}:  {workloads[i]}')
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
                      fioWorkload['readTypes'].split(',')
                      if 'readTypes' in fioWorkload
                      else ['read', 'randread']
                  ),
              )
          )
  return fioWorkloads
