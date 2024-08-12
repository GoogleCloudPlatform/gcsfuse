# Copyright 2024 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import unittest
import renaming_benchmark
from mock import patch, call, mock_open

class TestRenamingBenchmark(unittest.TestCase):

  @patch('subprocess.call')
  @patch('time.time')
  def test_record_time_for_folder_rename(self,mock_time,mock_subprocess):
    mount_point="gcs_bucket"
    folder = {
        'name': "test_folder",
        "num_files": 1,
        "file_name_prefix": "file",
        "file_size": "1kb"
    }
    num_samples=2
    mock_time.side_effect = [1.0, 2.0, 3.0, 4.0]
    expected_time_of_operation=[1.0,1.0]
    expected_subprocess_calls=[call("mv ./gcs_bucket/test_folder ./gcs_bucket/test_folder_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder_renamed ./gcs_bucket/test_folder",shell=True)]

    time_op=renaming_benchmark._record_time_for_folder_rename(mount_point,folder,num_samples)

    self.assertEqual(time_op,expected_time_of_operation)
    mock_subprocess.assert_has_calls(expected_subprocess_calls)

  @patch('subprocess.call')
  @patch('time.time')
  def test_record_time_of_operation(self,mock_time,mock_subprocess):
    mount_point="gcs_bucket"
    dir = {
        "name": "gcs_bucket",
        "folders": {
            "folder_structure": [
                {
                    'name': "test_folder1",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                },
                {
                    'name': "test_folder2",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    num_samples=2
    mock_time.side_effect = [1.0, 2.0, 3.0, 4.0,1.0, 2.0, 3.0, 4.0]
    expected_time_of_operation={'test_folder1':[1.0,1.0] ,'test_folder2':[1.0,1.0]}
    expected_subprocess_calls=[call("mv ./gcs_bucket/test_folder1 ./gcs_bucket/test_folder1_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder1_renamed ./gcs_bucket/test_folder1",shell=True),
                               call("mv ./gcs_bucket/test_folder2 ./gcs_bucket/test_folder2_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder2_renamed ./gcs_bucket/test_folder2",shell=True),]

    time_op=renaming_benchmark._record_time_of_operation(mount_point,dir,num_samples)

    self.assertEqual(time_op,expected_time_of_operation)
    mock_subprocess.assert_has_calls(expected_subprocess_calls)

  @patch('renaming_benchmark.unmount_gcs_bucket')
  @patch('renaming_benchmark.mount_gcs_bucket')
  @patch('renaming_benchmark._record_time_of_operation')
  @patch('renaming_benchmark.log')
  def test_perform_testing_flat(self, mock_log, mock_record_time_of_operation,
      mock_mount_gcs_bucket, mock_unmount_gcs_bucket):
    dir = {
        "name":"flat_bucket",
        "folders":{
            "num_folders":1,
            "folder_structure":{
                'name': "test_folder",
                "num_files": 1,
                "file_name_prefix": "file",
                "file_size": "1kb"
            }
        }
    }
    test_type = "flat"
    num_samples = 4
    results = {}
    mount_flags = "--implicit-dirs --rename-dir-limit=1000000"
    mock_mount_gcs_bucket.return_value="flat_bucket"
    mock_record_time_of_operation.return_value = {"test_folder": [0.1, 0.2, 0.3, 0.4]}
    expected_results = {"flat": {"test_folder": [0.1, 0.2, 0.3, 0.4]}}

    renaming_benchmark._perform_testing(dir, test_type, num_samples, results)

    self.assertEqual(results, expected_results)
    # Verify calls to other functions.
    mock_mount_gcs_bucket.assert_called_once_with(dir["name"], mount_flags, mock_log)
    mock_record_time_of_operation.assert_called_once_with(mock_mount_gcs_bucket.return_value, dir, num_samples)
    mock_unmount_gcs_bucket.assert_called_once_with(dir["name"], mock_log)
    mock_log.error.assert_not_called()  # No errors should be logged

  def test_compute_metrics_from_op_time(self):
    num_samples=2
    results=[1,1]
    expected_metrics={
        'Number of samples':2,
        'Mean':1.0,
        'Median':1.0,
        'Standard Dev':0,
        'Min': 1.0,
        'Max':1.0,
        'Quantiles':{'0 %ile': 1.0, '20 %ile': 1.0, '50 %ile': 1.0,
                     '90 %ile': 1.0, '95 %ile': 1.0, '98 %ile': 1.0,
                     '99 %ile': 1.0, '99.5 %ile': 1.0, '99.9 %ile': 1.0,
                     '100 %ile': 1.0}
    }

    metrics=renaming_benchmark._compute_metrics_from_time_of_operation(num_samples,results)

    self.assertEqual(metrics,expected_metrics)

  def test_get_values_to_export(self):
    dir = {
        "name": "gcs_bucket",
        "folders": {
            "folder_structure": [
                {
                    'name': "test_folder1_0",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    metrics={
        "test_folder1_0": {
              'Number of samples':2,
              'Mean':1.0,
              'Median':1.0,
              'Standard Dev':0,
              'Min': 1.0,
              'Max':1.0,
              'Quantiles':{'0 %ile': 1.0, '20 %ile': 1.0, '50 %ile': 1.0,
                           '90 %ile': 1.0, '95 %ile': 1.0, '98 %ile': 1.0,
                           '99 %ile': 1.0, '99.5 %ile': 1.0, '99.9 %ile': 1.0,
                           '100 %ile': 1.0}
        }
    }
    test_type="flat"
    expected_export_values=[['Renaming Operation','flat',1,1,2,1.0,1.0,0,1.0,1.0,
                             1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0]]

    values_to_export = renaming_benchmark._get_values_to_export(dir,metrics,test_type)

    self.assertEqual(values_to_export,expected_export_values)

  @patch('os.chdir')
  @patch('renaming_benchmark.log')
  def test_upload_to_gsheet_no_spreadsheet_id_passed(self,mock_log,mock_os):
    worksheet='temp-worksheet'
    data=['fake data']
    spreadsheet_id=''

    exit_code = renaming_benchmark._upload_to_gsheet(worksheet,data,spreadsheet_id)

    self.assertEqual(exit_code,1)
    mock_log.error.assert_called_once_with('Empty spreadsheet id passed!')

  @patch('builtins.open', new_callable=mock_open)
  @patch('renaming_benchmark.log')
  @patch('renaming_benchmark._check_for_config_file_inconsistency')
  @patch('renaming_benchmark.json.load')
  def test_run_rename_benchmark_error_config_inconsistency(self,mock_json,mock_inconsistency,mock_log,mock_open):
    test_type="flat"
    dir_config="test-config.json"
    num_samples=10
    results=dict()
    upload_gs=True
    mock_inconsistency.return_value=1
    mock_json.return_value={}

    with self.assertRaises(SystemExit):
      renaming_benchmark._run_rename_benchmark(test_type,dir_config,num_samples,results,upload_gs)

    mock_log.error.assert_called_once_with('Exited with code 1')

  @patch('builtins.open', new_callable=mock_open)
  @patch('renaming_benchmark.log')
  @patch('renaming_benchmark._check_for_config_file_inconsistency')
  @patch('renaming_benchmark._check_if_dir_structure_exists')
  @patch('renaming_benchmark.json.load')
  def test_run_rename_benchmark_error_dir_does_not_exist(self,mock_json,mock_check_dir_exists,mock_inconsistency,mock_log,mock_open):
    test_type="flat"
    dir_config="test-config.json"
    num_samples=10
    results=dict()
    upload_gs=True
    mock_inconsistency.return_value=0
    mock_check_dir_exists.return_value=False
    mock_json.return_value={}

    with self.assertRaises(SystemExit) :
      renaming_benchmark._run_rename_benchmark(test_type,dir_config,num_samples,results,upload_gs)

    mock_log.error.assert_called_once_with("Test data does not exist.To create test data, run : \
        python3 generate_folders_and_files.py {} ".format(dir_config))

  @patch('renaming_benchmark.SPREADSHEET_ID','temp-gsheet-id')
  @patch('renaming_benchmark.WORKSHEET_NAME_FLAT','flat-sheet')
  @patch('builtins.open', new_callable=mock_open)
  @patch('renaming_benchmark.log')
  @patch('renaming_benchmark._check_for_config_file_inconsistency')
  @patch('renaming_benchmark._check_if_dir_structure_exists')
  @patch('renaming_benchmark._perform_testing')
  @patch('renaming_benchmark._parse_results')
  @patch('renaming_benchmark._get_values_to_export')
  @patch('renaming_benchmark._upload_to_gsheet')
  @patch('renaming_benchmark.json.load')
  def test_run_rename_benchmark_upload_true(self,mock_json,mock_upload,mock_get_values,mock_parse_results,mock_perform_testing,mock_check_dir_exists,mock_inconsistency,mock_log,mock_open):
    test_type="flat"
    dir_config="test-config.json"
    num_samples=10
    results={'flat':''}
    upload_gs=True
    worksheet= 'flat-sheet'
    spreadsheet_id='temp-gsheet-id'
    mock_inconsistency.return_value=0
    mock_check_dir_exists.return_value=True
    mock_parse_results.return_value={'key':'val'}
    mock_get_values.return_value=[['testdata','testdata2']]
    mock_upload.return_value=0
    mock_json.return_value={}


    renaming_benchmark._run_rename_benchmark(test_type,dir_config,num_samples,results,upload_gs)

    mock_log.info.assert_called_with('Uploading files to the Google Sheet\n')
    mock_upload.assert_called_with(worksheet,[['testdata','testdata2']],spreadsheet_id)


if __name__ == '__main__':
  unittest.main()
