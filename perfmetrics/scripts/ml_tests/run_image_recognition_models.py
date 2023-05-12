"""Python Script to automate running ML model on any Machine.

To run the script:
>> python3 run_image_recognition_model.py -- <ml_model_path> <req_file_path> [--install_gcsfuse] [--data_read_method DATA_READ_METHOD] [--gcsbucket_data_path GCSBUCKET_DATA_PATH] [--disk_data_path DISK_DATA_PATH] <directory_name>

-> <ml_model_path> Provide the path of the ML Model Script.
-> <req_file_path> Provide the path of corresponding requirements.txt file.

Flag --install_gcsfuse.
-> By defult the script will run gcsfuse from the source code 
-> If you want to install gcsfuse then put [--install_gcsfuse] flag while running

Flag --data_read_method.
-> If you want to run model only for gcsfuse put [--data_read_method gcsfuse].
-> If you want to run model only for disk data put [--data_read_method disk].
-> By defult it will run model for both gcsfuse and disk data.

Flag --gcsfuse_data_path
-> If you are reading data from GCS bucket, provide the path of relevent data inside GCS bucket.

Flag --disk_data_path.
-> Give the absolute path of the data on the disk to be processed [--disk_data_path /path/to/data]

-> <directory_name> Provide the directory_name when you want to run the model and store the output

The code takes input the ml model path, corresponding
requirements.txt file's path,directory name,installation_method,data read method
and absolute path of the data on the disk.Then it will install
gcsfuse or download the source code on the Machine and mount
the GCS bucket, after that it will install the required dependencies/modules
and then it will run the ML Model reading data from GCS bucket and/or reading
data from the disk. The output will be stored in the directory_name/output.txt
file in the same directory. At the end, it will unmount the GCS bucket.
"""

import argparse
import os
import time

from absl import app


COMMAND_NOT_FOUND_CODE = 32512
GCS_BUCKET = 'ml-models-data-gcsfuse'
GITHUB_REPO = 'https://github.com/GoogleCloudPlatform/gcsfuse'


def _check_gcsfuse():
  """Check wheather gcsfuse is installed or not on the Machine.

  Returns:
    exitcode(int)
  """
  exitcode = os.system('gcsfuse')

  return exitcode


def _install_gcsfuse() -> None:
  """Installs GCSFuse on the Machine.

  Args:
    None
  """
  os.system('''export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
            echo "deb http://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
            curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
            sudo apt-get update
            sudo apt-get install gcsfuse
            ''')


def _mount_gcsbucket(gcs_bucket, data_directory_name) -> None:
  """Mount Specific bucket to given directory.

  Args:
    gcs_bucket(str): Name of the gcs_bucket to be mounted.
    data_directory_name(str): Destination for mounting the gcs_bucket.
  """
  os.system(f'''mkdir {data_directory_name}
            gcsfuse --implicit-dirs --stat-cache-capacity 1000000 --max-conns-per-host 100 --stackdriver-export-interval=60s {gcs_bucket} {data_directory_name}
            ''')


def _unmount_gcsbucket(data_directory_name) -> None:
  """Unmount the bucket from the given directory.

  Args:
    data_directory_name(str): Directory where gcs bucket is mounted.
  """

  os.system(f'''fusermount -u {data_directory_name}
            rm -rf {data_directory_name}
            rm -rf gcsfuse
            ''')


def _run_from_source(gcs_bucket, data_directory_name) -> None:
  """Run the GCSFuse from the source code and mount specific bucket to given directory.

  Args:
    gcs_bucket(str): Name of the gcs_bucket to be mounted.
    data_directory_name(str): Destination for mounting the gcs_bucket.
  """
  os.system(f'''mkdir {data_directory_name}
            git clone {GITHUB_REPO}
            cd gcsfuse
            go run . --implicit-dirs --stat-cache-capacity 1000000 --max-conns-per-host 100 --stackdriver-export-interval=60s {gcs_bucket} ../{data_directory_name}
            cd ..
            ''')


