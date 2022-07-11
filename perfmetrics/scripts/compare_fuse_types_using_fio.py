"""Python script to do FIO test for GCSFuse Vs GCSFuse/Goofys.

To run the script:
python3 compare_fuse_types_using_fio.py -- --fuse_type_1=<value1> --fuse_type_1_version=<version1> --flags_1=<flags1> 
--fuse_type_2=<value2> --fuse_type_2_version=<version2> --flags_2=<flags2> --jobfile_path=<jobfile_path> --gcs_bucket=<gcs_bucket>

1) --fuse_type_1=<value1> & --fuse_type_2=<value2>
-> Value1 and value2 can be fuse based system name.
2) --fuse_type_1_version=<version1> & --fuse_type_2_version=<version2>
-> For GCSFuse you can mention any released version or ‘master’ as version if you want to build from source.
-> For other fuse based system provide the link of github repository.
3) --flags_1=<flags1> & --flags_2=<flags2>
-> Provide  the flags for mounting bucket using specific provided fuse based system.
-> Since there might be more than one flags, provided all of them inside single quotes.
-> Example for GCSFuse flags: --flags_1='--implicit-dirs --max-conns-per-host 100 --disable-http2'
4) --jobfile_path=<jobfile_path>
-> Job file path for the FIO load test.
5) --gcs_bucket=<gcs_bucket>
-> Name of the GCS bucket to be mounted for FIO load test.

The code takes input the fuse based system type and its version, required flags to mount the bucket, jobfile_path, 
and gcs bucket name as input for fio test. Then it will perform the fio test for each of the given 
fuse based system type at a specific version
and extract fio_metrics to 'output.txt' file in the 'out' directory.
"""

import argparse
import os
from fio import fio_metrics

from absl import app

GCSFUSE_REPO = 'https://github.com/GoogleCloudPlatform/gcsfuse'
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --disable-http2"


def _install_gcsfuse(version, gcs_bucket, gcsfuse_flags) -> None:
  """Install gcsfuse with Specific version.
  
  Args:
    version(str): Gcsfuse version to be installed. 
    gcs_bucket(str): GCS bucket to be mounted.
    gcsfuse_flags(str): Fuse flags for mounting the GCS bucket.
  """
  os.system(f'''curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v{version}/gcsfuse_{version}_amd64.deb
            sudo dpkg --install gcsfuse_{version}_amd64.deb
            mkdir gcs
            gcsfuse {gcsfuse_flags} {gcs_bucket} gcs
            ''')


def _install_gcsfuse_source(gcs_bucket, gcsfuse_flags) -> None:
  """Run gcsfuse from source code.
  
  Args:
    version(str): Gcsfuse version to be installed. 
    gcs_bucket(str): GCS bucket to be mounted.
  """
  os.system(f'''git clone {GCSFUSE_REPO}
            mkdir gcs
            cd gcsfuse
            go run . {gcsfuse_flags} {gcs_bucket} ../gcs
            cd ..
            ''')

  
def _remove_gcsfuse(version) -> None:
  """Remove gcsfuse with specific version.
  
  Args:
    version(str): Gcsfuse version to be removed.
  """
  os.system(f'''fusermount -u gcs
            rm -rf gcs
            rm -rf gcsfuse_{version}_amd64.deb
            sudo apt-get remove gcsfuse -y
            rm -rf gcsfuse
            ''')


def _install_fuse(fuse_type, fuse_url, fuse_flags, gcs_bucket) -> None:
  """Install latest version of given fuse based system.
  
  Args:
    fuse_type(str): Fuse type for fio test.
    fuse_url(str): URL of github repo of given fuse based system.
    fuse_flags(str): Fuse flags for mounting the bucket.
    gcs_bucket(str): GCS bucket to be mounted.
  """
  os.system(f'''git clone {fuse_url}
            export GOPATH=$HOME/work
            mkdir gcs
            cd {fuse_type}
            go run . {fuse_flags} gs://{gcs_bucket} ../gcs
            cd ..
            ''')


def _remove_fuse(fuse_type) -> None:
  """Remove the given fuse based system.
  
  Args:
    fuse_type(str): Fuse type for fio test.
  """
  os.system(f'''fusermount -u gcs
            rm -rf gcs
            rm -rf {fuse_type}
            ''')


