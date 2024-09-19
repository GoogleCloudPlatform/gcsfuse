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

from google.oauth2 import service_account
from googleapiclient.discovery import build

SCOPES = ['https://www.googleapis.com/auth/spreadsheets']
_DEFAULT_GSHEET_ID = '1UghIdsyarrV1HVNc6lugFZS1jJRumhdiWnPgoEC8Fe4'


def _default_service_account_key_file(project_id: str) -> str:
  if project_id == 'gcs-fuse-test':
    return '20240919-gcs-fuse-test-bc1a2c0aac45.json'
  elif project_id == 'gcs-fuse-test-ml':
    return '20240919-gcs-fuse-test-ml-d6e0247b2cf1.json'
  else:
    raise Exception(f'Unknown project-id: {project_id}')


def _get_sheets_service_client(serviceAccountKeyFile):
  creds = service_account.Credentials.from_service_account_file(
      serviceAccountKeyFile, scopes=SCOPES
  )
  service = build('sheets', 'v4', credentials=creds)
  return service


def append_to_gsheet(
    project_id: str,
    worksheet: str,
    data: list,
    spreadsheet_id=_DEFAULT_GSHEET_ID,
) -> None:
  """Calls the API to insert the values into the given worksheet of a gsheet.

  Args:
    worksheet: string, name of the worksheet to be edited appended by a "!"
    data: list of tuples/lists, data to be added to the worksheet

  Raises:
    HttpError: For any Google Sheets API call related errors
  """
  sheets_client = _get_sheets_service_client(
      _default_service_account_key_file(project_id)
  )

  # Getting the index of the last occupied row in the sheet
  spreadsheet_response = (
      sheets_client.spreadsheets()
      .values()
      .get(spreadsheetId=spreadsheet_id, range='{}!A1:A'.format(worksheet))
      .execute()
  )
  print(spreadsheet_response)
  entries = 0
  if 'values' in spreadsheet_response:
    entries = len(spreadsheet_response['values'])
    print(entries)

    # Clearing the occupied rows
    request = (
        sheets_client.spreadsheets()
        .values()
        .clear(
            spreadsheetId=spreadsheet_id,
            range='{}!A{}'.format(worksheet, entries + 1),
            body={},
        )
        .execute()
    )

  # Appending new rows
  sheets_client.spreadsheets().values().update(
      spreadsheetId=spreadsheet_id,
      valueInputOption='USER_ENTERED',
      body={'majorDimension': 'ROWS', 'values': data},
      range=f'{worksheet}!A{entries+1}',
  ).execute()
