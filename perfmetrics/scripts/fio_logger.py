import os
from datetime import date

files = os.listdir('gcs/fio_logs')
files.sort()

for file in files[9:]:
	# Logging only last 10 fio output files
	# Hence remove older files.
	os.remove(file)

os.system('cp output.json gcs/fio_logs/output-{}.json'.format(date.today().strftime("%d-%m-%Y")))