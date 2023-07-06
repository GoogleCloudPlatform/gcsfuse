"""Tests for gsheet."""
import unittest
from unittest import mock

from googleapiclient.errors import HttpError
from google.oauth2 import service_account
from googleapiclient.discovery import Resource

from gsheet import gsheet

SPREADSHEET_ID = '1T5ZSUAAYqANS_MUjS2KR2XGN4DhgMFS8zF2I4E8WN4og'
WORKSHEET_NAME = 'mock'

class MockCredentials:

  def authorize(self, request):
    return request

class GsheetTest(unittest.TestCase):

  def test_get_sheets_service_client(self):
    # Mocking service account
    mock_credentials = MockCredentials()
    service_account_mock = mock.MagicMock()
    service_account_mock.return_value = mock_credentials
    service_account.Credentials.from_service_account_file = service_account_mock
    sheets_client = gsheet._get_sheets_service_client()
    self.assertIsInstance(sheets_client, Resource)

  def test_write_to_google_sheet(self):
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

    sheets_service_mock.assert_has_calls(calls, any_order=True)

  def test_write_to_google_sheet_missing_permissions_raises_http_error(
      self):
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

    with mock.patch.object(gsheet, '_get_sheets_service_client'
                           ) as get_sheets_service_client_mock:
      get_sheets_service_client_mock.return_value = sheets_service_mock
      with self.assertRaises(HttpError):
        gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_to_be_added)

    sheets_service_mock.assert_has_calls(calls, any_order=True)


if __name__ == '__main__':
  unittest.main()
