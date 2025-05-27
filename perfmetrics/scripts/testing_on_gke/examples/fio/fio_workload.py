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
from pathlib import Path


DefaultNumEpochs = 4
DefaultReadTypes = ['read', 'randread']


def validate_fio_workload(workload: dict, name: str):
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
      raise Exception(
          f"In {name}, the type of '{requiredWorkloadAttribute}' is of type"
          f" '{type(workload[requiredWorkloadAttribute])}', not {expectedType}"
      )
    if expectedType == str and ' ' in workload[requiredWorkloadAttribute]:
      raise Exception(
          f"{name} has space in the value of '{requiredWorkloadAttribute}'"
      )

  if 'numEpochs' in workload:
    if not type(workload['numEpochs']) is int:
      raise Exception(
          f"In {name}, the type of workload['numEpochs'] is of type"
          f" {type(workload['numEpochs'])}, not {int}"
      )
    if int(workload['numEpochs']) < 0:
      raise Exception(
          f"In {name}, the value of workload['numEpochs'] < 0, expected: >=0"
      )

  if 'dlioWorkload' in workload:
    print(f"{name} has 'dlioWorkload' key in it, which is unexpected.")
    return False

  fioWorkload = workload['fioWorkload']
  if 'jobFile' in fioWorkload:
    jobFile = fioWorkload['jobFile'].strip()
    if len(jobFile) == 0:
      raise Exception(
          '{name} has jobFile attribute in it, but it is empty, so ignoring'
          ' this workload.'
      )
    elif ' ' in jobFile:
      raise Exception(
          '{name} has jobFile attribute in it, but it has space (" ") in it, so'
          ' ignoring this workload.'
      )
    if not str.startswith(jobFile, 'gs://') and not Path(jobFile).is_file():
      raise Exception(
          f'{name} has fio.jobFile={jobFile} which is not a proper jobFile'
          ' argument. It neither starts with gs://, nor'
          ' it is a valid file on this machine.'
      )
  else:
    for requiredAttribute, expectedType in {
        'fileSize': str,
        'blockSize': str,
        'filesPerThread': int,
        'numThreads': int,
    }.items():
      if requiredAttribute not in fioWorkload:
        raise Exception(
            f'In {name}, fioWorkload does not have {requiredAttribute} in it'
        )
      if not type(fioWorkload[requiredAttribute]) is expectedType:
        raise Exception(
            f'In {name}, fioWorkload[{requiredAttribute}] is of type'
            f' {type(fioWorkload[requiredAttribute])}, expected:'
            f' {expectedType} '
        )

    if 'readTypes' in fioWorkload:
      readTypes = fioWorkload['readTypes']
      if not type(readTypes) is list:
        raise Exception(
            f"In {name}, fioWorkload['readTypes'] is of type {type(readTypes)},"
            " not 'list'."
        )
      for readType in readTypes:
        if not type(readType) is str:
          raise Exception(
              f'In {name}, one of the values in'
              f" fioWorkload['readTypes'] is '{readType}', which is of type"
              f' {type(readType)}, not str'
          )
        if not readType == 'read' and not readType == 'randread':
          raise Exception(
              f"In {name}, fioWorkload['readTypes'] is"
              f" '{readType}' which is not a supported value. Supported values"
              ' are read, randread'
          )

  return True


