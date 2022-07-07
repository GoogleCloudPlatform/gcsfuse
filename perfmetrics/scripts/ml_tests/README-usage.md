# Commands to run animal_image_recognition_model

Run below commands before executing the script:

* source setup.sh

Run from gcsfuse + install gcsfuse:

*   python3 run_image_recognition_models.py -- {ml_model_path} {req_file_path}
    --install_gcsfuse --data_read_method gcsfuse
    --gcsbucket_data_path {gcsbucket_data_path} {directory_name}

Run from gcsfuse + from source code:

*   python3 run_image_recognition_models.py -- {ml_model_path} {req_file_path}
    --data_read_method gcsfuse --gcsbucket_data_path {gcsbucket_data_path} {directory_name}

Run from disk_data:

*   python3 run_image_recognition_models.py -- {ml_model_path} {req_file_path}
    --data_read_method disk --disk_data_path {disk_data_path} {directory_name}

Run from gcsfuse + from source code + Run from disk_data:

*   python3 run_image_recognition_models.py -- {ml_model_path} {req_file_path}
    --gcsbucket_data_path {gcsbucket_data_path} --disk_data_path
    {disk_data_path} {directory_name}

Run from gcsfuse + install gcsfuse + Run from disk_data:

*   python3 run_image_recognition_models.py -- {ml_model_path} {req_file_path}
    --install_gcsfuse --gcsbucket_data_path {gcsbucket_data_path}
    --disk_data_path {disk_data_path} {directory_name}

The output of the ML run will be stored in the directory_name/output.txt:

* Note down the Start time for the script run.
* Output.txt file will store the Start time, End time,
  and Total running time for the ML model to run for each data reading method.
