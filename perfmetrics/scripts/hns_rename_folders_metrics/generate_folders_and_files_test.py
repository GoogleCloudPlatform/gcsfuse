# Copyright 2024 Google LLC
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
import subprocess
import unittest
import generate_folders_and_files
import mock
from mock import patch, call ,mock_open


class TestCheckForConfigFileInconsistency(unittest.TestCase):
  def test_missing_bucket_name(self):
    config = {}
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 1)

  def test_missing_keys_from_folder(self):
    config = {
        "name": "test_bucket",
        "folders": {}
    }
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 1)

  def test_missing_keys_from_nested_folder(self):
    config = {
        "name": "test_bucket",
        "nested_folders": {}
    }
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 1)

  def test_folders_num_folder_mismatch(self):
    config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 2,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 10,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 1)

  def test_nested_folders_num_folder_mismatch(self):
    config = {
        "name": "test_bucket",
        "nested_folders": {
            "folder_name": "test_nested_folder",
            "num_folders": 2,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 10,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 1)

  def test_valid_config(self):
    config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = generate_folders_and_files._check_for_config_file_inconsistency(
        config)
    self.assertEqual(result, 0)


class TestListDirectory(unittest.TestCase):

  @patch('subprocess.check_output')
  @patch('generate_folders_and_files._logmessage')
  def test_listing_at_non_existent_path(self, mock_logmessage,mock_check_output):
    mock_check_output.side_effect = subprocess.CalledProcessError(
        returncode=1,
        cmd="gcloud storage ls gs://fake_bkt",
        output=b'Error while listing')

    dir_list = generate_folders_and_files._list_directory("gs://fake_bkt")

    self.assertEqual(dir_list, None)
    mock_logmessage.assert_called_once_with('Error while listing','error')

  @patch('subprocess.check_output')
  def test_listing_directory(self, mock_check_output):
    mock_check_output.return_value = b'gs://fake_bkt/fake_folder_0/\n' \
                                     b'gs://fake_bkt/fake_folder_1/\n' \
                                     b'gs://fake_bkt/nested_fake_folder/\n'
    expected_dir_list = ["gs://fake_bkt/fake_folder_0/",
                         "gs://fake_bkt/fake_folder_1/",
                         "gs://fake_bkt/nested_fake_folder/"]

    dir_list = generate_folders_and_files._list_directory("gs://fake_bkt")

    self.assertEqual(dir_list, expected_dir_list)


class TestCompareFolderStructure(unittest.TestCase):

  @patch('generate_folders_and_files._list_directory')
  def test_folder_structure_matches(self,mock_listdir):
    mock_listdir.return_value=['test_file_1.txt']
    test_folder={
        "name": "test_folder",
        "num_files": 1,
        "file_name_prefix": "test_file",
        "file_size": "1kb"
    }
    test_folder_url='gs://temp_folder_url'

    match = generate_folders_and_files._compare_folder_structure(test_folder, test_folder_url)

    self.assertEqual(match,True)

  @patch('generate_folders_and_files._list_directory')
  def test_folder_structure_mismatches(self,mock_listdir):
    mock_listdir.return_value=['test_file_1.txt']
    test_folder={
        "name": "test_folder",
        "num_files": 2,
        "file_name_prefix": "test_file",
        "file_size": "1kb"
    }
    test_folder_url='gs://temp_folder_url'

    match = generate_folders_and_files._compare_folder_structure(test_folder, test_folder_url)

    self.assertEqual(match,False)

  @patch('generate_folders_and_files._list_directory')
  def test_folder_does_not_exist_in_gcs_bucket(self,mock_listdir):
    mock_listdir.side_effect=subprocess.CalledProcessError(
        returncode=1,
        cmd="gcloud storage ls gs://fake_bkt/folder_does_not_exist",
        output=b'Error while listing')
    test_folder={
        "name": "test_folder",
        "num_files": 1,
        "file_name_prefix": "test_file",
        "file_size": "1kb"
    }
    test_folder_url='gs://fake_bkt/folder_does_not_exist'

    match = generate_folders_and_files._compare_folder_structure(test_folder, test_folder_url)

    self.assertEqual(match,False)
    self.assertRaises(subprocess.CalledProcessError)


class TestCheckIfDirStructureExists(unittest.TestCase):

  @patch("generate_folders_and_files._list_directory")
  def test_dir_already_exists_in_gcs_bucket(self, mock_list_directory):
    mock_list_directory.side_effect = [
        ["gs://test_bucket/test_folder/", "gs://test_bucket/nested/"],
        ["gs://test_bucket/test_folder/file_1.txt"],
        ["gs://test_bucket/nested/test_folder/"],
        ["gs://test_bucket/nested/test_folder/file_1.txt"]
    ]
    dir_config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 1,
            "folder_structure": [
                {
                  "name": "test_folder",
                  "num_files": 1,
                  "file_name_prefix": "file",
                  "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }

    dir_present = generate_folders_and_files._check_if_dir_structure_exists(
      dir_config)

    self.assertEqual(dir_present, 1)

  @patch("generate_folders_and_files._list_directory")
  def test_dir_does_not_exist_in_gcs_bucket(self, mock_list_directory):
    mock_list_directory.side_effect = [
        ["gs://test_bucket/test_folder/", "gs://test_bucket/nested/"],
        ["gs://test_bucket/test_folder/file_1.txt",
         "gs://test_bucket/test_folder/file_1.txt"],
        ["gs://test_bucket/nested/test_folder/"],
        ["gs://test_bucket/nested/test_folder/file_1.txt"]
    ]
    dir_config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }

    dir_present = generate_folders_and_files._check_if_dir_structure_exists(
      dir_config)

    self.assertEqual(dir_present, 0)


class TestDeleteExistingDataInGcsBucket(unittest.TestCase):

  @patch('subprocess.check_output')
  @patch('generate_folders_and_files._logmessage')
  def test_deleting_failure(self, mock_logmessage,
      mock_check_output):
    mock_check_output.side_effect = subprocess.CalledProcessError(
        returncode=1,
        cmd="gcloud alpha storage rm -r gs://fake_bkt",
        output=b'Error while deleting')

    exit_code = generate_folders_and_files\
      ._delete_existing_data_in_gcs_bucket("fake_bkt")

    self.assertEqual(exit_code, 1)
    mock_logmessage.assert_called_once_with('Error while deleting','error')

  @patch('subprocess.check_output')
  def test_deleting_success(self, mock_check_output):
    mock_check_output.return_value = 0

    exit_code = generate_folders_and_files \
      ._delete_existing_data_in_gcs_bucket("fake_bkt")

    self.assertEqual(exit_code, 0)


class TestGenerateFilesAndUploadToGcsBucket(unittest.TestCase):

  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('generate_folders_and_files.BATCH_SIZE', 10)
  @patch('builtins.open', new_callable=mock_open)
  @patch('os.listdir')
  @patch('subprocess.Popen')
  @patch('subprocess.call')
  @patch('generate_folders_and_files._logmessage')
  def test_files_generation_and_upload(self, mock_logmessage, mock_call,
      mock_popen, mock_listdir, mock_open
  ):
    """
    Tests that files are created,copied to destination bucket,deleted from the
    temporary directory and the log message is written correctly.
    """
    mock_listdir.return_value = ['file1.txt']
    mock_popen.return_value.communicate.return_value = 0
    destination_blob_name = 'gs://fake-bucket'
    num_of_files = 1
    file_size_unit = 'MB'
    file_size = 1
    filename_prefix = 'file'
    temp_file='./tmp/data_gen/file_1.txt'
    expected_size = 1024 * 1024 * int(file_size)
    expected_log_message = f'{num_of_files}/{num_of_files} files uploaded to {destination_blob_name}\n'

    exit_code = generate_folders_and_files._generate_files_and_upload_to_gcs_bucket(
        destination_blob_name, num_of_files, file_size_unit, file_size,
        filename_prefix)

    # Assert that temp_file is opened.
    mock_open.assert_called_once_with(temp_file, 'wb')
    # Assert that 'truncate' was called with the expected size.
    mock_open.return_value.truncate.assert_called_once_with(expected_size)
    # Assert that upload started to GCS bucket and exit code is 0 indicating
    # successful upload.
    mock_popen.assert_called_once_with(
        f'gcloud storage cp --recursive {generate_folders_and_files.TEMPORARY_DIRECTORY}/* {destination_blob_name}',
        shell=True)
    self.assertEqual(exit_code, 0)
    # Assert that files are deleted and correct logmessage is written.
    mock_call.assert_called_once_with(
        f'rm -rf {generate_folders_and_files.TEMPORARY_DIRECTORY}/*',
        shell=True)
    mock_logmessage.assert_has_calls([call(expected_log_message,'info')])


  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('generate_folders_and_files.BATCH_SIZE', 10)
  @patch('builtins.open', new_callable=mock_open)
  @patch('os.listdir')
  def test_files_not_created_locally(self, mock_listdir, mock_open):
    """
    Tests that files are created,copied to destination bucket,deleted from the
    temporary directory and the log message is written correctly.
    """
    mock_listdir.return_value = []
    destination_blob_name = 'gs://fake-bucket'
    num_of_files = 1
    file_size_unit = 'MB'
    file_size = 1
    filename_prefix = 'file'
    temp_file='./tmp/data_gen/file_1.txt'
    expected_size = 1024 * 1024 * int(file_size)

    exit_code = generate_folders_and_files._generate_files_and_upload_to_gcs_bucket(
        destination_blob_name, num_of_files, file_size_unit, file_size,
        filename_prefix)

    # Assert that temp_file is opened.
    mock_open.assert_has_calls([call(temp_file, 'wb')])
    # Assert that 'truncate' was called with the expected size and file is
    # created.
    mock_open.return_value.truncate.assert_called_once_with(expected_size)
    # Assert that error log message is written to logfile.
    mock_open.assert_has_calls([call().write("Files were not created locally")])
    self.assertEqual(exit_code, 1)


  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('generate_folders_and_files.BATCH_SIZE', 10)
  @patch('builtins.open', new_callable=mock_open)
  @patch('os.listdir')
  @patch('subprocess.Popen')
  def test_files_upload_failure(self, mock_popen, mock_listdir, mock_open):
    """
    Tests that files are created,copied to destination bucket,deleted from the
    temporary directory and the log message is written correctly.
    """
    mock_listdir.return_value = ['file1.txt']
    upload_cmd="gcloud storage cp --recursive ./tmp/data_gen/ gs://fake-bucket"
    mock_popen.side_effect=subprocess.CalledProcessError(returncode=1,cmd=upload_cmd)
    destination_blob_name = 'gs://fake-bucket'
    num_of_files = 1
    file_size_unit = 'MB'
    file_size = 1
    filename_prefix = 'file'
    temp_file='./tmp/data_gen/file_1.txt'
    expected_size = 1024 * 1024 * int(file_size)

    exit_code = generate_folders_and_files._generate_files_and_upload_to_gcs_bucket(
        destination_blob_name, num_of_files, file_size_unit, file_size,
        filename_prefix)

    # Assert that temp_file is opened.
    mock_open.assert_has_calls([call(temp_file, 'wb')])
    # Assert that 'truncate' was called with the expected size.
    mock_open.return_value.truncate.assert_called_once_with(expected_size)
    # Assert that upload to GCS bucket was attempted.
    mock_popen.assert_called_once_with(
        f'gcloud storage cp --recursive {generate_folders_and_files.TEMPORARY_DIRECTORY}/* {destination_blob_name}',
        shell=True)
    # Assert that except block is executed due to the upload failure.
    mock_open.assert_has_calls([call().write('Issue while uploading files to GCS bucket.Aborting...')])
    self.assertEqual(exit_code, 1)


class TestParseAndGenerateDirStructure(unittest.TestCase):

  @patch('generate_folders_and_files._logmessage')
  def test_no_dir_structure_passed(self,mock_logmessage):
    dir_str={}

    exit_code=generate_folders_and_files._parse_and_generate_directory_structure(dir_str)

    self.assertEqual(exit_code,1)
    mock_logmessage.assert_called_once_with("Directory structure not specified via config file.",'error')


  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('subprocess.call')
  @patch('generate_folders_and_files._logmessage')
  @patch('generate_folders_and_files._generate_files_and_upload_to_gcs_bucket')
  def test_valid_dir_str_with_folders(self, mock_generate, mock_log, mock_subprocess):
    dir_str = {
        "name": "test_bucket",
        "folders": {
            "num_folders":1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 2,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "test_nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_nested_folder1",
                    "num_files": 2,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    mock_generate.return_value= 0
    expected_subprocess_calls = [
        call(['mkdir', '-p', generate_folders_and_files.TEMPORARY_DIRECTORY]),
        call(['rm', '-r', generate_folders_and_files.TEMPORARY_DIRECTORY])
    ]
    expected_generate_and_upload_calls = [
        call( 'gs://test_bucket/test_folder/',2,'kb',1,'file'),
        call('gs://test_bucket/test_nested/test_nested_folder1/',2,'kb',1,'file')
    ]

    exit_code = generate_folders_and_files._parse_and_generate_directory_structure(dir_str)

    self.assertEqual(exit_code, 0)
    # Verify subprocess calls.
    mock_subprocess.assert_has_calls(expected_subprocess_calls)
    # Verify log messages.
    mock_log.assert_any_call('Making a temporary directory.\n', 'info')
    mock_log.assert_any_call('Deleting the temporary directory.\n', 'info')
    # Verify generate_files_and_upload_to_gcs_bucket call.
    mock_generate.assert_has_calls(expected_generate_and_upload_calls)


  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('subprocess.call')
  @patch('generate_folders_and_files._logmessage')
  @patch('generate_folders_and_files._generate_files_and_upload_to_gcs_bucket')
  def test_create_folder_failure(self, mock_generate, mock_log, mock_subprocess):
    dir_str = {
        "name": "test_bucket",
        "folders": {
            "num_folders":1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 2,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    mock_generate.return_value= 1
    expected_subprocess_calls = [
        call(['mkdir', '-p', generate_folders_and_files.TEMPORARY_DIRECTORY])
    ]
    expected_generate_and_upload_calls = [
        call( 'gs://test_bucket/test_folder/',2,'kb',1,'file')
    ]

    exit_code = generate_folders_and_files._parse_and_generate_directory_structure(dir_str)

    self.assertEqual(exit_code, 1)
    # Verify log messages.
    mock_log.assert_any_call('Making a temporary directory.\n', 'info')
    # Verify generate_files_and_upload_to_gcs_bucket call.
    mock_generate.assert_has_calls(expected_generate_and_upload_calls)
    # Verify subprocess calls.
    mock_subprocess.assert_has_calls(expected_subprocess_calls)


class TestParseAndGenerateDirStructure(unittest.TestCase):

  @patch('generate_folders_and_files.LOG_ERROR','error')
  @patch('generate_folders_and_files._logmessage')
  def test_no_dir_structure_passed(self,mock_logmessage):
    dir_str={}

    exit_code=generate_folders_and_files._parse_and_generate_directory_structure(dir_str)

    self.assertEqual(exit_code,1)
    mock_logmessage.assert_called_once_with("Directory structure not specified via config file.",generate_folders_and_files.LOG_ERROR)


  @patch('generate_folders_and_files.TEMPORARY_DIRECTORY', './tmp/data_gen')
  @patch('subprocess.call')
  @patch('generate_folders_and_files._logmessage')
  @patch('generate_folders_and_files._generate_files_and_upload_to_gcs_bucket')
  def test_valid_dir_str_with_folders(self, mock_generate, mock_log, mock_subprocess):
    dir_str = {
        "name": "test_bucket",
        "folders": {
            "num_folders":1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 2,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "test_nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_nested_folder1",
                    "num_files": 2,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }

    exit_code = generate_folders_and_files._parse_and_generate_directory_structure(dir_str)

    self.assertEqual(exit_code, 0)
    # Verify subprocess calls
    expected_subprocess_calls = [
        call(['mkdir', '-p', generate_folders_and_files.TEMPORARY_DIRECTORY]),
        call(['rm', '-r', generate_folders_and_files.TEMPORARY_DIRECTORY])
    ]
    mock_subprocess.assert_has_calls(expected_subprocess_calls)
    # Verify log messages
    mock_log.assert_any_call('Making a temporary directory.\n', generate_folders_and_files.LOG_INFO)
    mock_log.assert_any_call('Deleting the temporary directory.\n', generate_folders_and_files.LOG_INFO)
    # Verify generate_files_and_upload_to_gcs_bucket call
    expected_generate_and_upload_calls = [
        call( 'gs://test_bucket/test_folder/',2,'kb',1,'file'),
        call('gs://test_bucket/test_nested/test_nested_folder1/',2,'kb',1,'file')
    ]
    mock_generate.assert_has_calls(expected_generate_and_upload_calls)


if __name__ == '__main__':
  unittest.main()
