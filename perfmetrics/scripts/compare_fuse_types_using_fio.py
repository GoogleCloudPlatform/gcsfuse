"""Python script to do FIO test for GCSFuse Vs GCSFuse/Goofys.

To run the script:
>> python3 compare_fuse_types_using_fio.py -- method1 method1_version method2 method2_version jobfile_path
-> method1 & method can be gcsfuse or goofys.
-> Incase of goofys specify the version as latest.

The code takes input the fio test method and its version, jobfile_path as input.
Then it will perform the fio test for each of the given method at a specific version
and extract fio_metrics to output.txt file.
"""
import argparse
import os
from fio import fio_metrics

from absl import app

GCS_BUCKET = 'gke_load_test'
GOOFYS_REPO = 'https://github.com/kahing/goofys'


def _install_gcsfuse(version) -> None:
  """Install gcsfuse with Specific version.
  Args:
    version(str): gcsfuse version to be installed.  
  """
  os.system(f'''curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v{version}/gcsfuse_{version}_amd64.deb
            sudo dpkg --install gcsfuse_{version}_amd64.deb
            mkdir gcs
            gcsfuse --implicit-dirs --max-conns-per-host 100 --disable-http2 {GCS_BUCKET} gcs
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
            ''')


def _install_goofys() -> None:
  """Install latest version of goofys.
  """
  os.system(f'''git clone {GOOFYS_REPO}
            export GOPATH=$HOME/work
            mkdir gcs
            cd goofys
            go run . gs://{GCS_BUCKET} ../gcs
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


def _gcsfuse_test(version, jobfile_path, fio_metrics_obj) -> None:
  """FIO test for gcsfuse of given version.
  Args:
    version(str): gcsfuse version to perform fio test.
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
  """
  _install_gcsfuse(version)
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_gcsfuse(version)


def _goofys_test(jobfile_path, fio_metrics_obj) -> None:
  """FIO test for latest version of goofys.
  
  Args:
    jobfile(str): path of the job file.
    fio_metrics_obj(str): object for extracting fio metrics.
  """
  _install_goofys()
  _run_fio_test(jobfile_path, fio_metrics_obj)
  _remove_goofys()


def main(argv) -> None:

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'method1',
      help='Provide reading method1 for fio test')
  parser.add_argument(
      'method1_version',
      help='Provid the Specific version of method1 in approprite formate')
  parser.add_argument(
      'method2',
      help='Provide reading method2 for fio test')
  parser.add_argument(
      'method2_version',
      help='Provid the Specific version of method2 in approprite formate')
  parser.add_argument(
      'jobfile_path',
      help='Provid path of the jobfile')
  args = parser.parse_args(argv[1:])

  fio_metrics_obj = fio_metrics.FioMetrics()
  os.system('mkdir out')

  os.system(f'echo Method1: {args.method1} {args.method1_version} >> out/output.txt')
  if args.method1 == 'gcsfuse':
    _gcsfuse_test(args.method1_version, args.jobfile_path, fio_metrics_obj)
  else:
    _goofys_test(args.jobfile_path, fio_metrics_obj)

  os.system(f'echo Method2: {args.method2} {args.method2_version} >> out/output.txt')
  if args.method2 == 'gcsfuse':
    _gcsfuse_test(args.method2_version, args.jobfile_path, fio_metrics_obj)
  else:
    _goofys_test(args.jobfile_path, fio_metrics_obj)


if __name__ == '__main__':
  app.run(main)
  
