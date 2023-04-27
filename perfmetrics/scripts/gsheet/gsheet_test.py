"""Tests for gsheet."""
import os
import unittest
from unittest import mock

from googleapiclient.errors import HttpError
from google.oauth2 import service_account
from googleapiclient.discovery import Resource

import gsheet

SPREADSHEET_ID = '1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
WORKSHEET_NAME = 'mock'

class MockCredentials:

  def authorize(self, request):
    return request

def call_write_to_google_sheet_function(self):
  get_response = {
      'range': '{}!A1:A'.format(WORKSHEET_NAME),
      'majorDimension': 'ROWS',
      'values': [['read'], ['read'],['read'],['read'],['write'],['write']]
  }
  last_row = len(get_response['values'])+1
  update_response_clear = {
      'spreadsheetId': SPREADSHEET_ID,
      'updatedRange': '{0}!A2:{1}'.format(WORKSHEET_NAME, last_row),
      'updatedRows': 1,
      'updatedColumns': 10,
      'updatedCells': 10
  }
  update_response_write = {
      'spreadsheetId': SPREADSHEET_ID,
      'updatedRange': '{0}!A2'.format(WORKSHEET_NAME),
      'updatedRows': 1,
      'updatedColumns': 10,
      'updatedCells': 10
  }
  sheets_service_mock = mock.MagicMock()
  sheets_service_mock.spreadsheets().values().get(
  ).execute.return_value = get_response
  sheets_service_mock.spreadsheets().values().clear(
  ).execute.return_value = update_response_clear
  sheets_service_mock.spreadsheets().values().update(
  ).execute.return_value = update_response_write
  metrics_to_be_added = [
      ('read', 50000, 40, 1653027155, 1653027215, 1, 2, 3, 4, 5)
  ]
  calls = [
      mock.call.spreadsheets().values().get(
          spreadsheetId=SPREADSHEET_ID,
          range='{}!A1:A'.format(WORKSHEET_NAME)),
      mock.call.spreadsheets().values().clear(
          spreadsheetId=SPREADSHEET_ID,
          range='{}!A2:{}'.format(WORKSHEET_NAME, last_row),
          body={}),
      mock.call.spreadsheets().values().update(
          spreadsheetId=SPREADSHEET_ID,
          valueInputOption='USER_ENTERED',
          body={
              'majorDimension': 'ROWS',
              'values': [(
                  'read', 50000, 40, 1653027155, 1653027215, 1, 2, 3, 4, 5
              )]
          },
          range='{}!A2'.format(WORKSHEET_NAME))
  ]

  with mock.patch.object(gsheet, '_get_sheets_service_client'
                         ) as get_sheets_service_client_mock:
    get_sheets_service_client_mock.return_value = sheets_service_mock
    gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_to_be_added)

  return sheets_service_mock, calls

def check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_set(self,KOKORO_JOB_TYPE):
  os.environ["KOKORO_JOB_TYPE"] = KOKORO_JOB_TYPE
  sheets_service_mock,calls = call_write_to_google_sheet_function(self)

  sheets_service_mock.assert_has_calls(calls, any_order=True)
  del os.environ["KOKORO_JOB_TYPE"]

def check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self):
  # The logic for this function is the same as the assert_has_calls function from the python moc library.
  sheets_service_mock,calls = call_write_to_google_sheet_function(self)

  # Expected mock calls
  expected = list(mock._CallList(sheets_service_mock._call_matcher(c) for c in calls))
  # Actual mock calls got after calling write_to_google_sheet function
  all_calls = list(mock._CallList(sheets_service_mock._call_matcher(c) for c in sheets_service_mock.mock_calls))

  # not found stores difference between actual and expected.
  not_found = []
  for call in expected:
    try:
      all_calls.remove(call)
    except ValueError:
      not_found.append(call)

  # not found should be equal to calls for not updating the gsheet.
  if not_found != calls :
    raise AssertionError("sheet is updated")

