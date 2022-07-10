"""Tests for gsheet."""
import unittest
from unittest import mock
from google.oauth2 import service_account
from googleapiclient import discovery
import gsheet

SPREADSHEET_ID = '1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
WORKSHEET_NAME = 'mock'

class MockCredentials:

  def authorize(self, request):
    return request

class GsheetTest(unittest.TestCase):

  def test_get_sheets_service_client(self):
    # Mocking service account call:
    mock_credentials = MockCredentials()
    service_account_mock = mock.MagicMock()
    service_account_mock.return_value = mock_credentials
    service_account.Credentials.from_service_account_file = service_account_mock
    sheets_service = gsheet._get_sheets_service_client()
    self.assertEqual(type(sheets_service), discovery.Resource)


  def test_write_to_google_sheet(self):
    get_response = {
        'range': '{}!A2:A'.format(WORKSHEET_NAME),
        'majorDimension': 'ROWS',
        'values': [['read'], ['read'],['read'],['read'],['write'],['write']]
    }
    new_row = len(get_response['values'])+2
    update_response = {
        'spreadsheetId': SPREADSHEET_ID,
        'updatedRange': '{0}!A{1}:H{1}'.format(WORKSHEET_NAME, new_row),
        'updatedRows': 1,
        'updatedColumns': 10,
        'updatedCells': 10
    }
    sheets_service_mock = mock.MagicMock()
    sheets_service_mock.spreadsheets().values().get(
    ).execute.return_value = get_response
    sheets_service_mock.spreadsheets().values().update(
    ).execute.return_value = update_response
    metrics_to_be_added = [
        ('read', 50000, 40, 1653027155, 1653027215, 1, 2, 3, 4, 5)
    ]
    calls = [
        mock.call.spreadsheets().values().get(
            spreadsheetId=SPREADSHEET_ID,
            range='{}!A2:A'.format(WORKSHEET_NAME)),
        mock.call.spreadsheets().values().update(
            spreadsheetId=SPREADSHEET_ID,
            valueInputOption='USER_ENTERED',
            body={
                'majorDimension': 'ROWS',
                'values': [(
                    'read', 50000, 40, 1653027155, 1653027215, 1, 2, 3, 4, 5
                    )]
            },
            range='{}!A{}'.format(WORKSHEET_NAME, new_row))
    ]

    with mock.patch.object(gsheet, '_get_sheets_service_client'
                           ) as get_sheets_service_client_mock:
      get_sheets_service_client_mock.return_value = sheets_service_mock
      gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_to_be_added)

    sheets_service_mock.assert_has_calls(calls, any_order=True)


if __name__ == '__main__':
  unittest.main()
