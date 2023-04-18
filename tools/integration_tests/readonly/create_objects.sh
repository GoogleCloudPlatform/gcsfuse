echo "This is from file a" >> a.txt
gsutil mv a.txt gs://gcsfuse-read-only-test/Test/
echo "This is from file Test1" >> Test1.txt
gsutil mv Test1.txt gs://gcsfuse-read-only-test/
echo "This is from file b" >> b.txt
gsutil mv b.txt gs://gcsfuse-read-only-test/Test/b/