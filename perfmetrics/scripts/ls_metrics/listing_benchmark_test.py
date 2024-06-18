# Copyright 2023 Google Inc. All Rights Reserved.
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

"""Tests for listing_benchmark."""

import unittest
from unittest import mock

import directory_pb2 as directory_proto
from google.protobuf.json_format import ParseDict
import listing_benchmark
from mock import patch, call

# (Type 1) - 0 levels deep directory structure.
DIRECTORY_STRUCTURE1 = {
    'name': 'fake_bucket'
}

# (Type 2) - 1 level deep directory structure with an empty testing folder.
DIRECTORY_STRUCTURE2 = {
    'name': 'fake_bucket',
    'num_folders': 3,
    'num_files': 1,
    'file_size': '1kb',
    'file_name_prefix': 'file',
    'folders': [
        {
            'name': '2KB_3files_0subdir',
            'num_files': 3,
            'file_name_prefix': 'file',
            'file_size': '2kb'
        },
        {
            'name': '1KB_2files_0subdir',
            'num_files': 2,
            'file_size': '1kb',
            'file_name_prefix': 'file'
        },
        {
            'name': '1KB_0files_0subdir'
        }
    ]
}

# (Type 3) - Multilevel deep directory structure with many edge cases embedded.
DIRECTORY_STRUCTURE3 = {
    'name': 'fake_bucket',
    'num_folders': 3,
    'num_files': 0,
    'folders': [
        {
            'name': '1KB_4files_3subdir',
            'num_files': 4,
            'file_name_prefix': 'file',
            'file_size': '1kb',
            'num_folders': 3,
            'folders': [
                {
                    'name': 'subdir1',
                    'num_files': 1,
                    'file_name_prefix': 'file',
                    'file_size': '1kb',
                    'num_folders': 2,
                    'folders': [
                        {
                            'name': 'subsubdir1',
                            'num_files': 2,
                            'file_name_prefix': 'file',
                            'file_size': '1kb'
                        },
                        {
                            'name': 'subsubdir2'
                        }
                    ]
                },
                {
                    'name': 'subdir2',
                    'num_files': 1,
                    'file_name_prefix': 'file',
                    'file_size': '1kb'
                },
                {
                    'name': 'subdir3',
                    'num_files': 1,
                    'file_name_prefix': 'file',
                    'file_size': '1kb',
                    'num_folders': 1,
                    'folders': [
                        {
                            'name': 'subsubdir1',
                            'num_files': 1,
                            'file_name_prefix': 'file',
                            'file_size': '1kb'
                        }
                    ]
                }
            ]
        },
        {
            'name': '2KB_3files_1subdir',
            'num_files': 3,
            'file_name_prefix': 'file',
            'file_size': '2kb',
            'num_folders': 1,
            'folders': [
                {
                    'name': 'subdir1'
                }
            ]
        },
        {
            'name': '1KB_1files_0subdir',
            'num_files': 1,
            'file_size': '1kb',
            'file_name_prefix': 'file'
        }
    ]
}

# List of latencies (msec) of list operation to test _parse_results method.
METRICS1 = [1.234, 0.995, 0.121, 0.222, 0.01709]
METRICS2 = [90.45, 1.95, 0.334, 7.090, 0.001]
METRICS3 = [100, 7, 6, 51, 21]

SAMPLE_METRIC_FOR_DIRECTORY_STRUCTURE_2 = {
    '2KB_3files_0subdir':
      {
          'Test Desc.': 'fake_test',
          'Number of samples': 5,
          'Mean': 0.518,
          'Median': 0.222,
          'Standard Dev': 0.556,
          'Quantiles':
            {
                '0 %ile': 0.017,
                '20 %ile': 0.1,
                '50 %ile': 0.222,
                '90 %ile': 1.138,
                '95 %ile': 1.186,
                '98 %ile': 1.215,
                '99 %ile': 1.224,
                '99.5 %ile': 1.229,
                '99.9 %ile': 1.233,
                '100 %ile': 1.234
            }
      },
    '1KB_2files_0subdir':
      {
          'Test Desc.': 'fake_test',
          'Number of samples': 5,
          'Mean': 19.965,
          'Median': 1.95,
          'Standard Dev': 39.504,
          'Quantiles':
            {
                '0 %ile': 0.001,
                '20 %ile': 0.267,
                '50 %ile': 1.95,
                '90 %ile': 57.106,
                '95 %ile': 73.778,
                '98 %ile': 83.781,
                '99 %ile': 87.116,
                '99.5 %ile': 88.783,
                '99.9 %ile': 90.117,
                '100 %ile': 90.45
            }
      },
    '1KB_0files_0subdir':
      {
          'Test Desc.': 'fake_test',
          'Number of samples': 5,
          'Mean': 37,
          'Median': 21,
          'Standard Dev': 39.63,
          'Quantiles':
            {
                '0 %ile': 6.0,
                '20 %ile': 6.8,
                '50 %ile': 21.0,
                '90 %ile': 80.4,
                '95 %ile': 90.2,
                '98 %ile': 96.08,
                '99 %ile': 98.04,
                '99.5 %ile': 99.02,
                '99.9 %ile': 99.804,
                '100 %ile': 100.0
            }
      }
}

