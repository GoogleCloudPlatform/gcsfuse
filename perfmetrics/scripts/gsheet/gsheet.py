from google.oauth2 import service_account
from googleapiclient.discovery import build

SCOPES = ['https://www.googleapis.com/auth/spreadsheets']
SPREADSHEET_ID = '1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'

# cell containing the total number of entries in the sheet
# so that we know where the new entry has to be added
NUM_ENTRIES_CELL = 'P4'
CREDENTIALS_PATH = ('./gsheet/creds.json')


def _get_sheets_service_client():
  creds = service_account.Credentials.from_service_account_file(
      CREDENTIALS_PATH, scopes=SCOPES)
  service = build('sheets', 'v4', credentials=creds)
  return service


def write_to_google_sheet(worksheet: str, data) -> None:
  """Calls the API to update the values of a sheet.

  Args:
    worksheet: string, name of the worksheet to be edited appended by a "!"
      NUM_ENTRIES_CELL in the worksheet should have the total number of entries
      present in the worksheet
    data: list of tuples/lists, data to be added to the worksheet

  Raises:
    HttpError: For any Google Sheets API call related errors
  """
  sheets_client = _get_sheets_service_client()
  spreadsheet_response = sheets_client.spreadsheets().values().get(
      spreadsheetId=SPREADSHEET_ID,
      range='{}{}'.format(worksheet, NUM_ENTRIES_CELL)).execute()
  entries = int(spreadsheet_response.get('values', [])[0][0])

  sheets_client.spreadsheets().values().update(
      spreadsheetId=SPREADSHEET_ID,
      valueInputOption='USER_ENTERED',
      body={
          'majorDimension': 'ROWS',
          'values': data
      },
      range='{}A{}'.format(worksheet, entries+2)).execute()

