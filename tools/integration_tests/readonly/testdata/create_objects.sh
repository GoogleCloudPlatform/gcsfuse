echo "This is from directory Test file a" >> a.txt
gsutil mv a.txt gs://$TEST_BUCKET/Test/
echo "This is from file Test1" >> Test1.txt
gsutil mv Test1.txt gs://$TEST_BUCKET/
echo "This is from directory Test/b file b" >> b.txt
gsutil mv b.txt gs://$TEST_BUCKET/Test/b/
