This log analyzer takes log files and a few filters and outputs the analysis.

Install python if not installed already.

Create a python environment using the following steps-


(Replace python3 with python if you don't have python3)

1. Run this command to install python3-venv, `sudo apt install python3-venv`

2. Run `python3 -m venv /path`, replace path with the location you want to create virtual environment (preferably outside the repo, to avoid creating unwanted files)
3. Activate the environment using command `source venv_name/bin/activate`, replace venv_name with the location where you created the environment

(Deactivate the environment using the command `deactivate`, once you finish running the code)



Install numpy using command- `pip install numpy`

Install gspread using command - `pip install gspread`


Run the code using command-
`python3 main.py` ,if python3 is installed,
else use `python main.py`

Enter the name of the directory that contains log files (with absolute paths)
for exp- `/usr/local/google/home/patelvishvesh/tmp/test_dir`

Make sure that directory contains only files and not folders.

You can also give a zip file inside the directory. This zip should contain files only and not folders.


Choose if you want a time window filter (by pressing y/n)

If yes, enter the start and end time (epoch)

Enter the type of logs (gke/gcsfuse)

If chosen gke, enter the format in which logs are (CSV/JSON)

Enter your ldap (to give access of the created sheet)

Exp- `patelvishvesh`


Enter the name and location of the credential file

Exp- `/usr/local/google/home/patelvishvesh/Downloads/credentials.json`

After this a google sheet link will be generated.