# Converting JSON to protobuf.
DIRECTORY_STRUCTURE1 = ParseDict(
    DIRECTORY_STRUCTURE1, directory_proto.Directory())
DIRECTORY_STRUCTURE2 = ParseDict(
    DIRECTORY_STRUCTURE2, directory_proto.Directory())
DIRECTORY_STRUCTURE3 = ParseDict(
    DIRECTORY_STRUCTURE3, directory_proto.Directory())

WORKSHEET_NAME = 'ls_metrics_gcsfuse'

DIRECTORY_STRUCTURE_2_VALUES = [
    ['fake_test','ls', 3, 0, 5, 0.518, 0.222, 0.556, 0.017, 0.1, 0.222, 1.138, 1.186, 1.215, 1.224, 1.229, 1.233, 1.234],
    ['fake_test','ls', 2, 0, 5, 19.965, 1.95, 39.504, 0.001, 0.267, 1.95, 57.106, 73.778, 83.781, 87.116, 88.783, 90.117, 90.45],
    ['fake_test','ls', 0, 0, 5, 37, 21, 39.63, 6.0, 6.8, 21.0, 80.4, 90.2, 96.08, 98.04, 99.02, 99.804, 100.0]
]

class ListingBenchmarkTest(unittest.TestCase):

  def test_num_files_and_folders_single_level_dir(self):
    num_files, num_folders = listing_benchmark._count_number_of_files_and_folders(
        DIRECTORY_STRUCTURE1, 0, 0)
    self.assertEqual(num_files, 0)
    self.assertEqual(num_folders, 0)

  def test_num_files_and_folders_double_level_dir(self):
    num_files, num_folders = listing_benchmark._count_number_of_files_and_folders(
        DIRECTORY_STRUCTURE2, 0, 0)
    self.assertEqual(num_files, 6)
    self.assertEqual(num_folders, 3)

  def test_num_files_and_folders_multi_level_dir(self):
    num_files, num_folders = listing_benchmark._count_number_of_files_and_folders(
        DIRECTORY_STRUCTURE3, 0, 0)
    self.assertEqual(num_files, 14)
    self.assertEqual(num_folders, 10)

  def test_parse_results_single_level_dir(self):
    metrics = listing_benchmark._parse_results(
        DIRECTORY_STRUCTURE1.folders, {}, 'fake_test', 5)
    self.assertEqual(metrics, {})

  def test_get_values_to_export(self):
    values = listing_benchmark._get_values_to_export(
        DIRECTORY_STRUCTURE2.folders, SAMPLE_METRIC_FOR_DIRECTORY_STRUCTURE_2,
        'ls')
    self.assertEqual(values, DIRECTORY_STRUCTURE_2_VALUES)

  @patch('listing_benchmark.gsheet.write_to_google_sheet')
  def test_export_to_google_sheet(self, mock_sheet):
    listing_benchmark._export_to_gsheet(
        WORKSHEET_NAME, DIRECTORY_STRUCTURE_2_VALUES)
    self.assertEqual(mock_sheet.call_args_list, [
        call('ls_metrics_gcsfuse', DIRECTORY_STRUCTURE_2_VALUES)
    ])

  def test_parse_results_double_level_dir(self):
    metrics = listing_benchmark._parse_results(DIRECTORY_STRUCTURE2.folders, {
        '2KB_3files_0subdir': METRICS1,
        '1KB_2files_0subdir': METRICS2,
        '1KB_0files_0subdir': METRICS3
    }, 'fake_test', 5)
    self.assertEqual(metrics, SAMPLE_METRIC_FOR_DIRECTORY_STRUCTURE_2)

  @patch('listing_benchmark.subprocess.call', return_value=1)
  @patch('listing_benchmark.time.time', return_value=1)
  def test_record_time_of_operation_same_time(
      self, mock_time, mock_subprocess_call):
    result_list = listing_benchmark._record_time_of_operation(
        'ls', 'fakepath/', 5)
    self.assertEqual(mock_subprocess_call.call_count, 5)
    self.assertEqual(result_list, [0, 0, 0, 0, 0])

  @patch('listing_benchmark.subprocess.call', return_value=1)
  @patch('listing_benchmark.time.time')
  def test_record_time_of_operation_different_time(
      self, mock_time, mock_subprocess_call):
    mock_time.side_effect = [1, 2, 3, 5]
    result_list = listing_benchmark._record_time_of_operation(
        'ls', 'fakepath/', 2)
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(result_list, [1000, 2000])

  @patch('listing_benchmark._record_time_of_operation')
  def test_perform_testing_single_level_dir(
      self, mock_record_time_of_operation):
    mock_record_time_of_operation.return_value = [1, 1, 1]
    gcs_bucket_results, persistent_disk_results = listing_benchmark._perform_testing(
        DIRECTORY_STRUCTURE1.folders, 'fake_bucket', 'fake_disk', 3, 'ls -R')
    self.assertEqual(gcs_bucket_results, persistent_disk_results)
    self.assertFalse(mock_record_time_of_operation.called)
    self.assertEqual(gcs_bucket_results, {})

  @patch('listing_benchmark._record_time_of_operation')
  def test_perform_testing_double_level_dir(
      self, mock_record_time_of_operation):
    mock_record_time_of_operation.return_value = [1, 1, 1]
    gcs_bucket_results, persistent_disk_results = listing_benchmark._perform_testing(
        DIRECTORY_STRUCTURE2.folders, 'fake_bucket', 'fake_disk', 3, 'ls -R')
    self.assertEqual(gcs_bucket_results, persistent_disk_results)
    self.assertTrue(mock_record_time_of_operation.called)
    self.assertEqual(gcs_bucket_results, {
        '2KB_3files_0subdir': [1, 1, 1],
        '1KB_2files_0subdir': [1, 1, 1],
        '1KB_0files_0subdir': [1, 1, 1]
    })

  @patch('listing_benchmark._record_time_of_operation', return_value=[1])
  def test_perform_testing_multi_level_dir(self, mock_record_time_of_operation):
    mock_record_time_of_operation.return_value = [1, 1]
    gcs_bucket_results, persistent_disk_results = listing_benchmark._perform_testing(
        DIRECTORY_STRUCTURE3.folders, 'fake_bucket', 'fake_disk', 2, 'ls -R')
    self.assertEqual(gcs_bucket_results, persistent_disk_results)
    self.assertTrue(mock_record_time_of_operation.called)
    self.assertEqual(gcs_bucket_results, {
        '1KB_4files_3subdir': [1, 1],
        '2KB_3files_1subdir': [1, 1],
        '1KB_1files_0subdir': [1, 1]
    })

  @patch('listing_benchmark.subprocess.call', return_value=0)
  @patch('listing_benchmark.generate_files.generate_files_and_upload_to_gcs_bucket', return_value=0)
  def test_create_directory_structure_single_level_dir(
      self, mock_generate_files, mock_subprocess_call):
    exit_code = listing_benchmark._create_directory_structure(
        'fake_bucket_url/', 'fake_disk_url/', DIRECTORY_STRUCTURE1, True)
    self.assertEqual(exit_code, 0)
    self.assertEqual(mock_subprocess_call.call_count, 1)
    self.assertEqual(mock_generate_files.call_count, 0)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_disk_url/', shell=True)
    ])
    self.assertEqual(mock_generate_files.call_args_list, [])

  @patch('listing_benchmark.subprocess.call', return_value=0)
  @patch('listing_benchmark.generate_files.generate_files_and_upload_to_gcs_bucket', return_value=0)
  def test_create_directory_structure_double_level_dir(
      self, mock_generate_files, mock_subprocess_call):
    exit_code = listing_benchmark._create_directory_structure(
        'fake_bucket_url/', 'fake_disk_url/', DIRECTORY_STRUCTURE2, True)
    self.assertEqual(exit_code, 0)
    self.assertEqual(mock_subprocess_call.call_count, 4)
    self.assertEqual(mock_generate_files.call_count, 3)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_disk_url/', shell=True),
        call('mkdir fake_disk_url/2KB_3files_0subdir/', shell=True),
        call('mkdir fake_disk_url/1KB_2files_0subdir/', shell=True),
        call('mkdir fake_disk_url/1KB_0files_0subdir/', shell=True)
    ])
    self.assertEqual(mock_generate_files.call_args_list, [
        call('fake_bucket_url/', 1, 'kb', 1, 'file', 'fake_disk_url/', True),
        call('fake_bucket_url/2KB_3files_0subdir/', 3, 'kb', 2, 'file',
             'fake_disk_url/2KB_3files_0subdir/', True),
        call('fake_bucket_url/1KB_2files_0subdir/', 2, 'kb', 1, 'file',
             'fake_disk_url/1KB_2files_0subdir/', True)
    ])

  @patch('listing_benchmark.subprocess.call', return_value=0)
  @patch('listing_benchmark.generate_files.generate_files_and_upload_to_gcs_bucket', return_value=0)
  def test_create_directory_structure_multi_level_dir(
      self, mock_generate_files, mock_subprocess_call):
    exit_code = listing_benchmark._create_directory_structure(
        'fake_bucket_url/', 'fake_disk_url/', DIRECTORY_STRUCTURE3, True)
    self.assertEqual(exit_code, 0)
    self.assertEqual(mock_subprocess_call.call_count, 11)
    self.assertEqual(mock_generate_files.call_count, 8)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_disk_url/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir1/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir1/subsubdir1/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir1/subsubdir2/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir2/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir3/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/subdir3/subsubdir1/', shell=True),
        call('mkdir fake_disk_url/2KB_3files_1subdir/', shell=True),
        call('mkdir fake_disk_url/2KB_3files_1subdir/subdir1/', shell=True),
        call('mkdir fake_disk_url/1KB_1files_0subdir/', shell=True)
    ])
    self.assertEqual(mock_generate_files.call_args_list, [
        call('fake_bucket_url/1KB_4files_3subdir/', 4, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/', True),
        call('fake_bucket_url/1KB_4files_3subdir/subdir1/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/subdir1/', True),
        call('fake_bucket_url/1KB_4files_3subdir/subdir1/subsubdir1/', 2, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/subdir1/subsubdir1/', True),
        call('fake_bucket_url/1KB_4files_3subdir/subdir2/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/subdir2/', True),
        call('fake_bucket_url/1KB_4files_3subdir/subdir3/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/subdir3/', True),
        call('fake_bucket_url/1KB_4files_3subdir/subdir3/subsubdir1/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/subdir3/subsubdir1/', True),
        call('fake_bucket_url/2KB_3files_1subdir/', 3, 'kb', 2,
             'file', 'fake_disk_url/2KB_3files_1subdir/', True),
        call('fake_bucket_url/1KB_1files_0subdir/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_1files_0subdir/', True)
    ])

  @patch('listing_benchmark.subprocess.call', return_value=0)
  @patch('listing_benchmark.generate_files.generate_files_and_upload_to_gcs_bucket', return_value=1)
  def test_create_directory_structure_error_multi_level_dir(
      self, mock_generate_files, mock_subprocess_call):
    exit_code = listing_benchmark._create_directory_structure(
        'fake_bucket_url/', 'fake_disk_url/', DIRECTORY_STRUCTURE3, True)
    self.assertGreater(exit_code, 0)
    self.assertEqual(mock_subprocess_call.call_count, 4)
    self.assertEqual(mock_generate_files.call_count, 3)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_disk_url/', shell=True),
        call('mkdir fake_disk_url/1KB_4files_3subdir/', shell=True),
        call('mkdir fake_disk_url/2KB_3files_1subdir/', shell=True),
        call('mkdir fake_disk_url/1KB_1files_0subdir/', shell=True)
    ])
    self.assertEqual(mock_generate_files.call_args_list, [
        call('fake_bucket_url/1KB_4files_3subdir/', 4, 'kb', 1,
             'file', 'fake_disk_url/1KB_4files_3subdir/', True),
        call('fake_bucket_url/2KB_3files_1subdir/', 3, 'kb', 2,
             'file', 'fake_disk_url/2KB_3files_1subdir/', True),
        call('fake_bucket_url/1KB_1files_0subdir/', 1, 'kb', 1,
             'file', 'fake_disk_url/1KB_1files_0subdir/', True)
    ])

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_true_single_level_dir(self, mock_list):
    mock_list.side_effect = [['fake_bucket/']]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE1)
    self.assertTrue(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_true_double_level_dir(self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/file',
         'fake_bucket/2KB_3files_0subdir/',
         'fake_bucket/1KB_2files_0subdir/',
         'fake_bucket/1KB_0files_0subdir/'],
        ['fake_bucket/2KB_3files_0subdir/file_1',
         'fake_bucket/2KB_3files_0subdir/file_2',
         'fake_bucket/2KB_3files_0subdir/file_3'],
        ['fake_bucket/1KB_2files_0subdir/file_1',
         'fake_bucket/1KB_2files_0subdir/file_2'],
        []
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE2)
    self.assertTrue(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_false_file_double_level_dir_test1(
      self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/file',
         'fake_bucket/2KB_3files_0subdir/',
         'fake_bucket/1KB_2files_0subdir/',
         'fake_bucket/1KB_0files_0subdir/'],
        ['fake_bucket/2KB_3files_0subdir/file_1',
         'fake_bucket/2KB_3files_0subdir/file_2',
         'fake_bucket/2KB_3files_0subdir/file_3'],
        ['fake_bucket/1KB_2files_0subdir/file_1',
         'fake_bucket/1KB_2files_0subdir/file_2'],
        ['fake_bucket/1KB_0files_0subdir/file_1']
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE2)
    self.assertFalse(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_false_folder_double_level_dir(
      self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/file',
         'fake_bucket/2KB_3files_0subdir/',
         'fake_bucket/1KB_2files_0subdir/',
         'fake_bucket/1KB_0files_0subdir/'],
        ['fake_bucket/2KB_3files_0subdir/dummy_folder/',
         'fake_bucket/2KB_3files_0subdir/file_1',
         'fake_bucket/2KB_3files_0subdir/file_2',
         'fake_bucket/2KB_3files_0subdir/file_3'],
        ['fake_bucket/1KB_2files_0subdir/file_1',
         'fake_bucket/1KB_2files_0subdir/file_2'],
        []
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE2)
    self.assertFalse(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_false_file_double_level_dir_test2(
      self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/2KB_3files_0subdir/',
         'fake_bucket/1KB_2files_0subdir/',
         'fake_bucket/1KB_0files_0subdir/'],
        ['fake_bucket/2KB_3files_0subdir/file_1',
         'fake_bucket/2KB_3files_0subdir/file_2',
         'fake_bucket/2KB_3files_0subdir/file_3'],
        ['fake_bucket/1KB_2files_0subdir/file_1',
         'fake_bucket/1KB_2files_0subdir/file_2'],
        []
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE2)
    self.assertFalse(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_true_multi_level_dir(self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/1KB_4files_3subdir/',
         'fake_bucket/2KB_3files_1subdir/',
         'fake_bucket/1KB_1files_0subdir/'],
        ['fake_bucket/1KB_4files_3subdir/file_1',
         'fake_bucket/1KB_4files_3subdir/file_2',
         'fake_bucket/1KB_4files_3subdir/file_3',
         'fake_bucket/1KB_4files_3subdir/file_4',
         'fake_bucket/1KB_4files_3subdir/subdir1/',
         'fake_bucket/1KB_4files_3subdir/subdir2/',
         'fake_bucket/1KB_4files_3subdir/subdir3/'],
        ['fake_bucket/1KB_4files_3subdir/subdir1/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir2/'],
        ['fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/file_2'],
        [],
        ['fake_bucket/1KB_4files_3subdir/subdir2/file_1'],
        ['fake_bucket/1KB_4files_3subdir/subdir3/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir3/subsubdir1/'],
        ['fake_bucket/1KB_4files_3subdir/subdir3/subsubdir1/file_1'],
        ['fake_bucket/2KB_3files_1subdir/file_1',
         'fake_bucket/2KB_3files_1subdir/file_2',
         'fake_bucket/2KB_3files_1subdir/file_3',
         'fake_bucket/2KB_3files_1subdir/subdir1/'],
        [],
        ['fake_bucket/1KB_1files_0subdir/file_1']
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE3)
    self.assertTrue(result)

  @patch('listing_benchmark._list_directory')
  def test_compare_directory_structure_false_file_folder_multi_level_dir(
      self, mock_list):
    mock_list.side_effect = [
        ['fake_bucket/', 'fake_bucket/file1',
         'fake_bucket/1KB_4files_3subdir/',
         'fake_bucket/2KB_3files_1subdir/',
         'fake_bucket/1KB_1files_0subdir/'],
        ['fake_bucket/1KB_4files_3subdir/file_1',
         'fake_bucket/1KB_4files_3subdir/file_2',
         'fake_bucket/1KB_4files_3subdir/file_3',
         'fake_bucket/1KB_4files_3subdir/file_4',
         'fake_bucket/1KB_4files_3subdir/subdir1/',
         'fake_bucket/1KB_4files_3subdir/subdir2/',
         'fake_bucket/1KB_4files_3subdir/subdir3/'],
        ['fake_bucket/1KB_4files_3subdir/subdir1/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir2/'],
        ['fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir1/subsubdir1/file_2'],
        [],
        ['fake_bucket/1KB_4files_3subdir/subdir2/file_1'],
        ['fake_bucket/1KB_4files_3subdir/subdir3/file_1',
         'fake_bucket/1KB_4files_3subdir/subdir3/subsubdir1/'],
        ['fake_bucket/1KB_4files_3subdir/subdir3/subsubdir1/file_1'],
        ['fake_bucket/2KB_3files_1subdir/file_1',
         'fake_bucket/2KB_3files_1subdir/file_2',
         'fake_bucket/2KB_3files_1subdir/file_3',
         'fake_bucket/2KB_3files_1subdir/subdir1/'],
        ['fake_bucket/2KB_3files_1subdir/subdir1/file_1',
         'fake_bucket/2KB_3files_1subdir/subdir1/dummy_folder/'],
        ['fake_bucket/1KB_1files_0subdir/file_1']
    ]
    result = listing_benchmark._compare_directory_structure(
        'fake_bucket/', DIRECTORY_STRUCTURE3)
    self.assertFalse(result)

  @patch('listing_benchmark.subprocess.call', return_value=0)
  def test_unmount_gcs_bucket(self, mock_subprocess_call):
    listing_benchmark._unmount_gcs_bucket('fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list[0], call(
        'umount -l fake_bucket', shell=True))
    self.assertEqual(mock_subprocess_call.call_args_list[1], call(
        'rm -rf fake_bucket', shell=True))

  @patch('listing_benchmark.subprocess.call', return_value=1)
  def test_unmount_gcs_bucket_error(self, mock_subprocess_call):
    listing_benchmark._unmount_gcs_bucket('fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list[0], call(
        'umount -l fake_bucket', shell=True))
    self.assertEqual(
        mock_subprocess_call.call_args_list[1], call('bash', shell=True))

  @patch('listing_benchmark.subprocess.call', return_value=0)
  def test_mount_gcs_bucket(self, mock_subprocess_call):
    directory_name = listing_benchmark._mount_gcs_bucket('fake_bucket',
                                                         '--implicit-dirs')
    self.assertEqual(directory_name, 'fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_bucket', shell=True),
        call('gcsfuse --implicit-dirs fake_bucket fake_bucket', shell=True)
    ])

  @patch('listing_benchmark.subprocess.call', return_value=1)
  def test_mount_gcs_bucket_error(self, mock_subprocess_call):
    listing_benchmark._mount_gcs_bucket('fake_bucket', '--implicit-dirs')
    self.assertEqual(mock_subprocess_call.call_count, 3)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_bucket', shell=True),
        call('gcsfuse --implicit-dirs fake_bucket fake_bucket', shell=True),
        call('bash', shell=True)
    ])


if __name__ == '__main__':
  unittest.main()
