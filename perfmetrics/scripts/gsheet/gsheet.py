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
# To run the script, run in terminal :
# python3 generate_folders_and_files.py <config-file.json>

from google.oauth2 import service_account
from googleapiclient.discovery import build

SCOPES = ['https://www.googleapis.com/auth/spreadsheets']
SPREADSHEET_ID = '1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'

CREDENTIALS_PATH = ('./gsheet/creds.json')


def _get_sheets_service_client():
  creds = service_account.Credentials.from_service_account_file(
      CREDENTIALS_PATH, scopes=SCOPES)
  service = build('sheets', 'v4', credentials=creds)
  return service


def write_to_google_sheet(worksheet: str, data,
    spreadsheet_id=SPREADSHEET_ID) -> None:
  """Calls the API to update the values of a sheet.

  Args:
    worksheet: string, name of the worksheet to be edited appended by a "!"
    data: list of tuples/lists, data to be added to the worksheet

  Raises:
    HttpError: For any Google Sheets API call related errors
  """
  sheets_client = _get_sheets_service_client()

  # Getting the index of the last occupied row in the sheet
  spreadsheet_response = sheets_client.spreadsheets().values().get(
      spreadsheetId=spreadsheet_id,
      range='{}!A1:A'.format(worksheet)).execute()
  entries = len(spreadsheet_response['values'])

  # Clearing the occupied rows
  request = sheets_client.spreadsheets().values().clear(
      spreadsheetId=spreadsheet_id,
      range='{}!A2:{}'.format(worksheet, entries + 1),
      body={}).execute()

  # Appending new rows
  sheets_client.spreadsheets().values().update(
      spreadsheetId=spreadsheet_id,
      valueInputOption='USER_ENTERED',
      body={
          'majorDimension': 'ROWS',
          'values': data
      },
      range='{}!A2'.format(worksheet)).execute()
