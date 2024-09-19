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

_SCOPES = ['https://www.googleapis.com/auth/spreadsheets']
_DEFAULT_GSHEET_ID = '1UghIdsyarrV1HVNc6lugFZS1jJRumhdiWnPgoEC8Fe4'


def _get_sheets_service_client(serviceAccountKeyFile):
  creds = service_account.Credentials.from_service_account_file(
      serviceAccountKeyFile, scopes=_SCOPES
  )
  service = build('sheets', 'v4', credentials=creds)
  return service


def append_to_gsheet(
    serviceAccountKeyFile: str,
    worksheet: str,
    data: list,
    gsheet_id=_DEFAULT_GSHEET_ID,
) -> None:
  """Calls the API to append the given data at the end of the given worksheet in the given gsheet.

  Args:
    worksheet: string, name of the worksheet to be edited appended by a "!"
    data: list of tuples/lists, data to be added to the worksheet

  Raises:
    HttpError: For any Google Sheets API call related errors
  """
  client = _get_sheets_service_client(serviceAccountKeyFile)

  gsheet_response = (
      client.spreadsheets()
      .values()
      .get(spreadsheetId=gsheet_id, range='{}!A1:A'.format(worksheet))
      .execute()
  )
  # )
  print(gsheet_response)
  entries = 0
  if 'values' in gsheet_response:
    entries = len(gsheet_response['values'])
    print(entries)

    # Clearing the occupied rows
    request = (
        client.spreadsheets()
        .values()
        .clear(
            spreadsheetId=gsheet_id,
            range='{}!A{}'.format(worksheet, entries + 1),
            body={},
        )
        .execute()
    )

  # Appending new rows
  client.spreadsheets().values().update(
      spreadsheetId=gsheet_id,
      valueInputOption='USER_ENTERED',
      body={'majorDimension': 'ROWS', 'values': data},
      range=f'{worksheet}!A{entries+1}',
  ).execute()
