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

import os
import tempfile
from typing import Tuple
from google.oauth2 import service_account
from googleapiclient.discovery import build
from .run_tests_common import run_command

_SCOPES = ['https://www.googleapis.com/auth/spreadsheets']


def _get_sheets_service_client(localServiceAccountKeyFile):
  creds = service_account.Credentials.from_service_account_file(
      localServiceAccountKeyFile, scopes=_SCOPES
  )
  # Alternatively, use from_service_account_info for client-creation,
  # documented at
  # https://google-auth.readthedocs.io/en/master/reference/google.oauth2.service_account.html
  # .
  client = build('sheets', 'v4', credentials=creds)
  return client


def download_gcs_object_locally(gcsObjectUri: str) -> str:
  """Downloads the given gcs file-object locally (collision-file) and returns the full-path of the local-file.

  gcsObjectUri is of the form: gs://<bucket-name>/object-name .
  On failure, returned int will be non-zero.

  Caller has to delete the tempfile after usage if a proper tempfile was
  returned, to avoid disk-leak.
  """
  if not gcsObjectUri.startswith('gs:'):
    raise ValueError(
        f'Passed input {gcsObjectUri} is not proper. Expected:'
        ' gs://<bucket>/<object-name>'
    )
  with tempfile.NamedTemporaryFile(mode='w+b', dir='/tmp', delete=False) as fp:
    returncode = run_command(f'gcloud storage cp {gcsObjectUri} {fp.name}')
    if returncode == 0:
      return fp.name
    else:
      raise f'failed to copy gcs object {gcsObjectUri} to local-file {fp.name}: returncode={returncode}. Deleting tempfile {fp.name}...'
      os.remove(fp.name)


def append_data_to_gsheet(
    serviceAccountKeyFile: str,
    worksheet: str,
    data: dict,
    gsheet_id: str,
    repeat_header: bool = False,
) -> None:
  """Calls the API to append the given data at the end of the given worksheet in the given gsheet.

  If the passed header matches the first row of the file, then the
  header is not inserted again, unless repeat_header is passed as
  True.
  Args:
    serviceAccountKeyFile: Path of a service-account key-file for authentication
      read/write from/to the given gsheet.
    worksheet: string, name of the worksheet to be edited appended by a "!"
    data: Dictionary of {'header': tuple, 'values': list(tuples)}, to be added
      to the worksheet.
    gsheet_id: Unique ID to identify a gsheet.
    repeat_header: Always add the passed header as a new row in the file.

  Raises:
    HttpError: For any Google Sheets API call related errors
  """
  for arg in [serviceAccountKeyFile, worksheet, gsheet_id]:
    if not arg or not arg.strip():
      raise ValueError(
          f"Passed  argument '{serviceAccountKeyFile}' is not proper"
      )

  def _using_local_service_key_file(localServiceAccountKeyFile: str):
    # Open a read-write gsheet client using
    client = _get_sheets_service_client(localServiceAccountKeyFile)

    def _read_from_range(cell_range: str):
      """Returns a list of list of values for the given range in the worksheet."""
      gsheet_response = (
          client.spreadsheets()
          .values()
          .get(spreadsheetId=gsheet_id, range=f'{worksheet}!{cell_range}')
          .execute()
      )
      return gsheet_response['values'] if 'values' in gsheet_response else []

    def _write_at_address(cell_address: str, data):
      """Writes a list of tuple of values at the given cell_cell_address in the worksheet."""
      client.spreadsheets().values().update(
          spreadsheetId=gsheet_id,
          valueInputOption='USER_ENTERED',
          body={'majorDimension': 'ROWS', 'values': data},
          range=f'{worksheet}!{cell_address}',
      ).execute()

    data_in_first_column = _read_from_range('A1:A')
    num_rows = len(data_in_first_column)
    data_in_first_row = _read_from_range('A1:1')
    original_header = tuple(data_in_first_row[0]) if data_in_first_row else ()
    new_header = data['header']

    # Insert header in the file, if needed.
    if (
        not original_header
        or repeat_header
        or not original_header == new_header
    ):
      # Append header after last row.
      _write_at_address(f'A{num_rows+1}', [new_header])
      num_rows = num_rows + 1

    # Append given values after the last row.
    _write_at_address(f'A{num_rows+1}', data['values'])
    num_rows = num_rows + 1

  if serviceAccountKeyFile.startswith('gs://'):
    localServiceAccountKeyFile = download_gcs_object_locally(
        serviceAccountKeyFile
    )
    _using_local_service_key_file(localServiceAccountKeyFile)
    os.remove(localServiceAccountKeyFile)
  else:
    _using_local_service_key_file(serviceAccountKeyFile)


def url(gsheet_id: str) -> str:
  return f'https://docs.google.com/spreadsheets/d/{gsheet_id}'
