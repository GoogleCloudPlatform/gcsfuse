#!/bin/bash
# Script for deleting older log files in the folder
# Usage: ./smart_log_deleter.sh $FOLDER_NAME
num_logs=`ls $1 | wc -w`
echo $num_logs
if [ $num_logs -lt 3 ]
then
        exit 0
fi

logs_list=`ls -tr $1`

for log_file in $logs_list; do
        num_logs=$(expr $num_logs - 1)
        `rm -f $1/$log_file`

        if [ $num_logs -lt 3 ]
        then
                exit 0
        fi
done