def call_write_to_google_sheet_function_with_missing_permissions_raises_http_error(self):
  get_response = {
      'range': '{}!A1:A'.format(WORKSHEET_NAME),
      'majorDimension': 'ROWS',
      'values': [['read'], ['read'],['read'],['read'],['write'],['write']]
  }
  sheets_service_mock = mock.MagicMock()
  sheets_service_mock.spreadsheets().values().get(
  ).execute.return_value = get_response
  sheets_service_mock.spreadsheets().values().get(
  ).execute.side_effect = HttpError(
      mock.Mock(
          status=403,
          message='The caller does not have permission'),
      b'http content')
  metrics_to_be_added = [('2_thread', 50000, 40, 1653027155, 1653027215, 1, 2,
                          3, 4, 5)]
  calls = [
      mock.call.spreadsheets().values().get(
          spreadsheetId=SPREADSHEET_ID,
          range='{}!A1:A'.format(WORKSHEET_NAME))
  ]

  return sheets_service_mock,metrics_to_be_added,calls

def check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_set(self,KOKORO_JOB_TYPE):
  os.environ["KOKORO_JOB_TYPE"] = KOKORO_JOB_TYPE
  sheets_service_mock,metrics_to_be_added,calls = call_write_to_google_sheet_function_with_missing_permissions_raises_http_error(self)

  with mock.patch.object(gsheet, '_get_sheets_service_client'
                         ) as get_sheets_service_client_mock:
    get_sheets_service_client_mock.return_value = sheets_service_mock
    with self.assertRaises(HttpError):
      gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_to_be_added)

  sheets_service_mock.assert_has_calls(calls, any_order=True)
  del os.environ["KOKORO_JOB_TYPE"]

def check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self):
  # The logic for this function is the same as the assert_has_calls function from the python moc library.
  sheets_service_mock,metrics_to_be_added,calls = call_write_to_google_sheet_function_with_missing_permissions_raises_http_error(self)

  with mock.patch.object(gsheet, '_get_sheets_service_client'
                         ) as get_sheets_service_client_mock:
    get_sheets_service_client_mock.return_value = sheets_service_mock
    try:
      # Try to update the sheet and http error raised.
      with self.assertRaises(HttpError):
        gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_to_be_added)
        sheets_service_mock.assert_has_calls(calls, any_order=True)
    except:
       # In our case, it should not happen as we don't update the sheet in this case.
       assert True

class GsheetTest(unittest.TestCase):

  def test_get_sheets_service_client(self):
    # Mocking service account
    mock_credentials = MockCredentials()
    service_account_mock = mock.MagicMock()
    service_account_mock.return_value = mock_credentials
    service_account.Credentials.from_service_account_file = service_account_mock
    sheets_client = gsheet._get_sheets_service_client()
    self.assertIsInstance(sheets_client, Resource)

  def test_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_equals_RELEASE(self):
    check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"RELEASE")

  def test_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_equals_CONTINUOUS_INTEGRATION(self):
    check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"CONTINUOUS_INTEGRATION")

  def test_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_equals_PRESUBMIT_GITHUB(self):
    check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"PRESUBMIT_GITHUB")

  def test_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self):
    check_write_to_google_sheet_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self)

  def test_write_to_google_sheet_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_equals_RELEASE(self):
    check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"RELEASE")

  def test_write_to_google_sheet_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_equals_CONTINUOUS_INTEGRATION(self):
    check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"CONTINUOUS_INTEGRATION")

  def test_write_to_google_sheet_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_equals_PRESUBMIT_GITHUB(self):
    check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_set(self,"PRESUBMIT_GITHUB")

  def test_write_to_google_sheet_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self):
    check_write_to_google_sheet_with_missing_permissions_raises_http_error_when_KOKORO_JOB_TYPE_env_variable_is_not_set(self)

if __name__ == '__main__':
  unittest.main()
