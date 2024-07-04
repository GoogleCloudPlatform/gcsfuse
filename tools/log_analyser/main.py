from parser import log_parser
from outputgenerator import generate_gsheet
from inputreader.user_input import UserInput
user_input_obj = UserInput()
logs = user_input_obj.get_input()
global_data = log_parser.general_parser(logs)
generate_gsheet.main_gsheet_generator(global_data)
