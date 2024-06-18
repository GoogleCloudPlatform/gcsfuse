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
"""Python module for testing ExperimentsGCSFuseBQ module

Usage from perfmetrics/scripts directory:
  python3 -m bigquery.experiments_gcsfuse_bq_test

Note:
  Make sure BigQuery API is enabled for the project
"""
import unittest
import uuid
from unittest.mock import patch, MagicMock
from google.cloud.bigquery.table import Table

from bigquery import constants
from bigquery.experiments_gcsfuse_bq import ExperimentsGCSFuseBQ

GCSFUSE_FLAGS = "--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library"
CONFIG_FILE_FLAGS_AS_JSON = "config-file: { write: { create-empty-file: true }, logging: { file-path: ~/log.log, format: text, severity: info }"
BRANCH = "master"
END_DATE = "2023-10-08 05:30:00"
NEW_END_DATE = "2024-10-08 05:30:00"
CONFIG_NAME = "TestConfiguration"
VALID_CONFIG_ID = 'c4c354b3-98a6-48ca-9313-680db412ecec'
INVALID_CONFIG_ID = 'c4d554b3-98a6-ng67-9313-680db412fnzs'
START_TIME_BUILD = '2023-06-30 00:00:00'
TABLE_ID = constants.FIO_TABLE_ID

