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


import unittest
from unittest.mock import patch
import argparse
from metrics_collector import analyze_metrics
import pandas as pd
from io import StringIO
import logging
from training import main, training
from training import sequential_reader, full_random_reader

import unittest
import os
import shutil
from unittest.mock import patch
import argparse
from training import main
import tempfile
import pytest


class TestEndToEnd(unittest.TestCase):
    def setUp(self):
        self.test_dir = tempfile.mkdtemp()
        # Create dummy files for testing
        with open(os.path.join(self.test_dir, "file1.txt"), "w") as f:
            f.write("test data")
        with open(os.path.join(self.test_dir, "file2.txt"), "w") as f:
            f.write("more test data")

    def tearDown(self):
        shutil.rmtree(self.test_dir)

    @patch('training.arguments.parse_args')
    @patch('training.full_random_reader')
    @patch('training.close_metrics_logger')
    @patch('torch.distributed.destroy_process_group')
    def test_e2e_main_error_during_read(self, mock_destroy_process_group, mock_close_metrics_logger, mock_full_random_reader, mock_parse_args):
        mock_args = argparse.Namespace(
            prefix=[self.test_dir],
            epochs=1,
            steps=1,
            sample_size=1000,
            batch_size=10,
            read_order=["FullRandom"],
            background_queue_maxsize=2048,            
            background_threads=16,
            group_coordinator_address="localhost",
            group_coordinator_port="4567",
            group_member_id=0,
            group_size=1,
            log_level="INFO",
            label="test-label",
            log_metrics=False,
            log_file="",
            export_metrics=False,
            metrics_file="metrics.csv",
            clear_pagecache_after_epoch=False,
            object_count_limit=100,
        )
        mock_parse_args.return_value = mock_args
        mock_full_random_reader.side_effect = Exception("Read failed")
        
        with self.assertRaises(SystemExit) as ce:
            main()

        self.assertEqual(ce.exception.code, 1)
        mock_close_metrics_logger.assert_called_once()
        mock_destroy_process_group.assert_called_once()

    # Debug this - why are we getting process_group_init_twice in this scenario.
    # @patch('training.arguments.parse_args')
    # @patch('training.close_metrics_logger')
    # @patch('torch.distributed.destroy_process_group')
    # def test_e2e_main_success(self, mock_destroy_process_group, mock_close_metrics_logger, mock_parse_args):
    #     mock_args = argparse.Namespace(
    #         prefix=[self.test_dir],
    #         epochs=1,
    #         steps=1,
    #         sample_size=10,
    #         batch_size=10,
    #         read_order=["FullRandom"],
    #         background_queue_maxsize=2048,            
    #         background_threads=16,
    #         group_coordinator_address="localhost",
    #         group_coordinator_port="4567",
    #         group_member_id=0,
    #         group_size=1,
    #         log_level="DEBUG",
    #         label="test-label",
    #         log_metrics=False,
    #         log_file="",
    #         export_metrics=False,
    #         metrics_file="metrics.csv",
    #         clear_pagecache_after_epoch=False,
    #         object_count_limit=100,
    #     )
    #     mock_parse_args.return_value = mock_args
        
    #     with patch('sys.exit') as mock_exit:
    #         main()
        
    #     mock_exit.assert_called_once_with(0)
    #     mock_close_metrics_logger.assert_called_once()
    #     mock_destroy_process_group.assert_called_once()

        
