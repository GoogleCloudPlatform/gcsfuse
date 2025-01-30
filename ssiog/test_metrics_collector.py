#!/usr/bin/env python3
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Command to run the unit test:
    `python test_training.py`
"""

import unittest
from metrics_collector import analyze_metrics
import pandas as pd
from unittest.mock import patch
from google.cloud import storage
from io import StringIO

class TestAnalyzeMetrics(unittest.TestCase):

    @patch('gcsfs.GCSFileSystem')
    def test_analyze_metrics_success_no_filtering(self, mock_gcsfs):
        # Mock the GCSFileSystem to return sample data
        mock_fs = mock_gcsfs.return_value
        mock_fs.glob.return_value = ['gs://test-bucket/file1.csv', 'gs://test-bucket/file2.csv']
        mock_fs.open.side_effect = [StringIO("timestamp,sample_lat\n1678886400,10\n1678886460,20"), StringIO("timestamp,sample_lat\n1678886520,15\n1678886580,25")]

        # Call the function to analyze metrics
        result_df = analyze_metrics('gs://test-bucket/*.csv', False)

        # Assert that the result is a pandas DataFrame and has the expected data
        self.assertIsInstance(result_df, pd.DataFrame)
        self.assertEqual(len(result_df), 4)
        self.assertTrue('sample_lat' in result_df.columns)
        
    @patch('gcsfs.GCSFileSystem')
    def test_analyze_metrics_success_with_filtering(self, mock_gcsfs):
        # Mock the GCSFileSystem to return sample data
        mock_fs = mock_gcsfs.return_value
        mock_fs.glob.return_value = ['gs://test-bucket/file1.csv', 'gs://test-bucket/file2.csv']
        mock_fs.open.side_effect = [StringIO("timestamp,sample_lat\n1678886400,10\n1678886560,20"), StringIO("timestamp,sample_lat\n1678886520,15\n1678886580,25")]

        # Call the function to analyze metrics
        result_df = analyze_metrics('gs://test-bucket/*.csv', True)

        # Assert that the result is a pandas DataFrame and has the expected data
        print(result_df)
        self.assertIsInstance(result_df, pd.DataFrame)
        self.assertEqual(len(result_df), 2)
        self.assertTrue('sample_lat' in result_df.columns)

    @patch('gcsfs.GCSFileSystem')
    def test_analyze_metrics_no_files(self, mock_gcsfs):
        # Mock the GCSFileSystem to return no files
        mock_fs = mock_gcsfs.return_value
        mock_fs.glob.return_value = []

        # Call the function to analyze metrics
        result_df = analyze_metrics('gs://test-bucket/*.csv')

        # Assert that the result is None
        self.assertIsNone(result_df)

    @patch('gcsfs.GCSFileSystem')
    def test_analyze_metrics_empty_file(self, mock_gcsfs):
        # Mock the GCSFileSystem to return an empty file
        mock_fs = mock_gcsfs.return_value
        mock_fs.glob.return_value = ['gs://test-bucket/empty.csv']
        mock_fs.open.return_value = StringIO("")

        # Call the function to analyze metrics
        result_df = analyze_metrics('gs://test-bucket/*.csv')
        
    @patch('fsspec.filesystem')
    def test_analyze_metrics_local_success(self, mock_fsspec):
        # Mock the local filesystem to return sample data
        mock_fs = mock_fsspec.return_value
        mock_fs.glob.return_value = ['file1.csv', 'file2.csv']
        mock_fs.open.side_effect = [StringIO("timestamp,sample_lat\n1678886400,10\n1678886460,20"), StringIO("timestamp,sample_lat\n1678886520,15\n1678886580,25")]

        # Call the function to analyze metrics
        result_df = analyze_metrics('/tmp/*.csv', False)

        # Assert that the result is a pandas DataFrame and has the expected data
        self.assertIsInstance(result_df, pd.DataFrame)
        self.assertEqual(len(result_df), 4)
        self.assertTrue('sample_lat' in result_df.columns)

    @patch('fsspec.filesystem')
    def test_analyze_metrics_local_empty_file(self, mock_fsspec):
        # Mock the local filesystem to return an empty file
        mock_fs = mock_fsspec.return_value
        mock_fs.glob.return_value = ['empty.csv']
        mock_fs.open.return_value = StringIO("")

        # Call the function to analyze metrics
        result_df = analyze_metrics('/tmp/*.csv')

        # Assert that the result is None
        self.assertIsNone(result_df)

    @patch('fsspec.filesystem')
    def test_analyze_metrics_local_no_files(self, mock_fsspec):
        # Mock the local filesystem to return no files
        mock_fs = mock_fsspec.return_value
        mock_fs.glob.return_value = []

        # Call the function to analyze metrics
        result_df = analyze_metrics('/tmp/*.csv')

        # Assert that the result is None
        self.assertIsNone(result_df)
        

if __name__ == '__main__':
    loader = unittest.TestLoader()
    # Discover tests in the current directory and its subdirectories
    suite = loader.discover(".")  # "." specifies the starting directory

    # Run the tests
    runner = unittest.TextTestRunner(verbosity=2)
    runner.run(suite)

    unittest.main()