def _run_model(directory_name, data_path, data_read_method, ml_model_path, req_file_path) -> None:
  """Automates running the ML model by installing required modules.

  Args:
    directory_name(str): Name of the directory where the model will run.
    data_path(str): Path of the required data for the model.
    data_read_method(str): Data read method for the model.
    ml_model_path(str): Path of the ml model to Run.
    req_file_path(str): Path of the corresponding requirements.txt file.
  """
  os.system(f'''sudo -H pip3 install virtualenv
            mkdir {directory_name}
            cd {directory_name}
            echo ML model reading data using {data_read_method} >> output.txt
            pip install --require-hashes -r {req_file_path} --user
            ''')
  
  start_time = int(time.time())
  
  os.system(f'''cd {directory_name}
            python3 {ml_model_path} {data_path} >> output.txt
            ''')
  
  end_time = int(time.time())

  os.system(f'''cd ..
            chmod +x populate_metrics.sh
            ./populate_metrics.sh {start_time} {end_time}
            ''')


def _run_model_using_gcsfuse(install_gcsfuse, gcsbucket_data_path, ml_model_path, req_file_path, directory_name) -> None:
  """Run model which uses GCSFuse to read data.

  Args:
    install_gcsfuse(bool): Boolean to indicates whether to install gcsfuse or not.
    gcsbucket_data_path(str): Path of data inside GCS bucket.
    ml_model_path(str): Path of the ml model to Run.
    req_file_path(str): Path of the corresponding requirements.txt file.
    directory_name(str): Name of the directory where the model will run.
  """

  data_directory_name = 'data'
  data_path = os.path.abspath(data_directory_name) + '/' + gcsbucket_data_path

  if install_gcsfuse:
    exitcode = _check_gcsfuse()
    if exitcode == COMMAND_NOT_FOUND_CODE:
      _install_gcsfuse()
    _mount_gcsbucket(GCS_BUCKET, data_directory_name)
  else:
    _run_from_source(GCS_BUCKET, data_directory_name)

  _run_model(directory_name, data_path, 'gcsfuse', ml_model_path, req_file_path)
  _unmount_gcsbucket(data_directory_name)


def main(argv) -> None:

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'ml_model_path',
      help='Provide path of the ML model script')
  parser.add_argument(
      'req_file_path',
      help='Provide the path of corresponding requirements.txt file')
  parser.add_argument(
      'directory_name',
      help='Please Provide the directory name',
      default='test')
  parser.add_argument(
      '--install_gcsfuse',
      action='store_true',
      default=False,
      help='Specify installation method',
      required=False)
  parser.add_argument(
      '--data_read_method',
      action='store',
      default='both',
      help='Specify data read method [ GCSFuse or disk data]',
      required=False)
  parser.add_argument(
      '--gcsbucket_data_path',
      action='store',
      default='None',
      help='Specify the path of data inside GCS bucket',
      required=False)
  parser.add_argument(
      '--disk_data_path',
      action='store',
      default='None',
      help='Provide Absolute disk data path',
      required=False)
  args = parser.parse_args(argv[1:])

  os.system('export PATH=/home/kbuilder/.local/bin:$PATH')

  directory_name = args.directory_name
  data_read_method = args.data_read_method

  ml_model_path = os.path.abspath(args.ml_model_path)
  req_file_path = os.path.abspath(args.req_file_path)

  # Run the model which uses GCSFuse to read data from GCSBucket
  if data_read_method == 'gcsfuse' or data_read_method == 'both':
    if args.gcsbucket_data_path == 'None':
      app.UsageError('GCS_BUCKET data path must be provided')

    _run_model_using_gcsfuse(args.install_gcsfuse, args.gcsbucket_data_path, ml_model_path, req_file_path,directory_name)

  # Run the model which reads data from the disk.
  if data_read_method == 'disk' or data_read_method == 'both':
    if args.disk_data_path == 'None':
      app.UsageError('Disk data path must be provided')

    _run_model(directory_name, args.disk_data_path, 'disk', ml_model_path, req_file_path)


if __name__ == '__main__':
  app.run(main)