class TestTraining(unittest.TestCase):
    @patch('training.arguments.parse_args')
    @patch('training.configure_object_sources')
    @patch('training.configure_epoch')
    @patch('training.configure_samples')
    @patch('training.Epoch')
    @patch('training.setup_logger')
    @patch('training.setup_metrics_exporter')
    @patch('training.setup_metrics_logger')
    @patch('training.metrics_logger.AsyncMetricsLogger')
    @patch('training.monitoring.initialize_monitoring_provider')
    @patch('torch.distributed.init_process_group')
    @patch('torch.distributed.destroy_process_group')
    @patch('training.util.clear_kernel_cache')
    @patch('training.sequential_reader')
    def test_main_success(self, 
                          mock_sequential_reader,
                          mock_clear_kernel_cache, 
                          mock_destroy_process_group, 
                          mock_init_process_group, 
                          mock_initialize_monitoring_provider, 
                          mock_AsyncMetricsLogger, 
                          mock_setup_metrics_exporter, 
                          mock_setup_metrics_logger, 
                          mock_setup_logger, 
                          mock_Epoch, 
                          mock_configure_samples, 
                          mock_configure_epoch, 
                          mock_configure_object_sources, 
                          mock_parse_args, 
                          ):

        # Mock the arguments
        mock_args = argparse.Namespace(
            prefix=["gs://test-bucket/"],
            epochs=1,
            steps=1,
            sample_size=1024,
            batch_size=1024,
            read_order=["Sequential"],
            background_queue_maxsize=2048,
            background_threads=16,
            group_coordinator_address="localhost",
            group_coordinator_port="4567",
            group_member_id=0,
            group_size=1,
            label="test-label",
            log_metrics=True,
            export_metrics=True,
            metrics_file="metrics.csv",
            clear_pagecache_after_epoch=True,
        )
        
        mock_parse_args.return_value = mock_args

        # Mock the necessary functions
        mock_configure_object_sources.return_value = {"gs://test-bucket/": "sequential_reader"}
        mock_configure_epoch.return_value = (lambda *args: [], "Sequential", "test_fs", "test_fs", ["test_object"])
        mock_configure_samples.return_value = [("test_object", 0)]
        mock_Epoch.return_value = [f"Step: 0, Duration (ms): 100, Batch-sample: 1024"]

        # Mock the logger and metrics exporter
        mock_logger = logging.getLogger("test-label")
        mock_logger.propagate = False
        mock_logger.setLevel(logging.INFO)
        mock_setup_logger.return_value = mock_logger
        mock_initialize_monitoring_provider.return_value = "test_meter"
        mock_setup_metrics_exporter.return_value = "test_meter"
        mock_setup_metrics_logger.return_value = "test_metrics_logger"
        mock_AsyncMetricsLogger.return_value.log_metric.return_value = None
        mock_AsyncMetricsLogger.return_value.close.return_value = None
        mock_sequential_reader.return_value = [("test_object", 0)]
        mock_clear_kernel_cache.return_value = None

        # Call the main function
        training()

        # Assertions
        mock_parse_args.assert_called_once()
        mock_setup_logger.assert_called_once()
        mock_setup_metrics_exporter.assert_called_once()
        mock_setup_metrics_logger.assert_called_once()
        mock_configure_object_sources.assert_called_once()
        mock_configure_epoch.assert_called_once()
        mock_configure_samples.assert_called_once()
        mock_Epoch.assert_called_once()
        mock_init_process_group.assert_called_once()
        mock_clear_kernel_cache.assert_called_once()
        
    def test_sequential_reader_random_sample(self):
        mock_fs = type('MockFileSystem', (object,), {'open_input_stream': lambda self, path: StringIO("test")})()
            
        # Mock td.get_rank() and td.get_world_size() method for this test
        with patch('training.td.get_rank', return_value=0), patch('training.td.get_world_size', return_value=1):
            result = list(sequential_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 4)]))
        
        # Assertions
        self.assertEqual(len(result), 2)
        self.assertEqual(result[0][0], "test_file")
        self.assertEqual(result[0][1], 0)
        self.assertGreater(result[0][2], 0)
        
        self.assertEqual(result[1][0], "test_file")
        self.assertEqual(result[1][1], 2)
        self.assertGreater(result[1][2], 0)
        
    def test_sequential_reader_continuous_sample(self):
        mock_fs = type('MockFileSystem', (object,), {'open_input_stream': lambda self, path: StringIO("test")})()
            
        with patch('training.td.get_rank', return_value=0), patch('training.td.get_world_size', return_value=1):
            result = list(sequential_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 2)]))
        
        # Assertions
        self.assertEqual(len(result), 2)
        self.assertEqual(result[0][0], "test_file")
        self.assertEqual(result[0][1], 0)
        self.assertGreater(result[0][2], 0)
        
        self.assertEqual(result[1][0], "test_file")
        self.assertEqual(result[1][1], 2)
        self.assertGreater(result[1][2], 0)
        
    def test_full_random_reader_continuous_sample(self):        
        mock_fs = type('MockFileSystem', (object,), {'open_input_file': lambda self, path: type('MockFile', (object,), {'readall': lambda self: b"testing_random_reader", 'read_at': lambda self, size, offset: b"testing_random_reader"[offset:offset+size], 'close': lambda self: None})()})()

        with patch('training.td.get_rank', return_value=0), patch('training.td.get_world_size', return_value=1):
            result = list(full_random_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 2)]))
        
        # Assertions
        self.assertEqual(len(result), 2)
        
        self.assertEqual(result[0][0], 0)
        self.assertEqual(result[0][1], b"te")
        
        self.assertEqual(result[1][0], 2)
        self.assertEqual(result[1][1], b"st")
        
    def test_full_random_reader_random_sample(self):
        mock_fs = type('MockFileSystem', (object,), {'open_input_file': lambda self, path: type('MockFile', (object,), {'readall': lambda self: b"testing_random_reader", 'read_at': lambda self, size, offset: b"testing_random_reader"[offset:offset+size], 'close': lambda self: None})()})()

        with patch('training.td.get_rank', return_value=0), patch('training.td.get_world_size', return_value=1):
            result = list(full_random_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 10)]))
        
        # Assertions
        self.assertEqual(len(result), 2)
        
        self.assertEqual(result[0][0], 0)
        self.assertEqual(result[0][1], b"te")
        
        self.assertEqual(result[1][0], 10)
        self.assertEqual(result[1][1], b"nd")
        
    @patch('training.td.get_rank', return_value=0)
    @patch('training.td.get_world_size', return_value=1)
    def test_full_random_reader_exception(self, mock_get_world_size, mock_get_rank):
        class MockFileSystem:
            def open_input_file(self, path):
                class MockFile:
                    def readall(self):
                        return b"testing_random_reader"
                    
                    def read_at(self, size, offset):
                        if offset < 10:
                            return b"testing_random_reader"[offset:offset + size]

                    def close(self):
                        pass

                return MockFile()

        mock_fs = MockFileSystem()
        with self.assertRaises(ValueError) as context:
            list(full_random_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 10)]))
            
        self.assertEqual(str(context.exception), "chunk is nil.")
    
    @patch('training.td.get_rank', return_value=0)
    @patch('training.td.get_world_size', return_value=1)
    def test_full_random_reader_exception_during_read(self, mock_get_world_size, mock_get_rank):
        class MockFileSystem:
            def open_input_file(self, path):
                class MockFile:
                    def readall(self):
                        return b"testing_random_reader"
                    
                    def read_at(self, size, offset):
                        if offset < 10:
                            return b"testing_random_reader"[offset:offset + size]
                        else:
                            raise ValueError("Read failed")

                    def close(self):
                        pass

                return MockFile()

        mock_fs = MockFileSystem()
        with self.assertRaises(Exception) as context:
            list(full_random_reader(["test_file"], 0, 1, mock_fs, 2, [("test_file", 0), ("test_file", 10)]))
        self.assertEqual(str(context.exception), "Read failed")

if __name__ == '__main__':
    loader = unittest.TestLoader()
    # Discover tests in the current directory and its subdirectories
    suite = loader.discover(".")  # "." specifies the starting directory

    # Run the tests
    runner = unittest.TextTestRunner(verbosity=2)
    runner.run(suite)

    unittest.main()