class FioWorkload:
  """FioWorkload holds data needed to define a FIO workload

  (essentially the data needed to create a job file for FIO run).

  Members:
  1. scenario (string): One of "local-ssd", "gcsfuse-generic".
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
  10. jobFile: The path of a FIO job-file . When this is specified, it will
  override the values of
  fileSize, blockSize, filesPerThreads, numThreads, readTypes. This will be
  either of the form gs://<bucket>/<object-name> or a path to an existing
  file on this machine. If this is a local file, then another attribute
  jobFileContent is created and stored in this FioWorkload object.
  """

  def _job_file_content_from_job_file(self, jobFile: str) -> str:
    """Reads, escapes and serializes the content of a given FIO job-file into a single string

    which can be passed down as a helm config value to be used in a pod
    configuration.
    This escapes uncommon characters such as space, tabs, dollar signs etc to
    avoid issues with serialization/deserialization and use in a shell script.
    This encodes all newline characters as ';' as helm doesn't support passing
    multiline config values.
    """
    with open(jobFile, 'r') as jobFileReader:
      replaceNewlineWith = ';'
      jobFileRawContent = jobFileReader.read()

      # Encode newline characters as ';' .
      if replaceNewlineWith in jobFileRawContent:
        raise Exception(
            f'FIO jobFile {jobFile} has unsupported character'
            f' "{replaceNewlineWith}" in it. Full content:'
            f' "{jobFileRawContent}" .'
        )
      jobFileContent = jobFileRawContent.replace('\n', replaceNewlineWith)

      # Prepend triple-backslash to any dollar signs (config constants e.g.
      # $FILESIZE) in the FIO job-file content
      # before passing it through helm to the pod config, to escape them from
      # helm and shell evaluating them.
      # One backslash is needed to avoid the shell evaluating these constants.
      # Then two more backslashes are needed to avoid the helm install command
      # evaluating added backslash and the slash itself.
      # As an example, helm install ... --set jobFileContent="filesize=\\\$FILESIZE",
      # get passed down to pod config as variable jobFileContent with value
      # "\$FILESIZE". This when dumped by shell into a file becomes
      # "filesize=$FILESIZE" which is the intended original config.
      # TODO: handle it through a utility function.
      jobFileContent = jobFileContent.replace('$', r'\\\$')

      return jobFileContent

  def __init__(
      self,
      scenario: str,
      bucket: str,
      gcsfuseMountOptions: str,
      numEpochs: int = DefaultNumEpochs,
      fileSize: str = None,
      blockSize: str = None,
      filesPerThread: int = None,
      numThreads: int = None,
      readTypes: list = DefaultReadTypes,
      jobFile: str = None,
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
    self.jobFile = jobFile
    # If jobFile is a local file, then take its content,
    # replace all newline characters with ';' and
    # store it as self.jobFileContent .
    if (
        not str.startswith(self.jobFile, 'gs://')
        and Path(self.jobFile).is_file()
    ):
      self.jobFileContent = self._job_file_content_from_job_file(self.jobFile)

  def PPrint(self):
    print(
        f'scenario:{self.scenario}, fileSize:{self.fileSize},'
        f' blockSize:{self.blockSize}, filesPerThread:{self.filesPerThread},'
        f' numThreads:{self.numThreads}, bucket:{self.bucket},'
        f' readTypes:{self.readTypes}, gcsfuseMountOptions:'
        f' {self.gcsfuseMountOptions}, numEpochs: {self.numEpochs}, jobFile:'
        f' {self.jobFile}'
    )


def parse_test_config_for_fio_workloads(fioTestConfigFile: str):
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
      if not validate_fio_workload(workload, f'workload#{i}'):
        print(f'workloads#{i} is not a valid FIO workload, so ignoring it.')
      else:
        fioWorkload = workload['fioWorkload']
        fioWorkloadAttributes = dict()
        if 'jobFile' in fioWorkload:
          fioWorkloadAttributes['jobFile'] = fioWorkload['jobFile']
        else:
          for attr in [
              'fileSize',
              'blockSize',
              'numThreads',
              'filesPerThread',
          ]:
            fioWorkloadAttributes[attr] = fioWorkload[attr]
        for attr in ['bucket', 'gcsfuseMountOptions']:
          fioWorkloadAttributes[attr] = workload[attr]
          fioWorkloadAttributes['readTypes'] = (
              fioWorkload['readTypes']
              if 'readTypes' in fioWorkload
              else DefaultReadTypes
          )
        fioWorkloadAttributes['numEpochs'] = (
            workload['numEpochs']
            if 'numEpochs' in workload
            else DefaultNumEpochs
        )
        for scenario in scenarios:
          fioWorkloadAttributes['scenario'] = scenario
          fioWorkloads.append(FioWorkload(**fioWorkloadAttributes))
  return fioWorkloads


def FioChartNamePodName(
    fioWorkload: FioWorkload, experimentID: str, readType: str = None
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

  if not fioWorkload.jobFile:
    readTypeToShortReadType = {'read': 'sr', 'randread': 'rr'}
    if not readType or readType not in readTypeToShortReadType.keys():
      raise Exception(
          f'Unexpected scenario. Unsupported rw type {readType} passed for a'
          ' fioWorkload not containing a jobFile. Expected it to be one out of'
          f' {readTypeToShortReadType.keys()}'
      )
    shortForReadType = readTypeToShortReadType[readType]
    hashOfWorkload = str(hash((fioWorkload, experimentID, readType))).replace(
        '-', ''
    )
    return (
        f'fio-load-{shortForScenario}-{shortForReadType}-{fioWorkload.fileSize.lower()}-{hashOfWorkload}',
        f'fio-tester-{shortForScenario}-{shortForReadType}-{fioWorkload.fileSize.lower()}-{hashOfWorkload}',
        f'{experimentID}/{fioWorkload.fileSize}-{fioWorkload.blockSize}-{fioWorkload.numThreads}-{fioWorkload.filesPerThread}-{hashOfWorkload}/{fioWorkload.scenario}/{readType}',
    )
  else:
    hashOfWorkload = str(hash((fioWorkload, experimentID))).replace('-', '')
    return (
        f'fio-load-{shortForScenario}-{hashOfWorkload}',
        f'fio-tester-{shortForScenario}-{hashOfWorkload}',
        f'{experimentID}/{hashOfWorkload}/{fioWorkload.scenario}/unknown_rw',
    )
