# Here $1 refers to the testBucket argument
gsutil rm -a gs://$1/**

# If bucket is empty it will throw an CommandException.
if [ $? -eq 1 ]; then
  echo "Bucket is already empty."
  exit 0
fi
