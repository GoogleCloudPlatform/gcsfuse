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
import os
import time
from metrics_logger import AsyncMetricsLogger, NoOpMetricsLogger

class TestAsyncMetricsLogger(unittest.TestCase):

    def test_logging_metrics(self):
        """Tests that metrics are logged to the CSV file."""

        # Create a temporary file for testing
        file_name = "test_metrics.csv"
        logger = AsyncMetricsLogger(file_name=file_name)

        # Log some metrics
        logger.log_metric(400)
        logger.log_metric(300)        
        
        # Close to flush the metrics in the buffer queue.
        logger.close()

        # Check if the file exists and has the expected content
        self.assertTrue(os.path.exists(file_name))
        with open(file_name, "r") as csvfile:
            lines = csvfile.readlines()
            self.assertEqual(len(lines), 3)
            
            # Split the lines by comma and check individual values
            row0 = lines[0].strip().split(",")
            row1 = lines[1].strip().split(",")
            row2 = lines[2].strip().split(",")

            # Check timestamp (ensure it's a number)
            self.assertTrue(row1[0].replace(".", "", 1).isdigit())  
            self.assertTrue(row2[0].replace(".", "", 1).isdigit())

            # Check metric name and value
            self.assertEqual(row0[0], "timestamp")
            self.assertEqual(row0[1], "sample_lat")
            self.assertEqual(row1[1], "400")
            self.assertEqual(row2[1], "300")

        # Clean up the temporary csv file
        os.remove(file_name)

class TestNoOpMetricsLogger(unittest.TestCase):

    def test_no_op_metrics_logger(self):
        """Tests that the NoOpMetricsLogger doesn't record any metrics."""

        # Create a temporary file for testing
        file_name = "no_op_metrics.csv"
        logger = NoOpMetricsLogger(file_name=file_name)

        # Call the log_metric method with various arguments
        logger.log_metric(400)
        logger.log_metric(20)

        # Assert that the file was not created.
        self.assertFalse(os.path.exists(file_name))

