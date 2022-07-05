"""Python script to do FIO test for GCSFuse Vs GCSFuse/Goofys.

To run the script:
>> python3 compare_fuse_types_using_fio.py -- fuse_type_1 fuse_type_1_version fuser_type_2 fuse_type_2_version jobfile_path

-> Supported fuse_types are gcsfuse and goofys.
-> Incase of gcsfuse you can specify version as any of its released version or master to run from the source.
-> Incase of goofys specify the version as latest, as we will be comparing gcsfuse with the latest version of goofys.

The code takes input the fuse type and its version and, jobfile_path as input for fio test.
Then it will perform the fio test for each of the given fuse type at a specific version
and extract fio_metrics to output.txt file.
"""
import argparse
import os
from fio import fio_metrics

from absl import app

GOOFYS_REPO = 'https://github.com/kahing/goofys'
GCSFUSE_REPO = 'https://github.com/GoogleCloudPlatform/gcsfuse'
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --disable-http2"


def _install_gcsfuse(version, gcs_bucket) -> None:
  """Install gcsfuse with Specific version.
  
  Args:
    version(str): gcsfuse version to be installed. 
    gcs_bucket(str): GCS bucket to be mounted.
  """
  os.system(f'''curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v{version}/gcsfuse_{version}_amd64.deb
            sudo dpkg --install gcsfuse_{version}_amd64.deb
            mkdir gcs
            gcsfuse {GCSFUSE_FLAGS} {gcs_bucket} gcs
            ''')
  
def _install_gcsfuse_source(version, gcs_bucket) -> None:
  """Run gcsfuse from source code.
  
  Args:
    version(str): gcsfuse version to be installed. 
    gcs_bucket(str): GCS bucket to be mounted.
  """
  os.system(f'''git clone {GCSFUSE_REPO}
            mkdir gcs
            cd gcsfuse
            gcsfuse {GCSFUSE_FLAGS} {gcs_bucket} ../gcs
            cd ..
            ''')

  
def _remove_gcsfuse(version) -> None:
  """Remove gcsfuse with specific version.
  
  Args:
    version(str): gcsfuse version to be removed.
  """
  os.system(f'''fusermount -u gcs
            rm -rf gcs
            rm -rf gcsfuse_{version}_amd64.deb
            sudo apt-get remove gcsfuse -y
            rm -rf gcsfuse
            ''')


def _install_goofys(gcs_bucket) -> None:
  """Install latest version of goofys.
  
  Args:
    gcs_bucket(str): GCS bucket to be mounted.
  """
  os.system(f'''git clone {GOOFYS_REPO}
            export GOPATH=$HOME/work
            mkdir gcs
            cd goofys
            go run . gs://{gcs_bucket} ../gcs
            cd ..
            ''')


def _remove_goofys() -> None:
  """Remove goofys.
  """
  os.system('''fusermount -u gcs
            rm -rf gcs
            rm -rf goofys
            ''')


def _run_fio_test(jobfile_path, fio_metrics_obj) -> None:
  """Run fio test and extract metrics to output.txt file.
  
  Args:
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
  """
  os.system(f'''fio {jobfile_path} --lat_percentiles 1 --output-format=json --output='output.json'
            ''')
  fio_metricss = fio_metrics_obj.get_metrics('output.json', False)
  os.system(f'''echo {fio_metricss} >> out/output.txt
            rm output.json
            ''')


def _gcsfuse_test(version, jobfile_path, fio_metrics_obj, gcs_bucket) -> None:
  """FIO test for gcsfuse of given version.
  
  Args:
    version(str): gcsfuse version to perform fio test.
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
  """
  if version == 'master':
    _install_gcsfuse_source(version, gcs_bucket)
  else:
    _install_gcsfuse(version, gcs_bucket)
  
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_gcsfuse(version)


def _goofys_test(jobfile_path, fio_metrics_obj, gcs_bucket) -> None:
  """FIO test for latest version of goofys.
  
  Args:
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
  """
  _install_goofys(gcs_bucket)
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_goofys()
  
def _fuse_test(fuse_type, fuse_type_version, jobfile_path, fio_metric_obj, gcs_bucket) -> None:
  """FIO test for specific version of given fuse type.
  
  Args:
    fuse_type(str): fuse type for fio test.
    fuse_type_version(str): fuse type version for fio test.
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
    gcs_bucket(str): GCS bucket to be mounted.
  """
  if fuse_type == 'gcsfuse':
    _gcsfuse_test(fuse_type_version, jobfile_path, fio_metric_obj, gcs_bucket)
  elif fuse_type == 'goofys':
    _goofys_test(jobfile_path, fio_metric_obj, gcs_bucket)
  else:
    app.UsageError('Unsupported fuse type!')
      
  
def main(argv) -> None:

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'fuse_type_1',
      help='Provide reading fuse_type_1 for fio test')
  parser.add_argument(
      'fuse_type_1_version',
      help='Provid the Specific version of fuse_type_1 in approprite formate')
  parser.add_argument(
      'fuse_type_2',
      help='Provide reading fuse_type_2 for fio test')
  parser.add_argument(
      'fuse_type_2_version',
      help='Provid the Specific version of fuse_type_2 in approprite formate')
  parser.add_argument(
      'jobfile_path',
      help='Provid path of the jobfile')
  parser.add_argument(
      'gcs_bucket',
      help="Provide the gcs bucket name to be mounted")
  args = parser.parse_args(argv[1:])

  fio_metrics_obj = fio_metrics.FioMetrics()
  os.system('mkdir out')

  os.system(f'echo Fuse Type 1: {args.fuse_type_1} {args.fuse_type_1_version} >> out/output.txt')
  _fuse_test(args.fuse_type_1, args.fuse_type_1_version, args.jobfile_path, fio_metrics_obj, args.gcs_bucket)

  os.system(f'echo Fuse Type 2: {args.fuse_type_2} {args.fuse_type_2_version} >> out/output.txt')
  _fuse_test(args.fuse_type_2, args.fuse_type_2_version, args.jobfile_path, fio_metrics_obj, args.gcs_bucket)


if __name__ == '__main__':
  app.run(main)
  