def _run_fio_test(jobfile_path, fio_metrics_obj) -> None:
  """Run fio test and extract metrics to output.txt file.
  
  Args:
    jobfile(str): Path of the job file.
    fio_metrics_obj(str): Object for extracting fio metrics.
  """
  os.system(f'''fio {jobfile_path} --lat_percentiles 1 --output-format=json --output='output.json'
            ''')
  fio_metricss = fio_metrics_obj.get_metrics('output.json', False)
  os.system(f'''echo {fio_metricss} >> out/output.txt
            rm output.json
            ''')


def _gcsfuse_fio_test(version, jobfile_path, fio_metrics_obj, gcs_bucket, fuse_flags) -> None:
  """FIO test for gcsfuse of given version.
  
  Args:
    version(str): Gcsfuse version to perform fio test.
    jobfile(str): Path of the job file.
    fio_metrics_obj(str): Object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
    fuse_flags(str): Fuse flags for mounting the bucket.
  """
  if version == 'master':
    _install_gcsfuse_source(gcs_bucket, fuse_flags)
  else:
    _install_gcsfuse(version, gcs_bucket, fuse_flags)
  
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_gcsfuse(version)


def _fuse_fio_test(fuse_type, jobfile_path, fio_metrics_obj, gcs_bucket, fuse_flags, fuse_url) -> None:
  """FIO test for latest version of goofys.
  
  Args:
    fuse_type(str): Fuse type for fio test.
    jobfile(str): Path of the job file.
    fio_metrics_obj(str): Object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
    fuse_flags(str): Fuse flags for mounting the bucket.
    fuse_url(str): URL of github repo of given fuse based system.
  """
  _install_fuse(fuse_type, fuse_url, fuse_flags, gcs_bucket)
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_fuse(fuse_type)

  
def _fuse_test(fuse_type, fuse_type_version, jobfile_path, fio_metric_obj, gcs_bucket, fuse_flags) -> None:
  """FIO test for specific version of given fuse type.
  
  Args:
    fuse_type(str): Fuse type for fio test.
    fuse_type_version(str): Fuse type version for fio test.
    jobfile(str): Path of the job file.
    fio_metrics_obj(str): Object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
    fuse_flags(str): Fuse flags for mounting the bucket.
  """
  if fuse_type == 'gcsfuse':
    _gcsfuse_fio_test(fuse_type_version, jobfile_path, fio_metric_obj, gcs_bucket, fuse_flags)
  else:
    _fuse_fio_test(fuse_type, jobfile_path, fio_metric_obj, gcs_bucket, fuse_flags, fuse_type_version)
      
  
def main(argv) -> None:

  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--fuse_type_1',
      help='Provide reading fuse_type_1 for fio test',
      required=True)
  parser.add_argument(
      '--fuse_type_1_version',
      help='Provide the Specific version of fuse_type_1 in approprite formate',
      required=True)
  parser.add_argument(
      '--flags_1',
      help='Required flag while mounting bucket using fuse_type_1',
      required=True)
  parser.add_argument(
      '--fuse_type_2',
      help='Provide reading fuse_type_2 for fio test',
      required=True)
  parser.add_argument(
      '--fuse_type_2_version',
      help='Provide the Specific version of fuse_type_2 in approprite formate',
      required=True)
  parser.add_argument(
      '--flags_2',
      help='Required flag while mounting bucket using fuse_type_2',
      required=True)
  parser.add_argument(
      '--jobfile_path',
      help='Provide path of the jobfile',
      required=True)
  parser.add_argument(
      '--gcs_bucket',
      help="Provide the gcs bucket name to be mounted",
      required=True)
  args = parser.parse_args(argv[1:])

  fio_metrics_obj = fio_metrics.FioMetrics()
  os.system('mkdir out')

  os.system(f'echo Fuse Type 1: {args.fuse_type_1} {args.fuse_type_1_version} >> out/output.txt')
  _fuse_test(args.fuse_type_1, args.fuse_type_1_version, args.jobfile_path, fio_metrics_obj, args.gcs_bucket, args.flags_1)

  os.system(f'echo Fuse Type 2: {args.fuse_type_2} {args.fuse_type_2_version} >> out/output.txt')
  _fuse_test(args.fuse_type_2, args.fuse_type_2_version, args.jobfile_path, fio_metrics_obj, args.gcs_bucket, args.flags_2)


if __name__ == '__main__':
  app.run(main)
  
