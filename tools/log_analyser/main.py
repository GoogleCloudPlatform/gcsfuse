from parser import process
from outputgenerator import generate_csv
from inputreader.user_input import UserInput
user_input_obj = UserInput()
logs = user_input_obj.get_input()
global_data = process.gen_processor(logs)
generate_csv.main_csv_generator(global_data)