class TestExperimentsGCSFuseBQ(unittest.TestCase):
  @patch('bigquery.experiments_gcsfuse_bq.bigquery')
  def setUp(self, mock_bigquery):
    # Set up the necessary mock objects and initialize the ExperimentsGCSFuseBQ instance
    self.project_id = constants.PROJECT_ID
    self.dataset_id = constants.DATASET_ID
    self.client = mock_bigquery.Client.return_value
    self.experiments = ExperimentsGCSFuseBQ(self.project_id, self.dataset_id, self.client)

  def test_check_if_config_valid(self):
    # Test the _check_if_config_valid method
    # Create the mock objects
    query_mock = MagicMock()
    query_mock.result.return_value.total_rows = 1
    self.experiments._execute_query = MagicMock(return_value=query_mock)

    # Call the method under test
    result = self.experiments._check_if_config_valid(VALID_CONFIG_ID)

    # Assertions to check other method calls and behaviors
    self.assertTrue(result) # Assert that result is true
    self.experiments._execute_query.assert_called_once() # Assert that _execute_query was called once

  def test_check_if_config_valid_with_invalid_config(self):
    # Test the _check_if_config_valid method with an invalid config
    # Create the mock objects
    query_mock = MagicMock()
    query_mock.result.return_value.total_rows = 0
    self.experiments._execute_query = MagicMock(return_value=query_mock)

    # Call the method under test
    result = self.experiments._check_if_config_valid(INVALID_CONFIG_ID)

    # Assertions to check other method calls and behaviors
    self.assertFalse(result) # Assert that result is False
    self.experiments._execute_query.assert_called_once() # Assert that _execute_query was called once

  def test_execute_query(self):
    # Test the _execute_query method with an invalid config
    # Create the mock objects
    query = """SELECT * FROM `{}.{}.{}`""".format(self.project_id, self.dataset_id, TABLE_ID)
    job_mock = MagicMock()
    job_mock.errors = None
    self.client.query.return_value = job_mock

    # Call the method under test
    result = self.experiments._execute_query(query)

    # Assertions to check other method calls and behaviors
    self.assertEqual(result, job_mock) # Check the return QueryJob object
    self.client.query.assert_called_once_with(query) # Assert that client.query was called once with query

  def test_execute_query_with_error(self):
    # Test the _execute_query method with an error
    # Create the mock objects
    query = """SELECT * FROM `{}.{}.{}`""".format(self.project_id, self.dataset_id, TABLE_ID)
    job_mock = MagicMock()
    job_mock.errors = [{'message': 'Query execution failed.'}]
    self.client.query.return_value = job_mock

    # Call the method under test and assert that it raises an exception
    with self.assertRaises(Exception) as context:
      self.experiments._execute_query(query)

    # Additional assertions to check other method calls and behaviors
    self.assertEqual(str(context.exception), 'Error message: Query execution failed.') # Check the exception message
    self.client.query.assert_called_once_with(query) # Assert that client.query was called once with query

  def test_insert_rows(self):
    # Test the _insert_rows method
    # Create the mock objects
    result = []
    self.experiments.client.insert_rows = MagicMock(return_value=result)

    mock_table = MagicMock(spec=Table)
    self.experiments.client.get_table = MagicMock(return_value=mock_table)
    rows_to_insert = [()]

    # Call the method under test
    self.experiments._insert_rows(mock_table, rows_to_insert, TABLE_ID, VALID_CONFIG_ID, START_TIME_BUILD)

    # Assertions to check other method calls and behaviors
    self.assertEqual(self.client.insert_rows.call_count, 1) # Assert that client.insert_rows was called once
    # Assert that client.insert_rows was called once with table and row data to be inserted
    self.client.insert_rows.assert_called_once_with(mock_table, rows_to_insert)

  def test_insert_rows_with_error(self):
    # Test the _insert_rows method with error
    # Create the mock objects
    error_result = [{'error': 'Error message'}]
    self.experiments.client.insert_rows = MagicMock(return_value=error_result)

    mock_table = MagicMock(spec=Table)
    self.experiments.client.get_table = MagicMock(return_value=mock_table)
    rows_to_insert = [()]

    job_mock = MagicMock()
    job_mock.errors = None
    self.client.query.return_value = job_mock

    expected_query = """
        DELETE FROM `{}.{}.{}`
        WHERE configuration_id = '{}'
        AND start_time_build = '{}'
      """.format(self.project_id, self.dataset_id, TABLE_ID, VALID_CONFIG_ID, START_TIME_BUILD)

    # Call the method under test and assert that it raises an exception
    with self.assertRaises(Exception) as context:
      self.experiments._insert_rows(mock_table, rows_to_insert, TABLE_ID, VALID_CONFIG_ID, START_TIME_BUILD)

    # Additional assertions to check other method calls and behaviors
    self.assertEqual(
        str(context.exception),
        f'Error inserting data to BigQuery table: {error_result}'
    ) # Check the exception message
    # Assert that client.insert_rows was called once with table and row data to be inserted
    self.client.insert_rows.assert_called_once_with(mock_table, rows_to_insert)
    self.client.query.assert_called_once_with(expected_query) # Assert that client.query was called once with expected query

  def test_setup_dataset_and_tables(self):
    # Test the setup_dataset_and_tables method
    # Create the mock objects
    dataset_ref_mock = MagicMock()
    self.client.get_dataset.return_value = dataset_ref_mock
    self.experiments._execute_query = MagicMock()

    # Call the method under test
    self.experiments.setup_dataset_and_tables()

    # Assertions to check other method calls and behaviors
    self.assertEqual(self.client.create_dataset.call_count, 1) # Check that client.create_dataset was called once
    self.experiments._execute_query.assert_called() # Assert that _execute_query was called
    self.assertEqual(
        self.experiments._execute_query.call_count, 4  # Check the number of queries executed in setup_dataset_and_tables
    )

  @patch('bigquery.experiments_gcsfuse_bq.uuid.uuid4')
  def test_get_experiment_configuration_id_insert_new_configuration(self, mock_uuid4):
    # Test the get_experiment_configuration_id method with new configuration
    # Create the mock objects
    query_mock = MagicMock()
    query_mock.result.return_value.total_rows = 0
    self.experiments._execute_query = MagicMock(return_value=query_mock)

    table_mock = MagicMock()
    self.client.get_table.return_value = table_mock

    insert_rows_result = []
    self.client.insert_rows.return_value = insert_rows_result

    config_id = str(uuid.uuid4())
    mock_uuid4.return_value = config_id

    # Call the method under test
    result = self.experiments.get_experiment_configuration_id(
        GCSFUSE_FLAGS, BRANCH, CONFIG_FILE_FLAGS_AS_JSON, END_DATE, CONFIG_NAME
    )

    # Assertions to check other method calls and behaviors
    self.assertEqual(result, config_id)  # Check returned config ID
    self.assertEqual(self.client.insert_rows.call_count, 1)  # Assert insert_rows was called once
    self.assertEqual(len(self.client.insert_rows.call_args[0][1]), 1)  # Check number of rows to insert
    self.experiments._execute_query.assert_called_once() # Assert _execute_query was called once
    # Assert client.get_table was called once with the correct dataset
    self.client.get_table.assert_called_once_with(self.experiments.dataset_ref.table.return_value)
    self.client.insert_rows.assert_called_once_with(table_mock, [(
        config_id, CONFIG_NAME, GCSFUSE_FLAGS, BRANCH, CONFIG_FILE_FLAGS_AS_JSON, END_DATE
    )]) # Check inserted row data

  def test_get_experiment_configuration_id_get_existing_configuration(self):
    # Test the get_experiment_configuration_id method with existing configuration
    # Create the mock objects
    job_mock = MagicMock()
    job_mock.result.return_value.total_rows = 1
    mock_item = MagicMock()
    mock_item.get.side_effect = lambda key: {
        'gcsfuse_flags': GCSFUSE_FLAGS,
        'branch': BRANCH,
        'CONFIG_FILE_FLAGS_AS_JSON':CONFIG_FILE_FLAGS_AS_JSON,
        'end_date': END_DATE,
        'configuration_id': VALID_CONFIG_ID
    }.get(key)

    job_mock.__iter__.return_value = iter([mock_item])
    self.experiments._execute_query = MagicMock(return_value=job_mock)

    # Call the method under test
    result = self.experiments.get_experiment_configuration_id(
        GCSFUSE_FLAGS, CONFIG_FILE_FLAGS_AS_JSON, BRANCH, END_DATE, CONFIG_NAME
    )

    # Assertions to check other method calls and behaviors
    self.assertEqual(result, VALID_CONFIG_ID) # Check returned config ID
    self.assertEqual(self.experiments._execute_query.call_count, 1)  # Check that only 1 query was executed
    self.assertFalse(self.client.insert_rows.called) # Ensure insert_rows is not called

  def test_get_experiment_configuration_id_update_existing_configuration(self):
    # Test the get_experiment_configuration_id method with existing configuration but different end date
    # Create the mock objects
    job_mock = MagicMock()
    job_mock.result.return_value.total_rows = 1
    mock_item = MagicMock()
    mock_item.get.side_effect = lambda key: {
        'gcsfuse_flags': GCSFUSE_FLAGS,
        'branch': BRANCH,
        'CONFIG_FILE_FLAGS_AS_JSON' : CONFIG_FILE_FLAGS_AS_JSON,
        'end_date': END_DATE,
        'configuration_id': VALID_CONFIG_ID
    }.get(key)

    job_mock.__iter__.return_value = iter([mock_item])
    self.experiments._execute_query = MagicMock(return_value=job_mock)

    # Call the method under test
    result = self.experiments.get_experiment_configuration_id(
        GCSFUSE_FLAGS, CONFIG_FILE_FLAGS_AS_JSON, BRANCH, NEW_END_DATE, CONFIG_NAME
    )

    # Assertions to check other method calls and behaviors
    self.assertEqual(result, VALID_CONFIG_ID) # Check returned config ID
    self.assertEqual(self.experiments._execute_query.call_count, 2)  # Check that 2 queries were executed
    self.assertFalse(self.client.insert_rows.called) # Ensure insert_rows is not called

  def test_upload_metrics_to_table_invalid_config(self):
    # Test the upload_metrics_to_table method with invalid configuration
    # Create the mock objects
    self.experiments._check_if_config_valid = MagicMock(return_value=False)

    # Call the method under test and assert that it raises an exception
    with self.assertRaises(Exception):
      self.experiments.upload_metrics_to_table(TABLE_ID, INVALID_CONFIG_ID, START_TIME_BUILD, [[]])

    # Additional assertions to check other method calls and behaviors
    self.experiments._check_if_config_valid.assert_called_once()

  def test_upload_metrics_to_table_valid_config(self):
    # Test the upload_metrics_to_table method with valid configuration
    # Create the mock objects
    self.experiments._check_if_config_valid = MagicMock(return_value=True)

    mock_table = MagicMock(spec=Table)
    self.experiments.client.get_table = MagicMock(return_value=mock_table)

    self.experiments.client.insert_rows = MagicMock(return_value=[])

    # Call the method under test
    self.experiments.upload_metrics_to_table(TABLE_ID, VALID_CONFIG_ID, START_TIME_BUILD, [[]])

    # Assertions to check other method calls and behaviors
    self.experiments._check_if_config_valid.assert_called_once() # Ensure that check_if_config_valid was called once
    self.experiments.client.insert_rows.assert_called_once() # Ensure that insert_rows was called once
    self.experiments.client.get_table.assert_called_once() # Ensure that get_table was called once

  def test_upload_metrics_to_table_valid_config_insert_fails(self):
    # Test the upload_metrics_to_table method with valid configuration but insertion fails
    # Create the mock objects
    self.experiments._check_if_config_valid = MagicMock(return_value=True)

    mock_table = MagicMock(spec=Table)
    self.experiments.client.get_table = MagicMock(return_value=mock_table)

    error_result = [{'error': 'Error message'}]
    self.experiments.client.insert_rows = MagicMock(return_value=error_result)

    # Call the method under test and assert that it raises an exception
    with self.assertRaises(Exception) as context:
      self.experiments.upload_metrics_to_table(constants.LS_TABLE_ID, VALID_CONFIG_ID, START_TIME_BUILD, [[]])

    # Additional assertions to check other method calls and behaviors
    self.assertEqual(
        str(context.exception),
        f'Error inserting data to BigQuery table: {error_result}'
    ) # Check exception message
    self.experiments._check_if_config_valid.assert_called_once() # Ensure that check_if_config_valid was called once
    self.experiments.client.insert_rows.assert_called_once() # Ensure that insert_rows was called once
    self.experiments.client.get_table.assert_called_once() # Ensure that get_table was called once

if __name__ == '__main__':
  unittest.main()
